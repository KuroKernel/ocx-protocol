# OCX Reputation System - Prometheus Metrics

## Overview

The OCX Reputation System exposes comprehensive Prometheus metrics for monitoring, alerting, and performance analysis.

## Metrics Endpoint

**URL**: `http://localhost:8080/metrics`
**Format**: Prometheus text format
**Authentication**: None (typically behind firewall or VPN)

## Metric Categories

### 1. Computation Metrics

#### `ocx_reputation_compute_requests_total`
- **Type**: Counter
- **Labels**: `status`, `user_id`
- **Description**: Total number of reputation compute requests
- **Example**:
  ```
  ocx_reputation_compute_requests_total{status="success",user_id="alice"} 42
  ocx_reputation_compute_requests_total{status="error",user_id="bob"} 3
  ```

#### `ocx_reputation_compute_duration_milliseconds`
- **Type**: Histogram
- **Labels**: `platforms`
- **Buckets**: 0.1, 0.5, 1, 2, 5, 10, 25, 50, 100, 250, 500, 1000
- **Description**: Duration of reputation computation in milliseconds
- **Example**:
  ```
  ocx_reputation_compute_duration_milliseconds_bucket{platforms="3",le="1"} 1543
  ocx_reputation_compute_duration_milliseconds_sum{platforms="3"} 687.5
  ocx_reputation_compute_duration_milliseconds_count{platforms="3"} 1543
  ```

#### `ocx_reputation_compute_errors_total`
- **Type**: Counter
- **Labels**: `error_type`
- **Description**: Total number of reputation compute errors
- **Error Types**: `invalid_input`, `platform_error`, `signature_error`, `database_error`

#### `ocx_reputation_compute_success_rate`
- **Type**: Gauge
- **Labels**: `interval`
- **Range**: 0.0 to 1.0
- **Description**: Success rate of reputation computations
- **Intervals**: `1m`, `5m`, `15m`, `1h`

### 2. Score Metrics

#### `ocx_reputation_trust_score`
- **Type**: Histogram
- **Labels**: `platform_count`
- **Buckets**: 0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100
- **Description**: Distribution of trust scores (0-100)
- **Use Case**: Understand score distribution across users

#### `ocx_reputation_confidence`
- **Type**: Histogram
- **Labels**: `platform_count`
- **Buckets**: 0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0
- **Description**: Distribution of confidence values (0-1)
- **Use Case**: Track data completeness

#### `ocx_reputation_platform_score`
- **Type**: Gauge
- **Labels**: `platform`, `user_id`
- **Range**: 0.0 to 100.0
- **Description**: Current platform scores
- **Platforms**: `github`, `linkedin`, `uber`

### 3. Badge Metrics

#### `ocx_reputation_badge_requests_total`
- **Type**: Counter
- **Labels**: `style`, `status`
- **Description**: Total number of badge generation requests
- **Styles**: `flat`, `flat-square`, `for-the-badge`
- **Statuses**: `success`, `unverified`, `expired`, `not_found`, `unavailable`

#### `ocx_reputation_badge_duration_milliseconds`
- **Type**: Histogram
- **Labels**: `style`
- **Buckets**: 1, 2, 5, 10, 25, 50, 100
- **Description**: Duration of badge generation in milliseconds

#### `ocx_reputation_badge_style_total`
- **Type**: Counter
- **Labels**: `style`
- **Description**: Count of badge requests by style

### 4. Platform Metrics

#### `ocx_reputation_platform_requests_total`
- **Type**: Counter
- **Labels**: `platform`, `status`
- **Description**: Total number of platform API requests
- **Platforms**: `github`, `linkedin`, `uber`
- **Statuses**: `success`, `error`, `rate_limited`, `cached`

#### `ocx_reputation_platform_scores_collected_total`
- **Type**: Counter
- **Labels**: `platform`
- **Description**: Total number of platform scores collected

#### `ocx_reputation_platform_errors_total`
- **Type**: Counter
- **Labels**: `platform`, `error_type`
- **Description**: Total number of platform API errors
- **Error Types**: `auth_error`, `network_error`, `timeout`, `invalid_response`

### 5. Receipt Metrics

#### `ocx_reputation_receipts_total`
- **Type**: Counter
- **Labels**: `status`
- **Description**: Total number of reputation receipts generated
- **Statuses**: `success`, `error`

#### `ocx_reputation_receipt_gas_used`
- **Type**: Histogram
- **Labels**: `platforms`
- **Buckets**: 50, 100, 150, 200, 238, 250, 300, 400, 500
- **Description**: Gas used for reputation receipt generation
- **Target**: 238 units

#### `ocx_reputation_receipt_signature_duration_microseconds`
- **Type**: Histogram
- **Labels**: None
- **Buckets**: 100, 250, 500, 750, 1000, 2500, 5000
- **Description**: Duration of Ed25519 signature generation in microseconds

### 6. OAuth Metrics

#### `ocx_reputation_oauth_requests_total`
- **Type**: Counter
- **Labels**: `platform`, `flow`
- **Description**: Total number of OAuth requests
- **Flows**: `authorize`, `token`, `refresh`

#### `ocx_reputation_oauth_success_total`
- **Type**: Counter
- **Labels**: `platform`, `flow`
- **Description**: Total number of successful OAuth requests

#### `ocx_reputation_oauth_errors_total`
- **Type**: Counter
- **Labels**: `platform`, `error_type`
- **Description**: Total number of OAuth errors

