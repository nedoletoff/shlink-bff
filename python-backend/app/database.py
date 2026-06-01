"""Async SQLAlchemy engine and session factory (MySQL / aiomysql)."""
from collections.abc import AsyncGenerator

from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from app.config import get_settings

_engine = None
_session_factory = None


def _build_mysql_url(raw: str) -> str:
    """Normalise DATABASE_URL to use mysql+aiomysql:// scheme.

    Accepts the following input formats:
      mysql://...
      mysql+aiomysql://...
      mysql+pymysql://...
    """
    for prefix in (
        "mysql+pymysql://",
        "mysql+aiomysql://",
        "mysql://",
    ):
        if raw.startswith(prefix):
            return "mysql+aiomysql://" + raw[len(prefix):]
    # If already correct or unknown scheme — return as-is
    return raw


def _get_engine():
    global _engine
    if _engine is None:
        settings = get_settings()
        url = _build_mysql_url(settings.database_url)
        _engine = create_async_engine(
            url,
            echo=False,
            pool_pre_ping=True,
            # aiomysql: keep connections alive
            pool_recycle=3600,
        )
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
