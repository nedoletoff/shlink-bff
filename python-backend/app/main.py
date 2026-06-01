"""FastAPI application entry-point."""
import logging

import structlog
from fastapi import FastAPI, Request, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.config import get_settings
from app.routers import admin, health, me, shlink

# ---------------------------------------------------------------------------
# Structured logging
# ---------------------------------------------------------------------------

structlog.configure(
    processors=[
        structlog.contextvars.merge_contextvars,
        structlog.processors.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.JSONRenderer(),
    ],
    wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
    context_class=dict,
    logger_factory=structlog.PrintLoggerFactory(),
)

log = structlog.get_logger()

# ---------------------------------------------------------------------------
# Application factory
# ---------------------------------------------------------------------------


def create_app() -> FastAPI:
    settings = get_settings()

    app = FastAPI(
        title="Shlink BFF — Python/FastAPI Backend",
        description=(
            "Unified backend for Shlink with OAuth2/OIDC authorization "
            "(Python rewrite of the Go unified-backend)"
        ),
        version="0.1.0",
    )

    # CORS — only internal traffic expected, but useful during development
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # ---------------------------------------------------------------------------
    # Routers
    # ---------------------------------------------------------------------------
    app.include_router(health.router)
    app.include_router(me.router)
    app.include_router(shlink.router)
    app.include_router(admin.router)

    # ---------------------------------------------------------------------------
    # Global exception handler
    # ---------------------------------------------------------------------------
    @app.exception_handler(Exception)
    async def unhandled_exception_handler(request: Request, exc: Exception) -> JSONResponse:
        log.error("unhandled_exception", path=str(request.url), error=str(exc))
        return JSONResponse(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            content={"detail": "internal server error"},
        )

    # ---------------------------------------------------------------------------
    # Startup / shutdown
    # ---------------------------------------------------------------------------
    @app.on_event("startup")
    async def on_startup() -> None:
        log.info(
            "startup",
            http_addr=settings.http_addr,
            shlink_url=settings.shlink_internal_url,
        )

    return app


app = create_app()
