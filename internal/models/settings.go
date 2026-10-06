package models

// Режимы ключа Stripe, определяемые по его префиксу.
const (
	// StripeModeTest — тестовый ключ (sk_test_ / rk_test_).
	StripeModeTest = "test"
	// StripeModeLive — боевой ключ (sk_live_ / rk_live_).
	StripeModeLive = "live"
)

// StripeSettingsStatus — статус настроек Stripe для админки;
// сами ключи никогда не возвращаются.
type StripeSettingsStatus struct {
	Configured        bool    `json:"configured"`
	Mode              *string `json:"mode"`
	SecretKeyLast4    *string `json:"secretKeyLast4"`
	WebhookConfigured bool    `json:"webhookConfigured"`
	AccountName       *string `json:"accountName"`
}

// StripeSettingsUpdateRequest — запрос сохранения ключей Stripe в админке.
type StripeSettingsUpdateRequest struct {
	SecretKey     string `json:"secretKey"`
	WebhookSecret string `json:"webhookSecret"`
}
