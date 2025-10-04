# OCX Protocol: Deterministic Execution with Cryptographic Proofs

## 🎯 **What is OCX Protocol?**

OCX Protocol is a revolutionary system that provides **mathematical proof of execution authenticity**. It solves the fundamental trust problem in computing by creating tamper-proof certificates for software execution results.

**Think of it as a "digital notary" for software execution.**

## 🚀 **Quick Start - 60 Second "Prove It"**

### 1) Build
```bash
go build -o server ./cmd/server
go build -o verify-standalone ./cmd/tools/verify-standalone
```

### 2) Run demo
```bash
OCX_API_KEY=prod-ocx-key OCX_PORT=9001 demo/DEMO.sh
```

**Expected output:**
- ✅ Server starts on port 9001
- ✅ Two identical stdout hashes (deterministic execution)
- ✅ Receipt verification: `verified=true`
- ✅ Tamper detection: `verified=false`
- ✅ Final message: `DEMO ✅`

### 3) Manual verification (if demo fails)
```bash
# Start server
OCX_API_KEY=prod-ocx-key OCX_DISABLE_DB=true OCX_PORT=9001 ./server &

# Test execution
HASH="f8a4e38a18001ece6fe697503722847e18b241a74543bf25a6ffee3733a38a6c"
INPUT_HEX=$(printf "demo" | xxd -p -c 256)
curl -H "X-API-Key: prod-ocx-key" -H "Content-Type: application/json" \
  -d "{\"artifact_hash\":\"$HASH\",\"input\":\"$INPUT_HEX\"}" \
  http://127.0.0.1:9001/api/v1/execute | jq -r '.receipt_b64' | base64 -d > /tmp/rcpt.cbor

# Verify receipt
PUBHEX="8d42905855072bf422f91315db2009c372700fa0345980ae5a7af9c098941cdf"
printf "%s" "$PUBHEX" | xxd -r -p | base64 > /tmp/pub.b64
./verify-standalone /tmp/rcpt.cbor /tmp/pub.b64
```

### Prerequisites
- Go 1.18+
- Linux environment (Ubuntu 22.04+ recommended)
- Docker (optional, for PostgreSQL)

### Generate Keys
```bash
mkdir -p keys
openssl genpkey -algorithm ed25519 -out keys/ocx_signing.pem
openssl pkey -in keys/ocx_signing.pem -pubout -outform DER | tail -c 32 | base64 -w0 > keys/ocx_public.b64
```

### Run Complete Smoke Test
```bash
chmod +x scripts/smoke.sh
./scripts/smoke.sh
```

## 🔧 **Core Components**

### 1. Deterministic Virtual Machine (D-MVM)
- **Purpose**: Executes code in a completely predictable way
- **Key Feature**: Same input + same code = identical output, every time
- **Files**: `pkg/deterministicvm/`

### 2. Cryptographic Receipt System
- **Purpose**: Creates tamper-proof certificates for execution results
- **Key Feature**: Mathematical proof of execution authenticity
- **Files**: `pkg/receipt/`

### 3. Security Sandboxing
- **Purpose**: Prevents unauthorized system access
- **Key Feature**: Ensures execution can't be influenced externally
- **Files**: `pkg/security/`, `pkg/deterministicvm/seccomp.go`

### 4. API Server
- **Purpose**: HTTP interface for remote execution
- **Key Feature**: Production-ready REST API
- **Files**: `cmd/server/`

## 📖 **Usage Examples**

### Command Line Interface
```bash
# Execute a program
./ocx execute my_program.sh --env INPUT="test data"

# Generate a receipt
./ocx execute my_program.sh --output receipt.cbor

# Verify a receipt
./verify-standalone receipt.cbor "$(cat keys/ocx_public.b64)"
```

