package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"forma3d/server/internal/models"
)

// orderColumns — общий список колонок orders для SELECT.
const orderColumns = `id, kind, name, email, contact, message, category, detail, count, deadline_weeks, estimate, file_name, items, total, status, created_at`

// OrderRepo — репозиторий заказов (кастомные заявки и чекауты).
type OrderRepo struct {
	db *sql.DB
}

// Create сохраняет новый заказ.
func (r *OrderRepo) Create(ctx context.Context, o *models.Order) error {
	items := o.Items
	if items == nil {
		items = []models.CheckoutItem{}
	}
	itemsRaw, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("encode items: %w", err)
	}
	fileName := sql.NullString{}
	if o.FileName != nil {
		fileName = sql.NullString{String: *o.FileName, Valid: true}
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO orders (id, kind, name, email, contact, message, category, detail, count, deadline_weeks, estimate, file_name, items, total, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.ID, string(o.Kind), o.Name, o.Email, o.Contact, o.Message, string(o.Category), string(o.Detail),
		o.Count, o.DeadlineWeeks, o.Estimate, fileName, string(itemsRaw), o.Total, string(o.Status),
		o.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

// List возвращает все заказы, новые сверху.
func (r *OrderRepo) List(ctx context.Context) ([]models.Order, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+orderColumns+` FROM orders ORDER BY rowid DESC`)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]models.Order, 0)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}
	return orders, nil
}

// GetByID возвращает заказ по id или ErrNotFound.
func (r *OrderRepo) GetByID(ctx context.Context, id string) (*models.Order, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+orderColumns+` FROM orders WHERE id = ?`, id)
	o, err := scanOrder(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	return o, nil
}

// UpdateStatus меняет статус заказа; ErrNotFound, если заказа нет.
func (r *OrderRepo) UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error {
	res, err := r.db.ExecContext(ctx, `UPDATE orders SET status = ? WHERE id = ?`, string(status), id)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return requireAffected(res)
}

// UpdateStatusFrom меняет статус заказа, только если текущий статус равен from
// (атомарно, на уровне SQL). Возвращает false, если заказ не найден или уже
// в другом статусе.
func (r *OrderRepo) UpdateStatusFrom(ctx context.Context, id string, from, to models.OrderStatus) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE orders SET status = ? WHERE id = ? AND status = ?`,
		string(to), id, string(from))
	if err != nil {
		return false, fmt.Errorf("update order status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("update order status: %w", err)
	}
	return n > 0, nil
}

// Stats считает сводку по заказам одним запросом.
func (r *OrderRepo) Stats(ctx context.Context) (*models.AdminStats, error) {
	var s models.AdminStats
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN (CASE WHEN kind = ? THEN total ELSE estimate END) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN kind = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN kind = ? THEN 1 ELSE 0 END), 0)
		FROM orders`,
		string(models.OrderStatusNew), string(models.OrderStatusPaid), string(models.OrderKindCheckout),
		string(models.OrderKindCustom), string(models.OrderKindCheckout),
	).Scan(&s.OrdersTotal, &s.OrdersNew, &s.RevenuePaid, &s.CustomCount, &s.CheckoutCount)
	if err != nil {
		return nil, fmt.Errorf("orders stats: %w", err)
	}
	return &s, nil
}

// scanOrder читает строку orders, распаковывая items и created_at.
func scanOrder(row rowScanner) (*models.Order, error) {
	var (
		o          models.Order
		fileName   sql.NullString
		itemsRaw   string
		createdRaw string
	)
	if err := row.Scan(&o.ID, &o.Kind, &o.Name, &o.Email, &o.Contact, &o.Message, &o.Category, &o.Detail,
		&o.Count, &o.DeadlineWeeks, &o.Estimate, &fileName, &itemsRaw, &o.Total, &o.Status, &createdRaw); err != nil {
		return nil, err
	}
	if fileName.Valid {
		name := fileName.String
		o.FileName = &name
	}
	if err := json.Unmarshal([]byte(itemsRaw), &o.Items); err != nil {
		return nil, fmt.Errorf("decode items: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	o.CreatedAt = createdAt
	return &o, nil
}
