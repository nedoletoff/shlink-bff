"""FastAPI dependencies: identity extraction and RBAC."""

from __future__ import annotations

import dataclasses
from typing import Annotated

from fastapi import Depends, Header, HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_db
from app.models import AuditLog, Role, User


@dataclasses.dataclass
class Identity:
    """Parsed identity from oauth2-proxy headers."""

    sub: str
    email: str
    username: str
    role: str
    groups: list[str]


def _resolve_role(groups: list[str]) -> str:
    """Map OIDC groups to internal role (admin wins)."""
    for g in groups:
        if g.lower() in ("admin", "admins", "shlink-admins"):
            return Role.ADMIN
    return Role.USER


async def get_identity(
    x_auth_request_user: Annotated[str | None, Header(alias="X-Auth-Request-User")] = None,
    x_auth_request_email: Annotated[str | None, Header(alias="X-Auth-Request-Email")] = None,
    x_auth_request_preferred_username: Annotated[
        str | None, Header(alias="X-Auth-Request-Preferred-Username")
    ] = None,
    x_auth_request_groups: Annotated[str | None, Header(alias="X-Auth-Request-Groups")] = None,
) -> Identity:
    """Extract identity from oauth2-proxy injected headers."""
    if not x_auth_request_user:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="unauthorized")

    raw_groups = x_auth_request_groups or ""
    groups = [g.strip() for g in raw_groups.split(",") if g.strip()]
    role = _resolve_role(groups)

    return Identity(
        sub=x_auth_request_user,
        email=x_auth_request_email or "",
        username=x_auth_request_preferred_username or x_auth_request_user,
        role=role,
        groups=groups,
    )


async def get_or_create_user(
    identity: Annotated[Identity, Depends(get_identity)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> User:
    """Upsert user on every authenticated request."""
    result = await db.execute(select(User).where(User.sub == identity.sub))
    user = result.scalar_one_or_none()

    if user is None:
        user = User(
            sub=identity.sub,
            email=identity.email,
            username=identity.username,
            role=identity.role,
        )
        db.add(user)
        await db.commit()
        await db.refresh(user)
    else:
        # Sync role from OIDC groups on every request
        changed = False
        if user.role != identity.role:
            user.role = identity.role
            changed = True
        if user.email != identity.email:
            user.email = identity.email
            changed = True
        if user.username != identity.username:
            user.username = identity.username
            changed = True
        if changed:
            await db.commit()
            await db.refresh(user)

    return user


async def require_admin(
    user: Annotated[User, Depends(get_or_create_user)],
) -> User:
    """Raise 403 if user is not admin."""
    if user.role != Role.ADMIN:
        raise HTTPException(status_code=status.HTTP_403_FORBIDDEN, detail="forbidden")
    return user


async def write_audit_log(
    db: AsyncSession,
    user: User,
    action: str,
    resource: str | None,
    result: str,
    details: dict | None = None,
) -> None:
    log = AuditLog(
        user_sub=user.sub,
        username=user.username,
        role=user.role,
        action=action,
        resource=resource,
        result=result,
        details=details,
    )
    db.add(log)
    await db.commit()
