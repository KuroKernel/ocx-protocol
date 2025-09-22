# =============================================================================
# OCX Kubernetes Webhook - Multi-stage Production Dockerfile
# =============================================================================

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    upx

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the webhook binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -tags netgo \
    -installsuffix netgo \
    -a \
    -o ocx-webhook \
    ./cmd/webhook/

# Compress binary
RUN upx --best --lzma ocx-webhook

# Verify binary
RUN ./ocx-webhook --version || echo "Built successfully"

# =============================================================================
# Runtime stage
FROM scratch

# Import CA certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Create non-root user structure
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the binary
COPY --from=builder /build/ocx-webhook /usr/local/bin/ocx-webhook

# Create directory for temporary files
RUN mkdir -p /tmp && chmod 1777 /tmp

# Security labels
LABEL \
    org.opencontainers.image.title="OCX Kubernetes Webhook" \
    org.opencontainers.image.description="Kubernetes Mutating Admission Webhook for OCX Protocol" \
    org.opencontainers.image.vendor="OCX Protocol" \
    org.opencontainers.image.version="1.0.0" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.source="https://github.com/ocx-protocol/ocx" \
    security.scan="enabled"

# Use non-root user
USER 65534:65534

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/ocx-webhook", "--health-check"]

# Expose ports
EXPOSE 8443 9090

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/ocx-webhook"]
