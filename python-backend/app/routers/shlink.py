"""Shlink API proxy with per-user isolation (short-urls & tags)."""

from __future__ import annotations

from typing import Annotated, Any

import httpx
from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import get_settings
from app.database import get_db
from app.dependencies import get_or_create_user, write_audit_log
from app.models import User

router = APIRouter(prefix="/api/shlink", tags=["shlink"])


def _shlink_headers(user: User) -> dict[str, str]:
    """Build headers for internal Shlink API calls."""
    return {"X-Api-Key": user.shlink_api_key}


async def _shlink_request(
    method: str,
    path: str,
    user: User,
    *,
    params: dict | None = None,
    json: Any = None,
) -> Any:
    settings = get_settings()
    url = settings.shlink_internal_url.rstrip("/") + "/rest/v3" + path
    try:
        async with httpx.AsyncClient(timeout=30) as client:
            resp = await client.request(
                method,
                url,
                headers=_shlink_headers(user),
                params=params,
                json=json,
            )
    except httpx.ConnectError as e:
        raise HTTPException(status_code=502, detail="Shlink unavailable") from e
    if resp.status_code >= 400:
        raise HTTPException(status_code=resp.status_code, detail=resp.text)
    return resp.json()


# ---------------------------------------------------------------------------
# Short-URLs
# ---------------------------------------------------------------------------


@router.get("/short-urls")
async def list_short_urls(
    user: Annotated[User, Depends(get_or_create_user)],
    page: int = Query(1, ge=1),
    items_per_page: int = Query(10, ge=1, le=200),
    search_term: str | None = Query(None),
    tags: list[str] = Query(default=[]),
) -> Any:
    """List short URLs for current user (filtered by slug prefix)."""
    settings = get_settings()
    params: dict[str, Any] = {"page": page, "itemsPerPage": items_per_page}
    if search_term:
        params["searchTerm"] = search_term
    if tags:
        params["tags[]"] = tags
    # Apply user slug-prefix filter if feature enabled
    if settings.feature_user_slug_prefix and user.slug_prefix:
        params["searchTerm"] = (params.get("searchTerm") or "") + user.slug_prefix
    return await _shlink_request("GET", "/short-urls", user, params=params)


@router.post("/short-urls", status_code=status.HTTP_201_CREATED)
async def create_short_url(
    body: dict,
    user: Annotated[User, Depends(get_or_create_user)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Any:
    """Create a short URL. Prepend slug prefix if feature enabled."""
    settings = get_settings()
    if settings.feature_user_slug_prefix and user.slug_prefix:
        body.setdefault("customSlug", user.slug_prefix + body.get("customSlug", ""))
    result = await _shlink_request("POST", "/short-urls", user, json=body)
    await write_audit_log(db, user, "create_short_url", result.get("shortCode"), "success")
    return result


@router.patch("/short-urls/{short_code}")
async def update_short_url(
    short_code: str,
    body: dict,
    user: Annotated[User, Depends(get_or_create_user)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Any:
    result = await _shlink_request("PATCH", f"/short-urls/{short_code}", user, json=body)
    await write_audit_log(db, user, "update_short_url", short_code, "success")
    return result


@router.delete("/short-urls/{short_code}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_short_url(
    short_code: str,
    user: Annotated[User, Depends(get_or_create_user)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> None:
    await _shlink_request("DELETE", f"/short-urls/{short_code}", user)
    await write_audit_log(db, user, "delete_short_url", short_code, "success")


# ---------------------------------------------------------------------------
# Tags
# ---------------------------------------------------------------------------


@router.get("/tags")
async def list_tags(
    user: Annotated[User, Depends(get_or_create_user)],
) -> Any:
    return await _shlink_request("GET", "/tags", user)


@router.post("/tags", status_code=status.HTTP_201_CREATED)
async def create_tag(
    body: dict,
    user: Annotated[User, Depends(get_or_create_user)],
) -> Any:
    return await _shlink_request("POST", "/tags", user, json=body)


@router.put("/tags/{tag_name}")
async def rename_tag(
    tag_name: str,
    body: dict,
    user: Annotated[User, Depends(get_or_create_user)],
) -> Any:
    return await _shlink_request("PUT", f"/tags/{tag_name}", user, json=body)


@router.delete("/tags/{tag_name}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_tag(
    tag_name: str,
    user: Annotated[User, Depends(get_or_create_user)],
) -> None:
    await _shlink_request("DELETE", f"/tags/{tag_name}", user)
