#!/bin/bash
set -e

echo "=== OCX Deployment Script ==="

# Build API server
echo "Building API server..."
cd cmd/api-server
go mod init api-server 2>/dev/null || true
go build -o api-server .
cd ../..

# Install Node.js dependencies and build React app
echo "Building React frontend..."
if command -v npm &> /dev/null; then
    npm install
    npm run build
else
    echo "Node.js/npm not found. Please install Node.js to build the frontend."
    exit 1
fi

echo "=== Deployment complete ==="
echo "To run locally:"
echo "1. Start API server: ./cmd/api-server/api-server"
echo "2. Start frontend: npm start (dev) or npm run serve (production)"
