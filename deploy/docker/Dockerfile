# OCX Protocol - Production Dockerfile
# Multi-stage build for minimal image size

# ============================================
# Stage 1: Build
# ============================================
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build server binary (CGO_ENABLED=0 for static binary)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/bin/ocx-server ./cmd/server

# ============================================
# Stage 2: Runtime
# ============================================
FROM alpine:3.19 AS runtime

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 ocx && \
    adduser -u 1000 -G ocx -s /bin/sh -D ocx

# Create directories
RUN mkdir -p /app/bin /app/data /app/logs && \
    chown -R ocx:ocx /app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/ocx-server /app/bin/

# Set permissions
RUN chmod +x /app/bin/ocx-server

# Switch to non-root user
USER ocx

# Environment variables
ENV OCX_DATA_DIR=/app/data
ENV OCX_LOG_DIR=/app/logs
ENV OCX_PORT=8080

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# Default command
CMD ["/app/bin/ocx-server"]
