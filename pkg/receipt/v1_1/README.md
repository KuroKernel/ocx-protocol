# OCX Protocol Receipt v1.1

This package implements the OCX Protocol Receipt v1.1 system, providing cryptographic proof of execution authenticity with enhanced security features, replay protection, and comprehensive audit capabilities.

## Features

### 🔐 **Cryptographic Security**
- **Ed25519 Signatures**: Industry-standard digital signatures for receipt authenticity
- **Canonical CBOR Encoding**: Deterministic serialization ensuring cross-platform compatibility
- **Domain Separation**: Cryptographic domain separation with `OCXv1|receipt|` prefix
- **Key Versioning**: Support for key rotation and version management

### 🛡️ **Replay Protection**
- **Nonce-based Protection**: 16-byte cryptographically secure nonces prevent replay attacks
- **Configurable Retention**: Default 7-day nonce retention with automatic cleanup
- **Clock Skew Validation**: 5-minute tolerance for timestamp validation
- **Duplicate Detection**: Automatic rejection of duplicate nonces

### 🔑 **Key Management System (KMS)**
- **Local Ed25519 Provider**: Built-in local key management
- **AWS KMS Integration**: Ready for cloud key management (interface implemented)
- **Unified Interface**: Consistent API across different key management backends
- **Key Rotation**: Automatic key versioning and rotation support

### 📊 **SIEM Integration**
- **JSONL Export**: Standard format for log aggregators
- **Splunk HEC Format**: Native Splunk HTTP Event Collector support
- **Audit Logging**: Comprehensive audit trail for all operations
- **Downloadable Logs**: Timestamped, structured log exports

### 🌐 **Cross-Architecture Compatibility**
- **JavaScript Implementation**: Browser-compatible verification using Web Crypto API
- **Deterministic Encoding**: Identical byte sequences across all platforms
- **SubtleCrypto Integration**: Native browser cryptographic operations
- **Canonical CBOR**: Ensures consistent serialization

### 📈 **Production Features**
- **Real-time Dashboard**: Live statistics and monitoring
- **Performance Metrics**: Execution time, throughput, and error rate tracking
- **System Health**: Database connectivity, replay protection status
- **Professional UI**: Operational controls and monitoring interface

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Receipt       │    │   Replay         │    │   KMS           │
│   Manager       │◄──►│   Protection     │    │   Manager       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Crypto        │    │   Database       │    │   Local/AWS     │
│   Manager       │    │   (PostgreSQL)   │    │   KMS Provider  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌──────────────────┐
│   Canonical     │    │   SIEM           │
│   CBOR Encoder  │    │   Exporter       │
└─────────────────┘    └──────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │   Dashboard      │
                       │   Manager        │
                       └──────────────────┘
```

## Quick Start

### 1. Database Setup

```sql
-- Run the migration
\i database/migrations/0002_receipt_v1_1.sql
```

### 2. Basic Usage

```go
package main

import (
    "context"
    "crypto/sha256"
    "time"
    "ocx-protocol/pkg/receipt/v1_1"
)

