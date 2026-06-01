"""Tests for GET /healthz endpoint."""
import pytest


@pytest.mark.asyncio
async def test_healthz_returns_200(client):
    response = await client.get("/healthz")
    assert response.status_code == 200


@pytest.mark.asyncio
async def test_healthz_body(client):
    response = await client.get("/healthz")
    data = response.json()
    assert data["status"] == "ok"


@pytest.mark.asyncio
async def test_healthz_no_auth_required(client):
    """Health endpoint must NOT require authentication headers."""
    response = await client.get("/healthz")  # no OIDC headers
    assert response.status_code == 200


@pytest.mark.asyncio
async def test_healthz_content_type(client):
    response = await client.get("/healthz")
    assert "application/json" in response.headers["content-type"]
