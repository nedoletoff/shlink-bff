"""Pytest fixtures for python-backend tests.

Strategy: all DB interaction is replaced by in-memory SQLite
(using SQLAlchemy's aiosqlite driver) so tests never need a real
MySQL instance.  HTTP calls to Shlink API are intercepted with
httpx.AsyncClient + a custom ASGI transport.
"""

from __future__ import annotations

import uuid
from collections.abc import AsyncGenerator

import pytest
import pytest_asyncio
from httpx import ASGITransport, AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from app.database import get_db
from app.main import create_app
from app.models import Base, User

# ---------------------------------------------------------------------------
# In-memory SQLite engine (avoids the need for a real MySQL in CI)
# ---------------------------------------------------------------------------

TEST_DATABASE_URL = "sqlite+aiosqlite:///:memory:"


@pytest_asyncio.fixture(scope="session")
async def engine():
    """Create tables once per test session."""
    eng = create_async_engine(TEST_DATABASE_URL, echo=False)
    async with eng.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield eng
    await eng.dispose()


@pytest_asyncio.fixture
async def db_session(engine) -> AsyncGenerator[AsyncSession, None]:
    """Wrap every test in a savepoint that is rolled back afterwards."""
    factory = async_sessionmaker(engine, expire_on_commit=False, class_=AsyncSession)
    async with factory() as session:
        yield session
        await session.rollback()


# ---------------------------------------------------------------------------
# Default OIDC headers (simulate oauth2-proxy injecting user info)
# ---------------------------------------------------------------------------

ADMIN_HEADERS = {
    "X-Auth-Request-User": "admin-sub-001",
    "X-Auth-Request-Email": "admin@example.com",
    "X-Auth-Request-Groups": "shlink-admins",
    "X-Auth-Request-Preferred-Username": "admin",
}

USER_HEADERS = {
    "X-Auth-Request-User": "user-sub-001",
    "X-Auth-Request-Email": "user@example.com",
    "X-Auth-Request-Groups": "shlink-users",
    "X-Auth-Request-Preferred-Username": "alice",
}


# ---------------------------------------------------------------------------
# FastAPI test client
# ---------------------------------------------------------------------------


@pytest_asyncio.fixture
async def client(db_session: AsyncSession):
    """Return an AsyncClient wired to the FastAPI app.

    The real DB dependency is overridden to use the in-memory SQLite session.
    The real Shlink HTTP client is overridden with a mock.
    """
    app = create_app()

    # Override DB dependency
    async def _override_get_db():
        yield db_session

    app.dependency_overrides[get_db] = _override_get_db

    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as ac:
        yield ac


# ---------------------------------------------------------------------------
# Helper: pre-seed a user row
# ---------------------------------------------------------------------------


async def seed_user(
    session: AsyncSession,
    sub: str = "user-sub-001",
    username: str = "alice",
    email: str = "user@example.com",
    role: str = "user",
    status: str = "active",
    shlink_api_key: str = "test-key",
) -> User:
    user = User(
        id=str(uuid.uuid4()),
        sub=sub,
        username=username,
        email=email,
        role=role,
        status=status,
        shlink_api_key=shlink_api_key,
    )
    session.add(user)
    await session.flush()
    return user


@pytest.fixture
def auth_headers():
    """Headers simulating an authenticated regular user via oauth2-proxy."""
    return dict(USER_HEADERS)


@pytest.fixture
def admin_headers():
    """Headers simulating an authenticated admin user via oauth2-proxy."""
    return dict(ADMIN_HEADERS)
