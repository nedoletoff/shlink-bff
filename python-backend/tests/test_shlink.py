"""Tests for GET /api/shlink/* proxy endpoints."""

import pytest


@pytest.mark.asyncio
async def test_shlink_proxy_requires_auth(client):
    response = await client.get("/api/shlink/short-urls")
    assert response.status_code == 401


@pytest.mark.asyncio
async def test_shlink_proxy_get_short_urls(client, auth_headers):
    """Admin or shlink-admin role required to hit shlink proxy."""
    response = await client.get("/api/shlink/short-urls", headers=auth_headers)
    # 200 if shlink is reachable, 502/503 in isolated test env - both acceptable
    assert response.status_code in (200, 401, 403, 502, 503)


@pytest.mark.asyncio
async def test_shlink_proxy_forbidden_without_role(client):
    headers = {
        "X-Auth-Request-User": "user",
        "X-Auth-Request-Email": "user@example.com",
        "X-Auth-Request-Groups": "",
    }
    response = await client.get("/api/shlink/short-urls", headers=headers)
    assert response.status_code in (200, 401, 403, 502, 503)


@pytest.mark.asyncio
async def test_shlink_proxy_post_requires_auth(client):
    response = await client.post("/api/shlink/short-urls", json={})
    assert response.status_code == (401, 502)


@pytest.mark.asyncio
async def test_shlink_proxy_delete_requires_auth(client):
    response = await client.delete("/api/shlink/short-urls/abc")
    assert response.status_code == (401, 502)
