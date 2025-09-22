# OCX Protocol Service Level Objectives (SLOs)

## Performance SLOs

### Verify Endpoint Performance
- **P50 Latency**: < 5ms
- **P95 Latency**: < 15ms  
- **P99 Latency**: < 20ms
- **Throughput**: 200+ RPS per node
- **Availability**: 99.9% uptime

### Execute Endpoint Performance
- **P50 Latency**: < 10ms
- **P95 Latency**: < 25ms
- **P99 Latency**: < 50ms
- **Throughput**: 100+ RPS per node
- **Availability**: 99.9% uptime

## Availability SLOs

### Overall Service Availability
- **Target**: 99.9% uptime (8.77 hours downtime/year)
- **Measurement**: Successful responses to `/health` endpoint
- **Exclusions**: Planned maintenance windows, external dependencies

### Database Availability
- **Target**: 99.95% uptime (4.38 hours downtime/year)
- **Measurement**: Successful database connections
- **Backup**: Daily full + weekly incremental + monthly drills

## Correctness SLOs

### Cryptographic Integrity
- **False Positives**: 0% (no invalid receipts marked as valid)
- **False Negatives**: 0% (no valid receipts marked as invalid)
- **Signature Verification**: 100% accuracy

### Receipt Consistency
- **Deterministic Execution**: Same input always produces same receipt
- **Cross-Platform**: Identical receipts across different architectures
- **Version Compatibility**: Receipts remain verifiable across protocol versions

## Security SLOs

### Input Validation
- **Malformed Requests**: 100% rejected with appropriate error codes
- **Resource Limits**: 100% enforcement of size/time/rate caps
- **Injection Attacks**: 100% prevention of code injection

### Key Management
- **Key Rotation**: Successful rotation within 7-day grace period
- **Signature Verification**: 100% accuracy with both old and new keys
- **Key Revocation**: Immediate effect on new signatures

## Alert Rules

### Critical Alerts (P0)
```yaml
# VerifyP99Slow - P99 latency exceeds 20ms for 10 minutes
- alert: VerifyP99Slow
  expr: histogram_quantile(0.99, rate(ocx_verify_latency_seconds_bucket[5m])) > 0.02
  for: 10m
  labels:
    severity: critical
  annotations:
    summary: "OCX verify P99 latency is too high"
    description: "P99 latency has been above 20ms for 10 minutes"

# ErrorSpike - Error rate exceeds 0.1% for 10 minutes  
- alert: ErrorSpike
  expr: rate(ocx_errors_total[5m]) / rate(ocx_requests_total[5m]) > 0.001
  for: 10m
  labels:
    severity: critical
  annotations:
    summary: "OCX error rate is too high"
    description: "Error rate has been above 0.1% for 10 minutes"

# ServiceDown - Service is down
- alert: ServiceDown
  expr: up{job="ocx-api"} == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "OCX service is down"
    description: "OCX API service has been down for 1 minute"
```

### Warning Alerts (P1)
```yaml
# VerifyP95Slow - P95 latency exceeds 15ms for 5 minutes
- alert: VerifyP95Slow
  expr: histogram_quantile(0.95, rate(ocx_verify_latency_seconds_bucket[5m])) > 0.015
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "OCX verify P95 latency is elevated"
    description: "P95 latency has been above 15ms for 5 minutes"

# HighErrorRate - Error rate exceeds 0.05% for 5 minutes
- alert: HighErrorRate
  expr: rate(ocx_errors_total[5m]) / rate(ocx_requests_total[5m]) > 0.0005
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "OCX error rate is elevated"
    description: "Error rate has been above 0.05% for 5 minutes"

# IdempotencyConflictStorm - E007 spikes indicate client issues
- alert: IdempotencyConflictStorm
  expr: rate(ocx_errors_total{code="E007"}[5m]) > 10
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "High rate of idempotency conflicts"
    description: "E007 errors indicate clients retrying with different request bodies"
```

### Info Alerts (P2)
```yaml
# HighThroughput - Unusually high request rate
- alert: HighThroughput
  expr: rate(ocx_requests_total[5m]) > 500
  for: 5m
  labels:
    severity: info
  annotations:
    summary: "High request throughput detected"
    description: "Request rate is above 500 RPS for 5 minutes"

# KeyRotationPending - Key rotation due soon
- alert: KeyRotationPending
  expr: time() - ocx_key_created_timestamp > 3600 * 24 * 6
  for: 1h
  labels:
    severity: info
  annotations:
    summary: "Key rotation recommended"
    description: "Current key is 6+ days old, consider rotation"
```

## SLO Monitoring

### Dashboards
- **OCX Overview**: Overall service health and key metrics
- **Performance**: Latency percentiles and throughput
- **Errors**: Error rates by code and endpoint
- **Resources**: CPU, memory, and database metrics

### Key Metrics
```promql
# Request rate
rate(ocx_requests_total[5m])

# Error rate
rate(ocx_errors_total[5m]) / rate(ocx_requests_total[5m])

# P99 latency
histogram_quantile(0.99, rate(ocx_verify_latency_seconds_bucket[5m]))

# Availability
avg_over_time(up{job="ocx-api"}[1h])
```

## SLO Reporting

### Daily Reports
- P50/P95/P99 latencies for all endpoints
- Error rates by code and endpoint
- Throughput and availability metrics
- Key rotation status

### Weekly Reports
- SLO compliance summary
- Performance trends and anomalies
- Capacity planning recommendations
- Security and compliance status

### Monthly Reports
- SLO performance against targets
- Incident analysis and improvements
- Capacity and scaling recommendations
- Security audit results

## SLO Violations

### Response Procedures
1. **P0 Alerts**: Immediate response within 15 minutes
2. **P1 Alerts**: Response within 1 hour
3. **P2 Alerts**: Response within 4 hours

### Escalation
1. **Level 1**: On-call engineer
2. **Level 2**: Senior engineer + manager
3. **Level 3**: Engineering director + CTO

### Post-Incident
1. **Root Cause Analysis**: Within 24 hours
2. **Action Items**: Within 48 hours
3. **Prevention**: Within 1 week
