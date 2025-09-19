# OCX Protocol Makefile
.PHONY: all build test test-race test-coverage lint security-check safety-check clean install-tools

# Variables
COVERAGE_THRESHOLD = 80
RACE_DETECTION = true
BENCHMARK = false
VERBOSE = false
TIMEOUT = 30s

# Default target
all: clean install-tools lint security-check safety-check test

# Build the project
build:
	@echo "🔨 Building OCX Protocol..."
	go build -v ./...

# Build killer demo
build-demo:
	@echo "🚀 Building OCX Killer Demo..."
	go build -o ocx-killer-demo ./cmd/ocx-killer-demo/

# Run killer demo
demo: build-demo
	@echo "🎮 Running OCX Killer Applications Demo..."
	./ocx-killer-demo

# Build simple demo
build-simple-demo:
	@echo "🚀 Building OCX Simple Demo..."
	go build -o ocx-simple-demo ./cmd/ocx-simple-demo/

# Run simple demo
simple-demo: build-simple-demo
	@echo "🎮 Running OCX Simple Demo..."
	./ocx-simple-demo

# Build enhanced CLI
build-cli:
	@echo "🔧 Building OCX Enhanced CLI..."
	go build -o ocx ./cmd/ocx/

# Run conformance tests
conformance: build-cli
	@echo "🧪 Running OCX Conformance Tests..."
	./ocx conformance

# Run benchmarks
benchmark: build-cli
	@echo "⚡ Running OCX Benchmarks..."
	./ocx benchmark

# Generate test vectors (requires safety flag)
gen-vectors: build-cli
	@echo "🔧 Generating Conformance Test Vectors..."
	ALLOW_VECTOR_REGEN=1 ./ocx gen-vectors

# Verify receipts
verify: build-cli
	@echo "🔍 Verifying Receipts..."
	./ocx verify $(ARGS)

# Install development tools
install-tools:
	@echo "📦 Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v -timeout=$(TIMEOUT) ./...

# Run tests with race detection
test-race:
	@echo "🏃 Running tests with race detection..."
	go test -v -race -timeout=$(TIMEOUT) ./...

# Run tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -v -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
benchmark:
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./...

# Run integration tests
test-integration:
	@echo "🔗 Running integration tests..."
	go test -v -tags=integration ./...

# Run all tests with coverage and race detection
test-all: test-race test-coverage benchmark test-integration

# Lint the code
lint:
	@echo "🔍 Running linters..."
	golangci-lint run --timeout=5m
	staticcheck ./...

# Security check
security-check:
	@echo "🔒 Running security checks..."
	gosec ./...

# Safety check
safety-check:
	@echo "🛡️ Running safety checks..."
	./ocx-safety-check

# Check code coverage threshold
check-coverage:
	@echo "📈 Checking coverage threshold..."
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$coverage < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $$coverage% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	else \
		echo "✅ Coverage $$coverage% meets threshold $(COVERAGE_THRESHOLD)%"; \
	fi

# Generate test report
test-report: test-coverage
	@echo "📋 Generating test report..."
	@echo "# OCX Protocol Test Report" > test-report.md
	@echo "Generated: $$(date)" >> test-report.md
	@echo "" >> test-report.md
	@echo "## Coverage Summary" >> test-report.md
	@go tool cover -func=coverage.out | grep total >> test-report.md
	@echo "" >> test-report.md
	@echo "## Test Results" >> test-report.md
	@go test -v ./... 2>&1 | grep -E "(PASS|FAIL|SKIP)" >> test-report.md

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	go clean
	rm -f coverage.out coverage.html
	rm -f test-report.md
	rm -rf tests/coverage/

# Format code
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...
	goimports -w .

# Vet the code
vet:
	@echo "🔍 Vetting code..."
	go vet ./...

# Run all checks
check: fmt vet lint security-check safety-check test-race test-coverage check-coverage

# CI/CD pipeline
ci: clean install-tools check

# Development setup
dev-setup: install-tools
	@echo "🚀 Setting up development environment..."
	@echo "Installing pre-commit hooks..."
	@echo "#!/bin/bash" > .git/hooks/pre-commit
	@echo "make check" >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Development environment ready!"

# Help
help:
	@echo "Available targets:"
	@echo "  all           - Run all checks and tests"
	@echo "  build         - Build the project"
	@echo "  build-demo    - Build killer applications demo"
	@echo "  demo          - Run killer applications demo"
	@echo "  build-simple-demo - Build simple demo"
	@echo "  simple-demo   - Run simple demo"
	@echo "  build-cli     - Build enhanced CLI"
	@echo "  conformance   - Run conformance tests"
	@echo "  benchmark     - Run performance benchmarks"
	@echo "  gen-vectors   - Generate test vectors"
	@echo "  verify        - Verify receipts"
	@echo "  test          - Run tests"
	@echo "  test-race     - Run tests with race detection"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  benchmark     - Run benchmarks"
	@echo "  lint          - Run linters"
	@echo "  security-check - Run security checks"
	@echo "  safety-check  - Run safety checks"
	@echo "  check         - Run all checks"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  ci            - CI/CD pipeline"
	@echo "  dev-setup     - Setup development environment"
	@echo "  help          - Show this help"
