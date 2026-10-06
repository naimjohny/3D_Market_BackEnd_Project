# FORMA³ Backend — деплой

Образ: см. `Dockerfile` (multi-stage, alpine, нерутовый пользователь, порт 8090).
SQLite и ключ шифрования настроек лежат в `/data` — этому пути нужен persistent volume.

## Env-чеклист прода

При `APP_ENV=production` сервер **не стартует**, пока всё из списка не выполнено:

| Переменная            | Требование                                                        |
|-----------------------|-------------------------------------------------------------------|
| `APP_ENV`             | `production`                                                      |
| `JWT_SECRET`          | обязателен, ≥ 32 байт (`openssl rand -hex 32`)                    |
| `ADMIN_PASSWORD_HASH` | обязателен, bcrypt (`htpasswd -bnBC 10 "" '<пароль>' \| tr -d ':\n'`) |
| `FRONTEND_URL`        | боевой домен, БЕЗ localhost                                       |
| `CORS_ORIGINS`        | боевые домены, БЕЗ localhost                                      |

Остальное по ситуации:

- `TRUSTED_PROXY=true` — если сервер за прокси платформы (Fly/Railway — да);
- `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET` — или задать через админку;
- `STRIPE_CURRENCY` (`usd`/`eur`/`rub`) и `PRICE_FX_RATE` (₽ за единицу валюты);
- `TELEGRAM_BOT_TOKEN` + `TELEGRAM_CHAT_ID` — уведомления о новых заказах;
- `ADMIN_EMAIL`, `PORT`, `DB_PATH` (в образе уже `/data/forma3d.db`).

## Fly.io

```sh
cd server
cp fly.toml.example fly.toml         # поправить app/region/домены
fly launch --no-deploy               # привязать приложение к fly.toml
fly volumes create forma3d_data --size 1   # volume для /data (SQLite + ключ)

fly secrets set \
  JWT_SECRET="$(openssl rand -hex 32)" \
  ADMIN_PASSWORD_HASH='<bcrypt-хеш>' \
  STRIPE_SECRET_KEY='sk_live_...' \
  STRIPE_WEBHOOK_SECRET='whsec_...' \
  TELEGRAM_BOT_TOKEN='...' TELEGRAM_CHAT_ID='...'

fly deploy
fly scale count 1                    # SQLite = один писатель, ровно 1 машина
```

Проверка: `curl https://<app>.fly.dev/api/v1/health`.
Webhook в Stripe Dashboard: `https://<app>.fly.dev/api/v1/stripe/webhook`,
событие `checkout.session.completed`.

## Railway

1. New Project → Deploy from GitHub repo; Root Directory = `server`
   (Railway сам подхватит `server/Dockerfile`).
2. Сервис → Settings → Volumes → Add Volume, Mount Path = `/data`.
3. Variables — по чеклисту выше (`APP_ENV=production`, `TRUSTED_PROXY=true`,
   секреты, боевые `FRONTEND_URL`/`CORS_ORIGINS`). `PORT` Railway задаёт сам —
   сервер слушает `:$PORT` автоматически.
4. Settings → Networking → Generate Domain; проверить `/api/v1/health`
   и прописать домен в Stripe webhook.

Replicas оставить = 1 (SQLite — один писатель).

## После деплоя

- Логин в админку боевым паролем; ключи Stripe можно менять через
  админку без рестарта.
- Тестовый заказ: письмо-уведомление в Telegram (если включено),
  оплата → заказ переходит в `paid` только из `new` и только при
  совпадении суммы сессии с суммой заказа.
