# Multi-stage build — финальный образ не содержит Go toolchain
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

WORKDIR /build

# Кешируем зависимости отдельным слоем
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ARG TARGETARCH позволяет собирать под linux/arm64 через --platform
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s -extldflags=-static" \
    -o /unified-backend ./cmd/server

# Финальный образ — минимальный scratch
FROM scratch
COPY --from=builder /unified-backend /unified-backend
# CA-сертификаты для HTTPS-запросов к Keycloak/Shlink
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/unified-backend"]
