# Shlink BFF — Go Backend + Web UI

> **Ветка `main`** — реализация BFF на Go (`unified-backend`).
> Ветка `feature/python-fastapi-backend` содержит альтернативную реализацию на Python/FastAPI.

## Архитектура

```
Browser → nginx (HTTPS) → oauth2-proxy → unified-backend (Go) → shlink-api
                                                              ↓
                                                       web-ui (React SPA)
                                                              ↓
                                                       PostgreSQL (BFF)
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

# PostgreSQL для BFF
DB_USER=bff
DB_PASSWORD=changeme
DB_NAME=shlink_bff

# Начальный admin API key для shlink-api (INITIAL_API_KEY).
# unified-backend его НЕ использует — работает с per-user ключами из БД.
ADMIN_SHLINK_API_KEY=your-admin-shlink-api-key

# Маппинг Keycloak-групп → роли (формат: group=role,...)
# Группы сравниваются case-insensitive.
ROLE_GROUPS=shlink-admins=admin,shlink-users=user

# Имя «суперроли» с доступом к /api/admin/*
ADMIN_ROLE=admin

# Источник истины для роли: keycloak (default) или db
# keycloak — роль перечитывается из групп Keycloak при каждом запросе
# db       — роль берётся из users.role, управляется через admin API
ROLE_SOURCE=keycloak

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

Полный список параметров с комментариями см. в `.env.example`.

### Шаг 3. Настроить Keycloak

#### 3.1. Создать client

1. **Clients** → **Create client**
2. `Client ID`: `shlink-bff` (или любое — укажи то же в `oauth2-proxy/shlink.cfg`)
3. `Client authentication`: ON (confidential)
4. `Valid redirect URIs`: `https://shlink-create.example.com/oauth2/callback`
5. После создания: **Credentials** → скопировать `Client secret` в `.env` → `OAUTH2_CLIENT_SECRET_SHLINK`

#### 3.2. Создать группы и назначить пользователей

Каждая Keycloak-группа маппируется в роль через `ROLE_GROUPS`. Пример для дефолтной конфигурации:

1. **Groups** → **Create group** → имя `shlink-admins`
2. **Groups** → **Create group** → имя `shlink-users`
3. Добавить пользователей в группы:
   **Users** → выбрать пользователя → **Groups** → **Join group**

> Пользователь **обязан** состоять хотя бы в одной группе, присутствующей в `ROLE_GROUPS`,
> иначе бэкенд не сможет определить роль и вернёт `403` на все запросы кроме `/healthz`.

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

## Миграции БД

`unified-backend` применяет SQL-миграции **автоматически при каждом старте** — вручную ничего делать не нужно.

### Как это работает

- SQL-файлы из `unified-backend/internal/migrations/sql/*.sql` встроены в бинарник через `go:embed`
- При старте создаётся таблица `schema_migrations` (если не существует)
- Файлы применяются в алфавитном порядке (`001_...`, `002_...`, ...), уже применённые пропускаются
- Каждая миграция выполняется в отдельной транзакции; при ошибке — откат и `os.Exit(1)` с подробным логом

### Ожидаемый лог успешного старта

```json
{"level":"INFO","msg":"postgres: connected successfully"}
{"level":"INFO","msg":"migration applied","file":"001_init_schema.sql"}
{"level":"INFO","msg":"migration applied","file":"002_open_role_constraint.sql"}
{"level":"INFO","msg":"migration applied","file":"003_role_permissions.sql"}
{"level":"INFO","msg":"permissions cache loaded","roles":2}
{"level":"INFO","msg":"shlink_client: version validated"}
{"level":"INFO","msg":"server starting","port":"8080"}
```

При повторном запуске (миграции уже применены) строки `migration applied` отсутствуют — это норма.

### Если миграция упала

```bash
docker logs unified-backend 2>&1 | grep -E '(ERROR|migration)'
```

Примеры ошибок и их причины:

| Ошибка | Причина | Решение |
|---|---|---|
| `failed to connect to postgres` | postgres-bff не поднялся | `docker compose restart unified-backend` |
| `apply migration 001_...: ...already exists` | Таблица уже есть без `IF NOT EXISTS` | Миграция использует `CREATE TABLE IF NOT EXISTS` — не должно возникать |
| `apply migration NNN_...: syntax error` | Сломан SQL в файле миграции | Исправить файл, пересобрать образ |

### Добавление новой миграции

```bash
# Создать файл с порядковым номером
cat > unified-backend/internal/migrations/sql/004_new_table.sql << 'EOF'
CREATE TABLE IF NOT EXISTS new_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid()
);
EOF

# Пересобрать образ
docker compose build unified-backend
docker compose up -d unified-backend
```

