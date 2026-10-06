package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"forma3d/server/internal/models"
)

// productColumns — общий список колонок products для SELECT.
const productColumns = `id, "key", category, model, price, old_price, polycount, formats, textured, rigged, animated, colors, accent, featured`

// ProductRepo — репозиторий каталога продуктов.
type ProductRepo struct {
	db *sql.DB
}

// List возвращает продукты, опционально фильтруя по категории.
func (r *ProductRepo) List(ctx context.Context, category models.ProductCategory) ([]models.Product, error) {
	query := `SELECT ` + productColumns + ` FROM products`
	args := make([]any, 0, 1)
	if category != "" {
		query += ` WHERE category = ?`
		args = append(args, string(category))
	}
	query += ` ORDER BY rowid`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := make([]models.Product, 0)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return products, nil
}

// GetByID возвращает продукт по id или ErrNotFound.
func (r *ProductRepo) GetByID(ctx context.Context, id string) (*models.Product, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+productColumns+` FROM products WHERE id = ?`, id)
	p, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

// Count возвращает количество продуктов в каталоге.
func (r *ProductRepo) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM products`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count products: %w", err)
	}
	return n, nil
}

// Create вставляет новый продукт.
func (r *ProductRepo) Create(ctx context.Context, p *models.Product) error {
	formats, colors, err := marshalProductJSON(p)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO products (id, "key", category, model, price, old_price, polycount, formats, textured, rigged, animated, colors, accent, featured)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Key, string(p.Category), p.Model, p.Price, p.OldPrice, p.Polycount,
		formats, boolToInt(p.Textured), boolToInt(p.Rigged), boolToInt(p.Animated), colors, p.Accent, boolToInt(p.Featured))
	if err != nil {
		return fmt.Errorf("insert product: %w", err)
	}
	return nil
}

// Update перезаписывает продукт по id; ErrNotFound, если его нет.
func (r *ProductRepo) Update(ctx context.Context, p *models.Product) error {
	formats, colors, err := marshalProductJSON(p)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE products SET "key" = ?, category = ?, model = ?, price = ?, old_price = ?, polycount = ?,
		 formats = ?, textured = ?, rigged = ?, animated = ?, colors = ?, accent = ?, featured = ? WHERE id = ?`,
		p.Key, string(p.Category), p.Model, p.Price, p.OldPrice, p.Polycount,
		formats, boolToInt(p.Textured), boolToInt(p.Rigged), boolToInt(p.Animated), colors, p.Accent, boolToInt(p.Featured), p.ID)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	return requireAffected(res)
}

// Delete удаляет продукт по id; ErrNotFound, если его нет.
func (r *ProductRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	return requireAffected(res)
}

// rowScanner — общий интерфейс *sql.Row и *sql.Rows для scanProduct.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanProduct читает строку products и распаковывает JSON-колонки.
func scanProduct(row rowScanner) (*models.Product, error) {
	var (
		p                          models.Product
		formatsRaw, colorsRaw      string
		textured, rigged, animated int
		featured                   int
	)
	if err := row.Scan(&p.ID, &p.Key, &p.Category, &p.Model, &p.Price, &p.OldPrice, &p.Polycount,
		&formatsRaw, &textured, &rigged, &animated, &colorsRaw, &p.Accent, &featured); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(formatsRaw), &p.Formats); err != nil {
		return nil, fmt.Errorf("decode formats: %w", err)
	}
	if err := json.Unmarshal([]byte(colorsRaw), &p.Colors); err != nil {
		return nil, fmt.Errorf("decode colors: %w", err)
	}
	if p.Formats == nil {
		p.Formats = []string{}
	}
	if p.Colors == nil {
		p.Colors = []string{}
	}
	p.Textured, p.Rigged, p.Animated, p.Featured = textured != 0, rigged != 0, animated != 0, featured != 0
	return &p, nil
}

// marshalProductJSON сериализует срезы formats/colors в JSON-текст для БД.
func marshalProductJSON(p *models.Product) (formats, colors string, err error) {
	f, err := json.Marshal(p.Formats)
	if err != nil {
		return "", "", fmt.Errorf("encode formats: %w", err)
	}
	c, err := json.Marshal(p.Colors)
	if err != nil {
		return "", "", fmt.Errorf("encode colors: %w", err)
	}
	return string(f), string(c), nil
}

// boolToInt приводит bool к 0/1 для SQLite.
func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// requireAffected превращает "0 затронутых строк" в ErrNotFound.
func requireAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
