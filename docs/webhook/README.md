# OCX Kubernetes Mutating Webhook

**Enterprise-grade Kubernetes integration for OCX Protocol with zero-code adoption**

## 🚀 Executive Summary

The OCX Kubernetes Mutating Webhook transforms enterprise adoption from "rewrite your code" to "add one annotation". This Fortune 500-grade implementation provides:

- **Zero Code Changes**: Add `ocx-inject: "true"` annotation to any pod
- **Enterprise Security**: Production-ready with TLS, RBAC, and NetworkPolicies
- **Performance**: Sub-5ms injection latency with comprehensive monitoring
- **Reliability**: High availability with graceful degradation and health checks

## 📋 Features

### Core Capabilities
- ✅ **Init Container Injection**: Automatically adds OCX binary and keystore
- ✅ **Sidecar Verification**: Optional verification-only containers
- ✅ **Flexible Configuration**: Cycles, profiles, and keystore selection
- ✅ **Security Hardened**: Non-root execution, read-only filesystems, capability drops
- ✅ **Production Monitoring**: Prometheus metrics, health checks, distributed tracing
- ✅ **Certificate Management**: Auto-rotation with cert-manager integration
- ✅ **High Availability**: Multi-replica deployment with anti-affinity rules

### Injection Methods

| Annotation | Method | Use Case |
|------------|---------|----------|
| `ocx-inject: "true"` | Init Container | Full OCX integration with binary injection |
| `ocx-inject: "verify"` | Sidecar | Verification-only workloads |
| `ocx-inject: "sidecar"` | Sidecar | Custom verification scenarios |

### Advanced Configuration

```yaml
metadata:
  annotations:
    ocx-inject: "true"           # Enable injection
    ocx-cycles: "50000"          # Computation cycles (1-1,000,000)
    ocx-profile: "v1-min"        # Protocol profile
    ocx-keystore: "production"   # Keystore selection
    ocx-verify-only: "true"      # Verification mode only
```

## 🛠️ Installation

### Prerequisites

- Kubernetes 1.19+
- cert-manager (recommended) or manual certificate management
- kubectl with cluster admin permissions

### Quick Install

```bash
# Clone repository
git clone https://github.com/ocx-protocol/webhook
cd webhook

# Build and deploy
make install-deps
make build
make docker-build
make deploy

# Verify installation
kubectl get pods -n ocx-system
kubectl get mutatingwebhookconfiguration ocx-webhook
```

### Manual Installation

```bash
# Create namespace and apply manifests
kubectl create namespace ocx-system
kubectl apply -f k8s/

# Generate certificates (if not using cert-manager)
./scripts/generate-certs.sh
kubectl apply -f certs/webhook-certs-secret.yaml

# Wait for deployment
kubectl wait --for=condition=available deployment/ocx-webhook -n ocx-system
```

## 📖 Usage Examples

### Basic OCX Integration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    metadata:
      annotations:
        ocx-inject: "true"  # 👈 Single annotation enables OCX
    spec:
      containers:
      - name: app
        image: my-app:latest
        # OCX binary automatically available at /usr/local/bin/ocx
        # Environment variables: OCX_CYCLES, OCX_PROFILE, etc.
```

### Advanced Configuration

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: ocx-production-workload
  annotations:
    ocx-inject: "true"
    ocx-cycles: "100000"          # High-computation workload
    ocx-profile: "v1-enterprise"  # Enterprise protocol profile
    ocx-keystore: "prod-hsm"      # Production HSM keystore
spec:
  containers:
  - name: ml-training
    image: tensorflow:latest
    command: ["python", "train.py"]
    # OCX automatically injected, ready to use
```

### Verification-Only Sidecar

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: audit-workload
  annotations:
    ocx-inject: "verify"  # Sidecar for verification only
    ocx-cycles: "25000"
spec:
  containers:
  - name: audit-processor
    image: audit-app:latest
    # OCX verifier available at localhost:8081
```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_PORT` | `8443` | HTTPS port for webhook |
| `METRICS_PORT` | `9090` | Prometheus metrics port |
| `OCX_SERVER_URL` | `http://ocx-server:8080` | OCX Protocol server endpoint |
| `TLS_CERT_FILE` | `/etc/certs/tls.crt` | TLS certificate path |
| `TLS_KEY_FILE` | `/etc/certs/tls.key` | TLS private key path |
| `DEBUG_MODE` | `false` | Enable debug logging |
| `OCX_IMAGE` | `ocx-protocol:latest` | OCX container image to inject |

### Resource Limits

The webhook enforces secure defaults:

```yaml
# Init Container Resources
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 200m
    memory: 256Mi

# Sidecar Resources  
resources:
  requests:
    cpu: 50m
    memory: 64Mi
  limits:
    cpu: 100m
    memory: 128Mi
```

## 📊 Monitoring & Observability

