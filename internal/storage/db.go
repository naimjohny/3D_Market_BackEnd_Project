// Package storage — единственный слой доступа к SQLite:
// подключение, миграции и репозитории. SQL только с плейсхолдерами.
package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // регистрация pure-Go драйвера "sqlite"
)

// ErrNotFound возвращается репозиториями, когда сущность отсутствует в БД.
var ErrNotFound = errors.New("entity not found")

// schema — идемпотентные миграции, выполняются при каждом старте.
const schema = `
CREATE TABLE IF NOT EXISTS products (
	id        TEXT PRIMARY KEY,
	"key"     TEXT NOT NULL,
	category  TEXT NOT NULL,
	model     TEXT NOT NULL,
	price     INTEGER NOT NULL,
	old_price INTEGER NOT NULL DEFAULT 0,
	polycount INTEGER NOT NULL,
	formats   TEXT NOT NULL,
	textured  INTEGER NOT NULL,
	rigged    INTEGER NOT NULL,
	animated  INTEGER NOT NULL,
	colors    TEXT NOT NULL,
	accent    TEXT NOT NULL,
	featured  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
	id             TEXT PRIMARY KEY,
	kind           TEXT NOT NULL,
	name           TEXT NOT NULL,
	email          TEXT NOT NULL,
	contact        TEXT NOT NULL DEFAULT '',
	message        TEXT NOT NULL DEFAULT '',
	category       TEXT NOT NULL DEFAULT '',
	detail         TEXT NOT NULL DEFAULT '',
	count          INTEGER NOT NULL DEFAULT 0,
	deadline_weeks INTEGER NOT NULL DEFAULT 0,
	estimate       INTEGER NOT NULL DEFAULT 0,
	file_name      TEXT,
	items          TEXT NOT NULL DEFAULT '[]',
	total          INTEGER NOT NULL DEFAULT 0,
	status         TEXT NOT NULL,
	created_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);

CREATE TABLE IF NOT EXISTS settings (
	"key" TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// Store объединяет подключение к БД и репозитории поверх него.
type Store struct {
	db       *sql.DB
	Products *ProductRepo
	Orders   *OrderRepo
	Settings *SettingsRepo
}

// Open создаёт директорию под файл БД, открывает SQLite и применяет миграции.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// modernc/sqlite: одно соединение исключает SQLITE_BUSY на конкурентной записи.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db, Products: &ProductRepo{db: db}, Orders: &OrderRepo{db: db}, Settings: &SettingsRepo{db: db}}, nil
}

// Close закрывает подключение к БД.
func (s *Store) Close() error {
	return s.db.Close()
}