> **Важно:** уже применённые миграции **не изменять** — `schema_migrations` отслеживает их по имени файла. Правки существующего файла не применятся повторно. Для изменений создавайте новый файл.

---

## RBAC, роли и разрешения

`unified-backend` не читает JWT напрямую. Идентичность пользователя приходит уже распакованной из `oauth2-proxy` через HTTP-заголовки, которые nginx пробрасывает из ответа на `auth_request`:

| Заголовок | Содержимое |
|---|---|
| `X-Auth-Request-User` | Уникальный ID пользователя (sub из JWT) |
| `X-Auth-Request-Email` | E-mail |
| `X-Auth-Request-Preferred-Username` | Username |
| `X-Auth-Request-Groups` | Группы Keycloak через запятую: `shlink-admins,developers` |

### Маппинг групп → роли (`ROLE_GROUPS`)

Функция `resolveRole` сравнивает группы из `X-Auth-Request-Groups` с картой `ROLE_GROUPS`:

- Формат: `ROLE_GROUPS=group1=role1,group2=role2,...`
- Сравнение **case-insensitive**
- Если несколько групп совпадают, берётся первая по порядку записи в `ROLE_GROUPS`
- Если ни одна группа не совпала — роль пустая, пользователь получает `403` на все запросы кроме `/healthz`

```dotenv
# Пример: две Keycloak-группы → одна admin-роль
ROLE_GROUPS=shlink-admins=admin,superusers=admin

# Пример: разные группы → разные роли
ROLE_GROUPS=shlink-admins=admin,shlink-users=user,contractors=user
```

> **Устаревший формат `ADMIN_GROUPS`** (только список admin-групп через запятую) по-прежнему поддерживается для обратной совместимости, но выводит предупреждение в логе. Рекомендуется перейти на `ROLE_GROUPS`.

### Источник истины для роли (`ROLE_SOURCE`)

| `ROLE_SOURCE` | Поведение |
|---|---|
| `keycloak` (default) | Роль читается из `X-Auth-Request-Groups` **при каждом запросе**. Изменение групп в Keycloak применяется немедленно без перезапуска. Роль в БД обновляется при каждом входе (upsert). |
| `db` | Роль берётся из `users.role` в БД. Keycloak используется только при первом логине (провизионирование). Последующие изменения групп в Keycloak **не влияют** на роль. Управление ролями — через `PUT /api/admin/users/{sub}`. |

### Гранулярные разрешения (`role_permissions`)

Помимо роли, каждая роль имеет набор гранулярных флагов разрешений, хранящихся в таблице `role_permissions` (создаётся миграцией `003_role_permissions.sql`). Загружаются при старте и кешируются в `PermissionsCache`.

#### Дефолтные права при первом запуске

| Разрешение | `admin` | `user` |
|---|:---:|:---:|
| `canViewOwnLinks` | ✓ | ✓ |
| `canViewAllLinks` | ✓ | — |
| `canCreateLinks` | ✓ | ✓ |
| `canCreateWithCustomSlug` | ✓ | ✓ |
| `canCreateWithoutSlug` | ✓ | ✓ |
| `canEditOwnLinks` | ✓ | ✓ |
| `canEditAllLinks` | ✓ | — |
| `canDeleteOwnLinks` | ✓ | ✓ |
| `canDeleteAllLinks` | ✓ | — |
| `canManageOwnTags` | ✓ | ✓ |
| `canManageAllTags` | ✓ | — |
| `canViewOwnStats` | ✓ | ✓ |
| `canViewAllStats` | ✓ | — |
| `canViewAuditLogs` | ✓ | — |
| `canManageUsers` | ✓ | — |
| `canManageRoles` | ✓ | — |

> `canCreateWithCustomSlug` выдаётся пользователям по умолчанию на уровне permissions.
> Реальное ограничение переехало на уровень фич-флага `FEATURE_USER_CUSTOM_SLUG`
> (см. раздел «Фич-флаги»).

#### Управление разрешениями

Администратор может изменить разрешения для любой роли через API:

```bash
curl -X PUT https://shlink-create.example.com/api/admin/roles/editor/permissions \
  -H 'Content-Type: application/json' \
  -d '{
    "canViewOwnLinks": true,
    "canCreateLinks": true,
    "canEditOwnLinks": true,
    "canDeleteOwnLinks": true,
    "canManageUsers": false
  }'
```

Изменения вступают в силу **немедленно** — `RolesHandler.UpsertRolePermissions` вызывает `cache.Set(p)` после сохранения в БД, без рестарта.

