#!/bin/bash

set -euo pipefail

# OCX Webhook Production Deployment Script
# This script deploys the production-ready OCX Kubernetes webhook

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="ocx-system"
WEBHOOK_DIR="."
IMAGE_NAME="ocx-webhook"
IMAGE_TAG="latest"
REGISTRY="docker.io"
DEPLOYMENT_MODE="manual"  # manual or cert-manager

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
    
    # Check if openssl is installed (for manual certificate generation)
    if ! command -v openssl &> /dev/null; then
        log_error "openssl is not installed or not in PATH"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

check_cert_manager() {
    log_info "Checking cert-manager installation..."
    
    if kubectl get crd certificates.cert-manager.io &> /dev/null; then
        log_success "cert-manager is installed"
        return 0
    else
        log_warning "cert-manager is not installed"
        return 1
    fi
}

deploy_manual() {
    log_info "Deploying with manual certificate generation..."
    
    # Create namespace
    kubectl apply -f namespace.yaml
    
    # Deploy RBAC
    kubectl apply -f rbac.yaml
    
    # Deploy ConfigMap
    kubectl apply -f configmap.yaml
    
    # Generate certificates manually
    log_info "Generating TLS certificates..."
    if [ ! -f certs/tls.crt ]; then
        mkdir -p certs
        openssl genrsa -out certs/tls.key 2048
        openssl req -new -key certs/tls.key -out certs/tls.csr -subj "/CN=ocx-webhook.ocx-system.svc"
        openssl x509 -req -in certs/tls.csr -signkey certs/tls.key -out certs/tls.crt -days 365
        rm certs/tls.csr
    fi
    
    # Create TLS secret
    kubectl create secret tls ocx-webhook-certs \
        --cert=certs/tls.crt \
        --key=certs/tls.key \
        --namespace=$NAMESPACE \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Deploy webhook components
    kubectl apply -f deployment.yaml
    kubectl apply -f service.yaml
    kubectl apply -f poddisruptionbudget.yaml
    kubectl apply -f networkpolicy.yaml
    
    # Deploy webhook configuration (without cert-manager annotation)
    kubectl apply -f webhook-config.yaml
    
    log_success "Manual deployment completed"
}

deploy_cert_manager() {
    log_info "Deploying with cert-manager..."
    
    # Create namespace
    kubectl apply -f namespace.yaml
    
    # Deploy RBAC
    kubectl apply -f rbac.yaml
    
    # Deploy ConfigMap
    kubectl apply -f configmap.yaml
    
    # Deploy cert-manager resources
    kubectl apply -f cert-manager.yaml
    
    # Wait for certificate to be ready
    log_info "Waiting for certificate to be ready..."
    kubectl wait --for=condition=Ready certificate/ocx-webhook-cert -n $NAMESPACE --timeout=300s
    
    # Deploy webhook components
    kubectl apply -f deployment.yaml
    kubectl apply -f service.yaml
    kubectl apply -f poddisruptionbudget.yaml
    kubectl apply -f networkpolicy.yaml
    kubectl apply -f servicemonitor.yaml
    
    # Deploy webhook configuration (with cert-manager annotation)
    kubectl apply -f webhook-config.yaml
    
    log_success "cert-manager deployment completed"
}

wait_for_webhook() {
    log_info "Waiting for webhook to be ready..."
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=available --timeout=300s deployment/ocx-webhook -n $NAMESPACE
    
    # Wait for pods to be ready
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=ocx-webhook -n $NAMESPACE --timeout=300s
    
    log_success "Webhook is ready"
}

test_webhook() {
    log_info "Testing webhook functionality..."
    
    # Create a test pod with OCX annotation
    log_info "Creating test pod with OCX annotation..."
    kubectl run ocx-test-pod --image=nginx --restart=Never \
        --overrides='{"metadata":{"annotations":{"ocx-inject":"true"}}}' \
        --dry-run=client -o yaml | kubectl apply -f -
    
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
    kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=ocx-webhook 2>/dev/null || echo "  No pods found"
    echo ""
    
    echo "📋 Logs (last 10 lines):"
    kubectl logs -n $NAMESPACE -l app.kubernetes.io/name=ocx-webhook --tail=10 2>/dev/null || echo "  No logs found"
    echo ""
    
    echo "🔐 Certificates:"
    kubectl get certificate -n $NAMESPACE 2>/dev/null || echo "  No certificates found"
    echo ""
    
    echo "📈 ServiceMonitor:"
    kubectl get servicemonitor -n $NAMESPACE 2>/dev/null || echo "  No ServiceMonitor found"
}

cleanup() {
    log_info "Cleaning up OCX webhook..."
    
    # Remove webhook configuration
    kubectl delete -f webhook-config.yaml 2>/dev/null || true
    
    # Remove webhook components
    kubectl delete -f servicemonitor.yaml 2>/dev/null || true
    kubectl delete -f networkpolicy.yaml 2>/dev/null || true
    kubectl delete -f poddisruptionbudget.yaml 2>/dev/null || true
    kubectl delete -f service.yaml 2>/dev/null || true
    kubectl delete -f deployment.yaml 2>/dev/null || true
    
    # Remove certificates
    kubectl delete -f cert-manager.yaml 2>/dev/null || true
    kubectl delete secret ocx-webhook-certs -n $NAMESPACE 2>/dev/null || true
    
    # Remove RBAC and namespace
    kubectl delete -f configmap.yaml 2>/dev/null || true
    kubectl delete -f rbac.yaml 2>/dev/null || true
    kubectl delete -f namespace.yaml 2>/dev/null || true
    
    # Clean up local certificates
    rm -rf certs/
    
    log_success "Cleanup completed"
}

show_help() {
    echo "OCX Webhook Production Deployment Script"
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  deploy [manual|cert-manager]  Deploy the webhook (default: manual)"
    echo "  test                          Test the webhook functionality"
    echo "  status                        Show webhook status"
    echo "  cleanup                       Remove the webhook from Kubernetes"
    echo "  help                          Show this help message"
    echo ""
    echo "Options:"
    echo "  manual        Use manual certificate generation (default)"
    echo "  cert-manager  Use cert-manager for automatic certificate management"
    echo ""
    echo "Examples:"
    echo "  $0 deploy                    # Deploy with manual certificates"
    echo "  $0 deploy cert-manager       # Deploy with cert-manager"
    echo "  $0 test                      # Test webhook"
    echo "  $0 status                    # Check status"
    echo "  $0 cleanup                   # Remove webhook"
}

# Main script
main() {
    local command=${1:-deploy}
    local mode=${2:-manual}
    
    case $command in
        deploy)
            check_prerequisites
            
            if [ "$mode" = "cert-manager" ]; then
                if check_cert_manager; then
                    deploy_cert_manager
                else
                    log_error "cert-manager is not installed. Please install cert-manager first or use manual mode."
                    exit 1
                fi
            else
                deploy_manual
            fi
            
            wait_for_webhook
            test_webhook
            show_status
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