### API Usage
```bash
# Start server
OCX_API_KEYS=dev123 ./server

# Execute via API
curl -X POST -H "OCX-API-Key: dev123" \
  -H "Content-Type: application/json" \
  --data '{"artifact_hash":"abc123","input":"test"}' \
  http://localhost:8080/api/v1/execute

# Verify via API
curl -X POST -H "OCX-API-Key: dev123" \
  -H "X-OCX-Public-Key: $(cat keys/ocx_public.b64)" \
  -H "Content-Type: application/cbor" \
  --data-binary @receipt.cbor \
  http://localhost:8080/api/v1/verify
```

## 🎯 **Value Propositions**

### 1. AI/ML Model Verification
- **Problem**: "Did this AI model really produce this result?"
- **Solution**: Cryptographic proof of model execution
- **Value**: Prevents AI hallucination claims, ensures model integrity

### 2. Financial Calculations
- **Problem**: "Was this trading algorithm executed correctly?"
- **Solution**: Tamper-proof execution receipts
- **Value**: Regulatory compliance, audit trails, dispute resolution

### 3. Scientific Computing
- **Problem**: "Can I reproduce this research result?"
- **Solution**: Deterministic execution with cryptographic proof
- **Value**: Scientific reproducibility, peer review validation

### 4. Smart Contracts
- **Problem**: "Did this blockchain transaction execute as intended?"
- **Solution**: Off-chain execution with on-chain verification
- **Value**: Reduced gas costs, increased computation power

## 🏗️ **Architecture**

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client App    │───▶│   OCX Server     │───▶│   D-MVM Engine  │
│                 │    │                  │    │                 │
│ - Submit job    │    │ - API Gateway    │    │ - Sandboxed     │
│ - Get receipt   │    │ - Auth/Rate Limit│    │   execution     │
│ - Verify result │    │ - Receipt Store  │    │ - Deterministic │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │   PostgreSQL     │
                       │                  │
                       │ - Receipts       │
                       │ - Idempotency    │
                       │ - Audit Logs     │
                       └──────────────────┘
```

## 🔐 **Security Features**

- **Seccomp Sandboxing**: Restricts system calls
- **Cgroup Limits**: Prevents resource exhaustion
- **Ed25519 Signatures**: Cryptographic authenticity
- **Canonical CBOR**: Consistent serialization
- **Domain Separation**: Prevents signature reuse

## 📊 **Performance**

- **Execution Overhead**: <1ms for simple programs
- **Receipt Generation**: ~600µs
- **Verification**: ~670µs
- **Determinism**: 100% consistent across runs

## 🚀 **Production Deployment**

### Docker
```bash
docker build -t ocx-protocol .
docker run -p 8080:8080 -e OCX_API_KEYS=your-key ocx-protocol
```

### Kubernetes
```bash
kubectl apply -f k8s/
```

### Environment Variables
- `OCX_API_KEYS`: Comma-separated API keys
- `OCX_DB_URL`: Database connection string
- `OCX_SIGNING_KEY_PEM`: Path to signing key
- `OCX_LOG_LEVEL`: Logging level (debug, info, warn, error)

## 🔍 **Verification Process**

1. **Receipt Creation**: System signs execution result with private key
2. **Receipt Storage**: Receipt stored in database with metadata
3. **Independent Verification**: Anyone can verify with public key
4. **Mathematical Proof**: Ed25519 signature provides cryptographic certainty

## 📈 **Monitoring**

- **Health Endpoints**: `/livez`, `/readyz`
- **Metrics**: `/metrics` (Prometheus format)
- **Audit Logs**: All API calls logged for compliance
- **Performance**: Execution time, memory usage, gas consumption

## 🤝 **Contributing**

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 **License**

MIT License - see LICENSE file for details

## 🆘 **Support**

- **Documentation**: This README and inline code comments
- **Issues**: GitHub Issues for bug reports
- **Discussions**: GitHub Discussions for questions

---

**OCX Protocol: Mathematical Proof of Execution Authenticity**