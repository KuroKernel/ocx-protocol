# OCX Protocol Operations Guide

## Environment Variables

### Required Variables
```bash
# Database Configuration
export DATABASE_URL="postgres://ocx:ocx_password@postgres:5432/ocx?sslmode=disable"
export DB_TYPE="postgres"

# Server Configuration
export PORT="8080"
export LOG_LEVEL="info"

# Security Configuration
export KEYSTORE_DIR="/app/keys"
export OCX_MAX_BODY_BYTES=$((1024*1024))  # 1MB
export RATE_LIMIT_RPS="100"

# Metrics Configuration
export METRICS_ENABLED="true"
```

### Optional Variables
```bash
# Database Configuration
export POSTGRES_URL="postgres://ocx:ocx_password@postgres:5432/ocx?sslmode=disable"
export MAX_RECEIPT_SIZE=$((1024*1024))  # 1MB

# Performance Configuration
export READ_HEADER_TIMEOUT="3s"
export READ_TIMEOUT="5s"
export WRITE_TIMEOUT="15s"
export IDLE_TIMEOUT="60s"

# Logging Configuration
export LOG_FORMAT="json"
export LOG_OUTPUT="stdout"
```

## Timeout Configuration

### HTTP Server Timeouts
- **ReadHeaderTimeout**: 3 seconds - Time to read request headers
- **ReadTimeout**: 5 seconds - Time to read entire request body
- **WriteTimeout**: 15 seconds - Time to write response
- **IdleTimeout**: 60 seconds - Time to keep connection idle

### Application Timeouts
- **Execution Timeout**: 30 seconds - Maximum execution time
- **Verification Timeout**: 5 seconds - Maximum verification time
- **Database Timeout**: 10 seconds - Maximum database operation time

### Timeout Tuning
```bash
# For high-latency networks
export READ_HEADER_TIMEOUT="10s"
export READ_TIMEOUT="30s"
export WRITE_TIMEOUT="60s"

# For low-latency networks
export READ_HEADER_TIMEOUT="1s"
export READ_TIMEOUT="2s"
export WRITE_TIMEOUT="5s"
```

## Body Size Limits

### Request Body Limits
- **Maximum Size**: 1MB (configurable via `OCX_MAX_BODY_BYTES`)
- **Artifact Field**: 10KB base64-encoded maximum
- **Input Field**: 10KB base64-encoded maximum
- **Receipt Blob**: 1MB maximum for verification

### Limit Configuration
```bash
# Conservative limits
export OCX_MAX_BODY_BYTES=$((512*1024))  # 512KB
export MAX_RECEIPT_SIZE=$((512*1024))    # 512KB

# Aggressive limits
export OCX_MAX_BODY_BYTES=$((10*1024*1024))  # 10MB
export MAX_RECEIPT_SIZE=$((10*1024*1024))    # 10MB
```

### Limit Enforcement
- **Server Level**: `http.MaxBytesReader` for request body
- **Application Level**: Validation in request handlers
- **Database Level**: Column size constraints
- **Error Response**: 413 Payload Too Large

## Idempotency Configuration

### Idempotency Requirements
- **Header Required**: `Idempotency-Key` header for all execute requests
- **Key Format**: Any string, recommended UUID or timestamp-based
- **Key Length**: 1-255 characters
- **Key Uniqueness**: Must be unique per client

### Idempotency Behavior
- **Same Key + Same Body**: Returns cached response
- **Same Key + Different Body**: Returns 409 Conflict (E007)
- **Missing Key**: Returns 400 Bad Request (E001)
- **Cache TTL**: 24 hours (configurable)

### Idempotency Best Practices
```bash
# Generate unique keys
export IDEMPOTENCY_KEY="$(date +%s)-$(uuidgen | tr -d '-')"

# Use in requests
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -H "Content-Type: application/json" \
  -d '{"artifact":"...","input":"...","max_cycles":1000}'
```

## Database Operations

### PostgreSQL Configuration
```sql
-- Create database
CREATE DATABASE ocx;

-- Create user
CREATE USER ocx WITH PASSWORD 'ocx_password';

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE ocx TO ocx;

-- Create tables
\c ocx
CREATE TABLE receipts (
    receipt_hash BYTEA PRIMARY KEY,
    receipt_body BYTEA NOT NULL,
    artifact_hash BYTEA NOT NULL,
    input_hash BYTEA NOT NULL,
    cycles_used BIGINT NOT NULL,
    price_micro_units BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (digest(receipt_body, 'sha256') = receipt_hash)
);

-- Create indexes
CREATE INDEX idx_receipts_artifact_hash ON receipts(artifact_hash);
CREATE INDEX idx_receipts_input_hash ON receipts(input_hash);
CREATE INDEX idx_receipts_created_at ON receipts(created_at);

-- Create rules to prevent updates/deletes
CREATE RULE no_update_receipts AS ON UPDATE TO receipts DO INSTEAD NOTHING;
CREATE RULE no_delete_receipts AS ON DELETE TO receipts DO INSTEAD NOTHING;
```