При недоступности БД на старте используются fallback-значения:
- admin-роль → `DefaultAdminPermissions` (все флаги = true)
- любая другая роль → `DefaultUserPermissions` (базовый набор)

#### Взаимодействие с `FEATURE_USER_CUSTOM_SLUG`

`canCreateWithCustomSlug` в `role_permissions` — необходимое, но **не достаточное** условие для кастомного slug у пользователя:

- Если `FEATURE_USER_CUSTOM_SLUG=false` — обычные пользователи не могут задать custom slug вне зависимости от `role_permissions`. Admin не блокируется флагом.
- Если `FEATURE_USER_CUSTOM_SLUG=true` (default) — решает `canCreateWithCustomSlug` в `role_permissions`.

### Предупреждение `active_user: no known keycloak group`

Это сообщение появляется, когда в `X-Auth-Request-Groups` нет ни одной группы из `ROLE_GROUPS`.

**Причины:**

1. **Не настроен Group Membership mapper в Keycloak** — claim `groups` не попадает в токен. Проверьте Шаг 3.3.
2. **Неверный `oidc_groups_claim`** в `oauth2-proxy/shlink.cfg` — должен быть `groups`.
3. **Пользователь не состоит ни в одной группе** из `ROLE_GROUPS` — добавьте в нужную группу.
4. **Переменные `ROLE_GROUPS` и `ADMIN_ROLE` не пробрасываются в контейнер** — убедитесь, что они указаны в блоке `environment` сервиса `unified-backend` в `docker-compose.yml`.

**Диагностика:**

```bash
# Проверить, что ROLE_GROUPS попал в контейнер
docker exec unified-backend env | grep ROLE_GROUPS

# Декодировать токен пользователя
docker logs oauth2-proxy-shlink 2>&1 | grep 'AuthSuccess'

# Проверить заголовки, которые oauth2-proxy отдаёт nginx
docker logs unified-backend 2>&1 | grep 'active_user'
```

Пользователь с пустой ролью получает `403` на все запросы кроме `/healthz`.

### Кастомизация ролей

```dotenv
# .env
# Три группы Keycloak → две роли
ROLE_GROUPS=super-admins=admin,content-managers=editor,viewers=user
ADMIN_ROLE=admin
```

`ROLE_GROUPS` и `ADMIN_ROLE` читаются один раз при старте. После изменения — перезапустить `unified-backend`.

### TLS-сертификаты nginx

nginx ожидает **раздельные** файлы сертификата и ключа:

```bash
# Из combined PEM (cert+key в одном файле) извлекаем раздельно:
openssl x509 -in cert.pem -out nginx/ssl/fullchain.pem   # цепочка сертификатов
openssl pkey -in cert.pem -out nginx/ssl/privkey.pem      # приватный ключ
```

---

## Провижининг per-user API keys (Shlink CLI)

При первом логине пользователя (или при его отсутствии в БД) `unified-backend` автоматически создаёт персональный Shlink API key через CLI. Режим запуска CLI настраивается через `SHLINK_RUNNER_MODE`.

### Режимы

| `SHLINK_RUNNER_MODE` | Способ вызова | Когда использовать |
|---|---|---|
| `docker` (default) | `docker exec <SHLINK_CONTAINER> shlink api-key:generate ...` | Стандартный docker-compose деплой |
| `native` | `<SHLINK_BIN> api-key:generate ...` | Деплой без Docker; shlink установлен как бинарь |

### Переменные

```dotenv
# Режим запуска Shlink CLI
SHLINK_RUNNER_MODE=docker

# Имя контейнера (только для docker-режима)
SHLINK_CONTAINER=shlink-api

# Путь до бинаря (только для native-режима)
SHLINK_BIN=shlink
```

### Требования для docker-режима

`unified-backend` внутри контейнера должен иметь доступ к Docker daemon, чтобы вызывать `docker exec`. Для этого нужно пробросить сокет:

```yaml
# docker-compose.yml
unified-backend:
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
```

