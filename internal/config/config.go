// Package config загружает и валидирует конфигурацию из переменных окружения.
// Все значения имеют дев-дефолты; секреты в коде не хранятся (кроме
// fallback-хеша пароля для локальной разработки, о чём пишется предупреждение).
// При APP_ENV=production сервер отказывается стартовать с дев-фолбэками
// (fail-fast вместо предупреждений).
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Режимы работы сервера (env APP_ENV).
const (
	// EnvDev — локальная разработка: дев-фолбэки с предупреждениями.
	EnvDev = "dev"
	// EnvProduction — прод: обязательные секреты, без localhost в URL.
	EnvProduction = "production"
)

// Дефолты для локальной разработки.
const (
	defaultPort        = "8090"
	defaultDBPath      = "./data/forma3d.db"
	defaultCORSOrigins = "http://localhost:3000,http://localhost:5173"
	defaultAdminEmail  = "admin@forma3d.studio"
	defaultFrontendURL = "http://localhost:5173"
	// defaultAdminPasswordHash — bcrypt("admin"); ТОЛЬКО дев-fallback,
	// в проде обязателен env ADMIN_PASSWORD_HASH.
	defaultAdminPasswordHash = "$2a$10$p6iCCN2Fkz49gr7LOoB9muaL4ksqaVgcqOLMGu6EdvMOpTghPuGna"
	minJWTSecretBytes        = 32
	generatedSecretBytes     = 32
	// defaultStripeCurrency — валюта Stripe Checkout по умолчанию.
	defaultStripeCurrency = "usd"
	// defaultPriceFXRate — курс по умолчанию: рублей за единицу валюты.
	defaultPriceFXRate = 90.0
)

// Config — вся конфигурация сервера, собранная при старте.
type Config struct {
	// AppEnv — режим работы: dev или production.
	AppEnv string
	// Addr — адрес прослушивания HTTP-сервера (":<PORT>").
	Addr string
	// DBPath — путь к файлу SQLite.
	DBPath string
	// CORSOrigins — allowlist источников для CORS.
	CORSOrigins []string
	// JWTSecret — секрет подписи JWT (HS256), ≥ 32 байт.
	JWTSecret []byte
	// AdminEmail — email единственного администратора.
	AdminEmail string
	// AdminPasswordHash — bcrypt-хеш пароля администратора.
	AdminPasswordHash string
	// StripeSecretKey — ключ Stripe; пустой = mock-режим оплаты.
	StripeSecretKey string
	// StripeWebhookSecret — секрет подписи Stripe-webhook.
	StripeWebhookSecret string
	// StripeCurrency — валюта Stripe Checkout: usd, eur или rub.
	StripeCurrency string
	// PriceFXRate — курс конвертации цен из БД (₽) в валюту чекаута:
	// рублей за единицу валюты; используется для usd/eur.
	PriceFXRate float64
	// FrontendURL — база для success/cancel-URL чекаута.
	FrontendURL string
	// TrustedProxy — сервер за доверенным обратным прокси: клиентский IP
	// берётся из первого адреса X-Forwarded-For.
	TrustedProxy bool
	// TelegramBotToken — токен Telegram-бота для уведомлений (opt-in).
	TelegramBotToken string
	// TelegramChatID — id чата Telegram для уведомлений (opt-in).
	TelegramChatID string
}

