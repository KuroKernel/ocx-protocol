# Multi-stage build for minimal production image
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build optimized binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -extldflags=-static" -trimpath -o ocx-server ./cmd/minimal-server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -extldflags=-static" -trimpath -o ocx-cli ./cmd/minimal-cli

# Production stage
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binaries
COPY --from=builder /build/ocx-server /ocx-server
COPY --from=builder /build/ocx-cli /ocx-cli

# Expose port
EXPOSE 8080

# Run server
ENTRYPOINT ["/ocx-server"]
