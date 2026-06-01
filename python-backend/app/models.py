"""SQLAlchemy ORM models and Pydantic response schemas."""
from __future__ import annotations

import uuid
from datetime import datetime
from enum import StrEnum
from typing import Any

from pydantic import BaseModel, ConfigDict
from sqlalchemy import CheckConstraint, ForeignKey, Index, String, Text, func
from sqlalchemy.dialects.postgresql import JSONB, UUID
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


# ---------------------------------------------------------------------------
# Enums
# ---------------------------------------------------------------------------

class Role(StrEnum):
    ADMIN = "admin"
    USER = "user"


class Status(StrEnum):
    ACTIVE = "active"
    DISABLED = "disabled"
    PENDING = "pending"


class AuditResult(StrEnum):
    SUCCESS = "success"
    DENIED = "denied"
    ERROR = "error"


# ---------------------------------------------------------------------------
# ORM
# ---------------------------------------------------------------------------

class Base(DeclarativeBase):
    pass


class User(Base):
    __tablename__ = "users"
    __table_args__ = (
        CheckConstraint("role IN ('admin', 'user')", name="ck_users_role"),
        CheckConstraint("status IN ('active', 'disabled', 'pending')", name="ck_users_status"),
        Index("idx_users_sub", "sub", unique=True),
        Index("idx_users_role", "role"),
        Index("idx_users_status", "status"),
    )

    id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), primary_key=True, default=uuid.uuid4
    )
    sub: Mapped[str] = mapped_column(String, nullable=False, unique=True)
    username: Mapped[str] = mapped_column(Text, nullable=False)
    email: Mapped[str] = mapped_column(Text, nullable=False)
    role: Mapped[str] = mapped_column(String, nullable=False, default="user")
    shlink_api_key: Mapped[str] = mapped_column(Text, nullable=False, default="")
    slug_prefix: Mapped[str | None] = mapped_column(Text)
    status: Mapped[str] = mapped_column(String, nullable=False, default="active")
    created_at: Mapped[datetime] = mapped_column(server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(
        server_default=func.now(), onupdate=func.now()
    )

    tags: Mapped[list[UserTag]] = relationship(back_populates="user", cascade="all, delete")


class UserTag(Base):
    __tablename__ = "user_tags"
    __table_args__ = (
        Index("idx_user_tags_user_id", "user_id"),
        Index("idx_user_tags_internal_id", "internal_id"),
    )

    id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), primary_key=True, default=uuid.uuid4
    )
    user_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("users.id", ondelete="CASCADE"), nullable=False
    )
    tag_name: Mapped[str] = mapped_column(Text, nullable=False)
    internal_id: Mapped[str | None] = mapped_column(Text)
    created_at: Mapped[datetime] = mapped_column(server_default=func.now())

    user: Mapped[User] = relationship(back_populates="tags")


class AuditLog(Base):
    __tablename__ = "audit_logs"

    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)
    user_sub: Mapped[str] = mapped_column(Text, nullable=False)
    username: Mapped[str | None] = mapped_column(Text)
    role: Mapped[str | None] = mapped_column(Text)
    action: Mapped[str] = mapped_column(Text, nullable=False)
    resource: Mapped[str | None] = mapped_column(Text)
    result: Mapped[str] = mapped_column(
        Text, CheckConstraint("result IN ('success', 'denied', 'error')"), nullable=False
    )
    details: Mapped[dict[str, Any] | None] = mapped_column(JSONB)
    created_at: Mapped[datetime] = mapped_column(server_default=func.now())


# ---------------------------------------------------------------------------
# Pydantic schemas
# ---------------------------------------------------------------------------

class Permissions(BaseModel):
    can_create_short_url: bool
    can_edit_own_links: bool
    can_delete_own_links: bool
    can_manage_own_tags: bool
    can_view_audit_logs: bool
    can_manage_users: bool

    model_config = ConfigDict(populate_by_name=True)


def compute_permissions(role: str) -> Permissions:
    is_admin = role == Role.ADMIN
    return Permissions(
        can_create_short_url=True,
        can_edit_own_links=True,
        can_delete_own_links=True,
        can_manage_own_tags=True,
        can_view_audit_logs=is_admin,
        can_manage_users=is_admin,
    )


class UserResponse(BaseModel):
    """Public user profile (no shlink_api_key)."""
    model_config = ConfigDict(from_attributes=True)

    id: uuid.UUID
    sub: str
    username: str
    email: str
    role: str
    slug_prefix: str | None
    status: str
    created_at: datetime
    updated_at: datetime
    permissions: Permissions | None = None


class UserUpdateRequest(BaseModel):
    role: str | None = None
    status: str | None = None


class ApiKeyUpdateRequest(BaseModel):
    api_key: str


class PrefixUpdateRequest(BaseModel):
    prefix: str


class AuditLogResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    user_sub: str
    username: str | None
    role: str | None
    action: str
    resource: str | None
    result: str
    details: dict[str, Any] | None
    created_at: datetime
