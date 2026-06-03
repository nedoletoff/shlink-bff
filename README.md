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
