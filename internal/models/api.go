package models

import "time"

// APIErrorBody — тело единого формата ошибки API.
type APIErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIError — единый конверт ошибки API: {"error":{"code","message"}}.
type APIError struct {
	Error APIErrorBody `json:"error"`
}

// NewAPIError создаёт конверт ошибки с кодом и безопасным сообщением.
func NewAPIError(code, message string) APIError {
	return APIError{Error: APIErrorBody{Code: code, Message: message}}
}

// HealthResponse — ответ health-чека.
type HealthResponse struct {
	Status string `json:"status"`
}

// LoginRequest — запрос логина администратора.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse — выданный JWT и срок его действия.
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// OrderStatusUpdateRequest — смена статуса заказа в админке.
type OrderStatusUpdateRequest struct {
	Status OrderStatus `json:"status"`
}

// AdminStats — сводка по заказам для дашборда админки.
type AdminStats struct {
	OrdersTotal   int `json:"ordersTotal"`
	OrdersNew     int `json:"ordersNew"`
	RevenuePaid   int `json:"revenuePaid"`
	CustomCount   int `json:"customCount"`
	CheckoutCount int `json:"checkoutCount"`
}
