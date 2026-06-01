# shlink-bff

Backend-for-Frontend for [Shlink](https://shlink.io/) with OAuth2/OIDC authentication via Keycloak and role-based access control.

The project provides two interchangeable backend implementations:

| Implementation | Directory | Stack |
|---|---|---|
| **Python** (current default) | `python-backend/` | FastAPI · SQLAlchemy async · SQLite · uvicorn |
| **Go** (original) | *(unified-backend branch)* | net/http · SQLite · chi |

Both expose the same API contract, sit behind the same nginx + oauth2-proxy chain, and are drop-in replacements for each other.

---

## Architecture

```
Browser
  │
  ▼ HTTPS :443
nginx
  │
  ▼ HTTP :4180
oauth2-proxy  ──── OIDC ────► Keycloak
  │
  │  injects headers:
  │    X-Auth-Request-User
  │    X-Auth-Request-Email
  │    X-Auth-Request-Preferred-Username
  │    X-Auth-Request-Groups
  ▼
python-backend (FastAPI)  ──► shlink-api  (SQLite, internal)
  │                            :8080
  ▼
web-ui (React SPA, nginx)
```

**Security properties:**
- `shlink_api_key` lives only in the BFF database — it never reaches the browser
- `servers.json`, `/rest/`, `shlink-web-client` are removed entirely
- RBAC is enforced at the backend level, independent of the UI
- All mutating operations are written to an audit log (sensitive fields sanitised)
- `python-jose >= 3.5.0` — CVE-2024-33664 patched

---

## Repository layout

```
shlink-bff/
├── python-backend/          # FastAPI backend (active)
│   ├── app/
│   │   ├── config.py        # pydantic-settings, reads .env
│   │   ├── database.py      # SQLAlchemy async engine + session factory
│   │   ├── models.py        # ORM models + Pydantic response schemas
│   │   ├── dependencies.py  # Identity extraction from headers, RBAC guards
│   │   ├── main.py          # App factory, lifespan, CORS, router mount
│   │   └── routers/
│   │       ├── health.py    # GET /healthz
│   │       ├── me.py        # GET /api/me
│   │       ├── shlink.py    # /api/shlink/* proxy
│   │       └── admin.py     # /api/admin/* (admin role only)
│   ├── tests/
│   │   ├── conftest.py      # In-memory SQLite fixtures, AsyncClient, auth headers
│   │   ├── test_health.py
│   │   ├── test_me.py
│   │   ├── test_shlink.py
│   │   └── test_admin.py
│   ├── Dockerfile
│   └── pyproject.toml
├── web-ui/                  # React SPA
├── nginx/                   # nginx config templates
├── oauth2-proxy/            # oauth2-proxy config templates
├── docker-compose.yml       # Full-stack orchestration
└── .env.example             # Environment variable reference
```

---

## Prerequisites

- Docker ≥ 24 and Docker Compose v2
- A running **Keycloak** instance (realm + client configured)
- A DNS record / `/etc/hosts` entry for your short-link domain

---

## Quick start (Docker Compose — Python backend)

### 1. Clone and prepare config files

```bash
git clone https://github.com/nedoletoff/shlink-bff.git
cd shlink-bff

# Environment variables
cp .env.example .env
$EDITOR .env

# nginx virtual host
cp nginx/nginx.conf.example nginx/nginx.conf
$EDITOR nginx/nginx.conf   # replace example.com with your domain

# oauth2-proxy
cp oauth2-proxy/shlink.cfg.example oauth2-proxy/shlink.cfg
$EDITOR oauth2-proxy/shlink.cfg
```

### 2. Fill in `.env`

| Variable | Description | Example |
|---|---|---|
| `DOMAIN_SHORT` | Public domain for short links | `s.example.com` |
| `ADMIN_SHLINK_API_KEY` | Initial Shlink admin API key | `secret-key-here` |
| `KEYCLOAK_HOST` | Keycloak FQDN | `auth.example.com` |
| `KEYCLOAK_IP` | Keycloak IP (for extra_hosts resolution) | `192.168.1.10` |
| `OAUTH2_CLIENT_SECRET_SHLINK` | Client secret from Keycloak | `abc...` |
| `OAUTH2_COOKIE_SECRET` | 32-byte random string | `openssl rand -base64 32` |
| `FEATURE_USER_SLUG_PREFIX` | Prepend username to slug | `false` |
| `FEATURE_USER_TAG_INTERNAL_ID` | Use internal tag IDs | `false` |

### 3. SSL certificate

Place a PEM-encoded certificate (cert + chain + key) at `nginx/ssl/cert.pem`:

```bash
mkdir -p nginx/ssl
cp /path/to/fullchain+privkey.pem nginx/ssl/cert.pem
```

### 4. Start the stack

```bash
docker compose up -d --build
```

Services started:

| Container | Exposes | Role |
|---|---|---|
| `nginx-proxy` | 80, 443 | TLS termination, routing |
| `oauth2-proxy-shlink` | internal :4180 | OIDC auth, header injection |
| `python-backend` | internal :8080 | BFF API |
| `web-ui` | internal :80 | React SPA |
| `shlink-api` | internal :8080 | Shlink short-link engine |

### 5. Create the first user

```bash
docker compose exec python-backend python - <<'EOF'
import asyncio, uuid
from app.database import get_session_factory, _get_engine
from app.models import User, Role, Status

async def main():
    engine = _get_engine()
    factory = get_session_factory()
    async with factory() as session:
        user = User(
            id=str(uuid.uuid4()),
            sub="keycloak-subject-uuid",   # copy from Keycloak user details
            username="admin",
            email="admin@example.com",
            role=Role.ADMIN,
            shlink_api_key="your-shlink-api-key",
            status=Status.ACTIVE,
        )
        session.add(user)
        await session.commit()
    await engine.dispose()

asyncio.run(main())
EOF
```

> **`sub`** is the Keycloak subject UUID. Find it at:  
> Keycloak Admin → Realm → Users → (user) → Details → ID

---

## Local development (Python backend)

### Requirements

- Python ≥ 3.12
- `pip` / `venv`

### Install

```bash
cd python-backend

python3.12 -m venv .venv
source .venv/bin/activate          # Windows: .venv\Scripts\activate

pip install -e ".[dev]"
```

### Configure

Minimal `.env` for local dev (place in `python-backend/` or the repo root):

```dotenv
DATABASE_URL=sqlite+aiosqlite:///./shlink_bff.db
SHLINK_INTERNAL_URL=http://localhost:8888   # or a real Shlink instance
HTTP_ADDR=:8080
```

### Run with hot reload

```bash
uvicorn app.main:create_app --factory --reload --host 0.0.0.0 --port 8080
```

| URL | Description |
|---|---|
| http://localhost:8080/healthz | Health + DB probe |
| http://localhost:8080/api/me | Current user profile |
| http://localhost:8080/docs | Swagger UI |
| http://localhost:8080/redoc | ReDoc |

### Run tests

Tests use an **in-memory SQLite** database — no external services required.

```bash
cd python-backend

# All tests
pytest tests/ -v

# With coverage report
pytest tests/ -v --cov=app --cov-report=term-missing

# Single module
pytest tests/test_health.py -v
pytest tests/test_me.py -v
pytest tests/test_admin.py -v
pytest tests/test_shlink.py -v
```

### Lint and format

```bash
cd python-backend

# Lint (no auto-fix)
ruff check .

# Format check (CI mode)
ruff format --check .

# Apply formatting
ruff format .
```

### Dependency matrix

| Package | Min version | Purpose |
|---|---|---|
| `fastapi` | 0.130 | Web framework |
| `uvicorn[standard]` | 0.47 | ASGI server (uvloop + httptools) |
| `sqlalchemy[asyncio]` | 2.0.40 | Async ORM |
| `aiosqlite` | 0.21 | SQLite async driver |
| `pydantic` | 2.11 | Schema validation |
| `pydantic-settings` | 2.7 | Settings from `.env` |
| `httpx` | 0.28 | HTTP client for Shlink proxy |
| `python-jose[cryptography]` | 3.5 | JWT / JOSE (CVE-2024-33664 patched) |
| `structlog` | 25.0 | Structured JSON logging |
| `pytest-asyncio` | 1.0 | Async test support |
| `ruff` | 0.15 | Linter + formatter |

---

## Local development (Go backend)

> The Go backend lives in the `unified-backend` branch.

```bash
git checkout unified-backend
cd unified-backend

# Build
go build -o shlink-bff ./cmd/server

# Run
DATABASE_URL=./shlink_bff.db \
SHLINK_INTERNAL_URL=http://localhost:8888 \
HTTP_ADDR=:8080 \
./shlink-bff
```

The Go backend reads the same environment variables and serves the identical API contract. Switch between implementations by swapping the `build` context in `docker-compose.yml`.

---

## API reference

Authentication is handled by **oauth2-proxy** upstream. The backend trusts these headers injected by the proxy:

```
X-Auth-Request-User:                 <keycloak-subject-uuid>
X-Auth-Request-Email:                user@example.com
X-Auth-Request-Preferred-Username:   username
X-Auth-Request-Groups:               admins,users
```

Requests without these headers receive `401 Unauthorized`.

### Endpoints

| Method | Path | Roles | Description |
|---|---|---|---|
| `GET` | `/healthz` | — | Health check + DB probe |
| `GET` | `/api/me` | user · admin | Current user profile |
| `GET` | `/api/shlink/short-urls` | user · admin | List short URLs |
| `POST` | `/api/shlink/short-urls` | user · admin | Create short URL |
| `PATCH` | `/api/shlink/short-urls/{shortCode}` | user · admin | Update short URL |
| `DELETE` | `/api/shlink/short-urls/{shortCode}` | user · admin | Delete short URL |
| `GET` | `/api/shlink/tags` | user · admin | List tags |
| `POST` | `/api/shlink/tags` | user · admin | Create tag |
| `PUT` | `/api/shlink/tags/{tagId}` | user · admin | Rename tag |
| `DELETE` | `/api/shlink/tags/{tagId}` | user · admin | Delete tag |
| `GET` | `/api/admin/users` | **admin** | List all users |
| `GET` | `/api/admin/users/{sub}` | **admin** | Get user by Keycloak sub |
| `PUT` | `/api/admin/users/{sub}` | **admin** | Update user (role, status) |
| `PUT` | `/api/admin/users/{sub}/apikey` | **admin** | Replace Shlink API key |
| `PUT` | `/api/admin/users/{sub}/prefix` | **admin** | Update slug prefix |
| `GET` | `/api/admin/users/{sub}/links` | **admin** | List user's short URLs |
| `GET` | `/api/admin/logs` | **admin** | Audit log |

### Response: `GET /api/me`

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "username": "user",
  "role": "user",
  "permissions": ["read", "write"],
  "slug_prefix": null
}
```

### Response: `GET /healthz`

```json
{ "status": "ok", "db": "ok" }
```

Returns `503 Service Unavailable` when the database is unreachable:

```json
{ "status": "error", "db": "unable to open database file" }
```

---

## Keycloak setup

1. Create a realm (e.g. `shlink`)
2. Create a client `shlink-bff`:
   - Access type: **confidential**
   - Valid redirect URIs: `https://your-domain/oauth2/callback`
   - Copy the **Client Secret** → `OAUTH2_CLIENT_SECRET_SHLINK` in `.env`
3. Create groups `admins` and `users` in the realm
4. Add a mapper: **Group Membership** → token claim `groups` (full path: off)
5. The `sub` field in Keycloak user details is the value to use when creating users in the BFF database

---

## Useful commands

```bash
# View live logs
docker compose logs -f python-backend

# Rebuild a single service
docker compose up -d --build python-backend

# Open a shell inside the backend container
docker compose exec python-backend bash

# Check health endpoint
curl -sf http://localhost:8080/healthz | python3 -m json.tool

# Run one-off migration or script inside the container
docker compose exec python-backend python path/to/script.py
```

---

## Image versions

| Image | Version |
|---|---|
| `nginx` | 1.30-alpine |
| `quay.io/oauth2-proxy/oauth2-proxy` | v7.15.2 |
| `shlinkio/shlink` | 5.0.2 |
| `python` | 3.12-slim |
| `node` | 22-alpine |

---

## CI

GitHub Actions runs on every push to `feature/python-fastapi-backend`:

| Job | Steps |
|---|---|
| **Lint & Type Check** | `ruff check .` · `ruff format --check .` · `mypy app/` |
| **Tests** | `pytest tests/ -v --tb=short` (Python 3.12, in-memory SQLite) |

All checks must pass before merging.
