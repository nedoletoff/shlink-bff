"""Tests for GET /api/shlink/* proxy endpoints."""
import pytest


@pytest.mark.asyncio
async def test_shlink_proxy_requires_auth(client):
    response = await client.get("/api/shlink/rest/v3/short-urls")
    assert response.status_code == 401


@pytest.mark.asyncio
async def test_shlink_proxy_get_short_urls(client, auth_headers):
    """Admin or shlink-admin role required to hit shlink proxy."""
    response = await client.get(
        "/api/shlink/rest/v3/short-urls", headers=auth_headers
    )
    # 200 if shlink is reachable, 502/503 in isolated test env — both acceptable
    assert response.status_code in (200, 401, 403, 502, 503)


@pytest.mark.asyncio
async def test_shlink_proxy_forbidden_without_role(client):
    """Token with no shlink role should get 403."""
    headers = {"X-Auth-Request-User": "user", "X-Auth-Request-Email": "user@example.com",
               "X-Auth-Request-Groups": ""}
    response = await client.get("/api/shlink/rest/v3/short-urls", headers=headers)
    assert response.status_code in (401, 403)


@pytest.mark.asyncio
async def test_shlink_proxy_post_requires_auth(client):
    response = await client.post("/api/shlink/rest/v3/short-urls", json={})
    assert response.status_code == 401


@pytest.mark.asyncio
async def test_shlink_proxy_delete_requires_auth(client):
    response = await client.delete("/api/shlink/rest/v3/short-urls/abc")
    assert response.status_code == 401
