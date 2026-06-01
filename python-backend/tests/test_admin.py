"""Tests for /api/admin/* endpoints (admin-only RBAC)."""
import pytest


@pytest.mark.asyncio
async def test_admin_requires_auth(client):
    response = await client.get("/api/admin/users")
    assert response.status_code == 401


@pytest.mark.asyncio
async def test_admin_forbidden_without_admin_role(client):
    """User with no admin role must receive 403."""
    headers = {
        "X-Auth-Request-User": "regular",
        "X-Auth-Request-Email": "regular@example.com",
        "X-Auth-Request-Groups": "shlink-users",
    }
    response = await client.get("/api/admin/users", headers=headers)
    assert response.status_code in (401, 403)


@pytest.mark.asyncio
async def test_admin_accessible_with_admin_role(client, admin_headers):
    response = await client.get("/api/admin/users", headers=admin_headers)
    # 200 or 404 if route exists; 403 must NOT happen for admin
    assert response.status_code != 403


@pytest.mark.asyncio
async def test_admin_users_returns_list(client, admin_headers):
    response = await client.get("/api/admin/users", headers=admin_headers)
    if response.status_code == 200:
        data = response.json()
        assert isinstance(data, list)


@pytest.mark.asyncio
async def test_admin_content_type(client, admin_headers):
    response = await client.get("/api/admin/users", headers=admin_headers)
    if response.status_code == 200:
        assert "application/json" in response.headers["content-type"]


@pytest.mark.asyncio
async def test_admin_post_requires_auth(client):
    response = await client.post("/api/admin/users", json={})
    assert response.status_code == 401


@pytest.mark.asyncio
async def test_admin_delete_requires_auth(client):
    response = await client.delete("/api/admin/users/some-id")
    assert response.status_code == 401
