# Terraform Provider Build System
.PHONY: build test clean install benchmark

# Build configuration
BINARY_NAME = terraform-provider-ocx
VERSION = 1.0.0
BUILD_DIR = bin
GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME):
	@echo "Building Terraform provider..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

test:
	@echo "Running Terraform provider tests..."
	go test -v ./...

test-acc:
	@echo "Running Terraform acceptance tests..."
	TF_ACC=1 go test -v ./...

clean:
	@echo "Cleaning Terraform provider artifacts..."
	rm -rf $(BUILD_DIR)/
	go clean -cache

install: build
	@echo "Installing Terraform provider..."
	@mkdir -p ~/.terraform.d/plugins/registry.terraform.io/ocx-protocol/ocx/$(VERSION)/$(GOOS)_$(GOARCH)/
	cp $(BUILD_DIR)/$(BINARY_NAME) ~/.terraform.d/plugins/registry.terraform.io/ocx-protocol/ocx/$(VERSION)/$(GOOS)_$(GOARCH)/

benchmark:
	@echo "Running Terraform provider benchmarks..."
	go test -bench=. ./...

lint:
	@echo "Running linter..."
	golangci-lint run

fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

mod-tidy:
	@echo "Tidying modules..."
	go mod tidy

mod-download:
	@echo "Downloading modules..."
	go mod download

# Development tools
tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

docs:
	@echo "Generating documentation..."
	tfplugindocs generate

# Release
release: build test
	@echo "Creating release package..."
	@mkdir -p release
	tar -czf release/terraform-provider-ocx-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Release package created: release/terraform-provider-ocx-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz"
