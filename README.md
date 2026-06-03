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

## Настройка Keycloak

### Роли пользователей (RBAC)

Backend (`unified-backend`) читает роль пользователя из JWT-токена Keycloak. Без правильно настроенных ролей запросы к `/api/me` вернут `500` с ошибкой `rbac: db error on user lookup`.

#### 1. Создать роли в Realm

В Keycloak Admin Console → **Realm Roles** → **Create role**:

| Role name | Описание |
|---|---|
| `admin` | Полный доступ: создание/удаление ссылок, управление пользователями |
| `user` | Создание ссылок от своего имени |

#### 2. Назначить роль пользователю

**Users** → выбрать пользователя → **Role mapping** → **Assign role** → выбрать `admin` или `user`.

#### 3. Добавить роли в токен (Role Mapper)

По умолчанию Keycloak **не включает** realm roles в access token. Необходимо добавить mapper:

1. **Clients** → `shlink` → **Client scopes** → `shlink-dedicated`
2. **Add mapper** → **By configuration** → **User Realm Role**
3. Настройки mapper:
   - **Name**: `realm_roles`
   - **Token Claim Name**: `roles` _(должно совпадать с тем, что читает backend)_
   - **Claim JSON Type**: `String`
   - **Add to ID token**: `ON`
   - **Add to access token**: `ON`
   - **Multivalued**: `ON`

