# OCX Protocol Runbooks

## A) VerifyP99Slow (>20ms for 10m)

### Symptoms
- P99 latency exceeds 20ms for 10+ minutes
- Users reporting slow verification times
- Metrics showing elevated response times

### Immediate Actions
1. **Check QPS and CPU**:
   ```bash
   # Check current request rate
   curl -s http://localhost:8080/metrics | grep ocx_requests_total
   
   # Check CPU usage
   top -p $(pgrep -f api-server)
   ```

2. **Scale Out** (if CPU > 70% or QPS > 200/node):
   ```bash
   # Docker Compose
   docker-compose up -d --scale ocx-api=2
   
   # Kubernetes
   kubectl scale deployment ocx-api --replicas=2
   ```

3. **Check Database Latency**:
   ```sql
   -- Connect to PostgreSQL
   psql $DATABASE_URL
   
   -- Check active queries
   SELECT query, state, query_start, now() - query_start as duration 
   FROM pg_stat_activity 
   WHERE state = 'active' 
   ORDER BY duration DESC;
   ```

4. **Re-run Load Test**:
   ```bash
   ./scripts/load_test.sh 200 60
   ```

### Resolution
- **If CPU bound**: Scale out horizontally
- **If DB bound**: Increase DB CPU/memory/IOPS
- **If network bound**: Check load balancer configuration
- **If code bound**: Profile application for bottlenecks

### Prevention
- Set up auto-scaling based on CPU/memory metrics
- Monitor database performance continuously
- Implement circuit breakers for external dependencies

---

## B) ErrorSpike (>0.1% errors for 10m)

### Symptoms
- Error rate exceeds 0.1% for 10+ minutes
- High number of failed requests
- Users reporting verification failures

### Immediate Actions
1. **Identify Error Pattern**:
   ```bash
   # Check error codes
   curl -s http://localhost:8080/metrics | grep ocx_errors_total
   
   # Check logs
   docker logs ocx-api | tail -100
   ```

2. **Common Error Codes**:
   - **E001/E002**: Bad input/receipt → Check for client issues
   - **E003**: Rate limit → Check for DDoS or misconfigured clients
   - **E004**: Execution failed → Check OCX executor
   - **E005**: Storage failed → Check database connectivity
   - **E006**: Internal error → Check server logs
   - **E007**: Idempotency mismatch → Check client retry logic

3. **Throttle Problematic Clients**:
   ```bash
   # Add rate limiting for specific IPs
   iptables -A INPUT -s <problematic_ip> -j DROP
   ```

4. **Restart Service** (if 5xx errors):
   ```bash
   # Docker Compose
   docker-compose restart ocx-api
   
   # Kubernetes
   kubectl rollout restart deployment ocx-api
   ```

### Resolution
- **Client Issues**: Contact client team to fix request format
- **Rate Limiting**: Adjust rate limits or scale out
- **Database Issues**: Check DB connectivity and performance
- **Code Issues**: Deploy fix and monitor

### Prevention
- Implement proper input validation
- Set up rate limiting per client
- Monitor error patterns and trends
- Regular load testing

---

## C) IdempotencyConflictStorm (E007 spikes)

### Symptoms
- High rate of E007 errors
- Clients retrying with same Idempotency-Key but different body
- Verification failures due to idempotency conflicts

### Immediate Actions
1. **Identify Problematic Clients**:
   ```bash
   # Check E007 error rate
   curl -s http://localhost:8080/metrics | grep 'code="E007"'
   
   # Check logs for client patterns
   docker logs ocx-api | grep E007 | head -20
   ```

2. **Temporary Rate Limiting**:
   ```bash
   # Add per-tenant rate limit
   export OCX_RATE_LIMIT_RPS=10
   docker-compose restart ocx-api
   ```

3. **Contact Client Team**:
   - Provide error logs and examples
   - Explain idempotency requirements
   - Request immediate fix

### Resolution
- **Fix Client Code**: Ensure same request body for same Idempotency-Key
- **Implement Proper Retry Logic**: Use exponential backoff
- **Add Request Validation**: Validate request consistency

### Prevention
- Document idempotency requirements clearly
- Provide client SDKs with proper retry logic
- Monitor for idempotency violations
- Regular client education

---

## D) DB Unavailable

### Symptoms
- `/readyz` endpoint returns 503
- Database connection errors in logs
- Service stops accepting traffic

### Immediate Actions
1. **Check Database Status**:
   ```bash
   # Check PostgreSQL
   docker ps | grep postgres
   docker logs postgres
   
   # Check connectivity
   psql $DATABASE_URL -c "SELECT 1;"
   ```

2. **Restore from Backup**:
   ```bash
   # Find latest backup
   ls -la backups/ | tail -5
   
   # Restore database
   ./scripts/backup_restore_drill.sh restore
   ```

