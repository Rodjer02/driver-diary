# Дневник смен водителя

Сводка и список поездок водителя за день, добавление поездок с защитой от дублей.

| Часть | Стек |
|---|---|
| `backend/` | Go 1.26, `net/http`, `pgx/v5` |
| `frontend/` | Next.js 16 (App Router), React 19, TypeScript |
| БД | PostgreSQL 17 |

```
браузер ──> Next.js ──/api/*──> Go API ──> PostgreSQL
            (route handler-прокси, BACKEND_URL)
```

Браузер обращается только к Next.js, а тот проксирует `/api/*` в Go. CORS не нужен. Адрес бэкенда читается из `BACKEND_URL` при каждом запросе, поэтому один образ работает в любом окружении.

## Быстрый старт (Docker)

```bash
docker compose up -d --build
docker compose exec backend /seed /data/trips.json   # загрузить пример данных
```

Откройте http://localhost:3300. Порт можно сменить переменной `PORT=8000 docker compose up -d`.

> В Git Bash на Windows перед `exec` задайте `MSYS_NO_PATHCONV=1`, иначе путь `/seed` будет переписан.

## Локальная разработка

```bash
# БД
docker run -d --name diary-db -e POSTGRES_USER=diary -e POSTGRES_PASSWORD=diary -e POSTGRES_DB=diary \
  -p 55432:5432 postgres:17-alpine

# API
cd backend
export DATABASE_URL="postgres://diary:diary@127.0.0.1:55432/diary?sslmode=disable"
go run ./cmd/seed ../data/trips.json
go run ./cmd/server                      # :8080

# фронтенд
cd frontend
cp .env.example .env.local               # BACKEND_URL=http://localhost:8080
npm install && npm run dev
```

### Тесты

```bash
cd backend
TEST_DATABASE_URL="postgres://diary:diary@127.0.0.1:55432/diary?sslmode=disable" go test ./...
```

- `internal/trip` — расчёт сводки, валидация, отпечаток поездки. БД не нужна.
- `internal/store` — защита от дублей на настоящем Postgres: повтор, тот же id, то же содержимое под другим id, 20 параллельных отправок, границы дня.
- `internal/api` — HTTP-коды и ответы.

Каждый тест создаёт свою схему в БД и удаляет её после себя. Без `TEST_DATABASE_URL` тесты с БД пропускаются.

## Переменные окружения

| Где | Переменная | По умолчанию | Назначение |
|---|---|---|---|
| backend | `DATABASE_URL` | — | строка подключения PostgreSQL |
| backend | `DRIVER_TZ` | `+05:00` | пояс водителя: в нём определяются границы дня |
| backend | `PORT` | `8080` | порт HTTP |
| backend | `CORS_ORIGIN` | пусто | разрешённый Origin, если браузер обращается к API напрямую |
| frontend | `BACKEND_URL` | `http://localhost:8080` | адрес Go API |

## API

| Метод | Путь | Ответ |
|---|---|---|
| GET | `/api/days` | `{"days": ["2026-10-01", …], "tz": "+05:00"}` |
| GET | `/api/day?date=YYYY-MM-DD` | `{"date", "summary", "trips"}` |
| GET | `/api/trips?date=YYYY-MM-DD` | `{"date", "trips"}` |
| GET | `/api/summary?date=YYYY-MM-DD` | `{"date", "summary"}` |
| POST | `/api/trips` | `201` создана · `200` уже была, дубль не создан · `409` id занят другими данными · `422` ошибки валидации |
| GET | `/healthz` | `200` |

Поля сводки: `trips`, `revenue`, `commission`, `net` (выручка − комиссия, «на руки»), `cash` и `card` вида `{count, amount}`.

## Правила

- **Валидация:** `amount > 0`; `end > start` с учётом часовых поясов; `payment` — `cash` или `card`; `0 ≤ commission ≤ amount`; время в формате RFC 3339 с поясом. Ответ 422 перечисляет все ошибки сразу. Те же ограничения продублированы в БД через `CHECK`.
- **Дубли:** поездка считается дублем, если совпадает `id` или содержимое (моменты начала и окончания, сумма, оплата, комиссия). Это гарантирует сама БД: `PRIMARY KEY (id)`, `UNIQUE (fingerprint)` и `INSERT … ON CONFLICT DO NOTHING`, поэтому дубль не появится и при параллельных запросах. Если `id` не передан, сервер вычисляет его из содержимого.
- **Границы дня:** поездка относится к дню своего **начала** в поясе водителя. Поездка 23:50–00:20 попадает в предыдущий день. Время в ответах отдаётся в поясе водителя.
