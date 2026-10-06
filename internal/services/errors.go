// Package services — бизнес-логика: каталог, заказы, оплата, авторизация.
// Хендлеры зовут только этот слой; в БД ходит только storage.
package services

import "errors"

// Общие ошибки сервисного слоя; хендлеры мапят их на HTTP-статусы.
var (
	// ErrNotFound — сущность не найдена (→ 404).
	ErrNotFound = errors.New("not found")
	// ErrConflict — сущность с таким id уже существует (→ 409).
	ErrConflict = errors.New("already exists")
	// ErrInvalidCredentials — неверные email/пароль (→ 401).
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrStripeInvalidKey — Stripe отверг секретный ключ (→ 400 stripe_invalid_key).
	ErrStripeInvalidKey = errors.New("stripe rejected the secret key")
	// ErrWebhookNotConfigured — webhook-секрет Stripe не задан (→ 503 webhook_not_configured).
	ErrWebhookNotConfigured = errors.New("stripe webhook secret is not configured")
	// ErrOrderNotNew — заказ уже не в статусе new; webhook игнорируется,
	// чтобы не перетирать cancelled/done/paid.
	ErrOrderNotNew = errors.New("order is not in status new")
	// ErrAmountMismatch — сумма Stripe-сессии не совпала с суммой заказа из БД;
	// заказ оплаченным не помечается.
	ErrAmountMismatch = errors.New("session amount does not match order total")
)

// ValidationError — ошибка валидации входных данных (→ 400 с безопасным текстом).
type ValidationError struct {
	Msg string
}

// Error реализует error.
func (e *ValidationError) Error() string {
	return e.Msg
}

// invalid — шорткат для создания ValidationError.
func invalid(msg string) error {
	return &ValidationError{Msg: msg}
}
