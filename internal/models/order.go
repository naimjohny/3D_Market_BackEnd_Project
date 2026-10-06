package models

import "time"

// DetailLevel — уровень детализации кастомной модели (зеркало DetailLevelType).
type DetailLevel string

// Уровни детализации.
const (
	DetailLow  DetailLevel = "low"
	DetailMid  DetailLevel = "mid"
	DetailHigh DetailLevel = "high"
)

// ValidDetailLevel сообщает, входит ли значение в enum детализации.
func ValidDetailLevel(v DetailLevel) bool {
	switch v {
	case DetailLow, DetailMid, DetailHigh:
		return true
	}
	return false
}

// OrderKind — тип заказа: кастомная заявка или чекаут корзины.
type OrderKind string

// Типы заказов.
const (
	OrderKindCustom   OrderKind = "custom"
	OrderKindCheckout OrderKind = "checkout"
)

// OrderStatus — статус заказа в админке.
type OrderStatus string

// Статусы заказов.
const (
	OrderStatusNew        OrderStatus = "new"
	OrderStatusPaid       OrderStatus = "paid"
	OrderStatusInProgress OrderStatus = "in_progress"
	OrderStatusDone       OrderStatus = "done"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

// ValidOrderStatus сообщает, входит ли значение в enum статусов.
func ValidOrderStatus(v OrderStatus) bool {
	switch v {
	case OrderStatusNew, OrderStatusPaid, OrderStatusInProgress, OrderStatusDone, OrderStatusCancelled:
		return true
	}
	return false
}

// CheckoutItem — позиция корзины (зеркало CheckoutItemInterface).
type CheckoutItem struct {
	ProductID string `json:"productId"`
	Qty       int    `json:"qty"`
}

// OrderRequest — заявка на кастомную модель (зеркало OrderRequestInterface).
type OrderRequest struct {
	Name          string          `json:"name"`
	Email         string          `json:"email"`
	Contact       string          `json:"contact"`
	Message       string          `json:"message"`
	Category      ProductCategory `json:"category"`
	Detail        DetailLevel     `json:"detail"`
	Count         int             `json:"count"`
	DeadlineWeeks int             `json:"deadlineWeeks"`
	FileName      *string         `json:"fileName"`
}

// CheckoutRequest — оформление корзины (зеркало CheckoutRequestInterface).
type CheckoutRequest struct {
	Name  string         `json:"name"`
	Email string         `json:"email"`
	Items []CheckoutItem `json:"items"`
}

// OrderResponse — ответ на создание заявки/чекаута (зеркало OrderResponseInterface).
type OrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	PayURL string `json:"payUrl,omitempty"`
}

// Order — сохранённый заказ; отдаётся в админку как есть.
type Order struct {
	ID            string          `json:"id"`
	Kind          OrderKind       `json:"kind"`
	Name          string          `json:"name"`
	Email         string          `json:"email"`
	Contact       string          `json:"contact,omitempty"`
	Message       string          `json:"message,omitempty"`
	Category      ProductCategory `json:"category,omitempty"`
	Detail        DetailLevel     `json:"detail,omitempty"`
	Count         int             `json:"count,omitempty"`
	DeadlineWeeks int             `json:"deadlineWeeks,omitempty"`
	Estimate      int             `json:"estimate,omitempty"`
	FileName      *string         `json:"fileName,omitempty"`
	Items         []CheckoutItem  `json:"items,omitempty"`
	Total         int             `json:"total,omitempty"`
	Status        OrderStatus     `json:"status"`
	CreatedAt     time.Time       `json:"createdAt"`
}
