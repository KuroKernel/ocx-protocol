# OCX Protocol Interfaces & Contracts

## REST API Endpoints

### Core Execution API

#### `POST /api/v1/execute`
**Purpose**: Execute artifact with OCX verification
**Payload**:
```json
{
  "artifact": "base64-encoded-binary",
  "input": "base64-encoded-input",
  "max_cycles": 1000000,
  "timeout_seconds": 30
}
```
**Response**:
```json
{
  "receipt": "base64-encoded-cbor-receipt",
  "output": "base64-encoded-output",
  "cycles_used": 500000,
  "execution_time_ms": 1500
}
```
**Idempotency**: Not idempotent (execution is stateful)
**Error Codes**: E001 (Invalid artifact), E002 (Cycle limit exceeded), E003 (Timeout)

#### `POST /api/v1/verify`
**Purpose**: Verify receipt signature
**Payload**:
```json
{
  "receipt": "base64-encoded-cbor-receipt",
  "public_key": "base64-encoded-ed25519-public-key"
}
```
**Response**:
```json
{
  "verified": true,
  "receipt_fields": {
    "artifact_hash": "sha256-hash",
    "cycles_used": 500000,
    "timestamp": 1640995200
  }
}
```
**Idempotency**: Idempotent (verification is stateless)
**Error Codes**: E004 (Invalid signature), E005 (Malformed receipt), E006 (Invalid public key)

#### `GET /api/v1/receipts/{id}`
**Purpose**: Retrieve receipt by ID
**Response**:
```json
{
  "id": "receipt-uuid",
  "receipt": "base64-encoded-cbor-receipt",
  "created_at": "2022-01-01T00:00:00Z",
  "status": "verified"
}
```
**Idempotency**: Idempotent
**Error Codes**: E007 (Receipt not found), E008 (Access denied)

#### `POST /api/v1/batch-verify`
**Purpose**: Batch verify multiple receipts
**Payload**:
```json
{
  "receipts": [
    {
      "receipt": "base64-encoded-cbor-receipt-1",
      "public_key": "base64-encoded-public-key-1"
    }
  ]
}
```
**Response**:
```json
{
  "results": [
    {
      "verified": true,
      "receipt_id": "receipt-1"
    }
  ],
  "summary": {
    "total": 1,
    "verified": 1,
    "failed": 0
  }
}
```
**Idempotency**: Idempotent
**Error Codes**: E009 (Batch size exceeded), E010 (Invalid batch format)

### Status & Health API

#### `GET /api/v1/status`
**Purpose**: Health check and system status
**Response**:
```json
{
  "status": "healthy",
  "version": "1.0.0-rc.1",
  "uptime_seconds": 3600,
  "components": {
    "verifier": "healthy",
    "vm": "healthy",
    "storage": "healthy"
  }
}
```
**Idempotency**: Idempotent
**Error Codes**: E011 (Service unavailable)

## CLI Commands & Flags

### `ocx execute`
**Purpose**: Execute artifact with OCX verification
**Usage**: `ocx execute [flags] <artifact> <input>`
**Flags**:
- `--max-cycles int`: Maximum cycles allowed (default: 1000000)
- `--timeout duration`: Execution timeout (default: 30s)
- `--output string`: Output file path (default: stdout)
- `--receipt string`: Receipt output file path
- `--profile string`: OCX profile (v1-min, v1-fp, v1-gpu)

### `ocx verify`
**Purpose**: Verify receipt signature
**Usage**: `ocx verify [flags] <receipt> <public-key>`
**Flags**:
- `--format string`: Output format (json, text) (default: text)
- `--verbose`: Verbose output
- `--strict`: Strict verification mode

### `ocxctl`
**Purpose**: Administrative commands
**Usage**: `ocxctl <command> [flags]`
**Commands**:
- `status`: Show system status
- `config`: Manage configuration
- `keys`: Manage cryptographic keys
- `receipts`: List and manage receipts

## Kubernetes Webhook Integration

### Mutating Admission Webhook
**Endpoint**: `/mutate`
**Purpose**: Inject OCX binary and configuration into pods

### Pod Annotations
- `ocx-inject: "true"` - Enable OCX injection
- `ocx-cycles: "1000000"` - Set cycle limit
- `ocx-profile: "v1-min"` - Set OCX profile
- `ocx-keystore: "default"` - Set keystore

### Pod Mutation Rules
1. Add init container with OCX binary
2. Mount OCX configuration volume
3. Set environment variables
4. Add security context constraints

## Envoy Filter Integration

### HTTP Filter Configuration
```yaml
http_filters:
- name: envoy.filters.http.ocx_verify
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.ocx_verify.v3.OcxVerify
    verify_requests: true
    verify_responses: true
    public_key: "base64-encoded-public-key"
```

### Request/Response Headers
- `X-OCX-Receipt`: Receipt in base64-encoded CBOR
- `X-OCX-Cycles`: Cycle count used
- `X-OCX-Verified`: Verification status

