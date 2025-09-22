# OCX Webhook Upgrade Guide

## Overview

This guide covers upgrading the OCX Kubernetes Mutating Webhook between versions, including compatibility requirements, upgrade procedures, and rollback strategies.

## Version Compatibility Matrix

| Webhook Version | Kubernetes | OCX Protocol | cert-manager | Go Version |
|----------------|------------|--------------|--------------|------------|
| 1.0.x | 1.19+ | 1.0+ | 1.0+ | 1.18+ |
| 1.1.x | 1.20+ | 1.1+ | 1.2+ | 1.19+ |
| 1.2.x | 1.21+ | 1.2+ | 1.3+ | 1.20+ |
| 1.3.x | 1.22+ | 1.3+ | 1.4+ | 1.21+ |

## Pre-Upgrade Checklist

### 1. Verify Current Version

```bash
# Check current webhook version
kubectl get deployment ocx-webhook -n ocx-system -o jsonpath='{.spec.template.spec.containers[0].image}'

# Check webhook configuration version
kubectl get mutatingwebhookconfiguration ocx-webhook -o jsonpath='{.metadata.labels.app\.kubernetes\.io/version}'

# Check OCX Protocol version
kubectl exec -n ocx-system deployment/ocx-webhook -- ocx --version
```

### 2. Check Compatibility

```bash
# Verify Kubernetes version
kubectl version --short

# Check cert-manager version (if used)
kubectl get pods -n cert-manager -o jsonpath='{.items[0].spec.containers[0].image}'

# Verify cluster resources
kubectl top nodes
kubectl get nodes --no-headers | wc -l
```

### 3. Backup Current Configuration

```bash
# Create backup directory
mkdir -p backup/$(date +%Y%m%d-%H%M%S)
cd backup/$(date +%Y%m%d-%H%M%S)

# Backup webhook configuration
kubectl get mutatingwebhookconfiguration ocx-webhook -o yaml > webhook-config.yaml

# Backup deployment
kubectl get deployment ocx-webhook -n ocx-system -o yaml > deployment.yaml

# Backup certificates
kubectl get secret ocx-webhook-certs -n ocx-system -o yaml > certificates.yaml

# Backup RBAC
kubectl get clusterrole ocx-webhook -o yaml > clusterrole.yaml
kubectl get clusterrolebinding ocx-webhook -o yaml > clusterrolebinding.yaml

# Backup service account
kubectl get serviceaccount ocx-webhook -n ocx-system -o yaml > serviceaccount.yaml
```

### 4. Test Environment Validation

```bash
# Run pre-upgrade tests
./scripts/test-injection.sh

# Check webhook health
kubectl get pods -n ocx-system -l app.kubernetes.io/name=ocx-webhook

# Verify metrics
kubectl port-forward -n ocx-system svc/ocx-webhook 9090:9090 &
curl http://localhost:9090/metrics | grep ocx_webhook
```

## Upgrade Procedures

### Minor Version Upgrades (1.0.x → 1.1.x)

Minor upgrades are generally safe and backward compatible.

#### 1. Update Webhook Image

```bash
# Update to new version
kubectl set image deployment/ocx-webhook webhook=ocx-webhook:v1.1.0 -n ocx-system

# Monitor rollout
kubectl rollout status deployment/ocx-webhook -n ocx-system

# Verify new version
kubectl get deployment ocx-webhook -n ocx-system -o jsonpath='{.spec.template.spec.containers[0].image}'
```

#### 2. Update Configuration (if needed)

```bash
# Apply new configuration
kubectl apply -f k8s/webhook-config.yaml

# Verify configuration
kubectl get mutatingwebhookconfiguration ocx-webhook -o yaml
```

#### 3. Verify Upgrade

```bash
# Check webhook health
kubectl get pods -n ocx-system -l app.kubernetes.io/name=ocx-webhook

# Test injection
./scripts/test-injection.sh

# Check logs for errors
kubectl logs -n ocx-system deployment/ocx-webhook --tail=50
```

### Major Version Upgrades (1.x.x → 2.x.x)

Major upgrades may include breaking changes and require more careful planning.

#### 1. Review Breaking Changes

```bash
# Check release notes for breaking changes
curl -s https://api.github.com/repos/ocx-protocol/webhook/releases/latest | jq '.body'

# Review migration guide
cat docs/MIGRATION.md
```

#### 2. Update Dependencies

```bash
# Update Kubernetes manifests
kubectl apply -f k8s/

# Update certificates if needed
./scripts/generate-certs.sh
kubectl apply -f certs/webhook-certs-secret.yaml
```

