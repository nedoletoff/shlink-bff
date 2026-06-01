# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# Build stage: install dependencies
# ---------------------------------------------------------------------------
FROM python:3.12-slim AS builder

WORKDIR /build

# Install build tools
RUN pip install --no-cache-dir hatchling pip --upgrade

# Copy project manifest and install deps into /install
COPY pyproject.toml ./
RUN pip install --no-cache-dir --prefix=/install .

# ---------------------------------------------------------------------------
# Runtime stage
# ---------------------------------------------------------------------------
FROM python:3.12-slim AS runtime

ARG TARGETARCH

LABEL org.opencontainers.image.title="shlink-bff-python-backend"
LABEL org.opencontainers.image.description="FastAPI unified backend for shlink-bff"

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONPATH=/app

WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /install /usr/local

# Copy application source
COPY app/ ./app/

# Non-root user
RUN addgroup --system --gid 1001 appgroup && \
    adduser --system --uid 1001 --ingroup appgroup --no-create-home appuser
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8080/healthz')"

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080", "--workers", "1", "--log-level", "info"]