> **Проверка**: после логина декодируй access token на [jwt.io](https://jwt.io) — в payload должен быть массив `"roles": ["user"]` или `"roles": ["admin"]`.

#### 4. Миграции БД

При первом запуске `unified-backend` таблица `users` создаётся автоматически через GORM AutoMigrate. Если таблица отсутствует (`relation "users" does not exist`), убедись что:

```bash
# Контейнер postgres-bff поднялся раньше unified-backend
docker compose logs postgres-bff | tail -5
docker compose restart unified-backend
```

---

## Nginx + OAuth2 Proxy

### Передача Cookie в `auth_request`

При использовании `auth_request` nginx **не передаёт** Cookie-заголовок в subrequest по умолчанию. Без явной передачи oauth2-proxy не видит сессионный cookie и возвращает `401`, что вызывает бесконечный цикл редиректов на страницу авторизации.

**Обязательно добавить** `proxy_set_header Cookie $http_cookie;` в `location /oauth2/auth { internal; ... }`:

```nginx
location /oauth2/auth {
    internal;
    proxy_pass http://oauth2-proxy-shlink:4180;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Content-Length "";
    proxy_set_header Cookie $http_cookie;   # обязательно
    proxy_pass_request_body off;
}
```

### `session_cookie_minimal`

Параметр `session_cookie_minimal = true` в конфиге oauth2-proxy записывает в cookie только минимальный набор данных — **без refresh_token**. Если Keycloak выдаёт короткоживущие access token (по умолчанию 5 минут, но может быть настроен меньше), сессия истекает и refresh невозможен.

**Рекомендация**: отключить `session_cookie_minimal` или увеличить `Access Token Lifespan` в Keycloak (Realm Settings → Tokens → Access Token Lifespan, минимум 5 минут).

```cfg
# oauth2-proxy/shlink.cfg
# session_cookie_minimal = true   # закомментировать или удалить
```

### Диагностика цикла редиректов

Симптом: пользователь успешно логинится (`[AuthSuccess]` в логах oauth2-proxy), но тут же снова попадает на `/oauth2/sign_in`.

Проверить последовательность в `docker logs oauth2-proxy-shlink`:

```
[AuthSuccess] ...          # логин прошёл
GET /oauth2/auth → 202     # первый auth_request — ок
GET /oauth2/auth → 401     # второй auth_request — cookie не читается
GET /oauth2/sign_in → 200  # снова на логин
```

Если такая последовательность воспроизводится — отсутствует `proxy_set_header Cookie $http_cookie;`.

---

## Logout

`/oauth2/sign_out` при GET-запросе без активной сессии (например, сессия протухла) уходит в цикл редиректов: oauth2-proxy не может прочитать сессию → редиректит на `/oauth2/sign_in` → браузер снова GET `/oauth2/sign_out`.

**Решение**: logout выполняется через POST с CSRF-токеном, либо через промежуточную страницу.

Добавь в `nginx.conf` для `shlink-create.local` (перед `location /oauth2/`):

```nginx
location = /logout {
    default_type text/html;
    return 200 '<!DOCTYPE html><html><head><meta charset="utf-8"><title>Выход</title></head><body><form id="f" method="POST" action="/oauth2/sign_out"><input type="hidden" name="rd" value="/oauth2/sign_in"></form><script>document.getElementById("f").submit();</script></body></html>';
}
```

В UI используй ссылку `href="/logout"` вместо прямого перехода на `/oauth2/sign_out`.

После POST oauth2-proxy корректно чистит cookie и редиректит на `/oauth2/sign_in` через Keycloak.

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
├── test/          # Unit-тесты
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

---

## Тестирование

Все тесты находятся в `unified-backend/test/` и запускаются без внешних зависимостей (БД не нужна).

```bash
cd unified-backend
go test -v -race ./test/...
```

### `audit_sanitize_test.go` — sanitize чувствительных полей аудита

Проверяет, что функция `sanitizeDetails` (используется в audit-репозитории) корректно удаляет чувствительные ключи перед записью в БД.

| Тест | Что проверяет |
|---|---|
| `TestSanitizeDetails_RemovesSensitiveKeys` | Ключи `shlink_api_key`, `api_key`, `authorization`, `password` удаляются; безопасные поля (`method`, `shortCode`) сохраняются |
| `TestSanitizeDetails_NilInput` | Nil-вход не вызывает панику, возвращает nil |
| `TestSanitizeDetails_EmptyInput` | Пустой map возвращает пустой map |
| `TestSanitizeDetails_CaseInsensitive` | Matching работает case-insensitive: `SHLINK_API_KEY`, `Api_Key` — оба удаляются |

### `rbac_test.go` — middleware ExtractIdentity и RBAC

Проверяет middleware `ExtractIdentity`, который читает заголовки от oauth2-proxy и определяет роль пользователя.

| Тест | Что проверяет |
|---|---|
| `TestExtractIdentity_MissingHeader` | Запрос без `X-Auth-Request-User` → `401 Unauthorized`, handler не вызван |
| `TestExtractIdentity_WithHeader` | Запрос с валидными заголовками → handler вызван |
| `TestExtractIdentity_AdminGroup_Default` | Группа `shlink-admins` → роль `admin` (дефолтные группы) |
| `TestExtractIdentity_AdminGroup_LegacyAdmin` | Группа `admin` (legacy) → роль `admin` |
| `TestExtractIdentity_UserRole_NoAdminGroup` | Группы `developers,readonly` → роль `user` |
| `TestExtractIdentity_UserRole_EmptyGroups` | Пустые группы → роль `user` |
| `TestExtractIdentity_CustomAdminGroup` | `ADMIN_GROUPS=shadmin,superusers`: только эти группы дают `admin`; старые (`shlink-admins`, `admin`) — уже `user` |
| `TestExtractIdentity_CaseInsensitive` | Matching групп case-insensitive: `SHLINK-ADMINS` == `Shlink-Admins` |
| `TestExtractIdentity_FieldsPopulated` | Все поля Identity заполнены корректно: `Sub`, `Email`, `Username`, `Role`, `Groups` |
| `TestUserFromCtx_NilSafe` | `UserFromCtx` на пустом контексте возвращает nil без паники |
| `TestWithUser_RoundTrip` | `WithUser` + `UserFromCtx`: запись/чтение из контекста сохраняет все поля |

### `handler_me_test.go` — обработчик `GET /api/me`

Проверяет, что ответ `/api/me` содержит корректные поля и **никогда не раскрывает** `shlink_api_key`.

| Тест | Что проверяет |
|---|---|
| `TestMeHandler_ReturnsCorrectFields` | Ответ содержит `sub`, `username`, `role`, `hasApiKey=true`, `features`, `permissions`; поля `shlinkApiKey` / `shlink_api_key` / `apiKey` отсутствуют |
| `TestMeHandler_AdminPermissions` | Admin получает `canManageUsers=true`, `canViewAuditLogs=true`; API key по-прежнему не попадает в ответ |
| `TestMeHandler_NoUser_InternalError` | Запрос без user в контексте → `500 Internal Server Error` |

### `service_test.go` — бизнес-логика ShlinkService

Проверяет `EnforceSlugPrefix` (принудительный prefix для коротких ссылок) и `FilterShortURLsByUser`.

| Тест | Что проверяет |
|---|---|
| `TestEnforceSlugPrefix_AdminBypass` | Admin передаёт произвольный slug без prefix — slug не изменяется |
| `TestEnforceSlugPrefix_UserNoPrefix` | Feature включён, у пользователя нет `slug_prefix` → ошибка |
| `TestEnforceSlugPrefix_UserCorrectPrefix` | Slug начинается с `slug_prefix` пользователя → OK |
| `TestEnforceSlugPrefix_UserWrongPrefix` | Slug без нужного prefix → ошибка |
| `TestEnforceSlugPrefix_FeatureDisabled` | `FEATURE_USER_SLUG_PREFIX=false` → slug не трогается независимо от роли |
| `TestEnforceSlugPrefix_UserNilSlug` | Nil slug + prefix → возвращает prefix как slug |
| `TestFilterShortURLsByUser` | Фильтрация: пользователь видит только ссылки со своим prefix |
| `TestFilterShortURLsByUser_AdminGetAll` | Admin видит все ссылки без фильтрации |
| `TestComputePermissions_Admin` | Admin: `CanViewAuditLogs=true`, `CanManageUsers=true` |
| `TestComputePermissions_User` | User: `CanViewAuditLogs=false`, `CanManageUsers=false`, `CanCreateShortURL=true` |
