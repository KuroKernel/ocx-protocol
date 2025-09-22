# OCX Webhook API Reference

## Overview

The OCX Kubernetes Mutating Webhook provides a comprehensive API for integrating OCX Protocol into Kubernetes workloads through admission control.

## Admission Request Processing

### Request Format

The webhook receives `AdmissionReview` objects from the Kubernetes API server:

```json
{
  "apiVersion": "admission.k8s.io/v1",
  "kind": "AdmissionReview",
  "request": {
    "uid": "705ab4f5-6393-11e9-bbdd-42010a800002",
    "kind": {
      "group": "",
      "version": "v1",
      "kind": "Pod"
    },
    "resource": {
      "group": "",
      "version": "v1",
      "resource": "pods"
    },
    "operation": "CREATE",
    "object": {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {
        "name": "test-pod",
        "annotations": {
          "ocx-inject": "true",
          "ocx-cycles": "50000"
        }
      },
      "spec": {
        "containers": [...]
      }
    }
  }
}
```

### Response Format

The webhook responds with an `AdmissionResponse`:

```json
{
  "apiVersion": "admission.k8s.io/v1",
  "kind": "AdmissionReview",
  "response": {
    "uid": "705ab4f5-6393-11e9-bbdd-42010a800002",
    "allowed": true,
    "patch": "W3sib3AiOiJhZGQiLCJwYXRoIjoiL3NwZWMvaW5pdENvbnRhaW5lcnMvMCIsInZhbHVlIjp7Im5hbWUiOiJvY3gtc2V0dXAiLCJpbWFnZSI6Im9jeC1wcm90b2NvbDpsYXRlc3QifX1d",
    "patchType": "JSONPatch"
  }
}
```

## Injection Specifications

### InjectionSpec Structure

```go
type InjectionSpec struct {
    Type       string // Injection method: "true", "verify", "sidecar"
    Cycles     string // Computation cycles: "1" to "1000000"
    Profile    string // OCX protocol profile
    Keystore   string // Keystore identifier
    VerifyOnly bool   // Verification-only mode
}
```

### Annotation Processing

| Annotation | Type | Required | Default | Description |
|------------|------|----------|---------|-------------|
| `ocx-inject` | string | Yes | - | Injection method: "true", "verify", "sidecar" |
| `ocx-cycles` | string | No | "10000" | Computation cycles (1-1000000) |
| `ocx-profile` | string | No | "v1-min" | OCX protocol profile |
| `ocx-keystore` | string | No | "default" | Keystore selection |
| `ocx-verify-only` | string | No | "false" | Verification-only mode |

### Validation Rules

```go
// Cycles validation
if cycles, err := strconv.Atoi(spec.Cycles); err != nil || cycles <= 0 || cycles > 1000000 {
    return nil, fmt.Errorf("invalid ocx-cycles value: %s (must be 1-1000000)", spec.Cycles)
}

// Profile validation
validProfiles := []string{"v1-min", "v1-enterprise", "v1-ultra"}
if !contains(validProfiles, spec.Profile) {
    return nil, fmt.Errorf("invalid ocx-profile: %s", spec.Profile)
}
```

## JSON Patch Operations

### Init Container Injection

```json
[
  {
    "op": "add",
    "path": "/spec/initContainers/0",
    "value": {
      "name": "ocx-setup",
      "image": "ocx-protocol:latest",
      "command": ["/bin/bash", "-c"],
      "args": ["ocx-setup-script"],
      "volumeMounts": [
        {
          "name": "ocx-shared",
          "mountPath": "/shared"
        }
      ],
      "resources": {
        "requests": {"cpu": "100m", "memory": "128Mi"},
        "limits": {"cpu": "200m", "memory": "256Mi"}
      },
      "securityContext": {
        "runAsNonRoot": true,
        "runAsUser": 65534,
        "allowPrivilegeEscalation": false,
        "readOnlyRootFilesystem": true,
        "capabilities": {"drop": ["ALL"]}
      }
    }
  },
  {
    "op": "add",
    "path": "/spec/volumes/0",
    "value": {
      "name": "ocx-shared",
      "emptyDir": {}
    }
  }
]
```

### Sidecar Injection

```json
[
  {
    "op": "add",
    "path": "/spec/containers/-1",
    "value": {
      "name": "ocx-verifier",
      "image": "ocx-protocol:latest",
      "command": ["/usr/local/bin/ocx", "verify-daemon"],
      "args": [
        "--port=8081",
        "--cycles=50000",
        "--profile=v1-min"
      ],
      "ports": [
        {
          "name": "ocx-verify",
          "containerPort": 8081,
          "protocol": "TCP"
        }
      ],
      "resources": {
        "requests": {"cpu": "50m", "memory": "64Mi"},
        "limits": {"cpu": "100m", "memory": "128Mi"}
      },
      "livenessProbe": {
        "httpGet": {
          "path": "/livez",
          "port": 8081
        },
        "initialDelaySeconds": 10,
        "periodSeconds": 30
      },
      "readinessProbe": {
        "httpGet": {
          "path": "/readyz",
          "port": 8081
        },
        "initialDelaySeconds": 5,
        "periodSeconds": 10
      }
    }
  }
]
```

## Environment Variables

