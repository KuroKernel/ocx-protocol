#!/bin/bash
# =============================================================================
# OCX Webhook Load Testing Script
# =============================================================================

set -euo pipefail

# Configuration
NAMESPACE="ocx-system"
TEST_NAMESPACE="ocx-load-test"
WEBHOOK_NAME="ocx-webhook"
CONCURRENT_PODS=${1:-100}
TEST_DURATION=${2:-300}  # 5 minutes default

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
    echo -e "${GREEN}[$(date +'%H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%H:%M:%S')] $1${NC}"
    exit 1
}

# Cleanup function
cleanup() {
    log "Cleaning up load test resources..."
    kubectl delete namespace "$TEST_NAMESPACE" --ignore-not-found=true --wait=false
}

trap cleanup EXIT

main() {
    echo "================================================================================"
    echo "OCX Webhook Load Test - $CONCURRENT_PODS concurrent pods for ${TEST_DURATION}s"
    echo "================================================================================"
    
    # Create test namespace
    kubectl create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    log "Starting load test with $CONCURRENT_PODS concurrent pods..."
    
    # Record start time
    START_TIME=$(date +%s)
    
    # Create pods concurrently
    for i in $(seq 1 $CONCURRENT_PODS); do
        cat <<EOF | kubectl apply -f - &
apiVersion: v1
kind: Pod
metadata:
  name: load-test-$i
  namespace: $TEST_NAMESPACE
  labels:
    test-type: load-test
  annotations:
    ocx-inject: "true"
    ocx-cycles: "10000"
spec:
  containers:
  - name: load-test
    image: nginx:alpine
    command: ["sleep", "600"]
    resources:
      requests:
        cpu: 10m
        memory: 32Mi
      limits:
        cpu: 50m
        memory: 64Mi
EOF
    done
    
    # Wait for all background processes
    wait
    
    log "All pods submitted, monitoring progress..."
    
    # Monitor webhook metrics during test
    kubectl port-forward -n "$NAMESPACE" service/"$WEBHOOK_NAME" 9090:9090 &
    METRICS_PF_PID=$!
    sleep 5
    
    # Collect metrics every 10 seconds
    for i in $(seq 1 $((TEST_DURATION / 10))); do
        sleep 10
        
        # Get current metrics
        METRICS=$(curl -s "http://localhost:9090/metrics" 2>/dev/null || echo "")
        
        if [ -n "$METRICS" ]; then
            ADMISSION_REQUESTS=$(echo "$METRICS" | grep "ocx_webhook_admission_requests_total" | awk '{sum+=$2} END {print sum+0}')
            INJECTION_REQUESTS=$(echo "$METRICS" | grep "ocx_webhook_injection_requests_total" | awk '{sum+=$2} END {print sum+0}')
            ERROR_COUNT=$(echo "$METRICS" | grep "ocx_webhook_errors_total" | awk '{sum+=$2} END {print sum+0}')
            
            log "Metrics at ${i}0s: Admission=$ADMISSION_REQUESTS, Injection=$INJECTION_REQUESTS, Errors=$ERROR_COUNT"
        fi
        
        # Check pod status
        READY_PODS=$(kubectl get pods -n "$TEST_NAMESPACE" -l test-type=load-test --field-selector=status.phase=Running --no-headers | wc -l)
        log "Ready pods: $READY_PODS/$CONCURRENT_PODS"
    done
    
    kill $METRICS_PF_PID || true
    
    # Final statistics
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    READY_PODS=$(kubectl get pods -n "$TEST_NAMESPACE" -l test-type=load-test --field-selector=status.phase=Running --no-headers | wc -l)
    FAILED_PODS=$(kubectl get pods -n "$TEST_NAMESPACE" -l test-type=load-test --field-selector=status.phase=Failed --no-headers | wc -l)
    PENDING_PODS=$(kubectl get pods -n "$TEST_NAMESPACE" -l test-type=load-test --field-selector=status.phase=Pending --no-headers | wc -l)
    
    SUCCESS_RATE=$((READY_PODS * 100 / CONCURRENT_PODS))
    
    echo ""
    echo "================================================================================"
    echo "Load Test Results"
    echo "================================================================================"
    echo "Duration: ${DURATION}s"
    echo "Total Pods: $CONCURRENT_PODS"
    echo "Ready Pods: $READY_PODS"
    echo "Failed Pods: $FAILED_PODS"
    echo "Pending Pods: $PENDING_PODS"
    echo "Success Rate: ${SUCCESS_RATE}%"
    echo ""
    
    if [ $SUCCESS_RATE -ge 95 ]; then
        success "Load test PASSED: ${SUCCESS_RATE}% success rate"
    elif [ $SUCCESS_RATE -ge 80 ]; then
        echo -e "${YELLOW}Load test MARGINAL: ${SUCCESS_RATE}% success rate${NC}"
    else
        error "Load test FAILED: ${SUCCESS_RATE}% success rate"
    fi
    
    # Check webhook health
    WEBHOOK_PODS=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$WEBHOOK_NAME" --field-selector=status.phase=Running --no-headers | wc -l)
    if [ $WEBHOOK_PODS -gt 0 ]; then
        success "Webhook remained healthy during load test"
    else
        error "Webhook became unhealthy during load test"
    fi
}

main "$@"