// Load читает env-переменные, подставляет дев-дефолты и валидирует значения.
// При APP_ENV=production вместо дев-фолбэков возвращает ошибку (fail-fast).
func Load(logger *slog.Logger) (*Config, error) {
	appEnv := strings.ToLower(envOr("APP_ENV", EnvDev))
	if appEnv != EnvDev && appEnv != EnvProduction {
		return nil, fmt.Errorf("APP_ENV must be %q or %q, got %q", EnvDev, EnvProduction, appEnv)
	}
	trustedProxy, err := boolEnv("TRUSTED_PROXY", false)
	if err != nil {
		return nil, err
	}
	currency, err := currencyEnv()
	if err != nil {
		return nil, err
	}
	fxRate, err := fxRateEnv()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:              appEnv,
		Addr:                ":" + envOr("PORT", defaultPort),
		DBPath:              envOr("DB_PATH", defaultDBPath),
		CORSOrigins:         splitCSV(envOr("CORS_ORIGINS", defaultCORSOrigins)),
		AdminEmail:          envOr("ADMIN_EMAIL", defaultAdminEmail),
		AdminPasswordHash:   os.Getenv("ADMIN_PASSWORD_HASH"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripeCurrency:      currency,
		PriceFXRate:         fxRate,
		FrontendURL:         strings.TrimRight(envOr("FRONTEND_URL", defaultFrontendURL), "/"),
		TrustedProxy:        trustedProxy,
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:      os.Getenv("TELEGRAM_CHAT_ID"),
	}
	if len(cfg.CORSOrigins) == 0 {
		return nil, fmt.Errorf("CORS_ORIGINS must contain at least one origin")
	}

	secret := os.Getenv("JWT_SECRET")
	if appEnv == EnvProduction {
		if err := validateProduction(cfg, secret); err != nil {
			return nil, err
		}
		cfg.JWTSecret = []byte(secret)
		return cfg, nil
	}
	if err := applyDevFallbacks(cfg, secret, logger); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validateProduction — fail-fast проверки для APP_ENV=production: собирает ВСЕ
// проблемы конфигурации в одну понятную ошибку, чтобы оператор чинил за один заход.
func validateProduction(cfg *Config, jwtSecret string) error {
	var problems []string
	if jwtSecret == "" {
		problems = append(problems, fmt.Sprintf("JWT_SECRET is required (at least %d bytes; generate: openssl rand -hex 32)", minJWTSecretBytes))
	} else if len(jwtSecret) < minJWTSecretBytes {
		problems = append(problems, fmt.Sprintf("JWT_SECRET must be at least %d bytes", minJWTSecretBytes))
	}
	if cfg.AdminPasswordHash == "" {
		problems = append(problems, "ADMIN_PASSWORD_HASH is required (bcrypt hash; generate: htpasswd -bnBC 10 \"\" '<password>' | tr -d ':\\n')")
	}
	if strings.Contains(cfg.FrontendURL, "localhost") {
		problems = append(problems, fmt.Sprintf("FRONTEND_URL must not point to localhost, got %q", cfg.FrontendURL))
	}
	for _, origin := range cfg.CORSOrigins {
		if strings.Contains(origin, "localhost") {
			problems = append(problems, fmt.Sprintf("CORS_ORIGINS must not contain localhost origins, got %q", origin))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("refusing to start with APP_ENV=production: %s", strings.Join(problems, "; "))
}

// applyDevFallbacks — поведение dev-режима: дев-фолбэки секретов с предупреждениями.
func applyDevFallbacks(cfg *Config, jwtSecret string, logger *slog.Logger) error {
	switch {
	case jwtSecret == "":
		generated, err := randomSecret()
		if err != nil {
			return fmt.Errorf("generate jwt secret: %w", err)
		}
		cfg.JWTSecret = generated
		logger.Warn("JWT_SECRET is empty: generated a random dev secret; tokens reset on restart")
	case len(jwtSecret) < minJWTSecretBytes:
		return fmt.Errorf("JWT_SECRET must be at least %d bytes", minJWTSecretBytes)
	default:
		cfg.JWTSecret = []byte(jwtSecret)
	}

	if cfg.AdminPasswordHash == "" {
		cfg.AdminPasswordHash = defaultAdminPasswordHash
		logger.Warn("ADMIN_PASSWORD_HASH is empty: using dev fallback hash (password 'admin'); set it in production")
	}
	return nil
}

// envOr возвращает значение переменной окружения или дефолт, если она пуста.
func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// boolEnv разбирает булеву env-переменную (true/false/1/0) с дефолтом.
func boolEnv(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean (true/false), got %q", key, raw)
	}
	return v, nil
}

// currencyEnv читает STRIPE_CURRENCY и проверяет, что валюта поддерживается.
func currencyEnv() (string, error) {
	c := strings.ToLower(envOr("STRIPE_CURRENCY", defaultStripeCurrency))
	switch c {
	case "usd", "eur", "rub":
		return c, nil
	}
	return "", fmt.Errorf("STRIPE_CURRENCY must be one of usd, eur, rub; got %q", c)
}

// fxRateEnv читает PRICE_FX_RATE (рублей за единицу валюты) и проверяет, что курс положительный.
func fxRateEnv() (float64, error) {
	raw := strings.TrimSpace(os.Getenv("PRICE_FX_RATE"))
	if raw == "" {
		return defaultPriceFXRate, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("PRICE_FX_RATE must be a number (rubles per currency unit), got %q", raw)
	}
	if v <= 0 {
		return 0, fmt.Errorf("PRICE_FX_RATE must be positive, got %v", v)
	}
	return v, nil
}

// splitCSV разбирает csv-строку в срез непустых значений без пробелов.
func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// randomSecret генерирует криптостойкий hex-секрет для дев-режима.
func randomSecret() ([]byte, error) {
	buf := make([]byte, generatedSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return []byte(hex.EncodeToString(buf)), nil
}
