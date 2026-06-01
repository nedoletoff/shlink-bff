"""Tests for GET /api/me endpoint."""

import pytest


@pytest.mark.asyncio
async def test_me_returns_401_without_token(client):
    response = await client.get("/api/me")
    assert response.status_code == 401


@pytest.mark.asyncio
async def test_me_returns_200_with_valid_token(client, auth_headers):
    response = await client.get("/api/me", headers=auth_headers)
    assert response.status_code == 200


@pytest.mark.asyncio
async def test_me_body_contains_id(client, auth_headers):
    response = await client.get("/api/me", headers=auth_headers)
    data = response.json()
    assert "id" in data


@pytest.mark.asyncio
async def test_me_body_contains_email(client, auth_headers):
    response = await client.get("/api/me", headers=auth_headers)
    data = response.json()
    assert "email" in data


@pytest.mark.asyncio
async def test_me_body_contains_permissions(client, auth_headers):
    response = await client.get("/api/me", headers=auth_headers)
    data = response.json()
    assert "permissions" in data
    assert isinstance(data["permissions"], list)


@pytest.mark.asyncio
async def test_me_body_contains_role(client, auth_headers):
    response = await client.get("/api/me", headers=auth_headers)
    data = response.json()
    assert "role" in data


@pytest.mark.asyncio
async def test_me_content_type(client, auth_headers):
    response = await client.get("/api/me", headers=auth_headers)
    assert "application/json" in response.headers["content-type"]
