# OCX Protocol - Production Dockerfile
# Multi-stage build for minimal image size

# ============================================
# Stage 1: Build
# ============================================
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/bin/ocx-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/bin/ocx-cli ./cmd/cli
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/bin/ocx-worker ./cmd/worker

# ============================================
# Stage 2: Runtime
# ============================================
FROM alpine:3.19 AS runtime

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata bash

# Create non-root user
RUN addgroup -g 1000 ocx && \
    adduser -u 1000 -G ocx -s /bin/sh -D ocx

# Create directories
RUN mkdir -p /app/data /app/logs /app/config && \
    chown -R ocx:ocx /app

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/bin/ocx-server /app/bin/
COPY --from=builder /build/bin/ocx-cli /app/bin/
COPY --from=builder /build/bin/ocx-worker /app/bin/

# Copy config templates
COPY --from=builder /build/config/*.yaml /app/config/

# Set permissions
RUN chmod +x /app/bin/*

# Switch to non-root user
USER ocx

# Environment variables
ENV OCX_DATA_DIR=/app/data
ENV OCX_LOG_DIR=/app/logs
ENV OCX_CONFIG_DIR=/app/config
ENV OCX_STRICT_MODE=true
ENV GIN_MODE=release

# Expose ports
EXPOSE 8080 9090 50051

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# Default command
CMD ["/app/bin/ocx-server"]
