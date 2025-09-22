#!/bin/bash
set -euo pipefail

NAMESPACE="default"
POD_NAME="ocx-test-pod"

echo "Testing OCX injection..."

# Create test pod with OCX annotation
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: $POD_NAME
  namespace: $NAMESPACE
  annotations:
    ocx-inject: "true"
    ocx-cycles: "50000"
    ocx-profile: "v1-min"
spec:
  containers:
  - name: test-app
    image: nginx:alpine
    command: ["sleep", "3600"]
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
EOF

echo "Waiting for pod to be created..."
kubectl wait --for=condition=Ready pod/$POD_NAME -n $NAMESPACE --timeout=60s

echo "Checking if OCX was injected..."
kubectl describe pod $POD_NAME -n $NAMESPACE

echo "Checking init containers..."
kubectl get pod $POD_NAME -n $NAMESPACE -o jsonpath='{.spec.initContainers[*].name}'

echo ""
echo "Checking OCX binary in pod..."
kubectl exec $POD_NAME -n $NAMESPACE -- ls -la /usr/local/bin/ocx || echo "OCX binary not found"

echo ""
echo "Checking OCX environment variables..."
kubectl exec $POD_NAME -n $NAMESPACE -- env | grep OCX || echo "OCX env vars not found"

echo ""
echo "Cleaning up test pod..."
kubectl delete pod $POD_NAME -n $NAMESPACE

echo "Test complete!"