#### 3. Rolling Upgrade

```bash
# Update image
kubectl set image deployment/ocx-webhook webhook=ocx-webhook:v2.0.0 -n ocx-system

# Monitor each pod during rollout
kubectl get pods -n ocx-system -l app.kubernetes.io/name=ocx-webhook -w

# Verify all pods are ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=ocx-webhook -n ocx-system
```

#### 4. Post-Upgrade Validation

```bash
# Run comprehensive tests
make test

# Test injection with new features
kubectl apply -f examples/test-pod.yaml
kubectl describe pod ocx-demo-pod

# Check metrics and logs
kubectl logs -n ocx-system deployment/ocx-webhook --tail=100
```

## Rollback Procedures

### Quick Rollback (Same Session)

```bash
# Rollback to previous version
kubectl rollout undo deployment/ocx-webhook -n ocx-system

# Monitor rollback
kubectl rollout status deployment/ocx-webhook -n ocx-system

# Verify rollback
kubectl get deployment ocx-webhook -n ocx-system -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### Manual Rollback (From Backup)

```bash
# Restore from backup
cd backup/YYYYMMDD-HHMMSS

# Restore webhook configuration
kubectl apply -f webhook-config.yaml

# Restore deployment
kubectl apply -f deployment.yaml

# Restore certificates
kubectl apply -f certificates.yaml

# Verify restoration
kubectl get all -n ocx-system -l app.kubernetes.io/name=ocx-webhook
```

### Emergency Rollback

```bash
# Disable webhook temporarily
kubectl patch mutatingwebhookconfiguration ocx-webhook --type='json' -p='[{"op": "replace", "path": "/webhooks/0/failurePolicy", "value": "Ignore"}]'

# Scale down webhook
kubectl scale deployment ocx-webhook -n ocx-system --replicas=0

# Restore previous version
kubectl set image deployment/ocx-webhook webhook=ocx-webhook:v1.0.0 -n ocx-system

# Scale back up
kubectl scale deployment ocx-webhook -n ocx-system --replicas=2

# Re-enable webhook
kubectl patch mutatingwebhookconfiguration ocx-webhook --type='json' -p='[{"op": "replace", "path": "/webhooks/0/failurePolicy", "value": "Fail"}]'
```

## Upgrade Strategies

### Blue-Green Deployment

```bash
# Deploy new version alongside old
kubectl apply -f k8s/webhook-v2.yaml

# Test new version
kubectl get pods -n ocx-system -l version=v2.0.0

# Switch traffic
kubectl patch mutatingwebhookconfiguration ocx-webhook --type='json' -p='[{"op": "replace", "path": "/webhooks/0/clientConfig/service/name", "value": "ocx-webhook-v2"}]'

# Remove old version
kubectl delete deployment ocx-webhook-v1 -n ocx-system
```

### Canary Deployment

```bash
# Deploy canary version
kubectl apply -f k8s/webhook-canary.yaml

# Route small percentage of traffic
kubectl patch mutatingwebhookconfiguration ocx-webhook --type='json' -p='[{"op": "add", "path": "/webhooks/0/objectSelector", "value": {"matchLabels": {"canary": "true"}}}]'

# Monitor canary performance
kubectl logs -n ocx-system deployment/ocx-webhook-canary -f

# Gradually increase traffic
kubectl patch mutatingwebhookconfiguration ocx-webhook --type='json' -p='[{"op": "remove", "path": "/webhooks/0/objectSelector"}]'
```

## Post-Upgrade Tasks

### 1. Verify Functionality

```bash
# Run test suite
make test

# Test injection
./scripts/test-injection.sh

# Check metrics
kubectl port-forward -n ocx-system svc/ocx-webhook 9090:9090 &
curl http://localhost:9090/metrics | grep ocx_webhook
```

### 2. Update Monitoring

```bash
# Update ServiceMonitor if needed
kubectl apply -f k8s/servicemonitor.yaml

# Update Grafana dashboards
kubectl apply -f monitoring/grafana-dashboard.yaml

# Verify alerts
kubectl get prometheusrules -n ocx-system
```

### 3. Update Documentation

```bash
# Update cluster documentation
kubectl annotate deployment ocx-webhook -n ocx-system version=v1.1.0

# Update runbooks
echo "OCX Webhook upgraded to v1.1.0 on $(date)" >> runbook/upgrades.log
```

## Troubleshooting Upgrades

### Common Upgrade Issues

#### 1. Image Pull Errors

```bash
# Check image availability
docker pull ocx-webhook:v1.1.0

