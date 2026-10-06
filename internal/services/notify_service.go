package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"forma3d/server/internal/models"
)

// telegramAPIBase — база Telegram Bot API.
const telegramAPIBase = "https://api.telegram.org"

// telegramTimeout — таймаут HTTP-запроса к Telegram.
const telegramTimeout = 5 * time.Second

// NotifyService — Telegram-уведомления о новых заказах (opt-in).
// Включается, только если заданы ОБА env: TELEGRAM_BOT_TOKEN и
// TELEGRAM_CHAT_ID; иначе все методы — no-op. Отправка идёт в отдельной
// горутине и не влияет на ответ клиенту; ошибки — только в лог, без секретов.
type NotifyService struct {
	botToken string
	chatID   string
	client   *http.Client
	logger   *slog.Logger
}

// NewNotifyService создаёт сервис уведомлений; пустые token/chatID = no-op.
func NewNotifyService(botToken, chatID string, logger *slog.Logger) *NotifyService {
	return &NotifyService{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: telegramTimeout},
		logger:   logger,
	}
}

// Enabled сообщает, настроены ли уведомления (оба env заданы).
func (s *NotifyService) Enabled() bool {
	return s.botToken != "" && s.chatID != ""
}

// OrderCreated шлёт краткую сводку о новом заказе (тип, id, имя, email, сумма)
// в отдельной горутине; вызывается после успешного сохранения заказа.
func (s *NotifyService) OrderCreated(order *models.Order) {
	if !s.Enabled() {
		return
	}
	text := orderSummary(order)
	go s.send(text)
}

// send выполняет POST /bot<token>/sendMessage; ошибки пишет в лог без токена.
func (s *NotifyService) send(text string) {
	body, err := json.Marshal(map[string]string{"chat_id": s.chatID, "text": text})
	if err != nil {
		s.logger.Warn("telegram notify: encode payload failed", "error", err)
		return
	}
	url := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPIBase, s.botToken)
	resp, err := s.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		// В тексте сетевой ошибки присутствует URL с токеном — вырезаем его.
		s.logger.Warn("telegram notify failed", "error", strings.ReplaceAll(err.Error(), s.botToken, "<token>"))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("telegram notify failed", "status", resp.StatusCode)
	}
}

// orderSummary — краткая сводка заказа для уведомления.
func orderSummary(o *models.Order) string {
	amount := o.Total
	if o.Kind == models.OrderKindCustom {
		amount = o.Estimate
	}
	return fmt.Sprintf("Новый заказ (%s)\nID: %s\nИмя: %s\nEmail: %s\nСумма: %d ₽",
		o.Kind, o.ID, o.Name, o.Email, amount)
}
