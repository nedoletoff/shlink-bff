"""Async SQLAlchemy engine and session factory."""
from collections.abc import AsyncGenerator

from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from app.config import get_settings

_engine = None
_session_factory = None


def _get_engine():
    global _engine
    if _engine is None:
        settings = get_settings()
        # asyncpg driver: replace postgresql:// with postgresql+asyncpg://
        url = settings.database_url.replace(
            "postgresql://", "postgresql+asyncpg://", 1
        ).replace(
            "postgres://", "postgresql+asyncpg://", 1
        )
        _engine = create_async_engine(url, echo=False, pool_pre_ping=True)
    return _engine


def get_session_factory() -> async_sessionmaker[AsyncSession]:
    global _session_factory
    if _session_factory is None:
        _session_factory = async_sessionmaker(
            _get_engine(), expire_on_commit=False, class_=AsyncSession
        )
    return _session_factory


async def get_db() -> AsyncGenerator[AsyncSession, None]:
    """FastAPI dependency: yields a database session."""
    async with get_session_factory()() as session:
        yield session