### Database Monitoring
```sql
-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check active connections
SELECT 
    state,
    COUNT(*) as count
FROM pg_stat_activity 
GROUP BY state;

-- Check slow queries
SELECT 
    query,
    state,
    query_start,
    now() - query_start as duration
FROM pg_stat_activity 
WHERE state = 'active' 
ORDER BY duration DESC;
```

## Monitoring and Alerting

### Health Checks
```bash
# Basic health check
curl -s http://localhost:8080/health

# Readiness check
curl -s http://localhost:8080/readyz

# Liveness check
curl -s http://localhost:8080/livez

# Metrics endpoint
curl -s http://localhost:8080/metrics
```

### Key Metrics
```promql
# Request rate
rate(ocx_requests_total[5m])

# Error rate
rate(ocx_errors_total[5m]) / rate(ocx_requests_total[5m])

# P99 latency
histogram_quantile(0.99, rate(ocx_verify_latency_seconds_bucket[5m]))

# Active connections
ocx_active_connections
```

### Alert Rules
```yaml
# High error rate
- alert: HighErrorRate
  expr: rate(ocx_errors_total[5m]) / rate(ocx_requests_total[5m]) > 0.001
  for: 10m
  labels:
    severity: critical
  annotations:
    summary: "High error rate detected"

# High latency
- alert: HighLatency
  expr: histogram_quantile(0.99, rate(ocx_verify_latency_seconds_bucket[5m])) > 0.02
  for: 10m
  labels:
    severity: critical
  annotations:
    summary: "High latency detected"
```

## Logging Configuration

### Log Levels
- **DEBUG**: Detailed debugging information
- **INFO**: General information about operations
- **WARN**: Warning messages for potential issues
- **ERROR**: Error messages for failed operations
- **FATAL**: Fatal errors that cause service termination

### Log Format
```json
{
  "timestamp": "2025-09-20T15:30:45Z",
  "level": "info",
  "message": "Request processed",
  "request_id": "req_123456789",
  "method": "POST",
  "path": "/api/v1/execute",
  "status_code": 200,
  "duration_ms": 15.5
}
```

### Log Rotation
```bash
# Configure logrotate
cat > /etc/logrotate.d/ocx << EOF
/var/log/ocx/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 ocx ocx
    postrotate
        systemctl reload ocx
    endscript
}
EOF
```

## Performance Tuning

### Go Runtime Tuning
```bash
# Garbage collection tuning
export GOGC=100
export GOMAXPROCS=4

# Memory tuning
export GO_MEMLIMIT=2GiB

# Debugging
export GODEBUG=gctrace=1
```

### Database Tuning
```sql
-- PostgreSQL configuration
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET effective_io_concurrency = 200;
SELECT pg_reload_conf();
```

### Network Tuning
```bash
# Increase file descriptor limits
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# TCP tuning
echo 'net.core.somaxconn = 65536' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_max_syn_backlog = 65536' >> /etc/sysctl.conf
echo 'net.core.netdev_max_backlog = 5000' >> /etc/sysctl.conf
sysctl -p
```

## Backup and Recovery

### Backup Procedures
```bash
# Daily backup
pg_dump $DATABASE_URL -Fc -f backups/ocx_$(date +%F_%H%M).dump

# Weekly backup
pg_dump $DATABASE_URL -Fc -f backups/ocx_weekly_$(date +%F).dump

# Monthly backup
pg_dump $DATABASE_URL -Fc -f backups/ocx_monthly_$(date +%Y-%m).dump
```

### Recovery Procedures
```bash
# Restore from backup
pg_restore -d ocx_restore backups/ocx_2025-09-20_15:30.dump

# Verify restore
psql -d ocx_restore -c "SELECT COUNT(*) FROM receipts;"

# Switch to restored database
# (Update DATABASE_URL and restart service)
```

## Scaling Operations

### Horizontal Scaling
```bash
# Docker Compose
docker-compose up -d --scale ocx-api=3

# Kubernetes
kubectl scale deployment ocx-api --replicas=3
```

### Vertical Scaling
```bash
# Increase memory
docker run -m 2g ocx-protocol:latest

# Increase CPU
docker run --cpus=2 ocx-protocol:latest
```

### Load Balancing
```nginx
# Nginx configuration
upstream ocx_backend {
    server ocx-api-1:8080;
    server ocx-api-2:8080;
    server ocx-api-3:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://ocx_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Troubleshooting

### Common Issues
1. **High Memory Usage**: Check for memory leaks, increase limits
2. **Slow Queries**: Check database performance, add indexes
3. **Connection Timeouts**: Check network, increase timeouts
4. **Rate Limiting**: Check client behavior, adjust limits
5. **Key Rotation Issues**: Check key generation, verify signatures

### Debug Commands
```bash
# Check service status
systemctl status ocx

# Check logs
journalctl -u ocx -f

# Check metrics
curl -s http://localhost:8080/metrics | grep ocx_

# Check database
psql $DATABASE_URL -c "SELECT COUNT(*) FROM receipts;"

# Check key store
ls -la /app/keys/
```

### Performance Analysis
```bash
# CPU profiling
go tool pprof http://localhost:8080/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine profiling
go tool pprof http://localhost:8080/debug/pprof/goroutine
```
