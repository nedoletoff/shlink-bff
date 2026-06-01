# Shlink BFF — Go Backend + Web UI

> **Ветка `main`** — реализация BFF на Go (`unified-backend`).
> Ветка `feature/python-fastapi-backend` содержит альтернативную реализацию на Python/FastAPI.

## Архитектура

```
Browser → nginx (HTTPS) → oauth2-proxy → unified-backend (Go) → shlink-api
                                                              ↓
                                                       web-ui (React SPA)
```

**Принципы безопасности:**

- `shlink_api_key` хранится только в БД, никогда не попадает в браузер
- `servers.json`, `/rest/`, `shlink-web-client` удалены полностью
- RBAC принудителен на уровне backend, независимо от UI
- Аудит-логи: все операции с sanitize чувствительных полей

---

## Go Backend (unified-backend)

### Стек

| Компонент | Версия | Назначение |
|---|---|---|
| Go | ≥ 1.24 | Runtime |
| Gin | latest | HTTP-роутер |
| GORM | latest | ORM (PostgreSQL) |
| PostgreSQL | 16 | База данных |
| golang-jwt | latest | JWT-валидация |
| zap | latest | Структурированные логи |

### Структура

```
unified-backend/
├── cmd/           # Точка входа
├── internal/
│   ├── config/    # Конфигурация из .env
│   ├── handler/   # HTTP-обработчики
│   ├── middleware/ # Auth, RBAC, logging
│   ├── model/     # GORM-модели
│   └── service/   # Бизнес-логика
├── migrations/    # SQL-миграции
├── Dockerfile
└── go.mod
```

### Локальная разработка

#### 1. Переменные окружения

```bash
cp .env.example .env
# Отредактируйте .env — минимальный набор для локального запуска:
```

```dotenv
DOMAIN_SHORT=localhost
SHLINK_DB_NAME=shlink
SHLINK_DB_USER=shlink
SHLINK_DB_PASSWORD=secret
ADMIN_SHLINK_API_KEY=test-key
KEYCLOAK_HOST=keycloak.example.com
KEYCLOAK_IP=127.0.0.1
OAUTH2_CLIENT_SECRET_SHLINK=test-secret
OAUTH2_COOKIE_SECRET=test-cookie-secret-32bytes!
```

#### 2. Запуск через Docker Compose

```bash
# Скопируйте конфиги nginx и oauth2-proxy
cp nginx/nginx.conf.example nginx/nginx.conf
cp oauth2-proxy/shlink.cfg.example oauth2-proxy/shlink.cfg

docker compose up -d
```

#### 3. Запуск тестов (Go)

```bash
cd unified-backend
go test -v -race ./...
```

### CI/CD

GitHub Actions (`.github/workflows/ci.yml`):

- **test** — `go test -race ./...` с PostgreSQL-сервисом
- **lint** — `golangci-lint`
- **docker** — сборка образа `shlink-bff-go:ci`

Триггер: push/PR в `main`.

### Переменные окружения (полный список)

См. `.env.example` в корне репозитория.
