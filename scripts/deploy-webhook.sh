#!/bin/bash

set -euo pipefail

# OCX Webhook Deployment Script
# This script deploys the OCX Kubernetes mutating admission webhook

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="ocx-system"
WEBHOOK_DIR="k8s/webhook"
IMAGE_NAME="ocx-protocol/webhook"
IMAGE_TAG="latest"
REGISTRY="docker.io"

# Functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if kubectl can connect to cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check if openssl is installed (for certificate generation)
    if ! command -v openssl &> /dev/null; then
        log_error "openssl is not installed or not in PATH"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

build_webhook() {
    log_info "Building OCX webhook..."
    
    cd cmd/ocx-webhook
    
    # Build the binary
    if ! make build; then
        log_error "Failed to build webhook binary"
        exit 1
    fi
    
    # Build Docker image
    if ! make docker-build; then
        log_error "Failed to build Docker image"
        exit 1
    fi
    
    cd ../..
    log_success "Webhook built successfully"
}

deploy_webhook() {
    log_info "Deploying OCX webhook to Kubernetes..."
    
    cd $WEBHOOK_DIR
    
    # Create namespace
    log_info "Creating namespace..."
    kubectl apply -f namespace.yaml
    
    # Deploy RBAC
    log_info "Deploying RBAC..."
    kubectl apply -f rbac.yaml
    
    # Generate certificates
    log_info "Generating TLS certificates..."
    if ! ./generate-certs.sh; then
        log_error "Failed to generate certificates"
        exit 1
    fi
    
    # Deploy certificates
    log_info "Deploying certificates..."
    kubectl apply -f certs/secret.yaml
    
    # Deploy webhook
    log_info "Deploying webhook deployment..."
    kubectl apply -f deployment.yaml
    
    # Deploy service
    log_info "Deploying webhook service..."
    kubectl apply -f service.yaml
    
    # Deploy webhook configuration
    log_info "Deploying webhook configuration..."
    kubectl apply -f webhook-config-with-ca.yaml
    
    cd ../..
    log_success "Webhook deployed successfully"
}

wait_for_webhook() {
    log_info "Waiting for webhook to be ready..."
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=available --timeout=300s deployment/ocx-webhook -n $NAMESPACE
    
    # Wait for pods to be ready
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=webhook -n $NAMESPACE --timeout=300s
    
    log_success "Webhook is ready"
}

test_webhook() {
    log_info "Testing webhook functionality..."
    
    # Enable webhook for default namespace
    kubectl label namespace default ocx-inject=enabled --overwrite
    
    # Create a test pod
    log_info "Creating test pod..."
    kubectl run ocx-test-pod --image=nginx --restart=Never --dry-run=client -o yaml | kubectl apply -f -
    
    # Wait for pod to be ready
    kubectl wait --for=condition=ready pod/ocx-test-pod --timeout=60s
    
    # Check if OCX was injected
    log_info "Checking if OCX was injected..."
    if kubectl exec ocx-test-pod -- ls /usr/local/bin/ocx &> /dev/null; then
        log_success "OCX binary found in pod - injection successful!"
    else
        log_warning "OCX binary not found in pod - injection may have failed"
    fi
    
    # Check environment variables
    log_info "Checking OCX environment variables..."
    kubectl exec ocx-test-pod -- env | grep OCX || log_warning "No OCX environment variables found"
    
    # Clean up test pod
    kubectl delete pod ocx-test-pod
    
    log_success "Webhook test completed"
}

show_status() {
    log_info "OCX Webhook Status:"
    echo ""
    
    echo "📊 Deployment:"
    kubectl get deployment -n $NAMESPACE ocx-webhook 2>/dev/null || echo "  Not found"
    echo ""
    
    echo "🔌 Service:"
    kubectl get service -n $NAMESPACE ocx-webhook 2>/dev/null || echo "  Not found"
    echo ""
    
    echo "⚙️  Webhook Configuration:"
    kubectl get mutatingwebhookconfiguration ocx-webhook 2>/dev/null || echo "  Not found"
    echo ""
    
    echo "🪟 Pods:"
    kubectl get pods -n $NAMESPACE -l app.kubernetes.io/component=webhook 2>/dev/null || echo "  No pods found"
    echo ""
    
    echo "📋 Logs (last 10 lines):"
    kubectl logs -n $NAMESPACE -l app.kubernetes.io/component=webhook --tail=10 2>/dev/null || echo "  No logs found"
}

cleanup() {
    log_info "Cleaning up OCX webhook..."
    
    cd $WEBHOOK_DIR
    
    # Remove webhook configuration
    kubectl delete -f webhook-config-with-ca.yaml 2>/dev/null || true
    
    # Remove service
    kubectl delete -f service.yaml 2>/dev/null || true
    
    # Remove deployment
    kubectl delete -f deployment.yaml 2>/dev/null || true
    
    # Remove certificates
    kubectl delete -f certs/secret.yaml 2>/dev/null || true
    
    # Remove RBAC
    kubectl delete -f rbac.yaml 2>/dev/null || true
    
    # Remove namespace
    kubectl delete -f namespace.yaml 2>/dev/null || true
    
    cd ../..
    
    log_success "Cleanup completed"
}

show_help() {
    echo "OCX Webhook Deployment Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  deploy     Deploy the webhook (default)"
    echo "  build      Build the webhook binary and Docker image"
    echo "  test       Test the webhook functionality"
    echo "  status     Show webhook status"
    echo "  cleanup    Remove the webhook from Kubernetes"
    echo "  help       Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 deploy          # Deploy webhook"
    echo "  $0 test            # Test webhook"
    echo "  $0 status          # Check status"
    echo "  $0 cleanup         # Remove webhook"
}

# Main script
main() {
    local command=${1:-deploy}
    
    case $command in
        deploy)
            check_prerequisites
            build_webhook
            deploy_webhook
            wait_for_webhook
            test_webhook
            show_status
            ;;
        build)
            check_prerequisites
            build_webhook
            ;;
        test)
            test_webhook
            ;;
        status)
            show_status
            ;;
        cleanup)
            cleanup
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
