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

## Настройка с нуля

### Предусловия

- Docker + docker compose
- Keycloak (локальный контейнер или внешний) с созданным realm
- DNS или `/etc/hosts` для двух доменов:
  - `shlink.example.com` — публичный редиректор ссылок (без auth)
  - `shlink-create.example.com` — защищённый UI и API

### Шаг 1. Клонировать репозиторий и скопировать примеры конфигов

```bash
git clone https://github.com/nedoletoff/shlink-bff.git
cd shlink-bff

cp .env.example .env
cp nginx/nginx.conf.example nginx/nginx.conf
cp oauth2-proxy/shlink.cfg.example oauth2-proxy/shlink.cfg
```

### Шаг 2. Заполнить `.env`

Минимально необходимые переменные:

```dotenv
# Домены
DOMAIN_SHORT=shlink.example.com
DOMAIN_SHORT_CREATE=shlink-create.example.com

# PostgreSQL для BFF (shlink-api в docker-compose использует SQLite, отдельная БД ему не нужна)
DB_USER=bff
DB_PASSWORD=changeme

# Начальный admin API key для shlink-api (INITIAL_API_KEY).
# unified-backend его НЕ использует — работает с per-user ключами из БД (см. #18).
ADMIN_SHLINK_API_KEY=your-admin-shlink-api-key

# Группы Keycloak с ролью admin (дефолт: shlink-admins,admin)
ADMIN_GROUPS=shlink-admins,admin

# Keycloak (extra_hosts в docker compose: имя → IP)
KEYCLOAK_HOST=keycloak.example.com
KEYCLOAK_IP=192.168.1.100

# oauth2-proxy секреты
OAUTH2_CLIENT_SECRET_SHLINK=client-secret-from-keycloak
# Cookie secret — случайная строка 32+ байт, base64:
# python3 -c "import secrets,base64; print(base64.b64encode(secrets.token_bytes(32)).decode())"
OAUTH2_COOKIE_SECRET=your-32-byte-base64-secret
```

> Внутренние переменные backend (`HTTP_ADDR`, `DATABASE_URL`, `SHLINK_INTERNAL_URL`)
> задаются в `docker-compose.yml` напрямую и обычно не требуют правки.

Полный список параметров см. в `.env.example` — он является источником истины.

### Шаг 3. Настроить Keycloak

#### 3.1. Создать client

1. **Clients** → **Create client**
2. `Client ID`: `shlink-bff` (или любое — укажи то же в `oauth2-proxy/shlink.cfg`)
3. `Client authentication`: ON (confidential)
4. `Valid redirect URIs`: `https://shlink-create.example.com/oauth2/callback`
5. После создания: **Credentials** → скопировать `Client secret` в `.env` → `OAUTH2_CLIENT_SECRET_SHLINK`

#### 3.2. Создать группу для admin-доступа

1. **Groups** → **Create group** → имя `shlink-admins`
2. Добавить нужных пользователей в группу:
   **Users** → выбрать пользователя → **Groups** → **Join group** → `shlink-admins`

> Группа `shlink-admins` — дефолтное имя admin-группы в backend.
> При необходимости можно изменить через `ADMIN_GROUPS` в `.env`.

#### 3.3. Добавить mapper для groups в токен

По умолчанию Keycloak **не включает** группы в OIDC-токен. Необходимо добавить mapper:

1. **Clients** → `shlink-bff` → **Client scopes** → `shlink-bff-dedicated`
2. **Add mapper** → **By configuration** → **Group Membership**
3. Настройки mapper:
   - **Name**: `groups`
   - **Token Claim Name**: `groups`
   - **Full group path**: OFF (чтобы был просто `shlink-admins`, а не `/shlink-admins`)
   - **Add to ID token**: ON
   - **Add to access token**: ON