3. **Verify Service Recovery**:
   ```bash
   # Check readiness
   curl -s http://localhost:8080/readyz
   
   # Check health
   curl -s http://localhost:8080/health
   ```

### Resolution
- **Database Restart**: If service issue
- **Backup Restore**: If data corruption
- **Network Fix**: If connectivity issue
- **Resource Increase**: If resource exhaustion

### Prevention
- Regular database backups
- Database monitoring and alerting
- High availability setup
- Regular restore drills

---

## E) Key Rotation Incident

### Symptoms
- New receipts failing verification
- Key rotation process errors
- Verification inconsistencies

### Immediate Actions
1. **Verify Both Keys**:
   ```bash
   # Test old key receipts
   ./scripts/key_rotation_drill.sh test-old
   
   # Test new key receipts
   ./scripts/key_rotation_drill.sh test-new
   ```

2. **Rollback if Necessary**:
   ```bash
   # Rollback to previous key
   ./scripts/key_rotation_drill.sh rollback
   ```

3. **Re-run Rotation Drill**:
   ```bash
   # Complete rotation process
   ./scripts/key_rotation_drill.sh complete
   ```

### Resolution
- **Fix Key Issues**: Ensure proper key generation and storage
- **Rollback**: Use previous key if new key has issues
- **Retry Rotation**: Once issues are resolved

### Prevention
- Test key rotation in staging
- Monitor key generation process
- Maintain key backup procedures
- Regular rotation drills

---

## F) High Memory Usage

### Symptoms
- Memory usage > 80%
- Slow response times
- Potential OOM kills

### Immediate Actions
1. **Check Memory Usage**:
   ```bash
   # Check process memory
   ps aux | grep api-server
   
   # Check container memory
   docker stats ocx-api
   ```

2. **Check for Memory Leaks**:
   ```bash
   # Profile memory usage
   go tool pprof http://localhost:8080/debug/pprof/heap
   ```

3. **Restart Service**:
   ```bash
   # Graceful restart
   docker-compose restart ocx-api
   ```

### Resolution
- **Code Fix**: Fix memory leaks in application
- **Resource Increase**: Increase memory limits
- **Scale Out**: Distribute load across instances

### Prevention
- Regular memory profiling
- Set memory limits and alerts
- Monitor garbage collection
- Regular load testing

---

## G) Network Issues

### Symptoms
- Connection timeouts
- Intermittent failures
- High latency

### Immediate Actions
1. **Check Network Connectivity**:
   ```bash
   # Test local connectivity
   curl -v http://localhost:8080/health
   
   # Test external connectivity
   telnet <external_ip> 8080
   ```

2. **Check Load Balancer**:
   ```bash
   # Check LB health
   curl -s http://<lb_ip>/health
   
   # Check LB logs
   docker logs <lb_container>
   ```

3. **Check DNS**:
   ```bash
   # Test DNS resolution
   nslookup <service_domain>
   dig <service_domain>
   ```

### Resolution
- **Network Fix**: Resolve network connectivity issues
- **LB Configuration**: Fix load balancer settings
- **DNS Fix**: Resolve DNS issues

### Prevention
- Network monitoring and alerting
- Load balancer health checks
- DNS monitoring
- Regular connectivity tests

---

## Emergency Contacts

### Level 1 (On-Call Engineer)
- **Primary**: [On-call engineer contact]
- **Backup**: [Backup engineer contact]
- **Response Time**: 15 minutes

### Level 2 (Senior Engineer)
- **Primary**: [Senior engineer contact]
- **Backup**: [Engineering manager contact]
- **Response Time**: 1 hour

### Level 3 (Management)
- **Primary**: [Engineering director contact]
- **Backup**: [CTO contact]
- **Response Time**: 4 hours

### External Dependencies
- **Database**: [DB team contact]
- **Infrastructure**: [Infra team contact]
- **Security**: [Security team contact]

---

## Escalation Procedures

### P0 (Critical)
1. **Immediate**: Page on-call engineer
2. **5 minutes**: If no response, page backup
3. **15 minutes**: If no response, escalate to Level 2
4. **30 minutes**: If no response, escalate to Level 3

### P1 (High)
1. **Immediate**: Page on-call engineer
2. **30 minutes**: If no response, page backup
3. **1 hour**: If no response, escalate to Level 2

### P2 (Medium)
1. **Immediate**: Create ticket
2. **4 hours**: If no response, page on-call engineer
3. **8 hours**: If no response, escalate to Level 2

### P3 (Low)
1. **Immediate**: Create ticket
2. **24 hours**: If no response, page on-call engineer
3. **48 hours**: If no response, escalate to Level 2
