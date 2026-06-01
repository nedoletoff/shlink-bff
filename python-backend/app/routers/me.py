"""GET /api/me - current user profile."""
from typing import Annotated

from fastapi import APIRouter, Depends

from app.dependencies import get_or_create_user
from app.models import User, UserResponse, compute_permissions

router = APIRouter(prefix="/api", tags=["me"])


@router.get("/me", response_model=UserResponse)
async def get_me(
    user: Annotated[User, Depends(get_or_create_user)],
) -> UserResponse:
    """Return authenticated user profile with computed permissions."""
    return UserResponse.model_validate(
        {
            "id": user.id,
            "email": user.email,
            "display_name": user.display_name,
            "role": user.role,
            "is_active": user.is_active,
            "created_at": user.created_at,
            "updated_at": user.updated_at,
            "permissions": compute_permissions(user.role),
        }
    )