### Injected Environment Variables

The webhook automatically injects the following environment variables into application containers:

| Variable | Value | Description |
|----------|-------|-------------|
| `OCX_CYCLES` | Annotation value | Computation cycle limit |
| `OCX_PROFILE` | Annotation value | OCX protocol profile |
| `OCX_KEYSTORE` | Annotation value | Keystore identifier |
| `OCX_SERVER_URL` | Config value | OCX Protocol server URL |
| `OCX_VERIFY_ONLY` | "true"/"false" | Verification-only mode |

### Webhook Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_PORT` | "8443" | HTTPS port for webhook |
| `METRICS_PORT` | "9090" | Prometheus metrics port |
| `OCX_SERVER_URL` | "http://ocx-server:8080" | OCX Protocol server endpoint |
| `TLS_CERT_FILE` | "/etc/certs/tls.crt" | TLS certificate path |
| `TLS_KEY_FILE` | "/etc/certs/tls.key" | TLS private key path |
| `DEBUG_MODE` | "false" | Enable debug logging |
| `OCX_IMAGE` | "ocx-protocol:latest" | OCX container image |

## Health Endpoints

### Health Check

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy",
  "webhook": "ocx-mutating-webhook",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Readiness Check

**Endpoint**: `GET /readyz`

**Response** (Ready):
```json
{
  "status": "ready",
  "webhook": "ocx-mutating-webhook",
  "ocx_server": "connected",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Response** (Not Ready):
```json
{
  "status": "not_ready",
  "webhook": "ocx-mutating-webhook",
  "ocx_server": "disconnected",
  "error": "connection timeout",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Metrics

**Endpoint**: `GET /metrics`

**Response**: Prometheus-formatted metrics

## Prometheus Metrics

### Admission Metrics

```
# Total admission requests by operation, kind, and result
ocx_webhook_admission_requests_total{operation="CREATE",kind="Pod",result="success"} 1250
ocx_webhook_admission_requests_total{operation="CREATE",kind="Pod",result="error"} 5

# Admission request duration histogram
ocx_webhook_admission_duration_seconds_bucket{operation="CREATE",kind="Pod",le="0.001"} 100
ocx_webhook_admission_duration_seconds_bucket{operation="CREATE",kind="Pod",le="0.005"} 1200
ocx_webhook_admission_duration_seconds_bucket{operation="CREATE",kind="Pod",le="0.01"} 1250
ocx_webhook_admission_duration_seconds_sum{operation="CREATE",kind="Pod"} 6.25
ocx_webhook_admission_duration_seconds_count{operation="CREATE",kind="Pod"} 1250
```

### Injection Metrics

```
# Injection requests by type and result
ocx_webhook_injection_requests_total{injection_type="init_container",result="success"} 800
ocx_webhook_injection_requests_total{injection_type="sidecar",result="success"} 200
ocx_webhook_injection_requests_total{injection_type="init_container",result="error"} 10
```

### Error Metrics

```
# Webhook errors by type
ocx_webhook_errors_total{error_type="unmarshal_error"} 2
ocx_webhook_errors_total{error_type="annotation_error"} 3
ocx_webhook_errors_total{error_type="patch_generation_error"} 1
```

## Error Handling

### Error Response Format

```json
{
  "apiVersion": "admission.k8s.io/v1",
  "kind": "AdmissionReview",
  "response": {
    "uid": "705ab4f5-6393-11e9-bbdd-42010a800002",
    "allowed": false,
    "result": {
      "code": 400,
      "message": "Invalid OCX annotation: ocx-cycles must be between 1 and 1000000",
      "reason": "BadRequest"
    }
  }
}
```

### Error Types

| Error Type | HTTP Code | Description |
|------------|-----------|-------------|
| `unmarshal_error` | 400 | Failed to unmarshal pod object |
| `annotation_error` | 400 | Invalid annotation values |
| `patch_generation_error` | 500 | Failed to generate JSON patches |
| `marshal_response_error` | 500 | Failed to marshal response |
| `invalid_injection_type` | 400 | Unsupported injection type |

## Security Considerations

### RBAC Permissions

The webhook requires the following Kubernetes permissions:

```yaml
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["admissionregistration.k8s.io"]
  resources: ["mutatingwebhookconfigurations"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

### Security Context

All injected containers use the following security context:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: ["ALL"]
```

### Network Policies

The webhook supports NetworkPolicy for traffic control:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ocx-webhook
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: ocx-webhook
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: TCP
      port: 8443
```

## Rate Limiting

### Admission Request Limits

- **Max concurrent requests**: 100
- **Request timeout**: 30 seconds
- **Read timeout**: 30 seconds
- **Write timeout**: 30 seconds

### Resource Limits

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 200m
    memory: 256Mi
```

## Versioning

### API Version

The webhook uses `admission.k8s.io/v1` for AdmissionReview objects.

### Supported Kubernetes Versions

- **Minimum**: 1.19
- **Recommended**: 1.21+
- **Tested**: 1.19, 1.20, 1.21, 1.22, 1.23, 1.24, 1.25, 1.26, 1.27, 1.28

### Webhook Version

Current version: `1.0.0`

Version format: `MAJOR.MINOR.PATCH`
- **MAJOR**: Breaking changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible
