// Package seed — стартовый каталог: зеркало src/app/data/products.data.ts (10 продуктов).
package seed

import (
	"context"
	"fmt"
	"log/slog"

	"forma3d/server/internal/models"
	"forma3d/server/internal/storage"
)

// Products возвращает стартовые 10 продуктов каталога (копия мок-данных фронта).
func Products() []models.Product {
	return []models.Product{
		{
			ID: "chrome-torus", Key: "CHROME_TORUS", Category: models.CategoryArt, Model: "torus",
			Price: 12_900, Polycount: 48_600, Formats: []string{"GLB", "FBX", "OBJ"},
			Textured: true, Rigged: false, Animated: true,
			Colors: []string{"#c7cdd9", "#6c5ce7", "#ffa502", "#2ed573"}, Accent: "#6c5ce7", Featured: true,
		},
		{
			ID: "nebula-crystal", Key: "NEBULA_CRYSTAL", Category: models.CategoryArt, Model: "crystal",
			Price: 18_900, OldPrice: 24_900, Polycount: 12_400, Formats: []string{"GLB", "FBX", "OBJ"},
			Textured: true, Rigged: false, Animated: false,
			Colors: []string{"#a29bfe", "#74b9ff", "#ff7675", "#ffeaa7"}, Accent: "#a29bfe", Featured: true,
		},
		{
			ID: "liquid-blob", Key: "LIQUID_BLOB", Category: models.CategoryArt, Model: "blob",
			Price: 9_900, Polycount: 65_500, Formats: []string{"GLB", "OBJ"},
			Textured: false, Rigged: false, Animated: true,
			Colors: []string{"#8e9dbd", "#e84393", "#00cec9", "#fdcb6e"}, Accent: "#00cec9", Featured: true,
		},
		{
			ID: "lowpoly-planet", Key: "LOWPOLY_PLANET", Category: models.CategoryEnvironment, Model: "planet",
			Price: 14_900, Polycount: 8_200, Formats: []string{"GLB", "FBX", "OBJ", "STL"},
			Textured: true, Rigged: false, Animated: true,
			Colors: []string{"#55efc4", "#74b9ff", "#ff7675", "#ffeaa7"}, Accent: "#55efc4", Featured: true,
		},
		{
			ID: "cyber-helmet", Key: "CYBER_HELMET", Category: models.CategoryCharacter, Model: "helmet",
			Price: 34_900, Polycount: 96_300, Formats: []string{"GLB", "FBX", "OBJ"},
			Textured: true, Rigged: true, Animated: false,
			Colors: []string{"#2d3436", "#d63031", "#0984e3", "#e9edf5"}, Accent: "#0984e3", Featured: true,
		},
		{
			ID: "bauhaus-column", Key: "BAUHAUS_COLUMN", Category: models.CategoryArchitecture, Model: "column",
			Price: 11_900, Polycount: 21_700, Formats: []string{"GLB", "OBJ", "STL"},
			Textured: false, Rigged: false, Animated: false,
			Colors: []string{"#dfe6e9", "#fab1a0", "#81ecec", "#ffeaa7"}, Accent: "#fab1a0", Featured: false,
		},
		{
			ID: "spiral-sculpt", Key: "SPIRAL_SCULPT", Category: models.CategoryArt, Model: "spiral",
			Price: 16_900, Polycount: 54_100, Formats: []string{"GLB", "FBX", "OBJ"},
			Textured: false, Rigged: false, Animated: true,
			Colors: []string{"#e9edf5", "#6c5ce7", "#00b894", "#e17055"}, Accent: "#e17055", Featured: false,
		},
		{
			ID: "mech-drone", Key: "MECH_DRONE", Category: models.CategoryTech, Model: "drone",
			Price: 29_900, OldPrice: 35_900, Polycount: 74_800, Formats: []string{"GLB", "FBX", "OBJ"},
			Textured: true, Rigged: true, Animated: true,
			Colors: []string{"#636e72", "#fdcb6e", "#d63031", "#00cec9"}, Accent: "#fdcb6e", Featured: true,
		},
		{
			ID: "abstract-wave", Key: "ABSTRACT_WAVE", Category: models.CategoryEnvironment, Model: "wave",
			Price: 8_900, Polycount: 32_800, Formats: []string{"GLB", "OBJ"},
			Textured: false, Rigged: false, Animated: true,
			Colors: []string{"#6c5ce7", "#00cec9", "#ff7675", "#ffeaa7"}, Accent: "#6c5ce7", Featured: false,
		},
		{
			ID: "golden-shell", Key: "GOLDEN_SHELL", Category: models.CategoryArt, Model: "shell",
			Price: 21_900, Polycount: 58_900, Formats: []string{"GLB", "FBX", "OBJ", "STL"},
			Textured: false, Rigged: false, Animated: false,
			Colors: []string{"#f5c56b", "#c7cdd9", "#e17055", "#a29bfe"}, Accent: "#f5c56b", Featured: true,
		},
	}
}

// Ensure сидирует каталог, только если таблица products пуста.
func Ensure(ctx context.Context, repo *storage.ProductRepo, logger *slog.Logger) error {
	count, err := repo.Count(ctx)
	if err != nil {
		return fmt.Errorf("check catalog: %w", err)
	}
	if count > 0 {
		return nil
	}
	products := Products()
	for i := range products {
		if err := repo.Create(ctx, &products[i]); err != nil {
			return fmt.Errorf("seed product %q: %w", products[i].ID, err)
		}
	}
	logger.Info("catalog seeded", "products", len(products))
	return nil
}
