# Shlink BFF — Go Backend + Web UI

> **Ветка `master`** — реализация BFF на Go (`unified-backend`).
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

## Модель ролей

### Концепция

Backend не хранит жёстко закодированный список допустимых имён ролей. Единственная привилегированная роль — `admin` (константа `domain.RoleAdmin`). Любая другая строка является непривилегированной ролью и получает стандартный набор прав. Имена непривилегированных ролей (`"user"`, `"viewer"`, `"editor"`, `"moderator"`, etc.) задаются произвольно и могут быть любыми.

### Как определяется роль

1. oauth2-proxy передаёт заголовок `X-Auth-Request-Groups` со списком групп Keycloak
2. Middleware `ExtractIdentity` сверяет группы с множеством admin-групп (env `ADMIN_GROUPS`)
3. Если хотя бы одна группа совпадает → роль `"admin"`, иначе → роль `"user"` (или любое другое имя — см. ниже)

### Настройка имён ролей

| Переменная окружения | Дефолт | Описание |
|---|---|---|
| `ADMIN_GROUPS` | `shlink-admins,admin` | Comma-separated список групп Keycloak, которым присваивается роль `admin`. Сравнение case-insensitive. |

**Примеры:**

```bash
# Дефолт: группы shlink-admins и admin → роль admin
ADMIN_GROUPS=shlink-admins,admin

# Кастомные группы: только superusers и shadmin → admin
ADMIN_GROUPS=superusers,shadmin

# Пользователи в группах viewer, editor, moderator получат роль "user"
# (имя роли задаётся через поле role в таблице users)
```

### Права по ролям

Права вычисляются функцией `domain.User.ComputePermissions()` по следующим приоритетам:

1. **Кастомное переопределение** — `domain.RolePermissionsOverride[role]`, если задано
2. **Административная роль** — `domain.IsAdminRole(role)` → полный набор прав
3. **Дефолтные права непривилегированной роли** — `domain.DefaultUserPermissions`

| Право | `admin` | Непривилегированная роль (дефолт) |
|---|---|---|
| `canCreateShortUrl` | ✅ | ✅ |
| `canEditOwnLinks` | ✅ | ✅ |
| `canDeleteOwnLinks` | ✅ | ✅ |
| `canManageOwnTags` | ✅ | ✅ |
| `canViewAuditLogs` | ✅ | ❌ |
| `canManageUsers` | ✅ | ❌ |

### Кастомные роли с кастомными правами

Для добавления роли с индивидуальными правами (например, `viewer` без права создавать ссылки) используй `domain.RolePermissionsOverride`:

```go
// main.go или инициализация приложения
domain.RolePermissionsOverride[domain.Role("viewer")] = domain.Permissions{
    CanCreateShortURL: false,
    CanEditOwnLinks:   false,
    CanDeleteOwnLinks: false,
    CanManageOwnTags:  false,
    CanViewAuditLogs:  false,
    CanManageUsers:    false,
}

domain.RolePermissionsOverride[domain.Role("editor")] = domain.Permissions{
    CanCreateShortURL: true,
    CanEditOwnLinks:   true,
    CanDeleteOwnLinks: false,
    CanManageOwnTags:  true,
    CanViewAuditLogs:  false,
    CanManageUsers:    false,
}
```

Переопределение читается при каждом вызове `ComputePermissions()` — без перезапуска сервиса (если обновлять map во время работы).

### Ключевые функции domain

```go
// IsAdminRole — единственная точка проверки "является ли роль администраторской"
func IsAdminRole(r Role) bool

// ComputePermissions — вычисляет Permissions с учётом override и IsAdminRole
func (u *User) ComputePermissions() Permissions

// RolePermissionsOverride — map для кастомных прав конкретных ролей
var RolePermissionsOverride map[Role]Permissions

// DefaultUserPermissions — базовые права непривилегированных ролей
var DefaultUserPermissions Permissions
```

---

## Настройка Keycloak

### 1. Создать группы в Realm

В Keycloak Admin Console → **Groups** → **Create group**:

| Group name | Роль в BFF |
|---|---|
| `shlink-admins` (или любое имя из `ADMIN_GROUPS`) | `admin` |
| Любая другая группа | непривилегированная роль |

> Имена групп могут быть любыми — главное, что они перечислены в `ADMIN_GROUPS`.

### 2. Назначить пользователю группу

**Users** → выбрать пользователя → **Groups** → **Join Group** → выбрать нужную группу.

### 3. Добавить группы в токен (Group Mapper)

По умолчанию Keycloak **не включает** группы в access token. Необходимо добавить mapper:

