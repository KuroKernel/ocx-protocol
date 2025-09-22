#!/bin/bash
set -euo pipefail

NAMESPACE="ocx-system"
TIMEOUT="300s"

echo "Installing OCX Kubernetes Webhook..."
echo "====================================="

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed or not in PATH"
    exit 1
fi

# Check if cert-manager is available (optional)
if kubectl get crd certificates.cert-manager.io &> /dev/null; then
    echo "✓ cert-manager detected - will use automated certificate management"
    CERT_MANAGER=true
else
    echo "! cert-manager not found - will use self-signed certificates"
    CERT_MANAGER=false
fi

# Create namespace
echo "Creating namespace: $NAMESPACE"
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Generate certificates if cert-manager is not available
if [ "$CERT_MANAGER" = false ]; then
    echo "Generating self-signed certificates..."
    ./scripts/generate-certs.sh
    kubectl apply -f ./certs/webhook-certs-secret.yaml
fi

# Apply manifests
echo "Applying Kubernetes manifests..."
kubectl apply -f k8s/

# Wait for deployment
echo "Waiting for webhook deployment to be ready..."
kubectl wait --for=condition=available --timeout="$TIMEOUT" deployment/ocx-webhook -n "$NAMESPACE"

# Verify installation
echo "Verifying installation..."
kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=ocx-webhook
kubectl get mutatingwebhookconfiguration ocx-webhook

echo ""
echo "✓ OCX Kubernetes Webhook installed successfully!"
echo ""
echo "To test the webhook, apply a pod with OCX annotations:"
echo "  kubectl apply -f examples/test-pod.yaml"
echo ""
echo "To uninstall:"
echo "  make undeploy"
echo ""
