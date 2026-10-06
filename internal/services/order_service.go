package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	"forma3d/server/internal/models"
	"forma3d/server/internal/storage"
)

// Константы калькулятора цены (зеркало src/app/services/order.service.ts).
const (
	roundStep      = 100
	rushWeeksHard  = 1
	rushWeeksSoft  = 2
	rushFactorHard = 1.5
	rushFactorSoft = 1.2
	bulkCount      = 5
	bulkFactor     = 0.85
)

// Лимиты валидации заявок.
const (
	maxNameLen     = 200
	maxEmailLen    = 254
	maxTextLen     = 2000
	maxFileNameLen = 255
	minOrderCount  = 1
	maxOrderCount  = 50
	minWeeks       = 1
	maxWeeks       = 12
	maxCartItems   = 50
	minQty         = 1
	maxQty         = 99
)

// emailRe — серверная проверка формата email.
var emailRe = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)

// basePrice — базовая цена категории (зеркало BASE_PRICE фронта).
func basePrice(category models.ProductCategory) float64 {
	switch category {
	case models.CategoryArt:
		return 9_000
	case models.CategoryTech:
		return 15_000
	case models.CategoryCharacter:
		return 30_000
	case models.CategoryEnvironment:
		return 18_000
	case models.CategoryArchitecture:
		return 14_000
	}
	return 0
}

// detailFactor — множитель детализации (зеркало DETAIL_FACTOR фронта).
func detailFactor(detail models.DetailLevel) float64 {
	switch detail {
	case models.DetailLow:
		return 1
	case models.DetailMid:
		return 1.8
	case models.DetailHigh:
		return 3.2
	}
	return 0
}

// rushFactor — надбавка за срочность по неделям дедлайна.
func rushFactor(weeks int) float64 {
	if weeks <= rushWeeksHard {
		return rushFactorHard
	}
	if weeks <= rushWeeksSoft {
		return rushFactorSoft
	}
	return 1
}

// OrderService — заявки на кастом, чекаут корзины и админ-операции с заказами.
type OrderService struct {
	orders   *storage.OrderRepo
	products *storage.ProductRepo
	payments *PaymentService
	notify   *NotifyService
}

// NewOrderService создаёт сервис заказов.
func NewOrderService(orders *storage.OrderRepo, products *storage.ProductRepo, payments *PaymentService, notify *NotifyService) *OrderService {
	return &OrderService{orders: orders, products: products, payments: payments, notify: notify}
}

// Estimate считает стоимость кастомной модели по формуле фронта.
func Estimate(category models.ProductCategory, detail models.DetailLevel, count, deadlineWeeks int) int {
	bulk := 1.0
	if count >= bulkCount {
		bulk = bulkFactor
	}
	raw := basePrice(category) * detailFactor(detail) * float64(count) * rushFactor(deadlineWeeks) * bulk
	return int(math.Round(raw/roundStep)) * roundStep
}

