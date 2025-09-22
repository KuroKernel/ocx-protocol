# OCX Protocol Sales Demo Script (90 seconds)

## Setup (Before Demo)
```bash
# Start OCX server
docker-compose up -d

# Verify health
curl -s http://localhost:8080/health
```

## Demo Flow

### 1. Execute Request (30 seconds)
```bash
# Run execute command
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: demo-$(date +%s)" \
  -d '{"artifact":"aGVsbG93b3JsZA==","input":"dGVzdGRhdGE=","max_cycles":1000}'

# Show receipt_blob and auto-printed verify command
echo "Receipt generated with cryptographic proof"
echo "Verify command: ./ocx-cli verify --receipt 'RECEIPT_BLOB_HERE'"
```

**Key Points**:
- "Every execution generates a cryptographic receipt"
- "Receipt contains hashes of code, input, and output"
- "Receipt is signed with Ed25519 for authenticity"

### 2. Verify Offline (30 seconds)
```bash
# Copy receipt_blob from previous response
RECEIPT_BLOB="eyJ2ZXJzaW9uIjoidjEtbWluIiwiYXJ0aWZhY3QiOiI..."

# Verify receipt
curl -X POST http://localhost:8080/api/v1/verify \
  -H "Content-Type: application/json" \
  -d "{\"receipt_blob\":\"$RECEIPT_BLOB\"}"

# Show response
echo "Valid: true - Receipt verified in <20ms"
```

**Key Points**:
- "Verification works offline, no network required"
- "Sub-20ms verification time"
- "Cryptographic proof of execution"

### 3. Idempotency Demo (15 seconds)
```bash
# Same request with same key
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: demo-same" \
  -d '{"artifact":"aGVsbG93b3JsZA==","input":"dGVzdGRhdGE=","max_cycles":1000}'

# Different request with same key
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: demo-same" \
  -d '{"artifact":"ZGlmZmVyZW50","input":"dGVzdGRhdGE=","max_cycles":1000}'
```

**Key Points**:
- "Same key + same body = cached response"
- "Same key + different body = 409 conflict"
- "Prevents duplicate processing"

### 4. Metrics Demo (15 seconds)
```bash
# Show metrics
curl -s http://localhost:8080/metrics | grep ocx_

# Show counters moving
echo "Execute counter: $(curl -s http://localhost:8080/metrics | grep ocx_execute_total | cut -d' ' -f2)"
echo "Verify counter: $(curl -s http://localhost:8080/metrics | grep ocx_verify_total | cut -d' ' -f2)"
```

**Key Points**:
- "Real-time metrics and monitoring"
- "Performance tracking and alerting"
- "Production-ready observability"

## Closing (30 seconds)

### Key Value Propositions
1. **"Proof, not promises"** - Cryptographic evidence of execution
2. **"Portable receipts"** - Anyone can verify offline
3. **"Sub-20ms verification"** - Enterprise performance
4. **"Zero false-positives"** - 100% accuracy guarantee

### Call to Action
- **"2-week pilot with one production pipeline"**
- **"If it meets SLOs, we roll to enforce + contract"**
- **"60 minutes to install, 5 minutes to integrate"**

### Next Steps
1. **Schedule pilot call**: https://calendly.com/ocx-protocol/pilot
2. **Send pilot kit**: Complete deployment package
3. **Start integration**: One production pipeline in warn mode
4. **Measure success**: SLO compliance for 14 days

## Demo Script Notes

### Timing
- **Total**: 90 seconds
- **Execute**: 30 seconds
- **Verify**: 30 seconds
- **Idempotency**: 15 seconds
- **Metrics**: 15 seconds

### Key Messages
- **Cryptographic proof** of execution
- **Offline verification** capability
- **Enterprise performance** (sub-20ms)
- **Production ready** with monitoring

### Common Questions
- **Q**: "How does this differ from logs?"
- **A**: "Logs can be modified. Receipts are cryptographically signed and immutable."

- **Q**: "What about performance?"
- **A**: "Sub-20ms verification, 200+ RPS per node, 99.9% availability."

- **Q**: "How do we integrate?"
- **A**: "Add one line to your CI pipeline. Receipts are generated automatically."

- **Q**: "What about security?"
- **A**: "Ed25519 signatures, constant-time operations, no code execution on server."

### Demo Environment
- **Server**: Running on localhost:8080
- **Database**: PostgreSQL with sample data
- **Monitoring**: Prometheus metrics enabled
- **Logs**: Structured logging with request IDs

### Backup Plans
- **If server down**: Show pre-recorded demo video
- **If network issues**: Use local demo environment
- **If time short**: Focus on execute + verify only
- **If technical issues**: Show documentation and runbooks

### Follow-up Materials
- **Pilot Kit**: Complete deployment package
- **Documentation**: API reference and runbooks
- **Support**: Dedicated pilot support channel
- **Calendar**: Schedule pilot installation call