#### `ocx_reputation_oauth_duration_milliseconds`
- **Type**: Histogram
- **Labels**: `platform`, `flow`
- **Buckets**: 100, 250, 500, 1000, 2000, 5000, 10000
- **Description**: Duration of OAuth operations in milliseconds

#### `ocx_reputation_oauth_token_refreshes_total`
- **Type**: Counter
- **Labels**: `platform`
- **Description**: Total number of OAuth token refreshes

### 7. Cache Metrics

#### `ocx_reputation_cache_hits_total`
- **Type**: Counter
- **Labels**: `cache_type`
- **Description**: Total number of reputation cache hits
- **Cache Types**: `memory`, `redis`, `disk`

#### `ocx_reputation_cache_misses_total`
- **Type**: Counter
- **Labels**: `cache_type`
- **Description**: Total number of reputation cache misses

#### `ocx_reputation_cache_size_bytes`
- **Type**: Gauge
- **Description**: Current size of reputation cache in bytes

### 8. Rate Limiting Metrics

#### `ocx_reputation_rate_limit_exceeded_total`
- **Type**: Counter
- **Labels**: `platform`, `user_id`
- **Description**: Total number of rate limit exceeded events

#### `ocx_reputation_rate_limit_remaining`
- **Type**: Gauge
- **Labels**: `platform`, `user_id`
- **Description**: Remaining rate limit quota

## Example Queries

### Average Computation Duration
```promql
rate(ocx_reputation_compute_duration_milliseconds_sum[5m]) /
rate(ocx_reputation_compute_duration_milliseconds_count[5m])
```

### Success Rate (Last Hour)
```promql
sum(rate(ocx_reputation_compute_requests_total{status="success"}[1h])) /
sum(rate(ocx_reputation_compute_requests_total[1h]))
```

### 95th Percentile Badge Generation Time
```promql
histogram_quantile(0.95,
  rate(ocx_reputation_badge_duration_milliseconds_bucket[5m]))
```

### Cache Hit Rate
```promql
sum(rate(ocx_reputation_cache_hits_total[5m])) /
(sum(rate(ocx_reputation_cache_hits_total[5m])) +
 sum(rate(ocx_reputation_cache_misses_total[5m])))
```

### Platform Error Rate
```promql
sum(rate(ocx_reputation_platform_errors_total[5m])) by (platform)
```

### Trust Score Distribution
```promql
histogram_quantile(0.5,
  sum(rate(ocx_reputation_trust_score_bucket[1h])) by (le, platform_count))
```

## Alerting Rules

### High Error Rate
```yaml
- alert: HighReputationErrorRate
  expr: |
    sum(rate(ocx_reputation_compute_errors_total[5m])) /
    sum(rate(ocx_reputation_compute_requests_total[5m])) > 0.05
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High reputation computation error rate"
    description: "Error rate is {{ $value | humanizePercentage }}"
```

### Slow Computation
```yaml
- alert: SlowReputationComputation
  expr: |
    histogram_quantile(0.95,
      rate(ocx_reputation_compute_duration_milliseconds_bucket[5m])) > 100
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Slow reputation computation"
    description: "95th percentile is {{ $value }}ms"
```

### Low Cache Hit Rate
```yaml
- alert: LowCacheHitRate
  expr: |
    sum(rate(ocx_reputation_cache_hits_total[10m])) /
    (sum(rate(ocx_reputation_cache_hits_total[10m])) +
     sum(rate(ocx_reputation_cache_misses_total[10m]))) < 0.7
  for: 15m
  labels:
    severity: info
  annotations:
    summary: "Low reputation cache hit rate"
    description: "Hit rate is {{ $value | humanizePercentage }}"
```

### OAuth Token Refresh Failures
```yaml
- alert: OAuthTokenRefreshFailures
  expr: |
    sum(rate(ocx_reputation_oauth_errors_total{flow="refresh"}[5m])) by (platform) > 0.1
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "OAuth token refresh failures for {{ $labels.platform }}"
    description: "Refresh error rate is {{ $value }}/s"
```

## Grafana Dashboard

### Example Dashboard JSON
```json
{
  "dashboard": {
    "title": "OCX Reputation System",
    "panels": [
      {
        "title": "Compute Requests/sec",
        "targets": [{
          "expr": "sum(rate(ocx_reputation_compute_requests_total[5m])) by (status)"
        }]
      },
      {
        "title": "Average Duration",
        "targets": [{
          "expr": "rate(ocx_reputation_compute_duration_milliseconds_sum[5m]) / rate(ocx_reputation_compute_duration_milliseconds_count[5m])"
        }]
      },
      {
        "title": "Trust Score Distribution",
        "targets": [{
          "expr": "sum(ocx_reputation_trust_score_bucket) by (le)"
        }]
      }
    ]
  }
}
```

## Performance Targets

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Compute Duration (p95) | < 5ms | > 100ms |
| Compute Duration (p99) | < 10ms | > 250ms |
| Badge Duration (p95) | < 25ms | > 100ms |
| Success Rate | > 99.5% | < 95% |
| Cache Hit Rate | > 80% | < 70% |
| OAuth Success Rate | > 99% | < 95% |
| Gas Usage | 238 ± 5% | > 300 |

## Integration

### Prometheus Configuration
```yaml
scrape_configs:
  - job_name: 'ocx-reputation'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Docker Compose
```yaml
version: '3'
services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./alerts.yml:/etc/prometheus/alerts.yml
```

---

**Metrics are automatically initialized and exported on server startup.**

No additional configuration required - all metrics are available at `/metrics` endpoint.