## Kafka Interceptor Integration

### Producer Interceptor
**Purpose**: Sign outgoing messages with OCX receipts
**Configuration**:
```properties
interceptor.classes=dev.ocx.kafka.OCXProducerInterceptor
ocx.verify.enabled=true
ocx.public.key=base64-encoded-public-key
```

### Consumer Interceptor
**Purpose**: Verify incoming messages with OCX receipts
**Configuration**:
```properties
interceptor.classes=dev.ocx.kafka.OCXConsumerInterceptor
ocx.verify.enabled=true
ocx.public.key=base64-encoded-public-key
```

### Message Headers
- `ocx-receipt`: Receipt in base64-encoded CBOR
- `ocx-cycles`: Cycle count used
- `ocx-verified`: Verification status

## Terraform Provider Integration

### Resource: `ocx_provenance`
**Purpose**: Verify artifact provenance
```hcl
resource "ocx_provenance" "example" {
  artifact_path = "/path/to/artifact"
  input_path    = "/path/to/input"
  expected_output = "expected-output-hash"
  
  server_url = "https://ocx.example.com"
  api_key    = var.ocx_api_key
}
```

### Data Source: `ocx_receipt`
**Purpose**: Retrieve and verify receipt
```hcl
data "ocx_receipt" "example" {
  receipt_id = "receipt-uuid"
  server_url = "https://ocx.example.com"
}
```

## GitHub Action Integration

### Action Inputs
- `artifact`: Path to artifact file
- `input`: Path to input file
- `expected-output`: Expected output hash
- `max-cycles`: Maximum cycles allowed
- `timeout`: Execution timeout

### Action Outputs
- `receipt`: Generated receipt
- `verified`: Verification status
- `cycles-used`: Cycles consumed
- `output-hash`: Actual output hash

### Usage Example
```yaml
- name: Verify with OCX
  uses: ocx-protocol/ocx-verify-action@v1
  with:
    artifact: ./dist/app
    input: ./test/input.json
    expected-output: ${{ hashFiles('./test/expected-output.json') }}
    max-cycles: 1000000
```

## OCX Receipt Signing ABI

### Domain Separation Prefix
```
"OCXv1|receipt|"
```

### Fields Included in Signing
1. `program_hash` (32 bytes) - SHA-256 of executed program
2. `input_hash` (32 bytes) - SHA-256 of input data
3. `output_hash` (32 bytes) - SHA-256 of output data
4. `cycles` (uint64) - Computational cycles used
5. `started_at` (uint64) - Execution start timestamp
6. `finished_at` (uint64) - Execution finish timestamp
7. `issuer_id` (string) - Issuer identifier

### Fields Excluded from Signing
- `signature` - The signature field itself
- `prev_receipt_hash` - Optional chaining field
- `request_digest` - Optional request digest
- `witness_signatures` - Optional witness signatures

### Canonical CBOR Rules
1. **Integer Keys**: Use integer keys (1-8) instead of text keys
2. **Lexical Ordering**: Keys must be in numerical order (1, 2, 3, ...)
3. **Definite Lengths**: All lengths must be definite (no indefinite encoding)
4. **Minimal Encoding**: Integers use shortest possible encoding
5. **No Duplicates**: No duplicate keys allowed
6. **No Tags**: No CBOR tags unless explicitly documented

### Signing Process
1. Create receipt map with all fields except `signature`
2. Serialize to canonical CBOR with integer keys
3. Prepend domain separation prefix: `"OCXv1|receipt|"`
4. Sign with Ed25519 over the complete message
5. Add signature to receipt map
6. Serialize complete receipt to canonical CBOR

### Verification Process
1. Parse CBOR to extract all fields
2. Remove `signature` field to create receipt core
3. Serialize receipt core to canonical CBOR
4. Prepend domain separation prefix
5. Verify Ed25519 signature over the message
6. Validate field constraints (timestamps, cycles, etc.)

## Error Codes

| Code | Description | HTTP Status | Resolution |
|------|-------------|-------------|------------|
| E001 | Invalid artifact | 400 | Check artifact format and permissions |
| E002 | Cycle limit exceeded | 400 | Increase max_cycles or optimize artifact |
| E003 | Execution timeout | 408 | Increase timeout or optimize artifact |
| E004 | Invalid signature | 400 | Check public key and receipt integrity |
| E005 | Malformed receipt | 400 | Validate CBOR format and required fields |
| E006 | Invalid public key | 400 | Check Ed25519 public key format |
| E007 | Receipt not found | 404 | Check receipt ID and permissions |
| E008 | Access denied | 403 | Check authentication and authorization |
| E009 | Batch size exceeded | 400 | Reduce batch size or use pagination |
| E010 | Invalid batch format | 400 | Validate batch request structure |
| E011 | Service unavailable | 503 | Check service health and dependencies |
