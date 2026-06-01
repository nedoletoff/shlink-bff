"""SQLAlchemy ORM models and Pydantic response schemas."""
from __future__ import annotations

import uuid
from datetime import datetime
from enum import StrEnum

from pydantic import BaseModel, ConfigDict
from sqlalchemy import (
    BigInteger,
    CheckConstraint,
    ForeignKey,
    Index,
    String,
    Text,
    func,
)
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


def _new_uuid() -> str:
    return str(uuid.uuid4())


class User(Base):
    __tablename__ = "users"
    __table_args__ = (
        CheckConstraint("role IN ('admin', 'user')", name="ck_users_role"),
        CheckConstraint(
            "status IN ('active', 'disabled', 'pending')",
            name="ck_users_status",
        ),
        Index("idx_users_sub", "sub", unique=True),
        Index("idx_users_role", "role"),
        Index("idx_users_status", "status"),
    )

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=_new_uuid)
    sub: Mapped[str] = mapped_column(String(512), nullable=False, unique=True)
    username: Mapped[str] = mapped_column(Text, nullable=False)
    email: Mapped[str] = mapped_column(Text, nullable=False)
    display_name: Mapped[str] = mapped_column(Text, nullable=False, default="")
    role: Mapped[str] = mapped_column(String(16), nullable=False, default="user")
    shlink_api_key: Mapped[str] = mapped_column(Text, nullable=False, default="")
    slug_prefix: Mapped[str | None] = mapped_column(String(128))
    status: Mapped[str] = mapped_column(String(16), nullable=False, default="active")
    created_at: Mapped[datetime] = mapped_column(server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(
        server_default=func.now(), onupdate=func.now()
    )
    audit_logs: Mapped[list[AuditLog]] = relationship(
        back_populates="user", cascade="all, delete-orphan"
    )


class UserTag(Base):
    __tablename__ = "user_tags"
    __table_args__ = (
        Index("uq_user_tags_user_tag", "user_id", "tag_name", unique=True),
        Index("idx_user_tags_user_id", "user_id"),
    )

    id: Mapped[str] = mapped_column(String(36), primary_key=True, default=_new_uuid)
    user_id: Mapped[str] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    tag_name: Mapped[str] = mapped_column(String(255), nullable=False)
    internal_id: Mapped[str | None] = mapped_column(String(255))
    created_at: Mapped[datetime] = mapped_column(server_default=func.now())


class AuditLog(Base):
    __tablename__ = "audit_logs"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    user_sub: Mapped[str] = mapped_column(Text, nullable=False)
    username: Mapped[str | None] = mapped_column(Text)
    role: Mapped[str | None] = mapped_column(Text)
    action: Mapped[str] = mapped_column(Text, nullable=False)
    resource: Mapped[str | None] = mapped_column(Text)
    result: Mapped[str] = mapped_column(String(16), nullable=False)
    details: Mapped[str | None] = mapped_column(Text)  # JSON stored as text
    ip_address: Mapped[str | None] = mapped_column(String(64))
    user_agent: Mapped[str | None] = mapped_column(Text)
    created_at: Mapped[datetime] = mapped_column(server_default=func.now())
    user_id: Mapped[str | None] = mapped_column(ForeignKey("users.id", ondelete="SET NULL"), nullable=True)
    user: Mapped[User | None] = relationship(back_populates="audit_logs")


# ---------------------------------------------------------------------------
# Pydantic schemas
# ---------------------------------------------------------------------------


class UserResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    email: str
    display_name: str = ""
    role: str
    is_active: bool = True
    created_at: datetime
    updated_at: datetime
    permissions: list[str] = []


class UserUpdateRequest(BaseModel):
    role: str | None = None
    status: str | None = None


class ApiKeyUpdateRequest(BaseModel):
    api_key: str


class PrefixUpdateRequest(BaseModel):
    prefix: str | None = None


class AuditLogResponse(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    user_sub: str
    action: str
    resource: str | None = None
    result: str
    details: str | None = None
    created_at: datetime


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def compute_permissions(role: str) -> list[str]:
    """Return permission list based on role string."""
    if role == Role.ADMIN:
        return ["read", "write", "admin"]
    return ["read", "write"]
