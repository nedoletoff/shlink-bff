# Shlink BFF — Unified Backend + Web UI

## Архитектура

```
Browser → nginx (HTTPS) → oauth2-proxy → python-backend (FastAPI) → shlink-api
                                                                   → web-ui (React SPA)
```

> **Ветка `feature/python-fastapi-backend`** содержит переписанный бэкенд на Python/FastAPI  
> вместо оригинального Go-бэкенда (`unified-backend`). API-контракт полностью сохранён.

**Принципы безопасности:**

- `shlink_api_key` хранится только в БД, никогда не попадает в браузер
- `servers.json`, `/rest/`, `shlink-web-client` удалены полностью
- RBAC принудителен на уровне backend, независимо от UI
- Аудит-логи: все операции с sanitize чувствительных полей
- `python-jose>=3.5.0` — исправлена CVE-2024-33664

---

## Python Backend (FastAPI)

### Стек

| Компонент | Версия | Назначение |
|---|---|---|
| Python | ≥ 3.12 | Runtime |
| FastAPI | ≥ 0.136 | Web-фреймворк |
| uvicorn[standard] | ≥ 0.47 | ASGI-сервер |
| SQLAlchemy[asyncio] | ≥ 2.0.40 | ORM (async) |
| aiosqlite | ≥ 0.21 | Async SQLite-драйвер (dev/CI) |
| Pydantic v2 | ≥ 2.11 | Схемы и валидация |
| pydantic-settings | ≥ 2.7 | Конфигурация из .env |
| httpx | ≥ 0.28 | HTTP-клиент для проксирования в Shlink |
| python-jose[cryptography] | ≥ 3.5 | JOSE/JWT (CVE-2024-33664 patched) |
| structlog | ≥ 25.0 | Структурированные JSON-логи |

### Структура

```
python-backend/
├── app/
│   ├── config.py          # Настройки (pydantic-settings, .env)
│   ├── database.py        # SQLAlchemy async engine + session factory
│   ├── models.py          # ORM-модели + Pydantic-схемы ответов
│   ├── dependencies.py    # Identity из oauth2-proxy headers, RBAC
│   ├── main.py            # FastAPI app factory, lifespan, CORS
│   └── routers/
│       ├── health.py      # GET /healthz (DB readiness probe)
│       ├── me.py          # GET /api/me
│       ├── shlink.py      # GET|POST|PATCH|DELETE /api/shlink/*
│       └── admin.py       # /api/admin/* (только role=admin)
├── migrations/
│   └── 001_init_schema.sql  # DDL для SQLite (справочник; схема создаётся через ORM)
├── tests/
│   ├── conftest.py        # Фикстуры: in-memory SQLite, AsyncClient, auth headers
│   ├── test_health.py
│   ├── test_me.py
│   ├── test_shlink.py
│   └── test_admin.py
├── Dockerfile
└── pyproject.toml
```

### Локальная разработка

#### 1. Переменные окружения

```bash
cp .env.example .env
# Отредактируйте .env — минимальный набор для локального запуска:
```

```dotenv
# .env (пример для локального dev)
DATABASE_URL=sqlite+aiosqlite:///./shlink_bff.db
SHLINK_INTERNAL_URL=http://shlink-api:8080
HTTP_ADDR=:8080
```

#### 2. Установка зависимостей

```bash
cd python-backend

# Создать виртуальное окружение
python3.12 -m venv .venv
source .venv/bin/activate

# Установить пакет со всеми dev-зависимостями
pip install -e ".[dev]"
```

#### 3. Запуск в dev-режиме (hot reload)

```bash
uvicorn app.main:create_app --factory --reload --host 0.0.0.0 --port 8080
```

После запуска:
- API: http://localhost:8080
- Swagger UI: http://localhost:8080/docs
- ReDoc: http://localhost:8080/redoc
- Healthcheck: http://localhost:8080/healthz

### Тестирование

Тесты используют **in-memory SQLite** — никакой внешней БД не нужно.

```bash
cd python-backend

# Запустить все тесты
pytest tests/ -v

# С отчётом о покрытии
pytest tests/ -v --cov=app --cov-report=term-missing

# Запустить конкретный модуль
pytest tests/test_health.py -v
pytest tests/test_me.py -v
pytest tests/test_admin.py -v
pytest tests/test_shlink.py -v
```

**Что покрывают тесты:**

