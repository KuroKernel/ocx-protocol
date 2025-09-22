#!/bin/bash
set -euo pipefail

CERT_DIR="./certs"
NAMESPACE="ocx-system"
SERVICE_NAME="ocx-webhook"

echo "Generating self-signed certificates for OCX webhook..."

# Create certificates directory
mkdir -p "$CERT_DIR"

# Generate private key
openssl genrsa -out "$CERT_DIR/tls.key" 2048

# Generate certificate signing request
cat > "$CERT_DIR/csr.conf" <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = $SERVICE_NAME.$NAMESPACE.svc

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = $SERVICE_NAME
DNS.2 = $SERVICE_NAME.$NAMESPACE
DNS.3 = $SERVICE_NAME.$NAMESPACE.svc
DNS.4 = $SERVICE_NAME.$NAMESPACE.svc.cluster.local
EOF

# Generate certificate
openssl req -new -key "$CERT_DIR/tls.key" -out "$CERT_DIR/tls.csr" -config "$CERT_DIR/csr.conf"
openssl x509 -req -in "$CERT_DIR/tls.csr" -signkey "$CERT_DIR/tls.key" -out "$CERT_DIR/tls.crt" \
    -extensions v3_req -extfile "$CERT_DIR/csr.conf" -days 365

# Create Kubernetes secret
kubectl create secret tls ocx-webhook-certs \
    --cert="$CERT_DIR/tls.crt" \
    --key="$CERT_DIR/tls.key" \
    --namespace="$NAMESPACE" \
    --dry-run=client -o yaml > "$CERT_DIR/webhook-certs-secret.yaml"

echo "Certificates generated successfully in $CERT_DIR/"
echo "Apply the secret with: kubectl apply -f $CERT_DIR/webhook-certs-secret.yaml"

# Cleanup
rm -f "$CERT_DIR/tls.csr" "$CERT_DIR/csr.conf"
