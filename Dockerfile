# OCX Protocol - Production Dockerfile
# Multi-stage build for minimal image size

# ============================================
# Stage 1: Build
# ============================================
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy all source code
COPY . .

# Downgrade Go version and x/time to be compatible with Go 1.23, then build
RUN go mod edit -go=1.23 -require golang.org/x/time@v0.5.0 && \
    sed -i '/^toolchain/d' go.mod && \
    go mod tidy && \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/bin/ocx-server ./cmd/server

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
