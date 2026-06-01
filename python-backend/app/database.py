"""Async SQLAlchemy engine and session factory (SQLite / aiosqlite)."""
from collections.abc import AsyncGenerator

from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from app.config import get_settings

_engine = None
_session_factory = None


def _get_engine():
    global _engine
    if _engine is None:
        settings = get_settings()
        url = settings.database_url
        # Normalise plain sqlite:// -> sqlite+aiosqlite://
        if url.startswith("sqlite://") and "+" not in url.split(":")[0]:
            url = "sqlite+aiosqlite" + url[len("sqlite"):]
        _engine = create_async_engine(
            url,
            echo=False,
            # SQLite: allow shared cache across threads in async context
            connect_args={"check_same_thread": False},
        )
    return _engine


def get_session_factory() -> async_sessionmaker[AsyncSession]:
    global _session_factory
    if _session_factory is None:
        _session_factory = async_sessionmaker(
            _get_engine(),
            expire_on_commit=False,
            class_=AsyncSession,
        )
    return _session_factory


async def get_db() -> AsyncGenerator[AsyncSession, None]:
    """FastAPI dependency: yield an async DB session."""
    async with get_session_factory()() as session:
        yield session


async def init_db() -> None:
    """Create all tables (used in lifespan startup)."""
    from app.models import Base  # noqa: PLC0415
    async with _get_engine().begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
