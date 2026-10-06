# FORMA³ backend — команды разработки.

.PHONY: run build vet test tidy

# Запуск API-сервера (env читается из окружения; см. .env.example).
run:
	go run ./cmd/api

# Сборка всех пакетов.
build:
	go build ./...

# Статический анализ.
vet:
	go vet ./...

# Тесты.
test:
	go test ./...

# Синхронизация зависимостей.
tidy:
	go mod tidy
