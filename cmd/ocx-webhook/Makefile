# =============================================================================
# OCX Kubernetes Webhook - Production Build System
# =============================================================================

.PHONY: build test docker-build docker-push deploy clean help

# Variables
IMAGE_NAME ?= ocx-webhook
IMAGE_TAG ?= latest
REGISTRY ?= your-registry.com
NAMESPACE ?= ocx-system

help: ## Show this help message
	@echo "OCX Kubernetes Webhook - Build Commands"
	@echo "======================================="
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the webhook binary
	@echo "Building OCX webhook..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -extldflags "-static"' \
		-tags netgo \
		-installsuffix netgo \
		-a \
		-o bin/ocx-webhook \
		./cmd/webhook/
	@echo "Build complete: bin/ocx-webhook"

test: ## Run unit tests
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Unit tests complete. Coverage report: coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	./tests/fortune500-test-suite.sh
	@echo "Integration tests complete"

test-load: ## Run load tests
	@echo "Running load tests..."
	./tests/load-test.sh 100 300
	@echo "Load tests complete"

test-security: ## Run security tests
	@echo "Running security tests..."
	./tests/security-test.sh
	@echo "Security tests complete"

test-all: test test-integration test-load test-security ## Run all tests
	@echo "All tests completed successfully"

test-fortune500: ## Run Fortune 500 grade test suite
	@echo "Running Fortune 500 grade test suite..."
	./tests/fortune500-test-suite.sh
	@echo "Fortune 500 test suite complete"

docker-build: ## Build Docker image
	@echo "Building Docker image: $(IMAGE_NAME):$(IMAGE_TAG)"
	docker build \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		-t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
		-f Dockerfile .
	@echo "Docker build complete"

docker-push: docker-build ## Push Docker image to registry
	@echo "Pushing image to registry..."
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	@echo "Push complete"

deploy: ## Deploy to Kubernetes
	@echo "Deploying OCX webhook to Kubernetes..."
	kubectl apply -f k8s/
	kubectl wait --for=condition=available --timeout=300s deployment/ocx-webhook -n $(NAMESPACE)
	@echo "Deployment complete"

undeploy: ## Remove from Kubernetes
	@echo "Removing OCX webhook from Kubernetes..."
	kubectl delete -f k8s/ --ignore-not-found=true
	@echo "Undeployment complete"

cert-gen: ## Generate self-signed certificates for development
	@echo "Generating self-signed certificates..."
	./scripts/generate-certs.sh
	@echo "Certificates generated in ./certs/"

clean: ## Clean build artifacts
	@echo "Cleaning up..."
	rm -rf bin/ coverage.out coverage.html
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) 2>/dev/null || true
	@echo "Cleanup complete"

install-deps: ## Install development dependencies
	@echo "Installing dependencies..."
	go mod tidy
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Dependencies installed"

lint: ## Run linting
	@echo "Running linters..."
	golangci-lint run ./...
	@echo "Linting complete"

security-scan: ## Run security scan
	@echo "Running security scan..."
	docker run --rm -v "$(PWD):/app" securecodewarrior/docker-image-security-check:latest /app
	@echo "Security scan complete"

# Default target
.DEFAULT_GOAL := help
