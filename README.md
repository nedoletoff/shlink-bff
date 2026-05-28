# Shlink BFF — Unified Backend + Web UI

## Архитектура

```
Browser → nginx (HTTPS) → oauth2-proxy → unified-backend (Go) → shlink-api
                                       → web-ui (React SPA)
```

**Принципы безопасности:**
- `shlink_api_key` хранится только в PostgreSQL, никогда не попадает в браузер
- `servers.json`, `/rest/`, `shlink-web-client` удалены полностью
- RBAC принудителен на уровне backend, независимо от UI
- Аудит-логи: все операции с sanitize чувствительных полей

## Быстрый старт

```bash
# 1. Скопируйте .env и заполните секреты
cp .env .env.local && vi .env.local

# 2. Положите SSL-сертификат в nginx/ssl/haproxy.pem

# 3. Запустите
docker compose up -d

# 4. Создайте первого пользователя в БД
docker compose exec postgres psql -U shlink -d shlink -c "
  INSERT INTO users (sub, username, email, role, shlink_api_key)
  VALUES ('keycloak-sub-here', 'admin', 'admin@example.local', 'admin', 'shlink-api-key-here');
"
```

## Запуск тестов

```bash
cd unified-backend
go test ./test/... -v
```

## API контракт

| Method | Path | Auth | Описание |
|--------|------|------|----------|
| GET | /healthz | — | Healthcheck |
| GET | /api/me | user/admin | Профиль (без API key) |
| GET | /api/dashboard | user/admin | Статистика |
| GET | /api/shlink/short-urls | user/admin | Список ссылок |
| POST | /api/shlink/short-urls | user/admin | Создать ссылку |
| PATCH | /api/shlink/short-urls/{code} | user/admin | Обновить ссылку |
| DELETE | /api/shlink/short-urls/{code} | user/admin | Удалить ссылку |
| GET | /api/shlink/tags | user/admin | Теги |
| POST | /api/shlink/tags | user/admin | Создать тег |
| PUT | /api/shlink/tags/{id} | user/admin | Переименовать тег |
| DELETE | /api/shlink/tags/{id} | user/admin | Удалить тег |
| GET | /api/admin/users | **admin** | Список пользователей |
| GET | /api/admin/users/{sub} | **admin** | Пользователь |
| PUT | /api/admin/users/{sub} | **admin** | Обновить пользователя |
| PUT | /api/admin/users/{sub}/apikey | **admin** | Обновить API key |
| PUT | /api/admin/users/{sub}/prefix | **admin** | Обновить prefix |
| GET | /api/admin/users/{sub}/links | **admin** | Ссылки пользователя |
| GET | /api/admin/logs | **admin** | Журнал аудита |

## Версии образов

| Образ | Версия | Примечание |
|-------|--------|-----------|
| nginx | 1.27-alpine | Stable branch |
| postgres | 17-alpine | Minor updates безопасны; major→18 требует миграции |
| oauth2-proxy | v7.15.2 | Обновлено (auth-bypass fix) |
| privatebin | 2.0.4 | Актуальный релиз май 2026 |
| shlink | 4.4.3 | Зафиксировать после проверки upgrade path |
| golang | 1.24-alpine | Builder only, в финальном образе не присутствует |

## Структура проекта

```
shlink-bff/
├── docker-compose.yml
├── .env                         # секреты (не коммитить!)
├── nginx/
│   └── nginx.conf               # без /rest, без /servers.json
├── oauth2-proxy/
│   ├── shlink.cfg
│   └── pb.cfg
├── unified-backend/             # Go 1.24+
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── handler/             # me, dashboard, shlink_proxy, admin
│   │   ├── middleware/          # identity, rbac, logging, userctx
│   │   ├── repository/postgres/
│   │   ├── service/
│   │   └── shlink/client.go
│   ├── migrations/
│   │   └── 001_init_schema.sql
│   ├── test/                    # unit-тесты
│   └── Dockerfile
└── web-ui/                      # React + TS + Vite + Mantine
    ├── src/
    │   ├── api/client.ts
    │   ├── contexts/AuthContext.tsx
    │   ├── pages/
    │   │   ├── Dashboard.tsx
    │   │   ├── ShortUrls.tsx
    │   │   ├── Tags.tsx
    │   │   └── admin/{Users,AuditLogs}.tsx
    │   └── types/api.ts
    └── Dockerfile
```
