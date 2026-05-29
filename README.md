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
cp .env.example .env && vi .env

# 2. Скопируйте конфиги и отредактируйте под свой домен
cp nginx/nginx.conf.example nginx/nginx.conf
cp oauth2-proxy/shlink.cfg.example oauth2-proxy/shlink.cfg

# 3. Положите SSL-сертификат (cert + key в одном PEM)
mkdir -p nginx/ssl
# скопируйте cert.pem в nginx/ssl/cert.pem

# 4. Запустите
docker compose up -d --build

# 5. Создайте первого пользователя в БД
docker compose exec postgres psql -U shlink -d shlink -c "
  INSERT INTO users (sub, username, email, role, shlink_api_key)
  VALUES ('keycloak-sub-here', 'admin', 'admin@example.local', 'admin', 'shlink-api-key-here');
"
```

## Запуск тестов

```bash
cd unified-backend
go test ./... -v
```

## API контракт

| Method | Path | Auth | Описание |
|--------|------|------|----------|
| GET | /healthz | — | Healthcheck |
| GET | /api/me | user/admin | Профиль (без API key) |
| GET | /api/dashboard | user/admin | Статистика |
| GET | /api/shlink/short-urls | user/admin | Список ссылок |
| POST | /api/shlink/short-urls | user/admin | Создать ссылку |
| PATCH | /api/shlink/short-urls/{shortCode} | user/admin | Обновить ссылку |
| DELETE | /api/shlink/short-urls/{shortCode} | user/admin | Удалить ссылку |
| GET | /api/shlink/tags | user/admin | Теги |
| POST | /api/shlink/tags | user/admin | Создать тег |
| PUT | /api/shlink/tags/{tagId} | user/admin | Переименовать тег |
| DELETE | /api/shlink/tags/{tagId} | user/admin | Удалить тег |
| GET | /api/admin/users | **admin** | Список пользователей |
| GET | /api/admin/users/{sub} | **admin** | Пользователь |
| PUT | /api/admin/users/{sub} | **admin** | Обновить пользователя |
| PUT | /api/admin/users/{sub}/apikey | **admin** | Обновить API key |
| PUT | /api/admin/users/{sub}/prefix | **admin** | Обновить prefix |
| GET | /api/admin/users/{sub}/links | **admin** | Ссылки пользователя |
| GET | /api/admin/logs | **admin** | Журнал аудита |

## Версии образов

| Образ | Версия | Примечание |
|-------|--------|------------|
| nginx | 1.30-alpine | Stable branch (1.28+ LTS) |
| postgres | 17-alpine | Minor updates безопасны; major→18 требует миграции |
| oauth2-proxy | v7.15.2 | Актуальный стабильный релиз |
| shlink | 4.5.2 | Последний стабильный релиз |
| golang | 1.24-alpine | Builder only, в финальном образе не присутствует |
| node | 22-alpine | Builder only для web-ui |
