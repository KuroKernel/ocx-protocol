# Multi-stage build for OCX Protocol
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh ocx

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/ocx /app/ocx
COPY --from=builder /app/server /app/server
COPY --from=builder /app/verify-standalone /app/verify-standalone

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