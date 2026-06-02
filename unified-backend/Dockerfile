# Multi-stage build — финальный образ не содержит Go toolchain
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

WORKDIR /build

COPY . .
RUN go mod tidy

# ARG TARGETARCH позволяет собирать под linux/arm64 через --platform
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s -extldflags=-static" \
    -o /unified-backend ./cmd/server

# Финальный образ — alpine (нужен wget для healthcheck в docker-compose)
FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /unified-backend /unified-backend
ENTRYPOINT ["/unified-backend"]
