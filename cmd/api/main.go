// Команда api — точка входа HTTP-сервера FORMA³: wiring и graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"path/filepath"

	"forma3d/server/internal/config"
	"forma3d/server/internal/httpapi"
	"forma3d/server/internal/httpapi/handlers"
	"forma3d/server/internal/seed"
	"forma3d/server/internal/services"
	"forma3d/server/internal/services/crypto"
	"forma3d/server/internal/storage"
)

// Таймауты HTTP-сервера и graceful shutdown.
const (
	readTimeout     = 10 * time.Second
	writeTimeout    = 15 * time.Second
	idleTimeout     = 60 * time.Second
	shutdownTimeout = 10 * time.Second
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

// run собирает зависимости, запускает сервер и ждёт SIGINT/SIGTERM.
func run(logger *slog.Logger) error {
	cfg, err := config.Load(logger)
	if err != nil {
		return err
	}

	store, err := storage.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := seed.Ensure(context.Background(), store.Products, logger); err != nil {
		return err
	}

	encryptionKey, err := crypto.LoadOrCreateKey(filepath.Dir(cfg.DBPath))
	if err != nil {
		return err
	}
	settingsCipher, err := crypto.New(encryptionKey)
	if err != nil {
		return err
	}
	settings, err := services.NewSettingsService(context.Background(), store.Settings, settingsCipher, cfg.StripeSecretKey, cfg.StripeWebhookSecret)
	if err != nil {
		return err
	}

	payments := services.NewPaymentService(settings, cfg.FrontendURL, cfg.StripeCurrency, cfg.PriceFXRate)
	notify := services.NewNotifyService(cfg.TelegramBotToken, cfg.TelegramChatID, logger)
	products := services.NewProductService(store.Products)
	orders := services.NewOrderService(store.Orders, store.Products, payments, notify)
	auth := services.NewAuthService(cfg.AdminEmail, cfg.AdminPasswordHash, cfg.JWTSecret)

	router := httpapi.New(httpapi.Options{
		Handlers:      handlers.New(products, orders, auth, payments, settings, logger),
		Logger:        logger,
		CORSOrigins:   cfg.CORSOrigins,
		ValidateToken: auth.ValidateToken,
		TrustedProxy:  cfg.TrustedProxy,
	})

	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening",
			"addr", cfg.Addr,
			"env", cfg.AppEnv,
			"stripe", payments.Enabled(),
			"currency", cfg.StripeCurrency,
			"telegram", notify.Enabled(),
			"trustedProxy", cfg.TrustedProxy)
		serverErr <- server.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
