// Package models содержит все доменные структуры и DTO бэкенда
// (зеркало src/app/models фронта; JSON-теги camelCase 1-в-1).
package models

// ProductCategory — категория продукта каталога (зеркало ProductCategoryType).
type ProductCategory string

// Категории каталога.
const (
	CategoryArt          ProductCategory = "art"
	CategoryTech         ProductCategory = "tech"
	CategoryCharacter    ProductCategory = "character"
	CategoryEnvironment  ProductCategory = "environment"
	CategoryArchitecture ProductCategory = "architecture"
)

// ValidProductCategory сообщает, входит ли значение в enum категорий.
func ValidProductCategory(v ProductCategory) bool {
	switch v {
	case CategoryArt, CategoryTech, CategoryCharacter, CategoryEnvironment, CategoryArchitecture:
		return true
	}
	return false
}

// ValidModelFormat сообщает, входит ли формат файла в enum (зеркало ModelFormatType).
func ValidModelFormat(v string) bool {
	switch v {
	case "GLB", "FBX", "OBJ", "STL":
		return true
	}
	return false
}

// Product — продукт каталога (зеркало ProductInterface).
type Product struct {
	ID        string          `json:"id"`
	Key       string          `json:"key"`
	Category  ProductCategory `json:"category"`
	Model     string          `json:"model"`
	Price     int             `json:"price"`
	OldPrice  int             `json:"oldPrice,omitempty"`
	Polycount int             `json:"polycount"`
	Formats   []string        `json:"formats"`
	Textured  bool            `json:"textured"`
	Rigged    bool            `json:"rigged"`
	Animated  bool            `json:"animated"`
	Colors    []string        `json:"colors"`
	Accent    string          `json:"accent"`
	Featured  bool            `json:"featured"`
}
