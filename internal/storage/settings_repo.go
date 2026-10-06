package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SettingsRepo — репозиторий key-value настроек приложения.
type SettingsRepo struct {
	db *sql.DB
}

// Get возвращает значение настройки по ключу или ErrNotFound.
func (r *SettingsRepo) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE "key" = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get setting: %w", err)
	}
	return value, nil
}

// Set сохраняет значение настройки (insert или update).
func (r *SettingsRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO settings ("key", value) VALUES (?, ?) ON CONFLICT("key") DO UPDATE SET value = excluded.value`,
		key, value)
	if err != nil {
		return fmt.Errorf("set setting: %w", err)
	}
	return nil
}

// Delete удаляет настройку; отсутствие ключа ошибкой не считается.
func (r *SettingsRepo) Delete(ctx context.Context, key string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM settings WHERE "key" = ?`, key); err != nil {
		return fmt.Errorf("delete setting: %w", err)
	}
	return nil
}