### Prometheus Metrics

```promql
# Admission request rate
rate(ocx_webhook_admission_requests_total[5m])

# Injection success rate
rate(ocx_webhook_injection_requests_total{result="success"}[5m]) /
rate(ocx_webhook_injection_requests_total[5m])

# Webhook latency percentiles
histogram_quantile(0.95, rate(ocx_webhook_admission_duration_seconds_bucket[5m]))

# Error rates by type
rate(ocx_webhook_errors_total[5m])
```

### Health Endpoints

| Endpoint | Purpose | Status Codes |
|----------|---------|--------------|
| `/health` | Basic liveness | `200` (healthy) |
| `/readyz` | Readiness with dependencies | `200` (ready), `503` (not ready) |
| `/metrics` | Prometheus metrics | `200` (metrics data) |

### Logging

```bash
# View webhook logs
kubectl logs -n ocx-system deployment/ocx-webhook

# Debug mode logging
kubectl set env deployment/ocx-webhook DEBUG_MODE=true -n ocx-system
```

## 🧪 Testing

### Fortune 500 Grade Test Suite

The OCX webhook includes a comprehensive Fortune 500-grade test suite that validates enterprise readiness:

```bash
# Run complete Fortune 500 test suite
make test-fortune500

# Or run individual test categories
make test              # Unit tests
make test-integration  # Integration tests  
make test-load         # Load tests
make test-security     # Security tests
make test-all          # All tests
```

### Test Categories

#### 1. **Fortune 500 Test Suite** (`tests/fortune500-test-suite.sh`)
- **Prerequisites validation** - Cluster and webhook readiness
- **Health endpoint testing** - Liveness, readiness, metrics
- **Basic injection testing** - Init container injection validation
- **Sidecar injection testing** - Verification sidecar validation
- **Annotation validation** - Input validation and error handling
- **Security context testing** - Non-root, read-only, capabilities
- **Resource limits testing** - CPU and memory constraints
- **TLS configuration testing** - Certificate validation
- **Performance under load** - 50+ concurrent pods
- **Webhook latency testing** - Sub-5ms injection latency
- **Failover scenarios** - High availability testing
- **Metrics collection** - Prometheus metrics validation
- **Namespace isolation** - Security boundary testing

#### 2. **Load Testing** (`tests/load-test.sh`)
```bash
# Run load test with 100 concurrent pods for 5 minutes
./tests/load-test.sh 100 300

# Monitor webhook performance during load
kubectl top pods -n ocx-system
```

#### 3. **Security Testing** (`tests/security-test.sh`)
```bash
# Run comprehensive security assessment
./tests/security-test.sh

# Tests include:
# - RBAC permissions validation
# - Security context enforcement
# - Network policy validation
# - TLS security assessment
# - Pod Security Standards compliance
# - Resource limits validation
# - Image security scanning
```

### Manual Testing

```bash
# Test basic injection
kubectl apply -f examples/test-pod.yaml

# Verify injection worked
kubectl describe pod ocx-demo-pod
kubectl exec ocx-demo-pod -- ocx --version

# Test sidecar injection
kubectl apply -f examples/sidecar-pod.yaml

# Check sidecar is running
kubectl get pod ocx-sidecar-demo -o jsonpath='{.spec.containers[*].name}'
```

### CI/CD Integration

The webhook includes GitHub Actions workflows for automated testing:

```yaml
# .github/workflows/fortune500-tests.yml
- Unit tests with coverage
- Integration tests with kind cluster
- Security scanning with Trivy
- Performance testing under load
- Fortune 500 validation report
```

### Load Testing

```bash
# Generate load
for i in {1..100}; do
  kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: load-test-$i
  annotations:
    ocx-inject: "true"
spec:
  containers:
  - name: test
    image: nginx:alpine
    command: ["sleep", "60"]
EOF
done

# Monitor webhook performance
kubectl top pod -n ocx-system
kubectl logs -n ocx-system deployment/ocx-webhook --tail=100
```

## 🔒 Security

### Security Features

- **Non-root Execution**: All containers run as user 65534 (nobody)
- **Read-only Filesystems**: No writable filesystem access
- **Capability Dropping**: All Linux capabilities removed
- **Network Policies**: Restricted ingress/egress rules
- **TLS Encryption**: All webhook communication encrypted
- **RBAC**: Minimal required permissions only

### Security Scanning

```bash
# Run security scan
make security-scan

# Check for vulnerabilities
docker run --rm -v $(PWD):/app clair-scanner /app

# Verify RBAC permissions
kubectl auth can-i --list --as=system:serviceaccount:ocx-system:ocx-webhook
```

## 🚀 Production Deployment

### High Availability Setup

```yaml
# Production configuration
spec:
  replicas: 3  # Multi-replica for HA
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  affinity:
    podAntiAffinity:  # Spread across nodes
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app.kubernetes.io/name: ocx-webhook
        topologyKey: kubernetes.io/hostname
```