1. **Clients** → `shlink` → **Client scopes** → `shlink-dedicated`
2. **Add mapper** → **By configuration** → **Group Membership**
3. Настройки mapper:
   - **Name**: `groups`
   - **Token Claim Name**: `groups`
   - **Add to ID token**: `ON`
   - **Add to access token**: `ON`
   - **Full group path**: `OFF`

> **Проверка**: декодируй access token на [jwt.io](https://jwt.io) — в payload должен быть массив `"groups": ["shlink-admins"]`.

### 4. Миграции БД

При первом запуске `unified-backend` таблица `users` создаётся автоматически через GORM AutoMigrate. Если таблица отсутствует (`relation "users" does not exist`):

```bash
docker compose logs postgres-bff | tail -5
docker compose restart unified-backend
```

---

## Logout

`/oauth2/sign_out` при GET-запросе без активной сессии уходит в цикл редиректов.

**Решение**: добавь в `nginx.conf` для `shlink-create.local` (перед `location /oauth2/`):

```nginx
location = /logout {
    default_type text/html;
    return 200 '<!DOCTYPE html><html><head><meta charset="utf-8"><title>Выход</title></head><body><form id="f" method="POST" action="/oauth2/sign_out"><input type="hidden" name="rd" value="/oauth2/sign_in"></form><script>document.getElementById("f").submit();</script></body></html>';
}
```

В UI используй `href="/logout"` вместо прямого перехода на `/oauth2/sign_out`.

---

## Go Backend (unified-backend)

### Стек

| Компонент | Версия | Назначение |
|---|---|---|
| Go | ≥ 1.24 | Runtime |
| Gin | latest | HTTP-роутер |
| GORM | latest | ORM (PostgreSQL) |
| PostgreSQL | 17 | База данных |
| golang-jwt | latest | JWT-валидация |
| slog | stdlib | Структурированные логи |

### Структура

```
unified-backend/
├── cmd/             # Точка входа
├── internal/
│   ├── config/      # Конфигурация из env
│   ├── domain/      # Доменные модели (User, Role, Permissions)
│   ├── handler/     # HTTP-обработчики
│   ├── middleware/  # ExtractIdentity, RequireRole, RequireActiveUser, logging
│   ├── repository/  # PostgreSQL (UserRepository, AuditRepository)
│   └── service/     # Бизнес-логика (ShlinkService)
├── test/            # Интеграционные и unit-тесты
├── Dockerfile
└── go.mod
```

### Локальная разработка

```bash
cp .env.example .env
# Отредактируйте .env под локальное окружение

docker compose up -d
```

### Переменные окружения (полный список)

См. `.env.example` в корне репозитория.

---

## Тестирование

### Запуск

```bash
cd unified-backend
go test -v -race ./...
```

### Покрытие

Все тесты находятся в `unified-backend/test/`, пакет `test`.

---

#### `rbac_test.go` — Middleware: ExtractIdentity, RBAC, context

| Тест | Что проверяет |
|---|---|
| `TestExtractIdentity_MissingHeader` | Нет `X-Auth-Request-User` → 401, handler не вызывается |
| `TestExtractIdentity_WithHeader` | Заголовок есть → handler вызван |
| `TestExtractIdentity_AdminGroup_Default` | Группа `shlink-admins` → роль `admin` (дефолтный `ADMIN_GROUPS`) |
| `TestExtractIdentity_AdminGroup_LegacyAdmin` | Группа `admin` → роль `admin` |
| `TestExtractIdentity_UserRole_NoAdminGroup` | Группы `developers,readonly` → роль `user` |
| `TestExtractIdentity_UserRole_EmptyGroups` | Пустые группы → роль `user` |
| `TestExtractIdentity_CustomAdminGroup` | Кастомный `ADMIN_GROUPS=shadmin,superusers`: `shadmin` → admin; `shlink-admins` → user; `editor`, `viewer` → user |
| `TestExtractIdentity_CaseInsensitive` | Сравнение групп case-insensitive (`SHLINK-ADMINS` == `shlink-admins`) |
| `TestExtractIdentity_FieldsPopulated` | Sub, Email, Username, Role, Groups корректно заполнены в Identity |
| `TestUserFromCtx_NilSafe` | `UserFromCtx` на пустом контексте возвращает nil без паники |
| `TestWithUser_RoundTrip` | `WithUser` + `UserFromCtx`: user сохраняется/читается из контекста |
| `TestWithUser_RoundTrip_CustomRole` | Round-trip для произвольных имён ролей: `viewer`, `editor`, `moderator`, `power-user` |
| `TestIsAdminRole_ViaMiddlewareIdentity` | Middleware возвращает `"user"` для кастомных групп; `IsAdminRole` = false для них |

---

#### `domain_permissions_test.go` — `IsAdminRole`, `ComputePermissions`, overrides

| Тест | Что проверяет |
|---|---|
| `TestIsAdminRole_Admin` | `RoleAdmin` → `IsAdminRole` = true |
| `TestIsAdminRole_UserString` | `Role("user")` → `IsAdminRole` = false |
| `TestIsAdminRole_CustomRole` | `viewer`, `editor`, `moderator`, `readonly`, `""` → все false |
| `TestComputePermissions_Admin` | admin получает все 6 прав |
| `TestComputePermissions_UserRoleString` | `Role("user")` → нет `canViewAuditLogs`, нет `canManageUsers`, есть `canCreateShortUrl` |
| `TestComputePermissions_CustomRoleViewer` | `viewer` → дефолтные права (нет audit/manage) |
| `TestComputePermissions_CustomRoleEditor` | `editor` → дефолтные права (есть `canEditOwnLinks`) |
| `TestComputePermissions_Override_RestrictedViewer` | Роль `viewer-restricted` с override: все права false |
| `TestComputePermissions_Override_PowerUser` | Роль `power-user` с override: `canViewAuditLogs=true`, `canManageUsers=false` |
| `TestComputePermissions_Override_DoesNotAffectAdmin` | Override на `RoleAdmin` работает (override имеет наивысший приоритет) |
| `TestComputePermissions_Override_NotRegistered_FallsBackToDefault` | Незарегистрированная роль → `DefaultUserPermissions` |
| `TestDefaultUserPermissions_Override` | Изменение `DefaultUserPermissions` через var применяется ко всем непривилегированным ролям |

---

#### `service_test.go` — `ShlinkService`: slug prefix и фильтрация

| Тест | Что проверяет |
|---|---|
| `TestEnforceSlugPrefix_AdminBypass` | admin: prefix не применяется, slug не изменяется |
| `TestEnforceSlugPrefix_CustomRoleBypass_IsNotAdmin` | `viewer` с prefix `v1-`: slug проверяется (не admin-bypass) |
| `TestEnforceSlugPrefix_UserNoPrefix` | `Role("user")` без prefix → ошибка |
| `TestEnforceSlugPrefix_CustomRoleNoPrefix` | `Role("editor")` без prefix → ошибка |
| `TestEnforceSlugPrefix_UserCorrectPrefix` | slug с правильным prefix → OK, slug не меняется |
| `TestEnforceSlugPrefix_UserWrongPrefix` | slug без нужного prefix → ошибка |
| `TestEnforceSlugPrefix_FeatureDisabled` | feature flag выключен → slug не трогается |
| `TestEnforceSlugPrefix_UserNilSlug` | nil slug + prefix → возвращает prefix (Shlink добавит суффикс) |
| `TestEnforceSlugPrefix_CustomRole_NilSlug` | `viewer`, nil slug + prefix → возвращает prefix |
| `TestFilterShortURLsByUser` | `Role("user")` с prefix `u1-`: отдаёт только свои ссылки |
| `TestFilterShortURLsByUser_CustomRole` | `viewer` с prefix `vw-`: фильтрует по своему prefix |
| `TestFilterShortURLsByUser_AdminGetAll` | admin видит все ссылки без фильтрации |
| `TestFilterShortURLsByUser_EmptyPrefix` | пустой prefix → возвращает все ссылки |
| `TestFilterShortURLsByUser_FeatureDisabled` | feature flag выключен → все ссылки видны всем |

---

#### `handler_me_test.go` — HTTP handler `/api/me`

| Тест | Что проверяет |
|---|---|
| `TestMeHandler_ReturnsCorrectFields` | Ответ содержит sub, username, role, hasApiKey, features, permissions; `shlinkApiKey` / `shlink_api_key` / `apiKey` — отсутствуют (security) |
| `TestMeHandler_AdminPermissions` | admin: `canManageUsers=true`, `canViewAuditLogs=true`; ни один вариант API key не утекает |
| `TestMeHandler_NoUser_InternalError` | Нет user в контексте → 500 |

---

#### `audit_sanitize_test.go` — Sanitize чувствительных полей аудита

| Тест | Что проверяет |
|---|---|
| `TestSanitizeDetails_RemovesSensitiveKeys` | `shlink_api_key`, `api_key`, `authorization`, `password` удаляются; безопасные поля сохраняются |
| `TestSanitizeDetails_NilInput` | nil → nil, без паники |
| `TestSanitizeDetails_EmptyInput` | пустой map → пустой map |
| `TestSanitizeDetails_CaseInsensitive` | `SHLINK_API_KEY`, `Api_Key` (uppercase/mixedcase) тоже удаляются |

---

### CI/CD

GitHub Actions (`.github/workflows/ci.yml`):

- **test** — `go test -race ./...` с PostgreSQL-сервисом
- **lint** — `golangci-lint`
- **docker** — сборка образа `shlink-bff-go:ci`

Триггер: push/PR в `master`.
