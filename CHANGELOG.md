# Changelog

All notable changes to the OCX Protocol project will be documented in this file.

## [1.0.0-rc.1-pilot1] - 2025-09-20

### Added
- **Production-Ready OCX Protocol**: Complete implementation with all 7 critical gaps closed
- **HTTP Security**: Defensive timeouts, body size limits, slowloris protection
- **Idempotency Semantics**: 409 + E007 for request body mismatches, request caching
- **Load Testing**: 200 RPS SLO testing with P99 < 20ms verification
- **Real Metrics**: Working Prometheus metrics with P50/P95/P99 percentiles
- **Key Rotation**: Complete drill procedures with 7-day grace periods
- **Backup/Restore**: Production PostgreSQL procedures with automated schedules
- **Health Probes**: Readiness/liveness endpoints with graceful shutdown
- **Professional Logo System**: Complete brand identity with 4 logo variants
- **Pilot Kit**: Complete deployment package for enterprise pilots

### Technical Specifications
- **Image**: `ocx-protocol:1.0.0-rc.1-pilot1`
- **Commit SHA**: `$(git rev-parse HEAD)`
- **OpenAPI**: `/api/openapi.yaml`
- **Ports**: 8080 (API), 5432 (PostgreSQL)
- **Database**: PostgreSQL 15+ (production), SQLite (development)
- **SLOs**: P99 < 20ms @ 200 RPS/node, 99.9% availability
- **Resource Limits**: 1MB body size, 1M max cycles, 10KB artifact/input

### Security
- **Cryptography**: Ed25519 signatures with domain separation
- **Constant-Time**: All cryptographic operations use constant-time comparisons
- **Input Validation**: Size, time, and rate caps enforced server-side
- **No Code Execution**: Server only builds receipts from provided inputs

### Performance
- **Response Times**: P50 < 5ms, P95 < 15ms, P99 < 20ms
- **Throughput**: 200+ RPS per node verified
- **Memory**: Constant memory usage under load
- **Concurrency**: 10+ simultaneous requests without degradation

### Operations
- **Monitoring**: Prometheus metrics with alert rules
- **Logging**: Structured logging with request IDs
- **Backups**: Daily full + weekly incremental + monthly drills
- **Key Management**: Automated rotation with grace periods
- **Graceful Shutdown**: 10-second grace period for in-flight requests

### Known Limits
- **Body Size**: 1MB maximum request body
- **Cycles**: 1,000,000 maximum cycles per execution
- **Payload**: 10KB maximum for artifact and input fields
- **Concurrency**: 100 RPS rate limit per client
- **Storage**: 90-day default retention (configurable)

### Breaking Changes
- **Field Names**: Standardized on `receipt_blob` (was `receipt`)
- **Error Codes**: Added E007 for idempotency mismatches
- **Endpoints**: Added `/readyz` and `/livez` health probes

### Dependencies
- **Go**: 1.21+ required
- **PostgreSQL**: 15+ for production
- **Docker**: 20.10+ for containerized deployment
- **Kubernetes**: 1.20+ for Helm deployment

### Migration Guide
- Update client code to use `receipt_blob` field name
- Add health probe endpoints to monitoring
- Configure resource limits in environment variables
- Update backup procedures to use new scripts

### Support
- **Documentation**: Complete pilot kit with runbooks
- **Scripts**: Smoke tests, load tests, key rotation, backup/restore
- **Monitoring**: Prometheus alert rules and dashboards
- **Contact**: See SUPPORT.md for response times and escalation
