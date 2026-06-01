"""GET /healthz – liveness + DB readiness probe."""
from typing import Annotated

from fastapi import APIRouter, Depends, status
from fastapi.responses import JSONResponse
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_db

router = APIRouter(tags=["health"])


@router.get("/healthz")
async def healthcheck(
    db: Annotated[AsyncSession, Depends(get_db)],
) -> JSONResponse:
    """Return 200 when the service is up and the DB is reachable."""
    try:
        await db.execute(text("SELECT 1"))
        db_status = "ok"
    except Exception as exc:  # noqa: BLE001
        return JSONResponse(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            content={"status": "error", "db": str(exc)},
        )
    return JSONResponse(
        status_code=status.HTTP_200_OK,
        content={"status": "ok", "db": db_status},
    )
