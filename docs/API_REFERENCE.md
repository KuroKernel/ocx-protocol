# OCX Protocol API Reference

Complete API documentation for OCX Protocol v1.0.

## Table of Contents

- [Authentication](#authentication)
- [Receipts API](#receipts-api)
- [Verification API](#verification-api)
- [Merkle Tree API](#merkle-tree-api)
- [Execution API](#execution-api)
- [Health & Monitoring](#health--monitoring)
- [Error Handling](#error-handling)

---

## Authentication

All API endpoints (except health checks) require authentication via API key.

### API Key Header

```http
X-API-Key: your-api-key-here
```

### Query Parameter (Alternative)

```
GET /api/v1/endpoint?api_key=your-api-key-here
```

---

## Receipts API

### Create Receipt

Create and sign a new computation receipt.

**Endpoint:** `POST /api/v1/receipts`

**Request Body:**

```json
{
  "program_hash": "64-char-hex-sha256",
  "input_hash": "64-char-hex-sha256",
  "output_hash": "64-char-hex-sha256",
  "gas_used": 1000,
  "issuer_id": "my-service-v1",
  "float_mode": "strict",
  "metadata": "optional-metadata"
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| program_hash | string | Yes | SHA256 hash of program/code (64 hex chars) |
| input_hash | string | Yes | SHA256 hash of input data |
| output_hash | string | Yes | SHA256 hash of output data |
| gas_used | integer | Yes | Gas consumed during execution |
| issuer_id | string | Yes | Identifier for the issuing service |
| float_mode | string | No | Float handling: `strict`, `deterministic`, `native` |
| metadata | string | No | Optional metadata string |

**Response (201 Created):**

```json
{
  "receipt": "hex-encoded-receipt-bytes",
  "signature": "hex-encoded-ed25519-signature",
  "public_key": "hex-encoded-ed25519-public-key",
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

### Verify Receipt

Verify a receipt signature.

**Endpoint:** `POST /api/v1/receipts/verify`

**Request Body:**

```json
{
  "receipt": "hex-encoded-receipt",
  "signature": "hex-encoded-signature",
  "public_key": "hex-encoded-public-key"
}
```

**Response (200 OK):**

```json
{
  "valid": true,
  "verified_at": "2024-01-01T00:00:00Z",
  "fields": {
    "program_hash": "abc123...",
    "input_hash": "def456...",
    "output_hash": "789abc...",
    "gas_used": 1000,
    "issuer_id": "my-service",
    "float_mode": "strict"
  }
}
```

---

### Batch Verify Receipts

Verify multiple receipts in a single request.

**Endpoint:** `POST /api/v1/receipts/batch-verify`

**Request Body:**

```json
{
  "receipts": [
    {
      "receipt": "hex-receipt-1",
      "signature": "hex-sig-1",
      "public_key": "hex-pubkey-1"
    },
    {
      "receipt": "hex-receipt-2",
      "signature": "hex-sig-2",
      "public_key": "hex-pubkey-2"
    }
  ]
}
```

**Limits:** Maximum 1000 receipts per batch.

**Response (200 OK):**

```json
{
  "results": [
    { "index": 0, "valid": true },
    { "index": 1, "valid": false, "error": "Signature verification failed" }
  ],
  "total_count": 2,
  "valid_count": 1,
  "invalid_count": 1,
  "duration": "1.234ms"
}
```

---

## Merkle Tree API

### Build Merkle Tree

Build a Merkle tree from a list of items and get proofs.

**Endpoint:** `POST /api/v1/merkle/tree`

**Request Body:**

```json
{
  "items": [
    "hex-encoded-item-1",
    "hex-encoded-item-2",
    "raw-string-item-3"
  ]
}
```

**Limits:** Maximum 10,000 items per tree.

**Response (200 OK):**

```json
{
  "root": "hex-encoded-merkle-root",
  "leaf_count": 3,
  "tree_height": 2,
  "proofs": {
    "0": ["sibling-hash-1", "sibling-hash-2"],
    "1": ["sibling-hash-1", "sibling-hash-2"],
    "2": ["sibling-hash-1", "sibling-hash-2"]
  }
}
```

---

### Verify Merkle Proof

Verify that an item is in a Merkle tree.

**Endpoint:** `POST /api/v1/merkle/verify`

**Request Body:**

```json
{
  "root": "hex-encoded-root",
  "item": "hex-encoded-item",
  "index": 0,
  "proof": ["hex-sibling-1", "hex-sibling-2"]
}
```

**Response (200 OK):**

```json
{
  "valid": true
}
```

---

## Execution API

### Execute WASM Program

Execute a WASM program deterministically and receive a receipt.

**Endpoint:** `POST /api/v1/execute`

**Request Body (Option A - Hash Reference):**

```json
{
  "artifact_hash": "64-char-hex-wasm-hash",
  "input": "hex-encoded-input"
}
```

**Request Body (Option B - Inline Program):**

```json
{
  "program": "base64-encoded-wasm",
  "input": "utf8-or-hex-input"
}
```

**Response (200 OK):**

```json
{
  "output": "hex-encoded-output",
  "output_hash": "sha256-of-output",
  "gas_used": 12345,
  "execution_time_ms": 50,
  "receipt": {
    "receipt": "hex-receipt",
    "signature": "hex-signature",
    "public_key": "hex-pubkey"
  }
}
```

---

### Get Artifact Info

Get information about a stored WASM artifact.

**Endpoint:** `GET /api/v1/artifact/info?hash={artifact_hash}`

**Response (200 OK):**

```json
{
  "hash": "artifact-hash",
  "size": 12345,
  "uploaded_at": "2024-01-01T00:00:00Z",
  "execution_count": 100
}
```

---

## Health & Monitoring

### Health Check

**Endpoint:** `GET /health` or `GET /healthz`

**Response (200 OK):**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "24h5m30s",
  "requests": 123456,
  "goroutines": 50,
  "memory_mb": 128
}
```

### Liveness Probe

**Endpoint:** `GET /livez`

**Response:** `200 OK` with body `live`

### Readiness Probe

**Endpoint:** `GET /readyz`

**Response:** `200 OK` with body `ready`

### Metrics (Prometheus)

**Endpoint:** `GET /metrics`

Returns Prometheus-formatted metrics including:

- `ocx_receipt_created_total` - Receipts created
- `ocx_receipt_verified_total` - Receipts verified
- `ocx_batch_latency_seconds` - Batch verification latency
- `ocx_vm_executions_total` - VM executions
- `ocx_api_requests_total` - API requests by endpoint
- `ocx_system_goroutines` - Goroutine count
- `ocx_system_memory_bytes` - Memory usage

---

## Error Handling

All errors return JSON with the following structure:

```json
{
  "error": "Human-readable error message",
  "code": 400,
  "request_id": "optional-request-id"
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request - Invalid parameters |
| 401 | Unauthorized - Missing or invalid API key |
| 404 | Not Found |
| 429 | Too Many Requests - Rate limited |
| 500 | Internal Server Error |

### Common Errors

**Invalid hash format:**
```json
{
  "error": "Invalid program_hash: must be 64 hex characters",
  "code": 400
}
```

**Rate limited:**
```json
{
  "error": "Rate limit exceeded",
  "code": 429
}
```

**Unauthorized:**
```json
{
  "error": "Unauthorized: invalid or missing API key",
  "code": 401
}
```

---

## Rate Limits

Default rate limits per client IP:

| Tier | Requests/Second | Burst |
|------|-----------------|-------|
| Default | 100 | 200 |
| Premium | 1000 | 2000 |

Rate limit headers:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704067200
```

---

## SDK Examples

### Go

```go
import "ocx.local/pkg/client"

client := client.New("https://ocx.example.com", "api-key")
receipt, err := client.CreateReceipt(ctx, &client.ReceiptRequest{
    ProgramHash: programHash,
    InputHash:   inputHash,
    OutputHash:  outputHash,
    GasUsed:     1000,
    IssuerID:    "my-service",
})
```

### TypeScript

```typescript
import { OCXClient, ReceiptBuilder } from '@ocx-protocol/sdk';

const client = new OCXClient({
    serverUrl: 'https://ocx.example.com',
    apiKey: 'your-api-key',
});

const receipt = new ReceiptBuilder()
    .programHash(programHash)
    .inputHash(inputHash)
    .outputHash(outputHash)
    .gasUsed(1000)
    .issuerId('my-service')
    .build();

const signed = await client.createReceipt(receipt);
```

### cURL

```bash
curl -X POST https://ocx.example.com/api/v1/receipts \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "program_hash": "abc123...",
    "input_hash": "def456...",
    "output_hash": "789abc...",
    "gas_used": 1000,
    "issuer_id": "my-service"
  }'
```

---

## WebSocket API (Future)

Real-time receipt streaming will be available at:

```
wss://ocx.example.com/ws/receipts
```

Subscribe to program receipts:
```json
{"action": "subscribe", "program_hash": "abc123..."}
```
