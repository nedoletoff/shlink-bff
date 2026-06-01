"""Admin-only endpoints: user management and audit logs."""
from __future__ import annotations

from typing import Annotated, Any

import httpx
from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import get_settings
from app.database import get_db
from app.dependencies import require_admin, write_audit_log
from app.models import (
    ApiKeyUpdateRequest,
    AuditLog,
    AuditLogResponse,
    PrefixUpdateRequest,
    User,
    UserResponse,
    UserUpdateRequest,
    compute_permissions,
)

router = APIRouter(prefix="/api/admin", tags=["admin"])


def _user_response(u: User) -> UserResponse:
    """Build UserResponse with computed permissions (Pydantic v2 safe)."""
    return UserResponse.model_validate(u).model_copy(
        update={"permissions": compute_permissions(u.role)}
    )


# ---------------------------------------------------------------------------
# Users
# ---------------------------------------------------------------------------

@router.get("/users", response_model=list[UserResponse])
async def list_users(
    admin: Annotated[User, Depends(require_admin)],
    db: Annotated[AsyncSession, Depends(get_db)],
    page: int = Query(1, ge=1),
    page_size: int = Query(50, ge=1, le=200),
) -> list[UserResponse]:
    offset = (page - 1) * page_size
    result = await db.execute(select(User).offset(offset).limit(page_size))
    users = result.scalars().all()
    return [_user_response(u) for u in users]


@router.get("/users/{sub}", response_model=UserResponse)
async def get_user(
    sub: str,
    admin: Annotated[User, Depends(require_admin)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> UserResponse:
    result = await db.execute(select(User).where(User.sub == sub))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="user not found")
    return _user_response(user)


@router.put("/users/{sub}", response_model=UserResponse)
async def update_user(
    sub: str,
    body: UserUpdateRequest,
    admin: Annotated[User, Depends(require_admin)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> UserResponse:
    result = await db.execute(select(User).where(User.sub == sub))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="user not found")
    if body.role is not None:
        user.role = body.role
    if body.status is not None:
        user.status = body.status
    await db.commit()
    await db.refresh(user)
    await write_audit_log(
        db, admin, "update_user", sub, "success",
        body.model_dump(exclude_none=True),
    )
    return _user_response(user)


@router.put("/users/{sub}/apikey", response_model=UserResponse)
async def update_apikey(
    sub: str,
    body: ApiKeyUpdateRequest,
    admin: Annotated[User, Depends(require_admin)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> UserResponse:
    result = await db.execute(select(User).where(User.sub == sub))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="user not found")
    user.shlink_api_key = body.api_key
    await db.commit()
    await db.refresh(user)
    await write_audit_log(db, admin, "update_apikey", sub, "success")
    return _user_response(user)


@router.put("/users/{sub}/prefix", response_model=UserResponse)
async def update_prefix(
    sub: str,
    body: PrefixUpdateRequest,
    admin: Annotated[User, Depends(require_admin)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> UserResponse:
    result = await db.execute(select(User).where(User.sub == sub))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="user not found")
    user.slug_prefix = body.prefix
    await db.commit()
    await db.refresh(user)
    await write_audit_log(db, admin, "update_prefix", sub, "success")
    return _user_response(user)


@router.get("/users/{sub}/links")
async def get_user_links(
    sub: str,
    admin: Annotated[User, Depends(require_admin)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Any:
    """Proxy: list all short-urls belonging to a specific user via their API key."""
    result = await db.execute(select(User).where(User.sub == sub))
    user = result.scalar_one_or_none()
    if not user:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="user not found")
    settings = get_settings()
    url = settings.shlink_internal_url.rstrip("/") + "/rest/v3/short-urls"
    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.get(url, headers={"X-Api-Key": user.shlink_api_key})
    if resp.status_code >= 400:
        raise HTTPException(status_code=resp.status_code, detail=resp.text)
    return resp.json()


# ---------------------------------------------------------------------------
# Audit logs
# ---------------------------------------------------------------------------

@router.get("/logs", response_model=list[AuditLogResponse])
async def list_audit_logs(
    admin: Annotated[User, Depends(require_admin)],
    db: Annotated[AsyncSession, Depends(get_db)],
    page: int = Query(1, ge=1),
    page_size: int = Query(50, ge=1, le=200),
) -> list[AuditLogResponse]:
    offset = (page - 1) * page_size
    result = await db.execute(
        select(AuditLog).order_by(AuditLog.created_at.desc()).offset(offset).limit(page_size)
    )
    return [AuditLogResponse.model_validate(log) for log in result.scalars().all()]