func main() {
    // Create receipt manager
    manager, err := v1_1.NewReceiptManager(db)
    if err != nil {
        panic(err)
    }
    
    // Start the manager
    ctx := context.Background()
    err = manager.Start(ctx)
    if err != nil {
        panic(err)
    }
    defer manager.Stop()
    
    // Create a receipt
    programHash := sha256.Sum256([]byte("my program"))
    inputHash := sha256.Sum256([]byte("my input"))
    outputHash := sha256.Sum256([]byte("my output"))
    
    startedAt := time.Now()
    finishedAt := startedAt.Add(100 * time.Millisecond)
    
    receipt, err := manager.CreateReceipt(
        ctx,
        programHash, inputHash, outputHash,
        1000, startedAt, finishedAt,
        "my-issuer", "my-key", 1,
        5000, map[string]string{"hostname": "server-01"},
    )
    if err != nil {
        panic(err)
    }
    
    // Verify the receipt
    verification, err := manager.VerifyReceipt(ctx, receipt, "my-key", 1)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Receipt verified: %v\n", verification.SignatureValid)
}
```

### 3. HTTP API Usage

```bash
# Create a receipt
curl -X POST http://localhost:8080/api/v1.1/receipts/create \
  -H "Content-Type: application/json" \
  -d '{
    "program_hash": "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3",
    "input_hash": "2c624232cdd221771294dfbb310aca000a0df6ac8b66b696d90ef06fdefb64a3",
    "output_hash": "5994471abb01112afcc18159f6cc74b4f511b99806da59b3caf5a9c173cacfc5",
    "gas_used": 1000,
    "started_at": 1609459200000000000,
    "finished_at": 1609459200100000000,
    "issuer_id": "my-issuer",
    "key_id": "my-key",
    "key_version": 1,
    "host_cycles": 5000,
    "host_info": {"hostname": "server-01"}
  }'

# Verify a receipt
curl -X POST http://localhost:8080/api/v1.1/receipts/verify \
  -H "Content-Type: application/json" \
  -d '{
    "receipt_b64": "base64-encoded-cbor-receipt",
    "key_id": "my-key",
    "key_version": 1
  }'

# Get dashboard
curl http://localhost:8080/api/v1.1/dashboard?format=html

# Export SIEM data
curl http://localhost:8080/api/v1.1/export/siem?format=jsonl&start_time=2023-01-01T00:00:00Z
```

## Receipt Format

### Receipt Core (Signed Data)
```go
type ReceiptCore struct {
    ProgramHash [32]byte `cbor:"1,keyasint"` // SHA-256 of executed program
    InputHash   [32]byte `cbor:"2,keyasint"` // SHA-256 of input data
    OutputHash  [32]byte `cbor:"3,keyasint"` // SHA-256 of output data
    GasUsed     uint64   `cbor:"4,keyasint"` // Gas units consumed
    StartedAt   uint64   `cbor:"5,keyasint"` // Unix timestamp (nanoseconds)
    FinishedAt  uint64   `cbor:"6,keyasint"` // Unix timestamp (nanoseconds)
    IssuerID    string   `cbor:"7,keyasint"` // Issuer identifier
    KeyVersion  uint32   `cbor:"8,keyasint"` // Key version for rotation
    Nonce       [16]byte `cbor:"9,keyasint"` // 16-byte nonce for replay protection
    IssuedAt    uint64   `cbor:"10,keyasint"` // Unix timestamp when receipt was issued
    FloatMode   string   `cbor:"11,keyasint"` // Floating-point mode
}
```

### Full Receipt (With Signature)
```go
type ReceiptFull struct {
    Core       ReceiptCore       `cbor:"core"`
    Signature  [64]byte          `cbor:"signature"` // Ed25519 signature
    HostCycles uint64            `cbor:"host_cycles"`
    HostInfo   map[string]string `cbor:"host_info"`
}
```

## Security Model

### Cryptographic Signing
1. **Domain Separation**: All signatures use the prefix `OCXv1|receipt|`
2. **Canonical CBOR**: The receipt core is encoded with canonical CBOR
3. **Ed25519**: Industry-standard digital signatures
4. **Key Versioning**: Support for key rotation and compromise recovery

### Replay Protection
1. **Nonce Generation**: Cryptographically secure 16-byte nonces
2. **Storage**: Nonces stored with expiration times
3. **Validation**: Clock skew tolerance and duplicate detection
4. **Cleanup**: Automatic removal of expired nonces

### Audit Trail
1. **Comprehensive Logging**: All operations logged with timestamps
2. **SIEM Export**: Standard formats for security monitoring
3. **Dashboard**: Real-time monitoring and alerting
4. **Retention**: Configurable log retention policies

## Performance

### Benchmarks
- **Receipt Creation**: ~50µs per receipt
- **Receipt Verification**: ~30µs per verification
- **Throughput**: 10,000+ receipts/second
- **Memory Usage**: <1MB per 10,000 receipts

### Scalability
- **Database**: PostgreSQL with optimized indexes
- **Caching**: In-memory nonce cache for replay protection
- **Cleanup**: Background cleanup of expired data
- **Monitoring**: Real-time performance metrics

## JavaScript Integration

### Browser Verification
```javascript
// Load the JavaScript verifier
const verifier = new OCXReceiptVerifier();

