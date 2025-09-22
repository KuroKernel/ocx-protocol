# OCX Webhook Troubleshooting Guide

## Quick Diagnostics

### Check Webhook Status

```bash
# Check if webhook is running
kubectl get pods -n ocx-system -l app.kubernetes.io/name=ocx-webhook

# Check webhook configuration
kubectl get mutatingwebhookconfiguration ocx-webhook

# Check service
kubectl get service -n ocx-system ocx-webhook

# Check logs
kubectl logs -n ocx-system deployment/ocx-webhook --tail=50
```

### Verify Installation

```bash
# Check all webhook components
kubectl get all -n ocx-system -l app.kubernetes.io/name=ocx-webhook

# Check RBAC
kubectl get clusterrolebinding ocx-webhook
kubectl get clusterrole ocx-webhook

# Check certificates
kubectl get secret -n ocx-system ocx-webhook-certs
```

## Common Issues

### 1. Webhook Not Injecting OCX

#### Symptoms
- Pods are created but no OCX binary is present
- No init containers or sidecars are added
- No OCX environment variables

#### Diagnosis

```bash
# Check webhook configuration
kubectl get mutatingwebhookconfiguration ocx-webhook -o yaml

# Verify namespace selector
kubectl get namespace default --show-labels

# Check webhook logs for errors
kubectl logs -n ocx-system deployment/ocx-webhook | grep -i error

# Test with explicit annotation
kubectl run test-pod --image=nginx --overrides='{"metadata":{"annotations":{"ocx-inject":"true"}}}'
```

#### Solutions

**Namespace not labeled:**
```bash
# Label namespace to enable webhook
kubectl label namespace default ocx-inject=enabled
```

**Webhook configuration issues:**
```bash
# Check webhook is targeting correct resources
kubectl get mutatingwebhookconfiguration ocx-webhook -o jsonpath='{.webhooks[0].rules}'

# Verify object selector
kubectl get mutatingwebhookconfiguration ocx-webhook -o jsonpath='{.webhooks[0].objectSelector}'
```

**Certificate issues:**
```bash
# Regenerate certificates
./scripts/generate-certs.sh
kubectl delete secret ocx-webhook-certs -n ocx-system
kubectl apply -f certs/webhook-certs-secret.yaml
kubectl rollout restart deployment/ocx-webhook -n ocx-system
```

### 2. Certificate Errors

#### Symptoms
- Webhook pods crash with TLS errors
- Admission requests fail with certificate errors
- `x509: certificate signed by unknown authority` errors

#### Diagnosis

```bash
# Check certificate secret
kubectl get secret ocx-webhook-certs -n ocx-system -o yaml

# Verify certificate validity
kubectl get secret ocx-webhook-certs -n ocx-system -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout

# Check webhook logs
kubectl logs -n ocx-system deployment/ocx-webhook | grep -i cert
```

#### Solutions

**Regenerate certificates:**
```bash
# Remove old certificates
kubectl delete secret ocx-webhook-certs -n ocx-system

# Generate new certificates
./scripts/generate-certs.sh

# Apply new certificates
kubectl apply -f certs/webhook-certs-secret.yaml

# Restart webhook
kubectl rollout restart deployment/ocx-webhook -n ocx-system
```

**cert-manager issues:**
```bash
# Check cert-manager status
kubectl get pods -n cert-manager

# Check certificate status
kubectl get certificate -n ocx-system

# Check certificate events
kubectl describe certificate ocx-webhook-cert -n ocx-system
```

### 3. Permission Errors

#### Symptoms
- Webhook logs show RBAC permission errors
- Admission requests fail with `Forbidden` errors
- Service account lacks permissions

#### Diagnosis

```bash
# Check service account
kubectl get serviceaccount ocx-webhook -n ocx-system

# Check RBAC bindings
kubectl get clusterrolebinding ocx-webhook
kubectl describe clusterrolebinding ocx-webhook

# Test permissions
kubectl auth can-i create pods --as=system:serviceaccount:ocx-system:ocx-webhook
kubectl auth can-i get pods --as=system:serviceaccount:ocx-system:ocx-webhook
```

