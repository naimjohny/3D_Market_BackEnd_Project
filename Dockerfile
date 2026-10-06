# FORMA³ backend — multi-stage образ: сборка на golang:1.25-alpine,
# рантайм на чистом alpine:3 под нерутовым пользователем.

# --- Стадия сборки -----------------------------------------------------------
FROM golang:1.25-alpine AS build

WORKDIR /src

# Зависимости кешируются отдельным слоем.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO не нужен (modernc.org/sqlite — pure Go): статический бинарник.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# --- Рантайм -----------------------------------------------------------------
FROM alpine:3

# ca-certificates — для исходящих HTTPS (Stripe, Telegram).
RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 app \
    && mkdir -p /data \
    && chown app:app /data

COPY --from=build /out/api /usr/local/bin/api

USER app

# БД и ключ шифрования настроек живут на volume /data.
ENV DB_PATH=/data/forma3d.db
VOLUME /data
EXPOSE 8090

ENTRYPOINT ["api"]