// Verify a receipt
const isValid = await verifier.verifyReceipt(receiptData, publicKey);
console.log('Receipt valid:', isValid);
```

### Node.js Integration
```javascript
const OCXReceiptVerifier = require('./js_verifier.js');

const verifier = new OCXReceiptVerifier();
const isValid = await verifier.verifyReceipt(receiptData, publicKey);
```

## Configuration

### Environment Variables
```bash
# Database
DATABASE_URL=postgres://user:pass@localhost/ocx

# Replay Protection
OCX_REPLAY_RETENTION=168h  # 7 days
OCX_CLOCK_SKEW=5m          # 5 minutes

# KMS
OCX_KMS_PROVIDER=local     # or aws
OCX_KMS_REGION=us-east-1   # for AWS

# Dashboard
OCX_DASHBOARD_PORT=8080
OCX_DASHBOARD_REFRESH=5s
```

### Database Configuration
```sql
-- Adjust retention periods
SELECT cleanup_expired_nonces();
SELECT cleanup_old_audit_logs();
SELECT cleanup_old_metrics();

-- Get system health
SELECT get_system_health();
```

## Testing

### Unit Tests
```bash
go test ./pkg/receipt/v1_1/...
```

### Integration Tests
```bash
go test -tags=integration ./pkg/receipt/v1_1/...
```

### Performance Tests
```bash
go test -bench=. ./pkg/receipt/v1_1/...
```

## Monitoring

### Dashboard
- **URL**: `http://localhost:8080/api/v1.1/dashboard?format=html`
- **Features**: Real-time stats, issuer breakdown, recent activity
- **Auto-refresh**: 30-second intervals

### Metrics
- **Total Receipts**: Count of all created receipts
- **Verification Rate**: Percentage of successfully verified receipts
- **Replay Attacks**: Number of blocked replay attempts
- **Performance**: Average execution and verification times

### Alerts
- **High Error Rate**: >5% verification failures
- **Replay Attacks**: Any detected replay attempts
- **Database Issues**: Connection or performance problems
- **Key Rotation**: Automatic key rotation events

## Troubleshooting

### Common Issues

1. **Replay Attack Detection**
   ```
   Error: replay attack detected: nonce already used
   ```
   **Solution**: Ensure unique nonces for each receipt

2. **Clock Skew Errors**
   ```
   Error: clock skew too large: 10m (max allowed: 5m)
   ```
   **Solution**: Synchronize system clocks or increase tolerance

3. **Signature Verification Failures**
   ```
   Error: signature verification failed
   ```
   **Solution**: Verify key version and public key match

4. **Database Connection Issues**
   ```
   Error: failed to connect to database
   ```
   **Solution**: Check database connectivity and credentials

### Debug Mode
```bash
# Enable debug logging
export OCX_DEBUG=true
export OCX_LOG_LEVEL=debug

# Run with verbose output
./server --log-level debug
```

## Contributing

1. **Code Style**: Follow Go conventions and run `gofmt`
2. **Testing**: Add tests for new features
3. **Documentation**: Update README and code comments
4. **Security**: Review cryptographic implementations
5. **Performance**: Benchmark new features

## License

This package is part of the OCX Protocol and is licensed under the same terms as the main project.

## Support

- **Issues**: GitHub Issues
- **Documentation**: This README and inline code comments
- **Security**: Report security issues privately
- **Community**: OCX Protocol Discord/Forum