### cert-manager Integration

```yaml
# Production certificate with Let's Encrypt
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ocx-webhook-cert
  namespace: ocx-system
spec:
  secretName: ocx-webhook-certs
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - ocx-webhook.your-domain.com
```

### Monitoring Setup

```yaml
# ServiceMonitor for Prometheus Operator
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: ocx-webhook
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: ocx-webhook
  endpoints:
  - port: metrics
    interval: 30s
```

## 🔧 Troubleshooting

### Common Issues

#### Webhook Not Injecting

```bash
# Check webhook configuration
kubectl get mutatingwebhookconfiguration ocx-webhook -o yaml

# Verify certificate
kubectl get secret ocx-webhook-certs -n ocx-system -o yaml

# Check webhook logs
kubectl logs -n ocx-system deployment/ocx-webhook
```

#### Certificate Issues

```bash
# Regenerate certificates
./scripts/generate-certs.sh
kubectl delete secret ocx-webhook-certs -n ocx-system
kubectl apply -f certs/webhook-certs-secret.yaml

# Restart webhook
kubectl rollout restart deployment/ocx-webhook -n ocx-system
```

#### Permission Errors

```bash
# Check RBAC
kubectl auth can-i create pods --as=system:serviceaccount:ocx-system:ocx-webhook

# Verify service account
kubectl get serviceaccount ocx-webhook -n ocx-system
kubectl describe clusterrolebinding ocx-webhook
```

### Debug Mode

```bash
# Enable debug logging
kubectl set env deployment/ocx-webhook DEBUG_MODE=true -n ocx-system

# Check detailed logs
kubectl logs -n ocx-system deployment/ocx-webhook -f
```

## 📚 API Reference

### Admission Request Processing

The webhook processes `CREATE` and `UPDATE` operations on Pod objects with OCX annotations:

```go
type InjectionSpec struct {
    Type       string // "true", "verify", "sidecar"
    Cycles     string // "1" to "1000000"
    Profile    string // Protocol profile identifier
    Keystore   string // Keystore selection
    VerifyOnly bool   // Verification-only mode
}
```

### JSON Patch Operations

The webhook generates RFC 6902 JSON Patch operations:

```json
[
  {
    "op": "add",
    "path": "/spec/initContainers/0",
    "value": { "name": "ocx-setup", ... }
  },
  {
    "op": "add", 
    "path": "/spec/volumes/0",
    "value": { "name": "ocx-shared", ... }
  }
]
```

### Metrics API

Prometheus metrics exposed at `/metrics`:

```
# Admission requests by operation and result
ocx_webhook_admission_requests_total{operation="CREATE",kind="Pod",result="success"} 1250

# Request duration histogram
ocx_webhook_admission_duration_seconds_bucket{operation="CREATE",kind="Pod",le="0.005"} 1200

# Injection requests by type
ocx_webhook_injection_requests_total{injection_type="init_container",result="success"} 800

# Error counts by type
ocx_webhook_errors_total{error_type="annotation_error"} 5
```

## 🔄 Upgrade Guide

### Version Compatibility

| Webhook Version | Kubernetes | OCX Protocol | cert-manager |
|----------------|------------|--------------|--------------|
| 1.0.x | 1.19+ | 1.0+ | 1.0+ |
| 1.1.x | 1.20+ | 1.1+ | 1.2+ |

### Upgrade Process

```bash
# Backup current configuration
kubectl get mutatingwebhookconfiguration ocx-webhook -o yaml > backup-webhook-config.yaml

# Update image
kubectl set image deployment/ocx-webhook webhook=ocx-webhook:v1.1.0 -n ocx-system

# Verify upgrade
kubectl rollout status deployment/ocx-webhook -n ocx-system
kubectl get pods -n ocx-system -l app.kubernetes.io/name=ocx-webhook
```

## 🤝 Contributing

### Development Setup

```bash
# Clone and setup
git clone https://github.com/ocx-protocol/webhook
cd webhook
make install-deps

# Run tests
make test
make lint

# Local development
make build
./bin/ocx-webhook --help
```

### Testing Guidelines

- All new features require unit tests with >90% coverage
- Integration tests must pass in kind cluster
- Security scans must pass with zero high/critical vulnerabilities
- Performance tests must maintain <5ms injection latency

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🏢 Enterprise Support

For enterprise support, custom integrations, or Fortune 500 deployments:

- **Email**: enterprise@ocx.dev
- **Slack**: [OCX Enterprise Channel](https://ocx-enterprise.slack.com)
- **Documentation**: [Enterprise Docs](https://docs.ocx.dev/enterprise)

---

**Built with ❤️ by the OCX Protocol Team**

*Transforming enterprise compute verification, one annotation at a time.*
