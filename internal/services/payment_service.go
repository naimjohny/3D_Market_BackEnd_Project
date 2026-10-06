package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

// checkoutSessionCompleted — тип события Stripe об успешной оплате сессии.
const checkoutSessionCompleted = "checkout.session.completed"

// minorUnitsPerMajor — минорных единиц в одной единице валюты (копейки/центы).
const minorUnitsPerMajor = 100

// currencyRUB — код рубля: цены каталога в этой валюте не конвертируются.
const currencyRUB = "rub"

// PaymentLineItem — позиция для Stripe Checkout (цена всегда из БД).
type PaymentLineItem struct {
	Name      string
	UnitPrice int
	Qty       int
}

// WebhookCheckout — разобранное webhook-событие checkout.session.completed.
type WebhookCheckout struct {
	// OrderID — id заказа из metadata сессии.
	OrderID string
	// Paid — session.payment_status == "paid" (иначе событие надо игнорировать).
	Paid bool
	// PaymentStatus — исходный payment_status сессии (для лога).
	PaymentStatus string
	// AmountTotal — итог сессии в минорных единицах валюты чекаута.
	AmountTotal int64
}

// PaymentService — обёртка над Stripe Checkout. Ключи читаются из
// SettingsService при каждой операции, поэтому смена ключей через
// админ-API применяется без перезапуска; при пустом ключе — mock-режим.
// Валюта и курс конвертации фиксируются при старте (env STRIPE_CURRENCY,
// PRICE_FX_RATE).
type PaymentService struct {
	settings    *SettingsService
	frontendURL string
	currency    string
	fxRate      float64
}

// NewPaymentService создаёт платёжный сервис поверх настроек Stripe.
// currency — валюта чекаута (usd|eur|rub), fxRate — рублей за единицу валюты.
func NewPaymentService(settings *SettingsService, frontendURL, currency string, fxRate float64) *PaymentService {
	return &PaymentService{settings: settings, frontendURL: frontendURL, currency: currency, fxRate: fxRate}
}

// Enabled сообщает, настроен ли Stripe (иначе чекаут работает в mock-режиме).
func (s *PaymentService) Enabled() bool {
	secretKey, _ := s.settings.GetStripe()
	return secretKey != ""
}

// AmountMinorUnits переводит цену каталога в минорные единицы валюты чекаута.
//
// Цены в БД хранятся в рублях. Формула:
//
//	rub:     amount = price × 100                      (копейки, точно)
//	usd/eur: amount = round(price / fx × 100)          (центы)
//
// где fx = PRICE_FX_RATE — курс в рублях за единицу валюты.
func (s *PaymentService) AmountMinorUnits(price int) int64 {
	if s.currency == currencyRUB {
		return int64(price) * minorUnitsPerMajor
	}
	return int64(math.Round(float64(price) / s.fxRate * minorUnitsPerMajor))
}

// TotalAmountMinor — итог по позициям в минорных единицах валюты чекаута.
// Считается той же формулой, что и строки Checkout-сессии (AmountMinorUnits
// на каждую позицию), поэтому пригоден для сверки amount_total из webhook.
func (s *PaymentService) TotalAmountMinor(items []PaymentLineItem) int64 {
	var total int64
	for _, item := range items {
		total += s.AmountMinorUnits(item.UnitPrice) * int64(item.Qty)
	}
	return total
}

// CreateCheckoutSession создаёт Stripe Checkout Session в валюте s.currency и
// возвращает URL страницы оплаты; orderId кладётся в metadata.
func (s *PaymentService) CreateCheckoutSession(ctx context.Context, orderID string, items []PaymentLineItem) (string, error) {
	secretKey, _ := s.settings.GetStripe()
	if secretKey == "" {
		return "", fmt.Errorf("stripe is not configured")
	}
	lineItems := make([]*stripe.CheckoutSessionCreateLineItemParams, 0, len(items))
	for _, item := range items {
		lineItems = append(lineItems, &stripe.CheckoutSessionCreateLineItemParams{
			Quantity: stripe.Int64(int64(item.Qty)),
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency:   stripe.String(s.currency),
				UnitAmount: stripe.Int64(s.AmountMinorUnits(item.UnitPrice)),
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
					Name: stripe.String(item.Name),
				},
			},
		})
	}
	params := &stripe.CheckoutSessionCreateParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:  lineItems,
		SuccessURL: stripe.String(s.frontendURL + "/?payment=success"),
		CancelURL:  stripe.String(s.frontendURL + "/?payment=cancel"),
	}
	params.AddMetadata("orderId", orderID)

	session, err := stripe.NewClient(secretKey).V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	return session.URL, nil
}

// CheckoutFromWebhook проверяет подпись webhook и возвращает данные сессии
// из события checkout.session.completed; для остальных событий — nil, nil.
// Если webhook-секрет не задан, возвращает ErrWebhookNotConfigured.
func (s *PaymentService) CheckoutFromWebhook(payload []byte, signatureHeader string) (*WebhookCheckout, error) {
	_, webhookSecret := s.settings.GetStripe()
	if webhookSecret == "" {
		return nil, ErrWebhookNotConfigured
	}
	event, err := webhook.ConstructEvent(payload, signatureHeader, webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("verify webhook signature: %w", err)
	}
	if string(event.Type) != checkoutSessionCompleted {
		return nil, nil
	}
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return nil, fmt.Errorf("decode checkout session: %w", err)
	}
	return &WebhookCheckout{
		OrderID:       session.Metadata["orderId"],
		Paid:          session.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid,
		PaymentStatus: string(session.PaymentStatus),
		AmountTotal:   session.AmountTotal,
	}, nil
}
