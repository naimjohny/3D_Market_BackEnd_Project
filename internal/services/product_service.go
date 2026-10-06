package services

import (
	"context"
	"errors"
	"fmt"

	"forma3d/server/internal/models"
	"forma3d/server/internal/storage"
)

// Лимиты валидации продуктов.
const (
	maxProductTextLen = 200
	maxProductFormats = 8
	maxProductColors  = 16
	maxColorLen       = 32
)

// ProductService — каталог: публичное чтение и админ-CRUD.
type ProductService struct {
	repo *storage.ProductRepo
}

// NewProductService создаёт сервис каталога.
func NewProductService(repo *storage.ProductRepo) *ProductService {
	return &ProductService{repo: repo}
}

// List возвращает каталог, опционально фильтруя по категории.
func (s *ProductService) List(ctx context.Context, category string) ([]models.Product, error) {
	cat := models.ProductCategory(category)
	if category != "" && !models.ValidProductCategory(cat) {
		return nil, invalid("unknown category")
	}
	return s.repo.List(ctx, cat)
}

// Get возвращает продукт по id.
func (s *ProductService) Get(ctx context.Context, id string) (*models.Product, error) {
	p, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return p, err
}

// Create добавляет продукт; при пустом id генерирует его.
func (s *ProductService) Create(ctx context.Context, p models.Product) (*models.Product, error) {
	if p.ID == "" {
		fragment, err := newIDFragment()
		if err != nil {
			return nil, err
		}
		p.ID = "prd-" + fragment
	}
	if err := validateProduct(&p); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByID(ctx, p.ID); err == nil {
		return nil, ErrConflict
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	if err := s.repo.Create(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Update перезаписывает продукт по id из пути.
func (s *ProductService) Update(ctx context.Context, id string, p models.Product) (*models.Product, error) {
	p.ID = id
	if err := validateProduct(&p); err != nil {
		return nil, err
	}
	err := s.repo.Update(ctx, &p)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Delete удаляет продукт по id.
func (s *ProductService) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// validateProduct проверяет все поля продукта и нормализует nil-срезы.
func validateProduct(p *models.Product) error {
	if p.ID == "" || len(p.ID) > maxProductTextLen {
		return invalid("id is required and must be at most 200 characters")
	}
	if p.Key == "" || len(p.Key) > maxProductTextLen {
		return invalid("key is required and must be at most 200 characters")
	}
	if !models.ValidProductCategory(p.Category) {
		return invalid("unknown category")
	}
	if p.Model == "" || len(p.Model) > maxProductTextLen {
		return invalid("model is required and must be at most 200 characters")
	}
	if p.Price <= 0 {
		return invalid("price must be positive")
	}
	if p.OldPrice < 0 {
		return invalid("oldPrice must not be negative")
	}
	if p.Polycount < 0 {
		return invalid("polycount must not be negative")
	}
	if len(p.Formats) == 0 || len(p.Formats) > maxProductFormats {
		return invalid(fmt.Sprintf("formats must contain 1..%d entries", maxProductFormats))
	}
	for _, f := range p.Formats {
		if !models.ValidModelFormat(f) {
			return invalid("unknown format: allowed GLB, FBX, OBJ, STL")
		}
	}
	if len(p.Colors) == 0 || len(p.Colors) > maxProductColors {
		return invalid(fmt.Sprintf("colors must contain 1..%d entries", maxProductColors))
	}
	for _, c := range p.Colors {
		if c == "" || len(c) > maxColorLen {
			return invalid("each color must be a non-empty string up to 32 characters")
		}
	}
	if p.Accent == "" || len(p.Accent) > maxColorLen {
		return invalid("accent is required and must be at most 32 characters")
	}
	return nil
}