| Файл | Покрытие |
|---|---|
| `test_health.py` | `GET /healthz` — статус 200, поля `status`/`db`, без auth, content-type |
| `test_me.py` | `GET /api/me` — 401 без токена, 200 с auth, поля id/email/role/permissions |
| `test_shlink.py` | `GET|POST|PATCH|DELETE /api/shlink/*` — проксирование, изоляция пользователей |
| `test_admin.py` | `/api/admin/*` — список юзеров, обновление, audit-логи, 403 для non-admin |

### Линтинг и типы

```bash
cd python-backend

# Ruff — линтер + форматтер
ruff check app/ tests/
ruff format app/ tests/

# Mypy — статическая типизация (strict mode)
mypy app/
```

### Docker

```bash
# Собрать образ локально
docker build -t shlink-bff-python python-backend/

# Запустить контейнер
docker run --rm -p 8080:8080 \
  -e DATABASE_URL=sqlite+aiosqlite:///./shlink_bff.db \
  -e SHLINK_INTERNAL_URL=http://shlink-api:8080 \
  shlink-bff-python
```

### Docker Compose (полный стек)

```bash
# 1. Скопируйте .env и заполните секреты
cp .env.example .env && vi .env

# 2. Скопируйте конфиги под свой домен
cp nginx/nginx.conf.example nginx/nginx.conf
cp oauth2-proxy/shlink.cfg.example oauth2-proxy/shlink.cfg

# 3. SSL-сертификат (cert + key в одном PEM)
mkdir -p nginx/ssl
# скопируйте cert.pem в nginx/ssl/cert.pem

# 4. Запустите
docker compose up -d --build

# 5. Создайте первого пользователя в БД
docker compose exec python-backend python - <<'EOF'
import asyncio
from app.database import get_session_factory, _get_engine
from app.models import User, Role, Status
import uuid

async def main():
    engine = _get_engine()
    factory = get_session_factory()
    async with factory() as session:
        user = User(
            id=str(uuid.uuid4()),
            sub="keycloak-sub-here",
            username="admin",
            email="admin@example.local",
            role=Role.ADMIN,
            shlink_api_key="shlink-api-key-here",
            status=Status.ACTIVE,
        )
        session.add(user)
        await session.commit()
    await engine.dispose()

asyncio.run(main())
EOF
```

### Healthcheck

```bash
curl http://localhost:8080/healthz
# {"status": "ok", "db": "ok"}
```

При недоступности БД вернёт `503 Service Unavailable`:
```json
{"status": "error", "db": "<причина ошибки>"}
```

---

## API контракт

| Method | Path | Auth | Описание |
|---|---|---|---|
| GET | /healthz | — | Healthcheck + DB probe |
| GET | /api/me | user/admin | Профиль текущего пользователя |
| GET | /api/shlink/short-urls | user/admin | Список коротких ссылок |
| POST | /api/shlink/short-urls | user/admin | Создать ссылку |
| PATCH | /api/shlink/short-urls/{shortCode} | user/admin | Обновить ссылку |
| DELETE | /api/shlink/short-urls/{shortCode} | user/admin | Удалить ссылку |
| GET | /api/shlink/tags | user/admin | Теги пользователя |
| POST | /api/shlink/tags | user/admin | Создать тег |
| PUT | /api/shlink/tags/{tagId} | user/admin | Переименовать тег |
| DELETE | /api/shlink/tags/{tagId} | user/admin | Удалить тег |
| GET | /api/admin/users | **admin** | Список пользователей |
| GET | /api/admin/users/{sub} | **admin** | Профиль пользователя |
| PUT | /api/admin/users/{sub} | **admin** | Обновить пользователя |
| PUT | /api/admin/users/{sub}/apikey | **admin** | Обновить Shlink API key |
| PUT | /api/admin/users/{sub}/prefix | **admin** | Обновить slug prefix |
| GET | /api/admin/users/{sub}/links | **admin** | Ссылки пользователя |
| GET | /api/admin/logs | **admin** | Журнал аудита |

**Auth** — заголовки, которые инжектирует `oauth2-proxy`:

```
X-Auth-Request-User: <keycloak-sub>
X-Auth-Request-Email: user@example.com
X-Auth-Request-Preferred-Username: username
X-Auth-Request-Groups: admins,users
```

---

## Версии образов

| Образ | Версия |
|---|---|
| nginx | 1.30-alpine |
| oauth2-proxy | v7.15.2 |
| shlink | 5.0.2 |
| python | 3.12-slim |
| node | 22-alpine |
