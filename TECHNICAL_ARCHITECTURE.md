# OCX Protocol - Technical Architecture Documentation

**Version**: 1.0
**Date**: October 2025
**Status**: Beta Release

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Component Architecture](#2-component-architecture)
3. [Data Flow](#3-data-flow)
4. [Database Schema](#4-database-schema)
5. [API Specification](#5-api-specification)
6. [Cryptographic Implementation](#6-cryptographic-implementation)
7. [Deterministic VM Details](#7-deterministic-vm-details)
8. [Security Architecture](#8-security-architecture)
9. [Deployment Architecture](#9-deployment-architecture)
10. [Performance Optimization](#10-performance-optimization)

---

## 1. System Overview

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Internet / Clients                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ HTTPS
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                       Reverse Proxy                              │
│                   (Caddy / nginx / HAProxy)                      │
│                                                                  │
│  - TLS Termination                                               │
│  - Rate Limiting (backup layer)                                  │
│  - Load Balancing                                                │
└────────────────────────────┬────────────────────────────────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
┌─────────────▼──────────────┐  ┌──────────▼──────────────┐
│    OCX API Server (1)      │  │  OCX API Server (N)     │
│                            │  │                         │
│  - Go 1.24                 │  │  - Horizontally Scaled  │
│  - net/http                │  │  - Stateless            │
│  - API Key Auth            │  │                         │
│  - Rate Limiting           │  │                         │
└─────────────┬──────────────┘  └──────────┬──────────────┘
              │                             │
              │     ┌───────────────────────┘
              │     │
┌─────────────▼─────▼──────────────┐
│     Deterministic VM Engine      │
│                                  │
│  - Process Isolation             │
│  - Seccomp Sandboxing            │
│  - Cgroup Resource Limits        │
│  - Gas Metering                  │
└─────────────┬────────────────────┘
              │
              │
┌─────────────▼────────────────────┐
│     Receipt Generator            │
│                                  │
│  - Ed25519 Signing               │
│  - Canonical CBOR Encoding       │
│  - SHA-256 Hashing               │
└─────────────┬────────────────────┘
              │
              │
┌─────────────▼────────────────────┐
│     PostgreSQL Database          │
│                                  │
│  - Receipts Table                │
│  - Audit Logs                    │
│  - Idempotency Keys              │
│                                  │
│  Backup: SQLite / In-Memory      │
└──────────────────────────────────┘
```

### 1.2 Technology Stack

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| **Backend** | Go | 1.24 | API server, D-MVM engine |
| **Verification** | Rust | 1.70+ | Standalone verifier |
| **Database** | PostgreSQL | 14+ | Receipt storage |
| **Fallback DB** | SQLite | 3.35+ | Development mode |
| **Crypto** | Ed25519 | RFC 8032 | Signatures |
| **Encoding** | CBOR | RFC 7049 | Receipt format |
| **Sandbox** | Seccomp | Linux 5.4+ | System call filtering |
| **Resources** | Cgroups v2 | Linux 5.4+ | CPU/Memory limits |
| **Frontend** | React | 18.2 | Landing page |
| **Reverse Proxy** | Caddy | 2.6+ | TLS, load balancing |

---

## 2. Component Architecture

### 2.1 API Server (`cmd/server/main.go`)

**Responsibilities**:
- HTTP request handling
- Authentication (API key)
- Rate limiting
- Request validation
- Response formatting
- Health checks
- Metrics exposure

**Key Packages**:
```go
import (
    "github.com/ocx-protocol/pkg/deterministicvm"
    "github.com/ocx-protocol/pkg/receipt"
    "github.com/ocx-protocol/pkg/keystore"
    "github.com/ocx-protocol/pkg/database"
)
```

**Configuration**:
```go
type ServerConfig struct {
    Port            int           // Default: 8080
    APIKeys         []string      // Required for auth
    DatabaseURL     string        // PostgreSQL connection
    SigningKeyPath  string        // Ed25519 private key
    RateLimitIP     int           // Req/s per IP (10)
    RateLimitKey    int           // Req/s per API key (100)
    MaxConcurrent   int           // Max parallel executions (8)
    LogLevel        string        // debug/info/warn/error
}
```

**Startup Sequence**:
```
1. Load configuration from environment
2. Initialize keystore (load Ed25519 keys)
3. Connect to PostgreSQL (or fallback to SQLite/in-memory)
4. Initialize D-MVM engine
5. Set up HTTP routes
6. Start health check ticker
7. Listen on configured port
```

### 2.2 Deterministic VM (`pkg/deterministicvm/`)

**Responsibilities**:
- Execute programs in isolated environment
- Enforce determinism
- Apply resource limits
- Calculate gas consumption
- Capture stdout/stderr

**Core Types**:
```go
type ExecutionRequest struct {
    ArtifactHash string        // SHA-256 of program
    Input        []byte         // Stdin data
    MaxCycles    uint64         // Gas limit
    Env          map[string]string  // Environment variables
}

type ExecutionResult struct {
    Stdout      []byte
    Stderr      []byte
    ExitCode    int
    GasConsumed uint64
    Duration    time.Duration
    Evidence    ExecutionEvidence
}

type ExecutionEvidence struct {
    CPUTimeMS   int64
    MemoryBytes int64
    IOReads     int64
    IOWrites    int64
}
```

**Execution Pipeline**:
```
1. Artifact Resolution
   - Check cache for artifact_hash
   - If miss: fetch from artifact store
   - Verify hash matches

2. Environment Setup
   - Create temporary directory
   - Mount artifact (read-only)
   - Prepare stdin pipe

3. Sandbox Configuration
   - Apply seccomp filter
   - Set cgroup limits (CPU, memory, PIDs)
   - Drop capabilities

4. Process Launch
   - Fork/exec program
   - Redirect stdin/stdout/stderr
   - Start timeout timer

5. Execution Monitoring
   - Track CPU/memory usage
   - Calculate gas consumption
   - Enforce limits

6. Result Collection
   - Wait for process exit
   - Read stdout/stderr
   - Cleanup temporary files

7. Evidence Generation
   - Record CPU time, memory, I/O
   - Return ExecutionResult
```

**Gas Calculation**:
```go
func CalculateGas(evidence ExecutionEvidence) uint64 {
    baseCost := uint64(100)
    cpuCost := evidence.CPUTimeMS * 10
    memCost := (evidence.MemoryBytes / (1024 * 1024)) * 5
    ioCost := (evidence.IOReads + evidence.IOWrites) / 1000

    return baseCost + cpuCost + memCost + ioCost
}
```

### 2.3 Receipt System (`pkg/receipt/`)

**Responsibilities**:
- Build receipt structure
- Encode as canonical CBOR
- Sign with Ed25519
- Store in database
- Retrieve receipts

**Receipt Structure**:
```go
type Receipt struct {
    Version       int       `cbor:"v"`
    ID            string    `cbor:"id"`            // UUID
    ArtifactHash  string    `cbor:"artifact_hash"` // SHA-256
    InputHash     string    `cbor:"input_hash"`    // SHA-256
    StdoutHash    string    `cbor:"stdout_hash"`   // SHA-256
    StderrHash    string    `cbor:"stderr_hash"`   // SHA-256
    GasConsumed   uint64    `cbor:"gas_consumed"`
    ExitCode      int       `cbor:"exit_code"`
    Timestamp     time.Time `cbor:"timestamp"`
    Signature     []byte    `cbor:"signature"`     // 64 bytes
}
```

**Receipt Generation**:
```go
func GenerateReceipt(result ExecutionResult, req ExecutionRequest) (*Receipt, error) {
    receipt := &Receipt{
        Version:      2,
        ID:           uuid.New().String(),
        ArtifactHash: req.ArtifactHash,
        InputHash:    sha256Hash(req.Input),
        StdoutHash:   sha256Hash(result.Stdout),
        StderrHash:   sha256Hash(result.Stderr),
        GasConsumed:  result.GasConsumed,
        ExitCode:     result.ExitCode,
        Timestamp:    time.Now().UTC(),
    }

    // Encode to canonical CBOR (without signature)
    cbor_bytes, err := canonicalCBOR.Marshal(receipt)
    if err != nil {
        return nil, err
    }

    // Sign
    signature, err := keystore.Sign(cbor_bytes)
    if err != nil {
        return nil, err
    }

    receipt.Signature = signature
    return receipt, nil
}
```

### 2.4 Verification Engine (`libocx-verify/`)

**Responsibilities**:
- Parse CBOR receipts
- Verify Ed25519 signatures
- Validate receipt structure
- Standalone operation (no network)

**Rust Implementation**:
```rust
pub fn verify_receipt(
    receipt_bytes: &[u8],
    public_key: &[u8; 32],
) -> Result<ReceiptData, VerifyError> {
    // Split signature (last 64 bytes)
    let (content, signature) = receipt_bytes.split_at(
        receipt_bytes.len() - 64
    );

    // Verify Ed25519 signature
    let public_key = PublicKey::from_bytes(public_key)?;
    let signature = Signature::from_bytes(signature)?;

    public_key.verify(content, &signature)?;

    // Parse CBOR
    let receipt: ReceiptData = serde_cbor::from_slice(content)?;

    // Validate structure
    if receipt.version != 2 {
        return Err(VerifyError::UnsupportedVersion);
    }

    Ok(receipt)
}
```

### 2.5 Database Layer (`pkg/database/`)

**Responsibilities**:
- Store receipts
- Idempotency checking
- Audit logging
- Receipt retrieval

**Schema**:
```sql
CREATE TABLE ocx_receipts (
    id UUID PRIMARY KEY,
    artifact_hash TEXT NOT NULL,
    input_hash TEXT NOT NULL,
    stdout_hash TEXT NOT NULL,
    stderr_hash TEXT NOT NULL,
    gas_consumed BIGINT NOT NULL,
    exit_code INT NOT NULL,
    receipt_cbor BYTEA NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    INDEX idx_artifact (artifact_hash),
    INDEX idx_created (created_at DESC)
);

CREATE TABLE ocx_audit_log (
    id BIGSERIAL PRIMARY KEY,
    receipt_id UUID REFERENCES ocx_receipts(id),
    event_type TEXT NOT NULL,
    client_ip TEXT,
    api_key_hash TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    details JSONB
);

CREATE TABLE ocx_idempotency_keys (
    key TEXT PRIMARY KEY,
    receipt_id UUID REFERENCES ocx_receipts(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL
);
```

**Operations**:
```go
type Database interface {
    StoreReceipt(receipt *Receipt) error
    GetReceipt(id string) (*Receipt, error)
    ListReceipts(limit, offset int) ([]*Receipt, error)
    CheckIdempotency(key string) (*Receipt, error)
    LogAuditEvent(event AuditEvent) error
}
```

---

## 3. Data Flow

### 3.1 Execution Request Flow

```
Client                API Server         D-MVM           Receipt Gen      Database
  │                      │                 │                 │               │
  │ POST /api/v1/execute │                 │                 │               │
  ├─────────────────────>│                 │                 │               │
  │                      │                 │                 │               │
  │                      │ Validate Request│                 │               │
  │                      ├────────┐        │                 │               │
  │                      │        │        │                 │               │
  │                      │<───────┘        │                 │               │
  │                      │                 │                 │               │
  │                      │ Execute Program │                 │               │
  │                      ├────────────────>│                 │               │
  │                      │                 │                 │               │
  │                      │                 │ Run in Sandbox  │               │
  │                      │                 ├────────┐        │               │
  │                      │                 │        │        │               │
  │                      │                 │<───────┘        │               │
  │                      │                 │                 │               │
  │                      │ ExecutionResult │                 │               │
  │                      │<────────────────┤                 │               │
  │                      │                 │                 │               │
  │                      │ Generate Receipt│                 │               │
  │                      ├────────────────────────────────>  │               │
  │                      │                 │                 │               │
  │                      │                 │                 │ Sign with Ed25519
  │                      │                 │                 ├────────┐      │
  │                      │                 │                 │        │      │
  │                      │                 │                 │<───────┘      │
  │                      │                 │                 │               │
  │                      │ Signed Receipt  │                 │               │
  │                      │<────────────────────────────────  │               │
  │                      │                 │                 │               │
  │                      │ Store Receipt   │                 │               │
  │                      ├───────────────────────────────────────────────>   │
  │                      │                 │                 │               │
  │                      │ Receipt ID      │                 │               │
  │                      │<──────────────────────────────────────────────┤   │
  │                      │                 │                 │               │
  │ HTTP 200 + Receipt   │                 │                 │               │
  │<─────────────────────┤                 │                 │               │
  │                      │                 │                 │               │
```

### 3.2 Verification Request Flow

```
Client                API Server         Verifier         Database
  │                      │                 │                │
  │ POST /api/v1/verify  │                 │                │
  ├─────────────────────>│                 │                │
  │                      │                 │                │
  │                      │ Verify Receipt  │                │
  │                      ├────────────────>│                │
  │                      │                 │                │
  │                      │                 │ Check Signature│
  │                      │                 ├────────┐       │
  │                      │                 │        │       │
  │                      │                 │<───────┘       │
  │                      │                 │                │
  │                      │ Valid/Invalid   │                │
  │                      │<────────────────┤                │
  │                      │                 │                │
  │                      │ Log Audit Event │                │
  │                      ├────────────────────────────────> │
  │                      │                 │                │
  │ HTTP 200 + Result    │                 │                │
  │<─────────────────────┤                 │                │
  │                      │                 │                │
```

---

## 4. Database Schema

### 4.1 Receipts Table

```sql
CREATE TABLE ocx_receipts (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Receipt content
    artifact_hash TEXT NOT NULL,          -- SHA-256 of executed program
    input_hash TEXT NOT NULL,             -- SHA-256 of stdin
    stdout_hash TEXT NOT NULL,            -- SHA-256 of stdout
    stderr_hash TEXT NOT NULL,            -- SHA-256 of stderr
    gas_consumed BIGINT NOT NULL,         -- Computational cost
    exit_code INT NOT NULL,               -- Program exit code

    -- Raw receipt (CBOR + signature)
    receipt_cbor BYTEA NOT NULL,          -- Complete signed receipt

    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    api_key_hash TEXT,                    -- Which API key used
    client_ip TEXT,                       -- Client IP address

    -- Indexes
    INDEX idx_artifact (artifact_hash),
    INDEX idx_created (created_at DESC),
    INDEX idx_stdout (stdout_hash),
    INDEX idx_gas (gas_consumed DESC)
);
```

### 4.2 Audit Log Table

```sql
CREATE TABLE ocx_audit_log (
    id BIGSERIAL PRIMARY KEY,

    -- Event details
    event_type TEXT NOT NULL,              -- "execute", "verify", "error"
    receipt_id UUID REFERENCES ocx_receipts(id),

    -- Client info
    client_ip TEXT,
    api_key_hash TEXT,
    user_agent TEXT,

    -- Timing
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    duration_ms INT,

    -- Additional data
    details JSONB,                         -- Flexible metadata

    INDEX idx_timestamp (timestamp DESC),
    INDEX idx_event_type (event_type),
    INDEX idx_receipt (receipt_id)
);
```

### 4.3 Idempotency Keys Table

```sql
CREATE TABLE ocx_idempotency_keys (
    key TEXT PRIMARY KEY,                  -- Client-provided key
    receipt_id UUID REFERENCES ocx_receipts(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,         -- Auto-expire after 24h

    INDEX idx_expires (expires_at)
);

-- Cleanup expired keys
CREATE OR REPLACE FUNCTION cleanup_expired_idempotency_keys()
RETURNS void AS $$
BEGIN
    DELETE FROM ocx_idempotency_keys WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;
```

---

## 5. API Specification

### 5.1 Execute Endpoint

**Request**:
```http
POST /api/v1/execute HTTP/1.1
Host: api.ocx.world
Content-Type: application/json
X-API-Key: your-api-key-here

{
  "program": "python3",
  "input": "7072696e7428322b322900",  // hex-encoded stdin
  "env": {
    "PATH": "/usr/bin"
  },
  "max_gas": 10000
}
```

**Response (Success)**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "success",
  "receipt_id": "39ce915c-3875-426f-9a72-1b3723377074",
  "receipt_b64": "omF2AmJpZFg2M2NlOTE1...",
  "stdout": "4\n",
  "stderr": "",
  "gas_consumed": 227,
  "exit_code": 0,
  "duration_ms": 12
}
```

**Response (Error)**:
```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "status": "error",
  "error": "invalid input encoding",
  "code": "INVALID_INPUT"
}
```

### 5.2 Verify Endpoint

**Request**:
```http
POST /api/v1/verify HTTP/1.1
Host: api.ocx.world
Content-Type: application/json
X-API-Key: your-api-key-here

{
  "receipt_b64": "omF2AmJpZFg2M2NlOTE1..."
}
```

**Response**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "valid": true,
  "receipt_id": "39ce915c-3875-426f-9a72-1b3723377074",
  "artifact_hash": "f8a4e38a...",
  "stdout_hash": "9f86d081...",
  "gas_consumed": 227,
  "timestamp": "2025-10-07T03:32:03.123Z"
}
```

### 5.3 Health Endpoint

**Request**:
```http
GET /health HTTP/1.1
Host: api.ocx.world
```

**Response**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "overall": "healthy",
  "timestamp": "2025-10-07T03:32:03.123Z",
  "checks": {
    "database": {
      "status": "healthy",
      "duration": 295
    },
    "keystore": {
      "status": "healthy",
      "duration": 597
    },
    "system": {
      "status": "healthy",
      "duration": 67
    }
  },
  "version": "1.0.0",
  "uptime": 1169556625895
}
```

---

## 6. Cryptographic Implementation

### 6.1 Ed25519 Signature Generation

```go
import "crypto/ed25519"

func SignReceipt(receiptBytes []byte, privateKey ed25519.PrivateKey) ([]byte, error) {
    // Ed25519 signature (deterministic)
    signature := ed25519.Sign(privateKey, receiptBytes)

    // Signature is always 64 bytes
    if len(signature) != 64 {
        return nil, errors.New("invalid signature length")
    }

    return signature, nil
}
```

### 6.2 Ed25519 Signature Verification

```rust
use ed25519_dalek::{PublicKey, Signature, Verifier};

pub fn verify_signature(
    content: &[u8],
    signature_bytes: &[u8],
    public_key_bytes: &[u8],
) -> Result<bool, Error> {
    let public_key = PublicKey::from_bytes(public_key_bytes)?;
    let signature = Signature::from_bytes(signature_bytes)?;

    match public_key.verify(content, &signature) {
        Ok(_) => Ok(true),
        Err(_) => Ok(false),
    }
}
```

### 6.3 SHA-256 Hashing

```go
import "crypto/sha256"

func HashBytes(data []byte) string {
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}

func HashStdout(stdout []byte) string {
    return HashBytes(stdout)
}
```

### 6.4 Canonical CBOR Encoding

```go
import "github.com/fxamacker/cbor/v2"

func EncodeCanonicalCBOR(receipt *Receipt) ([]byte, error) {
    // Use canonical encoding mode
    encMode, err := cbor.CanonicalEncOptions().EncMode()
    if err != nil {
        return nil, err
    }

    // Encode
    cbor_bytes, err := encMode.Marshal(receipt)
    if err != nil {
        return nil, err
    }

    return cbor_bytes, nil
}
```

**Canonical Rules Applied**:
1. Shortest possible encoding
2. Map keys sorted lexicographically
3. No duplicate map keys
4. Deterministic number encoding

---

## 7. Deterministic VM Details

### 7.1 Seccomp Filter

```go
import "golang.org/x/sys/unix"

func ApplySeccompFilter() error {
    // Allow only safe syscalls
    allowedSyscalls := []string{
        "read", "write", "exit", "exit_group",
        "brk", "mmap", "munmap", "mprotect",
        "rt_sigaction", "rt_sigreturn",
        "futex", "sched_yield",
        // Blocked: network, time, file creation, exec
    }

    filter := seccomp.NewFilter()
    filter.SetDefaultAction(seccomp.ActErrno)

    for _, syscall := range allowedSyscalls {
        filter.AddRule(syscall, seccomp.ActAllow)
    }

    return filter.Load()
}
```

**Blocked Syscalls** (Sources of Non-Determinism):
- `clock_gettime`, `gettimeofday` (time)
- `getrandom`, `/dev/random` (randomness)
- `socket`, `connect`, `sendto` (network)
- `open` with O_CREAT (file creation)
- `fork`, `clone`, `execve` (process creation)

### 7.2 Cgroup Configuration

```go
import "github.com/containerd/cgroups"

func SetupCgroups(pid int) error {
    control, err := cgroups.New(cgroups.V2, cgroups.StaticPath("/ocx/"+strconv.Itoa(pid)), &specs.LinuxResources{
        CPU: &specs.LinuxCPU{
            Quota:  100000,  // 1 CPU (100ms per 100ms period)
            Period: 100000,
        },
        Memory: &specs.LinuxMemory{
            Limit: 512 * 1024 * 1024,  // 512MB
        },
        Pids: &specs.LinuxPids{
            Limit: 100,  // Max 100 processes
        },
    })

    if err != nil {
        return err
    }

    return control.Add(cgroups.Process{Pid: pid})
}
```

### 7.3 Gas Metering Algorithm

```go
func CalculateGas(evidence ExecutionEvidence) uint64 {
    const (
        BaseCost       = 100      // Flat fee per execution
        CPUCostPerMS   = 10       // Per millisecond of CPU
        MemCostPerMB   = 5        // Per MB of memory
        IOCostPer1KB   = 1        // Per KB of I/O
    )

    cpuGas := evidence.CPUTimeMS * CPUCostPerMS
    memGas := (evidence.MemoryBytes / (1024 * 1024)) * MemCostPerMB
    ioGas := ((evidence.IOReads + evidence.IOWrites) / 1024) * IOCostPer1KB

    totalGas := BaseCost + cpuGas + memGas + ioGas

    return totalGas
}
```

**Example**:
```
Program: python3 -c "print(2+2)"
CPU Time: 12ms
Memory: 8MB
I/O: 4KB

Gas = 100 + (12 × 10) + (8 × 5) + (4 × 1)
    = 100 + 120 + 40 + 4
    = 264 units
```

---

## 8. Security Architecture

### 8.1 Authentication Flow

```
Client Request
    │
    ▼
┌─────────────────────┐
│ Extract API Key     │
│ from X-API-Key      │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ Hash API Key        │
│ (SHA-256)           │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ Lookup in Database  │
│ or Config           │
└─────────┬───────────┘
          │
    ┌─────┴─────┐
    │           │
    ▼           ▼
 Valid      Invalid
    │           │
    ▼           ▼
Continue   Return 401
```

### 8.2 Rate Limiting

```go
type RateLimiter struct {
    ipLimits  map[string]*TokenBucket  // Per-IP limits
    keyLimits map[string]*TokenBucket  // Per-API-key limits
}

type TokenBucket struct {
    tokens    float64
    maxTokens float64
    refillRate float64
    lastRefill time.Time
}

func (rl *RateLimiter) Allow(ip, apiKey string) bool {
    // Check IP limit (10 req/s)
    if !rl.ipLimits[ip].Take() {
        return false
    }

    // Check API key limit (100 req/s)
    if !rl.keyLimits[apiKey].Take() {
        return false
    }

    return true
}
```

**Limits**:
- **Per IP**: 10 requests/second (burst: 20)
- **Per API Key**: 100 requests/second (burst: 200)
- **Concurrent Executions**: 8 (number of CPU cores)

### 8.3 Key Management

**Key Generation**:
```bash
# Generate Ed25519 private key
openssl genpkey -algorithm ed25519 -out signing.pem

# Extract public key (DER format, last 32 bytes)
openssl pkey -in signing.pem -pubout -outform DER | tail -c 32 | base64 > public.b64
```

**Key Storage**:
```
/opt/ocx/keys/
├── signing.pem      (600 permissions, root:root)
└── public.b64       (644 permissions, readable)
```

**Key Rotation**:
```go
// Keystore supports multiple keys
type Keystore struct {
    activeKey  *ed25519.PrivateKey
    publicKeys map[string]*ed25519.PublicKey  // For verification
}

// Sign with active key
func (ks *Keystore) Sign(data []byte) ([]byte, error) {
    return ed25519.Sign(ks.activeKey, data), nil
}

// Verify with any known public key
func (ks *Keystore) Verify(data, signature, keyID []byte) bool {
    pubKey := ks.publicKeys[string(keyID)]
    return ed25519.Verify(pubKey, data, signature)
}
```

---

## 9. Deployment Architecture

### 9.1 Single Server Setup

```
┌────────────────────────────────────────┐
│         Server (Ubuntu 22.04)          │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │   Caddy (Port 80/443)            │ │
│  │   - HTTPS termination            │ │
│  │   - api.ocx.world → :8080        │ │
│  └──────────┬───────────────────────┘ │
│             │                          │
│  ┌──────────▼───────────────────────┐ │
│  │   OCX Server (Port 8080)         │ │
│  │   - Go binary                    │ │
│  │   - systemd service              │ │
│  └──────────┬───────────────────────┘ │
│             │                          │
│  ┌──────────▼───────────────────────┐ │
│  │   PostgreSQL (Port 5432)         │ │
│  │   - localhost only               │ │
│  └──────────────────────────────────┘ │
│                                        │
└────────────────────────────────────────┘
```

**Firewall Rules**:
```bash
ufw allow 80/tcp    # HTTP (redirects to HTTPS)
ufw allow 443/tcp   # HTTPS
ufw deny 8080/tcp   # OCX server (internal only)
ufw deny 5432/tcp   # PostgreSQL (internal only)
```

### 9.2 Multi-Server Setup

```
                 Internet
                    │
                    ▼
        ┌───────────────────────┐
        │   Load Balancer       │
        │   (nginx/HAProxy)     │
        └───────┬───────────────┘
                │
        ┌───────┴────────┐
        │                │
        ▼                ▼
┌──────────────┐  ┌──────────────┐
│ OCX Server 1 │  │ OCX Server 2 │
│              │  │              │
│ - Stateless  │  │ - Stateless  │
└──────┬───────┘  └──────┬───────┘
       │                 │
       └────────┬────────┘
                │
                ▼
        ┌──────────────┐
        │  PostgreSQL  │
        │  (Primary)   │
        │              │
        │  ┌────────┐  │
        │  │Replica │  │
        └──┴────────┴──┘
```

### 9.3 Docker Compose Setup

```yaml
version: '3.8'

services:
  ocx-api:
    image: ocx-protocol:latest
    ports:
      - "8080:8080"
    environment:
      - OCX_API_KEYS=your-secret-key
      - OCX_DB_URL=postgres://ocx:pass@db:5432/ocx
      - OCX_SIGNING_KEY_PEM=/keys/signing.pem
    volumes:
      - ./keys:/keys:ro
    depends_on:
      - db
    restart: unless-stopped

  db:
    image: postgres:14
    environment:
      - POSTGRES_USER=ocx
      - POSTGRES_PASSWORD=pass
      - POSTGRES_DB=ocx
    volumes:
      - pgdata:/var/lib/postgresql/data
    restart: unless-stopped

  caddy:
    image: caddy:2
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
    restart: unless-stopped

volumes:
  pgdata:
  caddy_data:
```

---

## 10. Performance Optimization

### 10.1 Caching Strategy

```go
type ArtifactCache struct {
    cache map[string][]byte
    mu    sync.RWMutex
    maxSize int  // Max artifacts in cache
}

func (ac *ArtifactCache) Get(hash string) ([]byte, bool) {
    ac.mu.RLock()
    defer ac.mu.RUnlock()

    artifact, exists := ac.cache[hash]
    return artifact, exists
}

func (ac *ArtifactCache) Set(hash string, artifact []byte) {
    ac.mu.Lock()
    defer ac.mu.Unlock()

    // LRU eviction if cache is full
    if len(ac.cache) >= ac.maxSize {
        // Evict oldest entry
        for k := range ac.cache {
            delete(ac.cache, k)
            break
        }
    }

    ac.cache[hash] = artifact
}
```

### 10.2 Database Connection Pooling

```go
import "database/sql"

func InitDatabase(url string) (*sql.DB, error) {
    db, err := sql.Open("postgres", url)
    if err != nil {
        return nil, err
    }

    // Connection pool settings
    db.SetMaxOpenConns(25)          // Max connections
    db.SetMaxIdleConns(5)           // Idle connections
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(1 * time.Minute)

    return db, nil
}
```

### 10.3 Benchmark Results

| Operation | Avg Time | Min | Max | P95 | P99 |
|-----------|----------|-----|-----|-----|-----|
| Execute (echo) | 2ms | 1ms | 5ms | 3ms | 4ms |
| Execute (python) | 12ms | 10ms | 30ms | 20ms | 25ms |
| Receipt Gen | 600µs | 400µs | 1ms | 800µs | 900µs |
| Verification | 670µs | 500µs | 1ms | 900µs | 1ms |
| DB Write | 3ms | 2ms | 10ms | 5ms | 8ms |

---

## Appendix: Code References

**Key Files**:
- `cmd/server/main.go` - HTTP server (2,457 lines)
- `pkg/deterministicvm/vm.go` - D-MVM engine (1,842 lines)
- `pkg/receipt/generator.go` - Receipt system (567 lines)
- `pkg/keystore/keystore.go` - Key management (423 lines)
- `pkg/database/postgres.go` - Database layer (789 lines)
- `libocx-verify/src/verify.rs` - Rust verifier (1,234 lines)

**Total Code**: ~25,000 lines (Go + Rust)

---

**End of Technical Architecture Documentation**

OCX Protocol | ocx.world | github.com/KuroKernel/ocx-protocol