// SubmitCustom валидирует заявку на кастом, считает estimate и сохраняет заказ.
func (s *OrderService) SubmitCustom(ctx context.Context, req *models.OrderRequest) (*models.OrderResponse, error) {
	if err := validateOrderRequest(req); err != nil {
		return nil, err
	}
	id, err := newOrderID()
	if err != nil {
		return nil, err
	}
	order := &models.Order{
		ID:            id,
		Kind:          models.OrderKindCustom,
		Name:          req.Name,
		Email:         req.Email,
		Contact:       req.Contact,
		Message:       req.Message,
		Category:      req.Category,
		Detail:        req.Detail,
		Count:         req.Count,
		DeadlineWeeks: req.DeadlineWeeks,
		Estimate:      Estimate(req.Category, req.Detail, req.Count, req.DeadlineWeeks),
		FileName:      req.FileName,
		Status:        models.OrderStatusNew,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.orders.Create(ctx, order); err != nil {
		return nil, err
	}
	s.notify.OrderCreated(order)
	return &models.OrderResponse{ID: order.ID, Status: "accepted"}, nil
}

// Checkout валидирует корзину, считает total по ценам ИЗ БД и сохраняет заказ.
// При настроенном Stripe создаёт Checkout Session и возвращает payUrl.
func (s *OrderService) Checkout(ctx context.Context, req *models.CheckoutRequest) (*models.OrderResponse, error) {
	if err := validateCheckoutRequest(req); err != nil {
		return nil, err
	}
	total := 0
	lineItems := make([]PaymentLineItem, 0, len(req.Items))
	for _, item := range req.Items {
		product, err := s.products.GetByID(ctx, item.ProductID)
		if errors.Is(err, storage.ErrNotFound) {
			return nil, invalid("unknown productId: " + item.ProductID)
		}
		if err != nil {
			return nil, err
		}
		total += product.Price * item.Qty
		lineItems = append(lineItems, PaymentLineItem{Name: product.Key, UnitPrice: product.Price, Qty: item.Qty})
	}

	id, err := newOrderID()
	if err != nil {
		return nil, err
	}
	order := &models.Order{
		ID:        id,
		Kind:      models.OrderKindCheckout,
		Name:      req.Name,
		Email:     req.Email,
		Items:     req.Items,
		Total:     total,
		Status:    models.OrderStatusNew,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.orders.Create(ctx, order); err != nil {
		return nil, err
	}
	s.notify.OrderCreated(order)

	resp := &models.OrderResponse{ID: order.ID, Status: "accepted"}
	if s.payments.Enabled() {
		payURL, err := s.payments.CreateCheckoutSession(ctx, order.ID, lineItems)
		if err != nil {
			return nil, fmt.Errorf("create payment session: %w", err)
		}
		resp.PayURL = payURL
	}
	return resp, nil
}

// MarkPaid помечает заказ оплаченным по событию Stripe-webhook.
// Инварианты целостности:
//   - заказ существует (иначе ErrNotFound);
//   - amountTotal сессии совпадает с суммой заказа, пересчитанной по ценам
//     из БД (иначе ErrAmountMismatch — заказ не помечается);
//   - статус переводится ТОЛЬКО из new (иначе ErrOrderNotNew — cancelled/done
//     не перетираются); переход new→paid атомарен на уровне SQL.
func (s *OrderService) MarkPaid(ctx context.Context, orderID string, amountTotal int64) error {
	order, err := s.orders.GetByID(ctx, orderID)
	if errors.Is(err, storage.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if order.Status != models.OrderStatusNew {
		return ErrOrderNotNew
	}
	expected, err := s.expectedAmountMinor(ctx, order)
	if err != nil {
		return err
	}
	if expected != amountTotal {
		return ErrAmountMismatch
	}
	updated, err := s.orders.UpdateStatusFrom(ctx, orderID, models.OrderStatusNew, models.OrderStatusPaid)
	if err != nil {
		return err
	}
	if !updated {
		return ErrOrderNotNew
	}
	return nil
}

// expectedAmountMinor восстанавливает итог Checkout-сессии заказа той же
// формулой, что и при её создании: позиции заказа по текущим ценам из БД,
// каждая конвертируется через PaymentService.AmountMinorUnits.
// Если товар удалили или цену изменили между чекаутом и оплатой, сверка
// не сойдётся (ErrAmountMismatch) — такой платёж отмечается вручную в админке.
func (s *OrderService) expectedAmountMinor(ctx context.Context, order *models.Order) (int64, error) {
	items := make([]PaymentLineItem, 0, len(order.Items))
	for _, item := range order.Items {
		product, err := s.products.GetByID(ctx, item.ProductID)
		if errors.Is(err, storage.ErrNotFound) {
			return 0, ErrAmountMismatch
		}
		if err != nil {
			return 0, err
		}
		items = append(items, PaymentLineItem{Name: product.Key, UnitPrice: product.Price, Qty: item.Qty})
	}
	return s.payments.TotalAmountMinor(items), nil
}

// ListAll возвращает все заказы для админки, новые сверху.
func (s *OrderService) ListAll(ctx context.Context) ([]models.Order, error) {
	return s.orders.List(ctx)
}

// UpdateStatus меняет статус заказа и возвращает обновлённый заказ.
func (s *OrderService) UpdateStatus(ctx context.Context, id string, status models.OrderStatus) (*models.Order, error) {
	if !models.ValidOrderStatus(status) {
		return nil, invalid("unknown status: allowed new, paid, in_progress, done, cancelled")
	}
	err := s.orders.UpdateStatus(ctx, id, status)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	order, err := s.orders.GetByID(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return order, err
}

// Stats возвращает сводку по заказам для дашборда.
func (s *OrderService) Stats(ctx context.Context) (*models.AdminStats, error) {
	return s.orders.Stats(ctx)
}

// validateEmail проверяет формат и длину email.
func validateEmail(email string) error {
	if email == "" || len(email) > maxEmailLen || !emailRe.MatchString(email) {
		return invalid("email is invalid")
	}
	return nil
}

// validateOrderRequest проверяет все поля заявки на кастом.
func validateOrderRequest(req *models.OrderRequest) error {
	if req.Name == "" || len(req.Name) > maxNameLen {
		return invalid("name is required and must be at most 200 characters")
	}
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	if len(req.Contact) > maxTextLen {
		return invalid("contact must be at most 2000 characters")
	}
	if len(req.Message) > maxTextLen {
		return invalid("message must be at most 2000 characters")
	}
	if !models.ValidProductCategory(req.Category) {
		return invalid("unknown category")
	}
	if !models.ValidDetailLevel(req.Detail) {
		return invalid("unknown detail: allowed low, mid, high")
	}
	if req.Count < minOrderCount || req.Count > maxOrderCount {
		return invalid("count must be within 1..50")
	}
	if req.DeadlineWeeks < minWeeks || req.DeadlineWeeks > maxWeeks {
		return invalid("deadlineWeeks must be within 1..12")
	}
	if req.FileName != nil && (*req.FileName == "" || len(*req.FileName) > maxFileNameLen) {
		return invalid("fileName must be null or a non-empty string up to 255 characters")
	}
	return nil
}

// validateCheckoutRequest проверяет все поля чекаута корзины.
func validateCheckoutRequest(req *models.CheckoutRequest) error {
	if req.Name == "" || len(req.Name) > maxNameLen {
		return invalid("name is required and must be at most 200 characters")
	}
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	if len(req.Items) == 0 || len(req.Items) > maxCartItems {
		return invalid("items must contain 1..50 entries")
	}
	for _, item := range req.Items {
		if item.ProductID == "" || len(item.ProductID) > maxNameLen {
			return invalid("productId is required")
		}
		if item.Qty < minQty || item.Qty > maxQty {
			return invalid("qty must be within 1..99")
		}
	}
	return nil
}