> **Важно:** проброс Docker socket даёт контейнеру полный контроль над хостовым Docker. В production рекомендуется использовать [Docker Socket Proxy](https://github.com/Tecnativa/docker-socket-proxy) с ограниченным доступом.

### Требования для native-режима

Бинарь `shlink` должен быть доступен в PATH или по пути, указанному в `SHLINK_BIN`. Убедитесь, что у процесса backend есть права на его выполнение.

---

## Фич-флаги

Все фич-флаги задаются в `.env` и пробрасываются в `unified-backend` через `docker-compose.yml`.

| Переменная | Тип | Default | Описание |
|---|---|---|---|
| `FEATURE_USER_SLUG_PREFIX` | bool | `false` | Принудительный prefix в slug на основе username. При `true` slug каждого пользователя должен начинаться с его `slug_prefix` из БД. |
| `FEATURE_USER_TAG_INTERNAL_ID` | bool | `false` | Использовать внутренний ID как тег для ссылок пользователя (для изоляции без prefix). |
| `FEATURE_USER_CUSTOM_SLUG` | bool | `true` | Разрешить пользователям задавать кастомный slug. При `false` — только admin может использовать custom slug, независимо от `role_permissions`. |
| `SHLINK_SHORT_ID_LENGTH` | int | `0` | Длина short-кода, генерируемого Shlink. `0` — использовать дефолт Shlink (по умолчанию 5 символов). |

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

Параметр `session_cookie_minimal = true` убирает из cookie `refresh_token`. При коротком `Access Token Lifespan` в Keycloak токен истекает, обновить его нельзя — пользователь вылетает из сессии.

**Рекомендации:**
- не включать `session_cookie_minimal`
- или увеличить `Access Token Lifespan` в Keycloak: **Realm Settings → Tokens → Access Token Lifespan** (минимум 5 минут)

### Logout

Добавить в `nginx.conf` (перед `location /oauth2/`):

```nginx
location = /logout {
    default_type text/html;
    return 200 '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><form id="f" method="POST" action="/oauth2/sign_out"><input type="hidden" name="rd" value="/oauth2/sign_in"></form><script>document.getElementById("f").submit();</script></body></html>';
}
```

---

## Go Backend (unified-backend)

### Стек

| Компонент | Версия | Назначение |
|---|---|---|
| Go | ≥ 1.24 | Runtime |
| Chi | v5 | HTTP-роутер |
| pgx | v5 | PostgreSQL-клиент |
| PostgreSQL | 17 | База данных BFF |
| embed.FS | stdlib | Встроенные SQL-миграции |
| slog | stdlib | Структурированные логи |

### Структура

```
unified-backend/
├── cmd/server/         # Точка входа (main.go)
├── internal/
│   ├── config/         # Конфигурация из env
│   ├── domain/         # Типы, permissions
│   ├── handler/        # HTTP-обработчики
│   │   ├── admin.go    # Управление пользователями, аудит, настройки
│   │   ├── roles.go    # Управление ролями и permissions (с инвалидацией кеша)
│   │   └── ...         # me, dashboard, proxy, settings, url_detail
│   ├── middleware/      # Auth, RBAC, logging
│   ├── migrations/
│   │   └── sql/        # SQL-миграции (embed в бинарник)
│   ├── repository/
│   │   └── postgres/   # Репозитории + мигратор
│   ├── service/
│   │   ├── permissions_cache.go  # In-memory кеш прав с OR-семантикой для мультироли
│   │   └── shlink_service.go     # Бизнес-логика Shlink
│   └── shlink/         # Клиент Shlink API
├── test/               # Unit-тесты (бизнес-логика, без БД)
├── Dockerfile
└── go.mod
```

---

## Локальная разработка

### Запуск тестов (Go)

```bash
cd unified-backend
go test -v -race ./...
```

### Lint

```bash
cd unified-backend
golangci-lint run ./...
```

### CI/CD

GitHub Actions (`.github/workflows/ci.yml`):

- **test** — `go test -race ./...` с PostgreSQL-сервисом
- **lint** — `golangci-lint`
- **docker** — сборка и push образа в GHCR

Триггер: push/PR в `main`.

---

## Тестирование

Все тесты запускаются без внешних зависимостей (БД не нужна):

```bash
cd unified-backend
go test -v -race ./...
```

### Структура тестов

```
unified-backend/
├── internal/config/
│   └── config_test.go             # Конфигурация: defaults и overrides
└── test/
    ├── audit_sanitize_test.go     # Sanitize чувствительных полей в аудит-логах
    ├── handler_me_test.go         # GET /api/me: сборка ответа, permissions
    ├── permissions_cache_test.go  # PermissionsCache: Set, GetMerged, fallback
    ├── permissions_test.go        # DefaultAdminPermissions, DefaultUserPermissions
    ├── rbac_test.go               # ExtractIdentity middleware, ROLE_GROUPS маппинг
    └── service_test.go            # ShlinkService: slug prefix, фильтрация ссылок
```

### `permissions_cache_test.go` — кеш разрешений

| Тест | Что проверяет |
|---|---|
| `TestPermissionsCache_SetUpdatesCache` | `Set` немедленно инвалидирует кеш — изменения доступны без рестарта |
| `TestPermissionsCache_SetOverwritesExisting` | Повторный `Set` полностью перезаписывает права роли |
| `TestPermissionsCache_Get_AdminFallback` | Роль отсутствует в БД + это adminRole → `DefaultAdminPermissions` |
| `TestPermissionsCache_Get_UnknownRoleFallback` | Роль отсутствует в БД + не adminRole → `DefaultUserPermissions` (deny elevated) |
| `TestPermissionsCache_GetMerged_OR` | Объединение прав двух ролей: пользователь получает флаги из обоих |
| `TestPermissionsCache_GetMerged_SingleRole` | `GetMerged([role])` эквивалентен `Get(role)` |
| `TestPermissionsCache_GetMerged_Empty` | `GetMerged([])` → deny-all (нулевые права) |
| `TestPermissionsCache_GetAll_Empty` | `GetAll` на пустом кеше возвращает `[]` |
| `TestPermissionsCache_GetAll_AfterLoad` | `GetAll` возвращает все роли, загруженные через `Load` |
| `TestPermissionsCache_GetAll_AfterSet` | `GetAll` учитывает роли, добавленные через `Set` |

### `config_test.go` — конфигурация

| Тест | Что проверяет |
|---|---|
| `TestLoad_ShlinkRunnerDefaults` | Дефолтные значения: `ShlinkRunnerMode=docker`, `ShlinkContainerName=shlink-api`, `ShlinkBin=shlink` |
| `TestLoad_ShlinkRunnerOverrides` | Переопределение через env: `SHLINK_RUNNER_MODE=native`, `SHLINK_CONTAINER`, `SHLINK_BIN`, `ROLE_SOURCE=db`, `FEATURE_USER_CUSTOM_SLUG=false`, `SHLINK_SHORT_ID_LENGTH=12` |

### `service_test.go` — бизнес-логика ShlinkService

| Тест | Что проверяет |
|---|---|
| `TestEnforceSlugPrefix_AdminBypass` | Admin передаёт произвольный slug — slug не изменяется |
| `TestEnforceSlugPrefix_UserNoPrefix` | Feature включён, у пользователя нет `slug_prefix` → ошибка |
| `TestEnforceSlugPrefix_UserCorrectPrefix` | Slug начинается с `slug_prefix` → OK |
| `TestEnforceSlugPrefix_UserWrongPrefix` | Slug без нужного prefix → ошибка |
| `TestEnforceSlugPrefix_FeatureDisabled` | `FEATURE_USER_SLUG_PREFIX=false` → slug не трогается |
| `TestEnforceSlugPrefix_UserCustomSlugFeatureDisabled` | `FEATURE_USER_CUSTOM_SLUG=false` → user получает ошибку |
| `TestEnforceSlugPrefix_AdminIgnoresFeatureFlag` | `FEATURE_USER_CUSTOM_SLUG=false` → admin не блокируется |
| `TestFilterShortURLsByUser` | Пользователь видит только ссылки со своим prefix |
| `TestFilterShortURLsByUser_AdminGetAll` | Admin видит все ссылки |

### `rbac_test.go` — middleware и RBAC

| Тест | Что проверяет |
|---|---|
| `TestExtractIdentity_MissingHeader` | Нет `X-Auth-Request-User` → 401 |
| `TestExtractIdentity_WithHeader` | Валидные заголовки → handler вызван |
| `TestExtractIdentity_AdminGroup_Default` | `shlink-admins` → роль `admin` |
| `TestExtractIdentity_CustomRoleGroups_Admin` | Кастомный `ROLE_GROUPS`: проверка 6 сценариев |
| `TestExtractIdentity_CaseInsensitive` | `SHLINK-ADMINS` (upper) → `admin` |
| `TestExtractIdentity_FirstMatchWins` | Несколько совпавших групп — первая по порядку |
| `TestExtractIdentity_FieldsPopulated` | Sub, Email, Username, Role, KeycloakRole, Groups корректно распарсены |
| `TestClientIP_*` | X-Real-IP, X-Forwarded-For, RemoteAddr fallback |

### `permissions_test.go` — domain defaults

| Тест | Что проверяет |
|---|---|
| `TestDefaultAdminPermissions_AllGranted` | Все 16 флагов = true для admin |
| `TestDefaultUserPermissions_OwnGranted` | Базовые «own» флаги = true для user |
| `TestDefaultUserPermissions_AllDenied` | Elevated флаги = false для user |
| `TestPermissions_AdminHasAllUserCan` | Инвариант: всё что может user — умеет и admin |
| `TestRolePermissions_ZeroValueIsDenyAll` | Нулевое значение = deny-all (нет тихого расширения прав) |
