package handlers

import (
	"errors"
	"io"
	"net/http"

	"forma3d/server/internal/models"
	"forma3d/server/internal/services"
)

// stripeSignatureHeader — заголовок подписи Stripe-webhook.
const stripeSignatureHeader = "Stripe-Signature"

// StripeWebhook — POST /stripe/webhook: проверка подписи и отметка оплаты
// по событию checkout.session.completed (metadata.orderId → status=paid).
// Целостность: событие принимается только с payment_status=paid, сумма сессии
// сверяется с суммой заказа из БД, статус переводится только из new.
// На проблемы целостности отвечаем 200, чтобы Stripe не ретраил бесконечно;
// разбор — по логам и вручную в админке.
func (h *Handler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_payload", "cannot read webhook payload")
		return
	}
	checkout, err := h.payments.CheckoutFromWebhook(payload, r.Header.Get(stripeSignatureHeader))
	switch {
	case errors.Is(err, services.ErrWebhookNotConfigured):
		writeError(w, http.StatusServiceUnavailable, "webhook_not_configured", "stripe webhook secret is not configured")
		return
	case err != nil:
		h.logger.Warn("stripe webhook rejected", "error", err)
		writeError(w, http.StatusBadRequest, "invalid_signature", "webhook verification failed")
		return
	}
	if checkout != nil && checkout.OrderID != "" {
		if !checkout.Paid {
			// Сессия завершена без оплаты (например, отложенные методы) — игнор.
			h.logger.Warn("stripe webhook ignored: session is not paid",
				"orderId", checkout.OrderID, "paymentStatus", checkout.PaymentStatus)
			writeJSON(w, http.StatusOK, models.HealthResponse{Status: "ok"})
			return
		}
		switch err := h.orders.MarkPaid(r.Context(), checkout.OrderID, checkout.AmountTotal); {
		case errors.Is(err, services.ErrNotFound):
			h.logger.Warn("stripe webhook: order not found", "orderId", checkout.OrderID)
		case errors.Is(err, services.ErrOrderNotNew):
			h.logger.Info("stripe webhook ignored: order is not in status new", "orderId", checkout.OrderID)
		case errors.Is(err, services.ErrAmountMismatch):
			h.logger.Error("stripe webhook: session amount does not match order total; order NOT marked paid",
				"orderId", checkout.OrderID, "amountTotal", checkout.AmountTotal)
		case err != nil:
			h.writeServiceError(w, r, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, models.HealthResponse{Status: "ok"})
}