#### Solutions

**Reapply RBAC:**
```bash
# Apply RBAC configuration
kubectl apply -f k8s/rbac.yaml

# Verify permissions
kubectl auth can-i --list --as=system:serviceaccount:ocx-system:ocx-webhook
```

**Check service account:**
```bash
# Verify service account exists
kubectl get serviceaccount ocx-webhook -n ocx-system

# Check if deployment uses correct service account
kubectl get deployment ocx-webhook -n ocx-system -o jsonpath='{.spec.template.spec.serviceAccountName}'
```

### 4. Performance Issues

#### Symptoms
- Slow pod creation times
- High webhook latency
- Admission request timeouts

#### Diagnosis

```bash
# Check webhook metrics
kubectl port-forward -n ocx-system svc/ocx-webhook 9090:9090
curl http://localhost:9090/metrics | grep ocx_webhook_admission_duration

# Check resource usage
kubectl top pod -n ocx-system -l app.kubernetes.io/name=ocx-webhook

# Check webhook logs for performance issues
kubectl logs -n ocx-system deployment/ocx-webhook | grep -i "slow\|timeout\|latency"
```

#### Solutions

**Scale webhook:**
```bash
# Increase replicas
kubectl scale deployment ocx-webhook -n ocx-system --replicas=3

# Check resource limits
kubectl get deployment ocx-webhook -n ocx-system -o jsonpath='{.spec.template.spec.containers[0].resources}'
```

**Optimize configuration:**
```bash
# Enable debug mode for detailed logs
kubectl set env deployment/ocx-webhook DEBUG_MODE=true -n ocx-system

# Check OCX server connectivity
kubectl exec -n ocx-system deployment/ocx-webhook -- curl -s http://ocx-server:8080/health
```

### 5. OCX Server Connectivity Issues

#### Symptoms
- Webhook readiness probe fails
- OCX server unreachable errors
- Verification failures

#### Diagnosis

```bash
# Check OCX server status
kubectl get pods -n ocx-system -l app.kubernetes.io/name=ocx-server

# Test connectivity from webhook
kubectl exec -n ocx-system deployment/ocx-webhook -- curl -v http://ocx-server:8080/health

# Check network policies
kubectl get networkpolicy -n ocx-system

# Check DNS resolution
kubectl exec -n ocx-system deployment/ocx-webhook -- nslookup ocx-server.ocx-system.svc.cluster.local
```

#### Solutions

**Fix OCX server:**
```bash
# Restart OCX server
kubectl rollout restart deployment/ocx-server -n ocx-system

# Check OCX server logs
kubectl logs -n ocx-system deployment/ocx-server --tail=50
```

**Update webhook configuration:**
```bash
# Update OCX server URL
kubectl set env deployment/ocx-webhook OCX_SERVER_URL=http://ocx-server.ocx-system.svc.cluster.local:8080 -n ocx-system

# Restart webhook
kubectl rollout restart deployment/ocx-webhook -n ocx-system
```

## Debug Mode

### Enable Debug Logging

```bash
# Enable debug mode
kubectl set env deployment/ocx-webhook DEBUG_MODE=true -n ocx-system

# Check debug logs
kubectl logs -n ocx-system deployment/ocx-webhook -f | grep -i debug
```

### Debug Information

Debug mode provides detailed information about:
- Annotation parsing
- Injection decisions
- JSON patch generation
- OCX server connectivity
- Performance metrics

## Log Analysis

### Common Log Patterns

**Successful injection:**
```
I0115 10:30:00.123456 webhook.go:234] OCX injection successful namespace=default name=test-pod injectionType=true cycles=50000 profile=v1-min
```

**Annotation parsing error:**
```
E0115 10:30:00.123456 webhook.go:156] Invalid OCX annotation: ocx-cycles must be between 1 and 1000000 namespace=default name=test-pod
```

**OCX server connection error:**
```
E0115 10:30:00.123456 webhook.go:445] OCX server not ready: connection timeout
```

