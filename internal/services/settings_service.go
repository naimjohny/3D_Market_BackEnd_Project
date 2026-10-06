package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	stripe "github.com/stripe/stripe-go/v82"

	"forma3d/server/internal/models"
	"forma3d/server/internal/services/crypto"
	"forma3d/server/internal/storage"
)

// Ключи таблицы settings для конфигурации Stripe.
const (
	settingStripeSecretKey     = "stripe_secret_key"
	settingStripeWebhookSecret = "stripe_webhook_secret"
	settingStripeAccountName   = "stripe_account_name"
)

// maxStripeKeyLen — верхняя граница длины ключа/секрета Stripe на входе.
const maxStripeKeyLen = 512

// secretKeyLast4Len — сколько последних символов ключа показывать в статусе.
const secretKeyLast4Len = 4

// SettingsService — потокобезопасный кэш настроек поверх репозитория.
// Ключи Stripe читаются с приоритетом: БД → env → пусто (mock-режим);
// в БД значения хранятся зашифрованными (AES-256-GCM).
type SettingsService struct {
	repo             *storage.SettingsRepo
	cipher           *crypto.Cipher
	envSecretKey     string
	envWebhookSecret string

	mu              sync.RWMutex
	dbSecretKey     string
	dbWebhookSecret string
	dbAccountName   string
}

