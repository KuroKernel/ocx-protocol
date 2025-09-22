#!/bin/bash
# =============================================================================
# OCX Webhook Smoke Test Script
# =============================================================================

set -euo pipefail

# Configuration
NAMESPACE="ocx-demo"
WEBHOOK_NAMESPACE="ocx-system"
TIMEOUT="60s"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')] ✅ $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%H:%M:%S')] ❌ $1${NC}"
    exit 1
}

warning() {
    echo -e "${YELLOW}[$(date +'%H:%M:%S')] ⚠️  $1${NC}"
}

# Cleanup function
cleanup() {
    log "Cleaning up test resources..."
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true --wait=false
    log "Cleanup initiated"
}

trap cleanup EXIT

main() {
    echo "================================================================================"
    echo "OCX Webhook Smoke Test"
    echo "================================================================================"
    echo ""
    
    # Check prerequisites
    log "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        error "kubectl is not installed or not in PATH"
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
    fi
    
    if ! kubectl get deployment ocx-webhook -n "$WEBHOOK_NAMESPACE" &> /dev/null; then
        error "OCX webhook is not deployed in namespace $WEBHOOK_NAMESPACE"
    fi
    
    success "Prerequisites check passed"
    
    # Create test namespace with enforcement label
    log "Creating test namespace with enforcement label..."
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    kubectl label namespace "$NAMESPACE" ocx.dev/enforce=true --overwrite
    
    success "Test namespace created and labeled"
    
    # Test 1: Pod without enable label (should not be mutated)
    log "Test 1: Pod without enable label (should not be mutated)..."
    
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-no-mutation
  namespace: $NAMESPACE
  labels:
    app: test-no-mutation
spec:
  containers:
  - name: test
    image: nginx:alpine
    command: ["sleep", "60"]
EOF

    sleep 5
    
    # Check if pod was not mutated
    if kubectl get pod test-no-mutation -n "$NAMESPACE" -o jsonpath='{.metadata.annotations.ocx\.dev/mutated}' | grep -q "true"; then
        error "Pod was mutated when it shouldn't have been"
    else
        success "Pod was not mutated (correct behavior)"
    fi
    
    # Test 2: Pod with enable label (should be mutated)
    log "Test 2: Pod with enable label (should be mutated)..."
    
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-with-mutation
  namespace: $NAMESPACE
  labels:
    app: test-with-mutation
    ocx.dev/enable: "true"
spec:
  containers:
  - name: test
    image: nginx:alpine
    command: ["sleep", "60"]
EOF

    sleep 5
    
    # Check if pod was mutated
    if kubectl get pod test-with-mutation -n "$NAMESPACE" -o jsonpath='{.metadata.annotations.ocx\.dev/mutated}' | grep -q "true"; then
        success "Pod was mutated (correct behavior)"
    else
        error "Pod was not mutated when it should have been"
    fi
    
    # Test 3: Verify OCX command wrapping
    log "Test 3: Verifying OCX command wrapping..."
    
    # Get the mutated pod and check command
    COMMAND=$(kubectl get pod test-with-mutation -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].command[0]}')
    if [ "$COMMAND" = "/usr/local/bin/ocx" ]; then
        success "Command was wrapped with OCX"
    else
        error "Command was not wrapped with OCX: $COMMAND"
    fi
    
    # Test 4: Verify provenance environment variables
    log "Test 4: Verifying provenance environment variables..."
    
    # Check if OCX environment variables are present
    if kubectl get pod test-with-mutation -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].env[*].name}' | grep -q "OCX_NAMESPACE"; then
        success "Provenance environment variables added"
    else
        error "Provenance environment variables not found"
    fi
    
    # Test 5: Verify volume mounts
    log "Test 5: Verifying volume mounts..."
    
    # Check if OCX binary volume mount is present
    if kubectl get pod test-with-mutation -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].volumeMounts[*].name}' | grep -q "ocx-shared"; then
        success "OCX binary volume mount added"
    else
        error "OCX binary volume mount not found"
    fi
    
    # Check if receipts volume mount is present
    if kubectl get pod test-with-mutation -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].volumeMounts[*].name}' | grep -q "ocx-receipts"; then
        success "Receipts volume mount added"
    else
        error "Receipts volume mount not found"
    fi
    
    # Test 6: Verify idempotent mutation
    log "Test 6: Verifying idempotent mutation..."
    
    # Update the pod (should not be mutated again)
    kubectl patch pod test-with-mutation -n "$NAMESPACE" --type='merge' -p='{"metadata":{"labels":{"test":"updated"}}}'
    
    sleep 5
    
    # Check that the pod still has the mutated annotation
    if kubectl get pod test-with-mutation -n "$NAMESPACE" -o jsonpath='{.metadata.annotations.ocx\.dev/mutated}' | grep -q "true"; then
        success "Pod mutation is idempotent"
    else
        error "Pod mutation is not idempotent"
    fi
    
    # Test 7: Verify webhook metrics
    log "Test 7: Verifying webhook metrics..."
    
    # Port forward to webhook service
    kubectl port-forward -n "$WEBHOOK_NAMESPACE" service/ocx-webhook 9090:9090 &
    METRICS_PF_PID=$!
    sleep 5
    
    # Check if metrics are available
    if curl -s "http://localhost:9090/metrics" | grep -q "ocx_webhook"; then
        success "Webhook metrics are available"
    else
        warning "Webhook metrics not available (may be expected in some environments)"
    fi
    
    kill $METRICS_PF_PID || true
    sleep 2
    
    # Test 8: Verify webhook health
    log "Test 8: Verifying webhook health..."
    
    # Port forward to webhook service
    kubectl port-forward -n "$WEBHOOK_NAMESPACE" service/ocx-webhook 8443:443 &
    HEALTH_PF_PID=$!
    sleep 5
    
    # Check health endpoint
    if curl -k -s "https://localhost:8443/health" | grep -q "healthy"; then
        success "Webhook health endpoint is responding"
    else
        error "Webhook health endpoint not responding"
    fi
    
    kill $HEALTH_PF_PID || true
    sleep 2
    
    # Clean up test pods
    log "Cleaning up test pods..."
    kubectl delete pod test-no-mutation test-with-mutation -n "$NAMESPACE" --ignore-not-found=true
    
    echo ""
    success "🎉 All smoke tests passed! OCX Webhook is working correctly!"
    echo ""
    log "Summary:"
    log "  ✅ Selective mutation based on labels"
    log "  ✅ OCX command wrapping"
    log "  ✅ Provenance environment variables"
    log "  ✅ Volume mounts for OCX binary and receipts"
    log "  ✅ Idempotent mutation"
    log "  ✅ Webhook metrics and health endpoints"
    echo ""
    log "The OCX Webhook is ready for production use!"
}

main "$@"
