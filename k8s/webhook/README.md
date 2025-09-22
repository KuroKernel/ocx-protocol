# OCX Kubernetes Mutating Webhook

This directory contains the Kubernetes mutating admission webhook for OCX Protocol integration.

## Overview

The OCX webhook automatically injects OCX Protocol components into Kubernetes pods based on annotations. It supports two injection methods:

1. **Init Container Injection** (`ocx-inject: "true"`) - Adds OCX binary and keystore to all containers
2. **Sidecar Injection** (`ocx-inject: "verify"`) - Adds OCX verification sidecar container

## Features

- ✅ **Automatic OCX Injection** - Based on pod annotations
- ✅ **Multiple Injection Methods** - Init container and sidecar support
- ✅ **Security Hardened** - Non-root containers, read-only filesystems
- ✅ **Prometheus Metrics** - Comprehensive monitoring
- ✅ **Health Checks** - Liveness and readiness probes
- ✅ **TLS Security** - Encrypted communication
- ✅ **Resource Management** - CPU and memory limits
- ✅ **High Availability** - PodDisruptionBudget and anti-affinity
- ✅ **Network Security** - NetworkPolicy for ingress/egress control
- ✅ **Certificate Management** - Manual and cert-manager support
- ✅ **Prometheus Integration** - ServiceMonitor for monitoring
- ✅ **Production Hardening** - Security contexts and resource limits

## Quick Start

### Production Deployment (Recommended)

```bash
cd k8s/webhook

# Deploy with manual certificates (default)
./deploy-production.sh deploy

# Or deploy with cert-manager (if installed)
./deploy-production.sh deploy cert-manager
```

### Manual Deployment

```bash
# Generate certificates
./generate-certs.sh

# Deploy all components
kubectl apply -f namespace.yaml
kubectl apply -f rbac.yaml
kubectl apply -f configmap.yaml
kubectl apply -f certs/secret.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f poddisruptionbudget.yaml
kubectl apply -f networkpolicy.yaml
kubectl apply -f webhook-config.yaml

# Optional: Add monitoring
kubectl apply -f servicemonitor.yaml
```

### Enable Webhook for Namespace

```bash
# Enable for default namespace
kubectl label namespace default ocx-inject=enabled

# Enable for specific namespace
kubectl label namespace <namespace> ocx-inject=enabled
```

## Usage

### Basic OCX Injection

Add the following annotation to your pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    ocx-inject: "true"
spec:
  containers:
  - name: app
    image: nginx
```

### Advanced Configuration

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    ocx-inject: "true"           # Enable OCX injection
    ocx-cycles: "50000"          # Custom cycle limit
    ocx-profile: "v1-min"        # OCX profile
    ocx-keystore: "production"   # Keystore name
    ocx-verify-only: "true"      # Verification only mode
spec:
  containers:
  - name: app
    image: nginx
```

### Sidecar Verification

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    ocx-inject: "verify"         # Sidecar injection
    ocx-cycles: "10000"
spec:
  containers:
  - name: app
    image: nginx
```

## Annotations

| Annotation | Values | Default | Description |
|------------|--------|---------|-------------|
| `ocx-inject` | `"true"`, `"verify"`, `"false"` | `"false"` | Enable OCX injection |
| `ocx-cycles` | `1-1000000` | `"10000"` | Maximum cycle limit |
| `ocx-profile` | `"v1-min"` | `"v1-min"` | OCX protocol profile |
| `ocx-keystore` | string | `"default"` | Keystore identifier |
| `ocx-verify-only` | `"true"`, `"false"` | `"false"` | Verification only mode |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_PORT` | `8443` | Webhook HTTPS port |
| `METRICS_PORT` | `9090` | Prometheus metrics port |
| `OCX_SERVER_URL` | `http://ocx-server:8080` | OCX API server URL |
| `TLS_CERT_FILE` | `/etc/certs/tls.crt` | TLS certificate file |
| `TLS_KEY_FILE` | `/etc/certs/tls.key` | TLS private key file |
| `DEBUG_MODE` | `false` | Enable debug logging |

## Monitoring

### Prometheus Metrics

The webhook exposes Prometheus metrics on port 9090:

- `ocx_webhook_admission_requests_total` - Total admission requests
- `ocx_webhook_admission_duration_seconds` - Request processing duration
- `ocx_webhook_injection_requests_total` - OCX injection requests
- `ocx_webhook_errors_total` - Webhook processing errors

### Health Endpoints

- `GET /health` - Basic health check
- `GET /readyz` - Readiness probe (checks OCX server connectivity)

## Security

### Pod Security

- **Non-root containers** - Runs as user 65534 (nobody)
- **Read-only filesystem** - Immutable container filesystem
- **Dropped capabilities** - All Linux capabilities dropped
- **No privilege escalation** - Privilege escalation disabled

### Network Security

- **TLS 1.2+** - Encrypted communication
- **Certificate validation** - Mutual TLS authentication
- **Network policies** - Isolated network access

## Troubleshooting

### Check Webhook Status

```bash
# Check webhook deployment
kubectl get deployment -n ocx-system

# Check webhook logs
kubectl logs -n ocx-system -l app.kubernetes.io/component=webhook

# Check webhook configuration
kubectl get mutatingwebhookconfiguration ocx-webhook
```

### Test Webhook

```bash
# Create test pod
kubectl run test-pod --image=nginx --dry-run=client -o yaml | kubectl apply -f -

# Check if OCX was injected
kubectl describe pod test-pod
kubectl exec test-pod -- ls -la /usr/local/bin/ocx
```

### Common Issues

1. **Webhook not triggered** - Check namespace labels
2. **Certificate errors** - Regenerate certificates
3. **OCX server unreachable** - Check OCX server URL
4. **Resource limits** - Adjust CPU/memory limits

## Development

### Building the Webhook

```bash
cd cmd/ocx-webhook
go build -o ocx-webhook .
```

### Building Docker Image

```bash
docker build -t ocx-protocol/webhook:latest .
```

### Local Testing

```bash
# Run webhook locally
./ocx-webhook --port=8443 --cert-file=certs/tls.crt --key-file=certs/tls.key
```

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Kubernetes    │    │   OCX Webhook    │    │   OCX Server    │
│   API Server    │◄──►│   (Mutating)     │◄──►│   (API)         │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Pod Creation  │    │   OCX Injection  │    │   Receipt       │
│   Request       │    │   (Init/Sidecar) │    │   Generation    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## License

MIT License - see LICENSE file for details.
