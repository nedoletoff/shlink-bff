"""GET /api/me — current user profile."""
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
    response = UserResponse.model_validate(user)
    response.permissions = compute_permissions(user.role)
    return response
