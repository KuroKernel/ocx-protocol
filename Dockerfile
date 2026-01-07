# Multi-stage build for OCX Protocol
FROM golang:1.24-alpine AS builder

# Install build dependencies (including CGO dependencies for sqlite3)
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server application
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o server ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh ocx

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Copy database schema
COPY --from=builder /app/database/migrations /app/database/migrations

# Set ownership
RUN chown -R ocx:ocx /app

# Switch to non-root user
USER ocx

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/livez || exit 1

# Default command
CMD ["./server"]