**Certificate error:**
```
E0115 10:30:00.123456 webhook.go:78] Failed to load TLS certificates: tls: failed to find any PEM data in certificate input
```

### Log Filtering

```bash
# Filter by log level
kubectl logs -n ocx-system deployment/ocx-webhook | grep -E "(ERROR|WARN)"

# Filter by namespace
kubectl logs -n ocx-system deployment/ocx-webhook | grep "namespace=default"

# Filter by injection type
kubectl logs -n ocx-system deployment/ocx-webhook | grep "injectionType=true"

# Filter by error type
kubectl logs -n ocx-system deployment/ocx-webhook | grep "error_type="
```

## Monitoring and Alerting

### Key Metrics to Monitor

```promql
# Webhook availability
up{job="ocx-webhook"}

# Admission request rate
rate(ocx_webhook_admission_requests_total[5m])

# Error rate
rate(ocx_webhook_errors_total[5m])

# Injection success rate
rate(ocx_webhook_injection_requests_total{result="success"}[5m]) / rate(ocx_webhook_injection_requests_total[5m])

# Webhook latency
histogram_quantile(0.95, rate(ocx_webhook_admission_duration_seconds_bucket[5m]))
```

### Recommended Alerts

```yaml
# Webhook down
- alert: OCXWebhookDown
  expr: up{job="ocx-webhook"} == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "OCX Webhook is down"

# High error rate
- alert: OCXWebhookHighErrorRate
  expr: rate(ocx_webhook_errors_total[5m]) > 0.1
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "OCX Webhook error rate is high"

# High latency
- alert: OCXWebhookHighLatency
  expr: histogram_quantile(0.95, rate(ocx_webhook_admission_duration_seconds_bucket[5m])) > 0.01
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "OCX Webhook latency is high"
```

## Recovery Procedures

### Complete Webhook Reset

```bash
# 1. Delete webhook configuration
kubectl delete mutatingwebhookconfiguration ocx-webhook

# 2. Delete webhook deployment
kubectl delete deployment ocx-webhook -n ocx-system

# 3. Delete certificates
kubectl delete secret ocx-webhook-certs -n ocx-system

# 4. Regenerate certificates
./scripts/generate-certs.sh

# 5. Redeploy webhook
kubectl apply -f k8s/

# 6. Verify deployment
kubectl wait --for=condition=available deployment/ocx-webhook -n ocx-system
```

### Certificate Renewal

```bash
# 1. Backup current certificates
kubectl get secret ocx-webhook-certs -n ocx-system -o yaml > backup-certs.yaml

# 2. Generate new certificates
./scripts/generate-certs.sh

# 3. Apply new certificates
kubectl apply -f certs/webhook-certs-secret.yaml

# 4. Restart webhook
kubectl rollout restart deployment/ocx-webhook -n ocx-system

# 5. Verify new certificates
kubectl get secret ocx-webhook-certs -n ocx-system -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout
```

## Support and Escalation

### Self-Service Resources

1. **Documentation**: [OCX Webhook Docs](https://docs.ocx.dev/webhook)
2. **Examples**: [GitHub Examples](https://github.com/ocx-protocol/webhook/tree/main/examples)
3. **Community**: [OCX Slack](https://ocx-community.slack.com)

### Escalation Process

1. **Level 1**: Check this troubleshooting guide
2. **Level 2**: Enable debug mode and collect logs
3. **Level 3**: Contact enterprise support with:
   - Webhook logs
   - Kubernetes cluster info
   - Error reproduction steps
   - Debug output

### Information to Collect

When reporting issues, collect:

```bash
# Webhook status
kubectl get all -n ocx-system -l app.kubernetes.io/name=ocx-webhook

# Webhook configuration
kubectl get mutatingwebhookconfiguration ocx-webhook -o yaml

# Recent logs
kubectl logs -n ocx-system deployment/ocx-webhook --tail=100

# Cluster info
kubectl version
kubectl get nodes

# Resource usage
kubectl top pod -n ocx-system -l app.kubernetes.io/name=ocx-webhook
```
