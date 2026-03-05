# OCX Protocol - Production Dockerfile with VDF (Rust + Go CGO)
# Multi-stage build: Rust library → Go binary → minimal runtime

# ============================================
# Stage 1: Build Rust VDF library
# ============================================
FROM rust:1.83-slim-bookworm AS rust-builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    pkg-config libssl-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY libocx-verify/ ./libocx-verify/

WORKDIR /build/libocx-verify
RUN cargo build --release --features ffi 2>&1 | tail -5

# ============================================
# Stage 2: Build Go server with CGO (links Rust)
# ============================================
FROM golang:1.23-bookworm AS go-builder

# Install C toolchain for CGO
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libc6-dev ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy Rust build artifacts
COPY --from=rust-builder /build/libocx-verify/target/release/liblibocx_verify.a /build/libocx-verify/target/release/
COPY --from=rust-builder /build/libocx-verify/target/release/liblibocx_verify.so /build/libocx-verify/target/release/

# Copy Rust headers
COPY libocx-verify/ocx_verify.h /build/libocx-verify/

# Copy Go source
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build with CGO enabled (links Rust VDF library)
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /build/bin/ocx-server ./cmd/server

# ============================================
# Stage 3: Minimal runtime
# ============================================
FROM debian:bookworm-slim AS runtime

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata wget && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1000 ocx && \
    useradd -u 1000 -g ocx -s /bin/sh -m ocx

# Create directories
RUN mkdir -p /app/bin /app/data /app/logs && \
    chown -R ocx:ocx /app

WORKDIR /app

# Copy binary and Rust shared library
COPY --from=go-builder /build/bin/ocx-server /app/bin/
COPY --from=rust-builder /build/libocx-verify/target/release/liblibocx_verify.so /usr/lib/

RUN ldconfig && chmod +x /app/bin/ocx-server

USER ocx

ENV OCX_DATA_DIR=/app/data
ENV OCX_LOG_DIR=/app/logs
ENV OCX_PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["/app/bin/ocx-server"]