# Verify image registry access
kubectl describe pod -n ocx-system -l app.kubernetes.io/name=ocx-webhook

# Check image pull secrets
kubectl get secrets -n ocx-system | grep docker
```

#### 2. Configuration Conflicts

```bash
# Check for configuration conflicts
kubectl get mutatingwebhookconfiguration ocx-webhook -o yaml

# Compare with previous version
kubectl get mutatingwebhookconfiguration ocx-webhook -o yaml > current-config.yaml
diff backup/webhook-config.yaml current-config.yaml
```

#### 3. Certificate Issues

```bash
# Check certificate validity
kubectl get secret ocx-webhook-certs -n ocx-system -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout

# Regenerate if needed
./scripts/generate-certs.sh
kubectl apply -f certs/webhook-certs-secret.yaml
```

### Recovery Procedures

#### 1. Partial Upgrade Failure

```bash
# Check pod status
kubectl get pods -n ocx-system -l app.kubernetes.io/name=ocx-webhook

# Check events
kubectl get events -n ocx-system --sort-by='.lastTimestamp'

# Restart failed pods
kubectl delete pod -n ocx-system -l app.kubernetes.io/name=ocx-webhook
```

#### 2. Complete Upgrade Failure

```bash
# Disable webhook
kubectl patch mutatingwebhookconfiguration ocx-webhook --type='json' -p='[{"op": "replace", "path": "/webhooks/0/failurePolicy", "value": "Ignore"}]'

# Rollback to previous version
kubectl rollout undo deployment/ocx-webhook -n ocx-system

# Re-enable webhook
kubectl patch mutatingwebhookconfiguration ocx-webhook --type='json' -p='[{"op": "replace", "path": "/webhooks/0/failurePolicy", "value": "Fail"}]'
```

## Best Practices

### 1. Upgrade Planning

- **Schedule upgrades** during maintenance windows
- **Test in staging** environment first
- **Have rollback plan** ready
- **Notify stakeholders** of upgrade schedule

### 2. Monitoring During Upgrade

```bash
# Monitor webhook health
kubectl get pods -n ocx-system -l app.kubernetes.io/name=ocx-webhook -w

# Monitor admission requests
kubectl logs -n ocx-system deployment/ocx-webhook -f | grep -i "admission\|injection"

# Monitor metrics
kubectl port-forward -n ocx-system svc/ocx-webhook 9090:9090 &
watch -n 5 'curl -s http://localhost:9090/metrics | grep ocx_webhook_admission_requests_total'
```

### 3. Post-Upgrade Validation

```bash
# Run comprehensive tests
make test
./scripts/test-injection.sh

# Check performance
kubectl top pod -n ocx-system -l app.kubernetes.io/name=ocx-webhook

# Verify functionality
kubectl apply -f examples/test-pod.yaml
kubectl exec ocx-demo-pod -- ocx --version
kubectl delete pod ocx-demo-pod
```

## Support and Escalation

### Pre-Upgrade Support

- **Documentation**: [Upgrade Guide](https://docs.ocx.dev/webhook/upgrade)
- **Community**: [OCX Slack](https://ocx-community.slack.com)
- **Issues**: [GitHub Issues](https://github.com/ocx-protocol/webhook/issues)

### Post-Upgrade Support

- **Enterprise Support**: enterprise@ocx.dev
- **Emergency Hotline**: +1-800-OCX-HELP
- **Documentation**: [Troubleshooting Guide](https://docs.ocx.dev/webhook/troubleshooting)

### Information to Collect

When reporting upgrade issues:

```bash
# Collect upgrade information
echo "Upgrade Information" > upgrade-report.txt
echo "===================" >> upgrade-report.txt
echo "From Version: $(kubectl get deployment ocx-webhook -n ocx-system -o jsonpath='{.metadata.annotations.previous-version}')" >> upgrade-report.txt
echo "To Version: $(kubectl get deployment ocx-webhook -n ocx-system -o jsonpath='{.spec.template.spec.containers[0].image}')" >> upgrade-report.txt
echo "Upgrade Time: $(date)" >> upgrade-report.txt
echo "" >> upgrade-report.txt

# Collect error logs
kubectl logs -n ocx-system deployment/ocx-webhook --tail=100 >> upgrade-report.txt

# Collect cluster info
kubectl version >> upgrade-report.txt
kubectl get nodes >> upgrade-report.txt
```
