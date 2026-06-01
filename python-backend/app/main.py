"""FastAPI application entry-point."""
from __future__ import annotations

import logging
from contextlib import asynccontextmanager
from typing import AsyncGenerator

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
# Lifespan (replaces deprecated @app.on_event)
# ---------------------------------------------------------------------------
@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator[None, None]:  # noqa: ARG001
    settings = get_settings()
    log.info(
        "startup",
        http_addr=settings.http_addr,
        shlink_url=settings.shlink_internal_url,
    )
    yield
    log.info("shutdown")


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
        lifespan=lifespan,
    )

    # CORS: allow_origins=["*"] with allow_credentials=True is invalid per spec.
    # Internal-only service — restrict to explicit origins from settings.
    # For local dev without a real origin list, use allow_credentials=False with "*".
    cors_origins = settings.cors_allowed_origins
    app.add_middleware(
        CORSMiddleware,
        allow_origins=cors_origins,
        allow_credentials=bool(cors_origins and cors_origins != ["*"]),
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

    return app


app = create_app()
