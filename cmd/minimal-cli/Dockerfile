# Multi-stage Dockerfile for OCX Protocol v1.0.0-rc.1 minimal-cli
# Supports cross-platform builds for amd64 and arm64 architectures

# Build stage
FROM golang:1.23-alpine AS builder

# Set build arguments for cross-compilation
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Set build environment
ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

# Build the minimal-cli binary with version information
RUN go build \
    -ldflags "-X ocx.local/internal/version.SpecHash=abc123def456 \
              -X ocx.local/internal/version.Build=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
              -X ocx.local/internal/version.GitCommit=abc123 \
              -X ocx.local/internal/version.GitBranch=main \
              -s -w" \
    -o minimal-cli \
    ./cmd/minimal-cli

# Runtime stage - distroless image for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Copy the binary from builder stage
COPY --from=builder /app/minimal-cli /minimal-cli

# Set the binary as entrypoint
ENTRYPOINT ["/minimal-cli"]

# Default command
CMD ["--help"]