// NewSettingsService создаёт сервис и загружает кэш настроек из БД.
func NewSettingsService(ctx context.Context, repo *storage.SettingsRepo, cipher *crypto.Cipher, envSecretKey, envWebhookSecret string) (*SettingsService, error) {
	s := &SettingsService{
		repo:             repo,
		cipher:           cipher,
		envSecretKey:     envSecretKey,
		envWebhookSecret: envWebhookSecret,
	}
	if err := s.reload(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// reload перечитывает настройки Stripe из БД в кэш.
func (s *SettingsService) reload(ctx context.Context) error {
	secretKey, err := s.readEncrypted(ctx, settingStripeSecretKey)
	if err != nil {
		return err
	}
	webhookSecret, err := s.readEncrypted(ctx, settingStripeWebhookSecret)
	if err != nil {
		return err
	}
	accountName, err := s.repo.Get(ctx, settingStripeAccountName)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	s.mu.Lock()
	s.dbSecretKey, s.dbWebhookSecret, s.dbAccountName = secretKey, webhookSecret, accountName
	s.mu.Unlock()
	return nil
}

// readEncrypted читает и расшифровывает настройку; отсутствие ключа = "".
func (s *SettingsService) readEncrypted(ctx context.Context, key string) (string, error) {
	raw, err := s.repo.Get(ctx, key)
	if errors.Is(err, storage.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	plain, err := s.cipher.Decrypt(raw)
	if err != nil {
		return "", fmt.Errorf("decrypt setting %s: %w", key, err)
	}
	return plain, nil
}

// GetStripe возвращает расшифрованные ключи Stripe с приоритетом
// БД → env → пусто; пустой secretKey означает mock-режим.
func (s *SettingsService) GetStripe() (secretKey, webhookSecret string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	secretKey, webhookSecret = s.dbSecretKey, s.dbWebhookSecret
	if secretKey == "" {
		secretKey = s.envSecretKey
	}
	if webhookSecret == "" {
		webhookSecret = s.envWebhookSecret
	}
	return secretKey, webhookSecret
}

// SetStripe валидирует секретный ключ через Stripe API (GET /v1/account),
// шифрует и сохраняет ключи вместе с именем аккаунта, обновляет кэш
// и возвращает актуальный статус настроек.
func (s *SettingsService) SetStripe(ctx context.Context, secretKey, webhookSecret string) (*models.StripeSettingsStatus, error) {
	secretKey = strings.TrimSpace(secretKey)
	webhookSecret = strings.TrimSpace(webhookSecret)
	if secretKey == "" || len(secretKey) > maxStripeKeyLen {
		return nil, invalid("secretKey is required and must be at most 512 characters")
	}
	if len(webhookSecret) > maxStripeKeyLen {
		return nil, invalid("webhookSecret must be at most 512 characters")
	}
	if stripeKeyMode(secretKey) == "" {
		return nil, invalid("secretKey must start with sk_test_, sk_live_, rk_test_ or rk_live_")
	}

	accountName, err := fetchStripeAccountName(ctx, secretKey)
	if err != nil {
		return nil, err
	}

	encryptedKey, err := s.cipher.Encrypt(secretKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt stripe secret key: %w", err)
	}
	if err := s.repo.Set(ctx, settingStripeSecretKey, encryptedKey); err != nil {
		return nil, err
	}
	if webhookSecret == "" {
		if err := s.repo.Delete(ctx, settingStripeWebhookSecret); err != nil {
			return nil, err
		}
	} else {
		encryptedWebhook, err := s.cipher.Encrypt(webhookSecret)
		if err != nil {
			return nil, fmt.Errorf("encrypt stripe webhook secret: %w", err)
		}
		if err := s.repo.Set(ctx, settingStripeWebhookSecret, encryptedWebhook); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Set(ctx, settingStripeAccountName, accountName); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.dbSecretKey, s.dbWebhookSecret, s.dbAccountName = secretKey, webhookSecret, accountName
	s.mu.Unlock()
	return s.StripeStatus(), nil
}

// ClearStripe удаляет настройки Stripe из БД и кэша;
// env-фолбэк после очистки продолжает работать.
func (s *SettingsService) ClearStripe(ctx context.Context) error {
	for _, key := range []string{settingStripeSecretKey, settingStripeWebhookSecret, settingStripeAccountName} {
		if err := s.repo.Delete(ctx, key); err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.dbSecretKey, s.dbWebhookSecret, s.dbAccountName = "", "", ""
	s.mu.Unlock()
	return nil
}

// StripeStatus собирает статус настроек Stripe для админки без самих ключей.
func (s *SettingsService) StripeStatus() *models.StripeSettingsStatus {
	secretKey, webhookSecret := s.GetStripe()
	s.mu.RLock()
	accountName := s.dbAccountName
	s.mu.RUnlock()

	status := &models.StripeSettingsStatus{
		Configured:        secretKey != "",
		WebhookConfigured: webhookSecret != "",
	}
	if secretKey != "" {
		if mode := stripeKeyMode(secretKey); mode != "" {
			status.Mode = &mode
		}
		last4 := secretKey
		if len(last4) > secretKeyLast4Len {
			last4 = last4[len(last4)-secretKeyLast4Len:]
		}
		status.SecretKeyLast4 = &last4
	}
	if accountName != "" {
		status.AccountName = &accountName
	}
	return status
}

// stripeKeyMode определяет режим ключа Stripe по префиксу;
// пустая строка = неизвестный префикс.
func stripeKeyMode(secretKey string) string {
	switch {
	case strings.HasPrefix(secretKey, "sk_test_"), strings.HasPrefix(secretKey, "rk_test_"):
		return models.StripeModeTest
	case strings.HasPrefix(secretKey, "sk_live_"), strings.HasPrefix(secretKey, "rk_live_"):
		return models.StripeModeLive
	}
	return ""
}

// fetchStripeAccountName проверяет ключ запросом GET /v1/account и
// возвращает имя аккаунта; отказ Stripe = ErrStripeInvalidKey.
func fetchStripeAccountName(ctx context.Context, secretKey string) (string, error) {
	account, err := stripe.NewClient(secretKey).V1Accounts.Retrieve(ctx, nil)
	if err != nil {
		var stripeErr *stripe.Error
		if errors.As(err, &stripeErr) {
			return "", fmt.Errorf("stripe account check: %w", ErrStripeInvalidKey)
		}
		return "", fmt.Errorf("stripe account check: %w", err)
	}
	if account.Settings != nil && account.Settings.Dashboard != nil && account.Settings.Dashboard.DisplayName != "" {
		return account.Settings.Dashboard.DisplayName, nil
	}
	if account.BusinessProfile != nil && account.BusinessProfile.Name != "" {
		return account.BusinessProfile.Name, nil
	}
	return account.Email, nil
}