> **Проверка**: после логина декодируй access token на [jwt.io](https://jwt.io) — в payload должен быть массив `"groups": ["shlink-admins"]`.

### Шаг 4. Заполнить конфиг oauth2-proxy

Отредактировать `oauth2-proxy/shlink.cfg`:

```cfg
oidc_issuer_url = "https://keycloak.example.com/realms/YOUR_REALM"
client_id       = "shlink-bff"
redirect_url    = "https://shlink-create.example.com/oauth2/callback"
```

Остальные настройки уже правильно выставлены в примере:
- `set_xauthrequest = true` — включает заголовки `X-Auth-Request-*`
- `oidc_groups_claim = "groups"` — берёт группы из claim `groups` в токене
- `session_cookie_minimal` — не включать (см. ниже)

### Шаг 5. Отредактировать nginx.conf

В `nginx/nginx.conf` заменить домены:

```nginx
server_name shlink-create.example.com;  # ваш домен
server_name shlink.example.com;         # ваш публичный домен
```

Сертификат положить в `nginx/ssl/cert.pem` (cert + key в одном PEM).

### Шаг 6. Запустить

```bash
docker compose up -d
```

### Шаг 7. Проверить

```bash
# Статус контейнеров
docker compose ps

# Логи oauth2-proxy (должны быть AuthSuccess и /oauth2/auth 200)
docker logs oauth2-proxy-shlink -f

# Проверить роль текущего пользователя
curl -sk https://shlink-create.example.com/api/me  # в браузере (после логина)
```

Если пользователь в группе `shlink-admins` — ответ `/api/me` будет содержать `"role": "admin"`.

---

## RBAC и роль admin

`unified-backend` не читает JWT напрямую. Идентичность пользователя приходит уже распакованной из `oauth2-proxy` через HTTP-заголовки, которые nginx пробрасывает из ответа на `auth_request`:

| Заголовок | Содержимое |
|---|---|
| `X-Auth-Request-User` | Уникальный ID пользователя (sub из JWT) |
| `X-Auth-Request-Email` | E-mail |
| `X-Auth-Request-Preferred-Username` | Username |
| `X-Auth-Request-Groups` | Группы Keycloak через запятую: `shlink-admins,developers` |

Функция `resolveRole` в `unified-backend/internal/middleware/identity.go` сравнивает группы пользователя со списком admin-групп:

- список берётся из env `ADMIN_GROUPS` (comma-separated)
- если `ADMIN_GROUPS` не задан, дефолт: `shlink-admins,admin`
- сравнение **case-insensitive**
- если хотя бы одна группа совпадает — роль `admin`, иначе `user`

### Кастомизация admin-групп

```dotenv
# .env
ADMIN_GROUPS=shadmin,superusers
```

`ADMIN_GROUPS` читается один раз при старте и хранится в иммутабельном `config.Config`
(без глобального состояния и гонок данных). После изменения — перезапустить `unified-backend`.

### Управление ролями

Роль назначается в два этапа:

1. **Первый логин (auto-provision).** Если пользователя нет в БД, он создаётся автоматически,
   роль берётся из групп Keycloak (`resolveRole`).
2. **Последующие логины.** Роль в БД — источник истины и **не перезаписывается** из Keycloak
   при каждом входе (см. `UserRepository.Upsert`, #32). Это позволяет админу вручную
   понижать/повышать роль через `PUT /api/admin/users/{sub}` без последующего затирания.

Если требуется, чтобы Keycloak всегда был источником истины для ролей, добавьте
`role = EXCLUDED.role` в `ON CONFLICT` внутри `Upsert`.

### TLS-сертификаты nginx

nginx ожидает **раздельные** файлы сертификата и ключа (#9):

```bash
# Из combined PEM (cert+key в одном файле) извлекаем раздельно:
openssl x509 -in cert.pem -out nginx/ssl/fullchain.pem   # цепочка сертификатов
openssl pkey -in cert.pem -out nginx/ssl/privkey.pem      # приватный ключ
```

---

## Nginx + OAuth2 Proxy

### Как работает auth_request

Каждый запрос к защищённому ресурсу (`/api/`, `/`) проходит через следующую цепочку:

```
browser → nginx → /_oauth2_auth (internal)
                        ↓
               oauth2-proxy /oauth2/auth
                        ↓
             200 OK: запрос идёт дальше
             401:    редирект на /oauth2/sign_in
```

После получения `200` nginx читает из ответа oauth2-proxy заголовки `X-Auth-Request-*` (через `auth_request_set`) и пробрасывает их в `unified-backend`.

### Передача Cookie в auth subrequest

При использовании `auth_request` nginx **по умолчанию не передаёт** Cookie-заголовок во внутренний subrequest. Без явной передачи oauth2-proxy не видит сессионный cookie и возвращает `401`.

Это приводит к бесконечному циклу редиректов, который выглядит так в логах:

```
[AuthSuccess] user@example.com ...  # логин прошёл
GET /oauth2/auth → 202              # первый auth_request — ок
GET /oauth2/auth → 401              # второй — cookie не дошёл
GET /oauth2/sign_in → 200           # снова на логин ← цикл
```

**Обязательно в блоке `/_oauth2_auth`:**

```nginx
location = /_oauth2_auth {
    internal;
    proxy_pass              http://oauth2_proxy_shlink/oauth2/auth;
    proxy_pass_request_body off;
    proxy_set_header        Content-Length    "";
    proxy_set_header        X-Forwarded-Proto $scheme;
    proxy_set_header        Cookie            $http_cookie;  # обязательно
}
```

### session_cookie_minimal

Параметр `session_cookie_minimal = true` в конфиге oauth2-proxy убирает из cookie `refresh_token`. При коротком `Access Token Lifespan` в Keycloak (по умолчанию 5 минут) токен истекает, обновить его нельзя — пользователь вылетает из сессии.

**Рекомендации:**
- не включать `session_cookie_minimal`
- или увеличить `Access Token Lifespan` в Keycloak: **Realm Settings → Tokens → Access Token Lifespan** (минимум 5 минут)

### Logout

`/oauth2/sign_out` с GET-запросом без активной сессии уходит в цикл редиректов. Решение — делать logout через POST:

Добавить в `nginx.conf` (перед `location /oauth2/`):

```nginx
location = /logout {
    default_type text/html;
    return 200 '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><form id="f" method="POST" action="/oauth2/sign_out"><input type="hidden" name="rd" value="/oauth2/sign_in"></form><script>document.getElementById("f").submit();</script></body></html>';
}
```

В UI использовать `href="/logout"` вместо `/oauth2/sign_out`.

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

### Миграции БД

При первом запуске `unified-backend` таблица `users` создаётся автоматически через GORM AutoMigrate. Если таблица отсутствует (`relation "users" does not exist`), убедись что postgres-bff поднялся раньше:

```bash
docker compose logs postgres-bff | tail -5
docker compose restart unified-backend
```

### Переменные окружения (полный список)

См. `.env.example` в корне репозитория.

---

## Локальная разработка

### Запуск тестов (Go)

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

---

## Тестирование

Все тесты находятся в `unified-backend/test/` и запускаются без внешних зависимостей (БД не нужна).

```bash
cd unified-backend
go test -v -race ./test/...
```

### `audit_sanitize_test.go` — sanitize чувствительных полей аудита

Проверяет, что функция `sanitizeDetails` корректно удаляет чувствительные ключи перед записью в БД.

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

Проверяет `EnforceSlugPrefix` и `FilterShortURLsByUser`.

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
