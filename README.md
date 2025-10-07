# OCX Protocol

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev)
[![Rust Version](https://img.shields.io/badge/Rust-1.70+-orange?logo=rust)](https://rust-lang.org)

> Deterministic execution with cryptographic receipts for verifiable computation

OCX Protocol provides mathematical proof of execution authenticity through deterministic virtual machines and cryptographic receipts. Execute code, generate tamper-proof certificates, verify results independently.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Documentation](#documentation)
- [Installation](#installation)
- [Usage](#usage)
- [Security](#security)
- [Contributing](#contributing)
- [License](#license)

## Overview

### Core Capabilities

- **Deterministic Execution**: Identical results across all platforms and runs
- **Cryptographic Receipts**: Ed25519-signed execution certificates
- **Offline Verification**: Verify results without network or trusted parties
- **Production Ready**: Rate limiting, security headers, comprehensive monitoring

### Use Cases

- AI/ML model output verification
- Financial calculation audit trails
- Scientific computation reproducibility
- Smart contract off-chain execution
- Regulatory compliance evidence

## Quick Start

### Prerequisites

- Go 1.24+
- Linux x86_64 (Ubuntu 22.04+ recommended)
- Optional: Docker, PostgreSQL

### Build

```bash
go build -o server ./cmd/server
go build -o verify-standalone ./cmd/tools/verify-standalone
```

### Run Demo

```bash
OCX_API_KEY=demo-key OCX_PORT=9001 demo/DEMO.sh
```

Expected output:
- Server starts on port 9001
- Two identical execution results (proving determinism)
- Receipt verification: `verified=true`
- Tamper detection: `verified=false`

### Generate Production Keys

```bash
mkdir -p keys
openssl genpkey -algorithm ed25519 -out keys/ocx_signing.pem
openssl pkey -in keys/ocx_signing.pem -pubout -outform DER | \
  tail -c 32 | base64 -w0 > keys/ocx_public.b64
```

## Architecture

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Client    │───▶│  OCX Server  │───▶│   D-MVM     │
│   Request   │    │   (Go)       │    │   Engine    │
└─────────────┘    └──────────────┘    └─────────────┘
                           │
                           ▼
                   ┌──────────────┐
                   │  PostgreSQL  │
                   │   Receipts   │
                   └──────────────┘
```

### Components

- **API Server** (`cmd/server/`): REST API with authentication, rate limiting, and idempotency
- **D-MVM Engine** (`pkg/deterministicvm/`): Sandboxed deterministic execution environment
- **Receipt System** (`pkg/receipt/`): Cryptographic certificate generation and storage
- **Verifier** (`libocx-verify/`): Standalone Rust library for receipt verification
- **Security** (`pkg/security/`): Rate limiting, request validation, security headers

## Documentation

Comprehensive documentation available in [`docs/`](docs/):

- [White Paper](docs/OCX_PROTOCOL_WHITEPAPER.md) - Technical and business overview
- [Technical Architecture](docs/TECHNICAL_ARCHITECTURE.md) - Detailed system design
- [Deployment Guide](docs/DEPLOYMENT_GUIDE.md) - Production deployment instructions
- [Security Audit](docs/COMPREHENSIVE_AUDIT_REPORT.md) - Security analysis and findings

## Installation

### Docker

```bash
docker build -t ocx-protocol .
docker run -p 8080:8080 \
  -e OCX_API_KEY=your-key \
  -e DATABASE_URL=postgres://... \
  ocx-protocol
```

### Kubernetes

```bash
kubectl apply -f k8s/
```

### From Source

```bash
git clone https://github.com/KuroKernel/ocx-protocol.git
cd ocx-protocol
go build -o server ./cmd/server
./server
```

## Usage

### API Endpoints

#### Execute Code

```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "artifact_hash": "abc123...",
    "input": "base64-encoded-input"
  }'
```

Response:
```json
{
  "receipt_b64": "base64-encoded-receipt",
  "cycles_used": 423,
  "stdout_hash": "sha256-hash"
}
```

#### Verify Receipt

```bash
./verify-standalone receipt.cbor public_key.b64
```

Output:
```
Verification: SUCCESS
Receipt is valid and tamper-proof
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OCX_API_KEY` | API authentication key | Required |
| `OCX_PORT` | Server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | In-memory mode if unset |
| `OCX_SIGNING_KEY_PEM` | Path to Ed25519 signing key | `keys/ocx_signing.pem` |
| `OCX_LOG_LEVEL` | Logging level (debug/info/warn/error) | `info` |
| `OCX_DISABLE_DB` | Disable database (in-memory only) | `false` |

## Security

### Features

- **Seccomp sandboxing**: Restricts system calls to prevent unauthorized access
- **Cgroup limits**: CPU, memory, and PID constraints
- **Rate limiting**: 10 requests/second per client, burst capacity 20
- **Request size limits**: 10MB maximum request body
- **Security headers**: HSTS, CSP, X-Frame-Options, X-XSS-Protection
- **Ed25519 signatures**: 128-bit security level with fast verification
- **Canonical CBOR**: RFC 7049 deterministic encoding

### Reporting Vulnerabilities

Please report security vulnerabilities to: security@ocx.world

## Performance

- **Execution overhead**: <1ms for simple programs
- **Receipt generation**: ~600µs
- **Verification**: ~670µs
- **API latency**: P99 < 20ms
- **Throughput**: 200+ requests/second per node

## Monitoring

- **Health checks**: `/livez`, `/readyz`, `/health`
- **Metrics**: `/metrics` (Prometheus format)
- **Audit logs**: All API calls logged with timestamps and results

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone repository
git clone https://github.com/KuroKernel/ocx-protocol.git
cd ocx-protocol

# Install dependencies
go mod download

# Run tests
go test ./...

# Run smoke tests
./scripts/smoke.sh
```

### Commit Message Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: Add new feature
fix: Fix bug
docs: Update documentation
refactor: Code refactoring
test: Add or update tests
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Links

- **Website**: https://ocx.world
- **Documentation**: https://ocx.world/documentation
- **API Reference**: https://ocx.world/api-reference
- **GitHub**: https://github.com/KuroKernel/ocx-protocol
- **Issues**: https://github.com/KuroKernel/ocx-protocol/issues

## Acknowledgments

Built with:
- [Go](https://go.dev) - Backend server and APIs
- [Rust](https://rust-lang.org) - Verification library
- [Ed25519](https://ed25519.cr.yp.to/) - Cryptographic signatures
- [CBOR](https://cbor.io/) - Receipt encoding format

---

**OCX Protocol**: Mathematical proof for computational integrity
