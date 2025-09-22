#!/bin/bash

set -euo pipefail

# Configuration
NAMESPACE="ocx-system"
SERVICE_NAME="ocx-webhook"
SECRET_NAME="ocx-webhook-certs"
CERT_DIR="./certs"

echo "🔐 Generating TLS certificates for OCX Webhook..."

# Create certs directory
mkdir -p "$CERT_DIR"

# Generate private key
openssl genrsa -out "$CERT_DIR/tls.key" 2048

# Generate certificate signing request
openssl req -new -key "$CERT_DIR/tls.key" -out "$CERT_DIR/tls.csr" -subj "/CN=$SERVICE_NAME.$NAMESPACE.svc"

# Generate self-signed certificate
openssl x509 -req -in "$CERT_DIR/tls.csr" -signkey "$CERT_DIR/tls.key" -out "$CERT_DIR/tls.crt" -days 365

# Clean up CSR
rm "$CERT_DIR/tls.csr"

echo "✅ Certificates generated in $CERT_DIR/"

# Create Kubernetes secret
echo "📦 Creating Kubernetes secret..."

kubectl create secret tls "$SECRET_NAME" \
  --cert="$CERT_DIR/tls.crt" \
  --key="$CERT_DIR/tls.key" \
  --namespace="$NAMESPACE" \
  --dry-run=client -o yaml > "$CERT_DIR/secret.yaml"

echo "✅ Secret YAML created at $CERT_DIR/secret.yaml"

# Extract CA bundle for webhook configuration
echo "🔧 Extracting CA bundle..."

CA_BUNDLE=$(base64 -w 0 < "$CERT_DIR/tls.crt")
echo "CA Bundle: $CA_BUNDLE"

# Update webhook configuration with CA bundle
sed "s/caBundle: \"\"/caBundle: \"$CA_BUNDLE\"/" webhook-config.yaml > webhook-config-with-ca.yaml

echo "✅ Webhook configuration updated with CA bundle"
echo ""
echo "🚀 To deploy the webhook:"
echo "1. kubectl apply -f namespace.yaml"
echo "2. kubectl apply -f rbac.yaml"
echo "3. kubectl apply -f $CERT_DIR/secret.yaml"
echo "4. kubectl apply -f deployment.yaml"
echo "5. kubectl apply -f service.yaml"
echo "6. kubectl apply -f webhook-config-with-ca.yaml"
echo ""
echo "🧪 To test the webhook:"
echo "kubectl label namespace default ocx-inject=enabled"
echo "kubectl run test-pod --image=nginx --dry-run=client -o yaml | kubectl apply -f -"
