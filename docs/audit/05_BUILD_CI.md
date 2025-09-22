# OCX Protocol Build & CI System

## Build System Overview

The OCX Protocol uses a multi-language build system with a central Makefile orchestrating builds across Go, Rust, C++, Node.js, and Java components.

## Language-Specific Build Systems

### Go Build System
**Tool**: `go build` with Go modules
**Configuration**: `go.mod` (Go 1.21)
**Build Command**: `go build -o bin/ocx-server cmd/ocx-server/main.go`
**Dependencies**: Managed via `go mod tidy`
**Output**: Native binaries in `bin/` directory

**Key Targets**:
- `build-go`: Build all Go binaries
- `test-go`: Run Go tests with race detection
- `clean-go`: Remove Go build artifacts

### Rust Build System
**Tool**: `cargo` with Cargo.toml
**Configuration**: `libocx-verify/Cargo.toml`
**Build Command**: `cargo build --release`
**Dependencies**: Managed via `cargo update`
**Output**: Static/dynamic libraries in `libocx-verify/target/`

**Key Targets**:
- `build-rust`: Build Rust library
- `test-rust`: Run Rust tests
- `clean-rust`: Remove Rust build artifacts

### C++ Build System
**Tool**: CMake with Makefile wrapper
**Configuration**: `adapters/ad3-envoy/CMakeLists.txt`
**Build Command**: `cmake . && make`
**Dependencies**: Envoy headers, Abseil, Protobuf
**Output**: Shared library in `adapters/ad3-envoy/build/`

**Key Targets**: 
- `build-envoy`: Build Envoy filter
- `test-envoy`: Run C++ tests
- `clean-envoy`: Remove C++ build artifacts

### Node.js Build System
**Tool**: npm with webpack
**Configuration**: `adapters/ad4-github/package.json`
**Build Command**: `npm run build`
**Dependencies**: Managed via `npm ci`
**Output**: Bundled action in `adapters/ad4-github/dist/`

**Key Targets**:
- `build-github`: Build GitHub Action
- `test-github`: Run Node.js tests
- `clean-github`: Remove Node.js build artifacts

### Java Build System
**Tool**: Maven with pom.xml
**Configuration**: `adapters/ad6-kafka/pom.xml`
**Build Command**: `mvn package`
**Dependencies**: Managed via Maven
**Output**: JAR files in `adapters/ad6-kafka/target/`

**Key Targets**:
- `build-kafka`: Build Kafka interceptors
- `test-kafka`: Run Java tests
- `clean-kafka`: Remove Java build artifacts

## Docker Multi-Stage Build

### Dockerfile.complete
**Purpose**: Multi-stage Docker build for entire ecosystem
**Stages**:
1. **Rust Stage**: Build Rust verifier library
2. **Go Stage**: Build Go binaries
3. **C++ Stage**: Build Envoy filter
4. **Java Stage**: Build Kafka interceptors
5. **Node.js Stage**: Build GitHub Action
6. **Final Stage**: Combine all artifacts

**Key Features**:
- Multi-architecture support (amd64, arm64)
- Dependency caching for faster builds
- Optimized final image size
- Security scanning integration

### Docker Compose
**File**: `deployment/docker-compose.yml`
**Services**:
- `ocx-api`: Main API server
- `ocx-webhook`: Kubernetes webhook
- `ocx-verifier`: Standalone verifier
- `postgres`: Database
- `redis`: Caching layer

## CI/CD Pipeline

### GitHub Actions Workflow
**File**: `.github/workflows/build.yml`
**Triggers**: Push to main, pull requests
**Matrix Strategy**:
- **OS**: ubuntu-latest, macos-latest, windows-latest
- **Go Version**: 1.21, 1.22
- **Rust Version**: stable, beta
- **Node Version**: 18, 20

**Pipeline Stages**:
1. **Checkout**: Check out code
2. **Setup**: Install language runtimes
3. **Dependencies**: Install dependencies
4. **Lint**: Run linters (golangci-lint, clippy, eslint)
5. **Test**: Run test suites
6. **Build**: Build all components
7. **Security**: Run security scans
8. **Package**: Create release artifacts
9. **Deploy**: Deploy to staging (on main branch)

### Linting & Code Quality
**Go**: `golangci-lint` with custom configuration
**Rust**: `cargo clippy` with strict warnings
**Node.js**: `eslint` with React configuration
**Java**: `spotbugs` and `checkstyle`
**C++**: `clang-tidy` and `cppcheck`

### Security Scanning
**Tools**:
- `gosec` for Go security issues
- `cargo audit` for Rust vulnerabilities
- `npm audit` for Node.js dependencies
- `trivy` for container scanning
- `snyk` for dependency vulnerabilities

## Build Commands Reference

### Main Build Targets
```bash
# Build everything
make build-all

# Build specific components
make build-rust
make build-go
make build-envoy
make build-github
make build-terraform
make build-kafka

# Test everything
make test-all

# Clean everything
make clean-all

# Deploy everything
make deploy-all
```

### Development Commands
```bash
# Install system dependencies
make install-system-deps

# Start development environment
make start-dev-env

# Stop development environment
make stop-dev-env

# Health check
make health-check

# View logs
make logs

# Performance monitoring
make monitor-performance
```

### Testing Commands
```bash
# Run all tests
make test-all

# Run specific test suites
make test-rust
make test-go
make test-envoy
make test-github
make test-terraform
make test-kafka

# Run integration tests
make integration-test

# Run benchmarks
make benchmark
```

### Deployment Commands
```bash
# Deploy to local environment
make deploy-local

# Deploy to staging
make deploy-staging

# Deploy to production
make deploy-prod

# Security scan
make security-scan
```

## Build Artifacts

### Go Binaries
**Location**: `bin/`
**Files**:
- `ocx-server`: Main API server
- `ocx-verifier`: Standalone verifier
- `ocx-webhook`: Kubernetes webhook
- `ocxctl`: Administrative CLI

### Rust Library
**Location**: `libocx-verify/target/release/`
**Files**:
- `liblibocx_verify.a`: Static library
- `liblibocx_verify.so`: Dynamic library (Linux)
- `liblibocx_verify.dylib`: Dynamic library (macOS)
- `liblibocx_verify.dll`: Dynamic library (Windows)

### C++ Filter
**Location**: `adapters/ad3-envoy/build/`
**Files**:
- `libocx_filter.so`: Envoy filter shared library

### Node.js Action
**Location**: `adapters/ad4-github/dist/`
**Files**:
- `index.js`: Bundled GitHub Action

### Java JARs
**Location**: `adapters/ad6-kafka/target/`
**Files**:
- `ocx-kafka-1.0.0.jar`: Kafka interceptor JAR

### Docker Images
**Registry**: `ghcr.io/ocx-protocol/ocx`
**Tags**:
- `latest`: Latest development build
- `v1.0.0-rc.1`: Release candidate
- `amd64`: AMD64 architecture
- `arm64`: ARM64 architecture

## Build System Issues

### Current Problems
1. **Complex Makefile**: 400+ lines, hard to maintain
2. **Missing Dependencies**: Some components have incomplete dependency management
3. **Cross-Platform Issues**: Limited Windows support
4. **Build Order**: No explicit dependency ordering
5. **Error Handling**: Limited error reporting and recovery

### Recommended Fixes
1. **Modular Makefiles**: Split into component-specific Makefiles
2. **Dependency Management**: Add explicit dependency tracking
3. **Cross-Platform**: Add Windows-specific build targets
4. **Build Order**: Implement proper dependency ordering
5. **Error Handling**: Add comprehensive error reporting

## Performance Considerations

### Build Performance
- **Parallel Builds**: Use `make -j` for parallel compilation
- **Dependency Caching**: Cache dependencies between builds
- **Incremental Builds**: Only rebuild changed components
- **Docker Layer Caching**: Optimize Docker layer caching

### Runtime Performance
- **Go**: Use `-ldflags="-s -w"` for smaller binaries
- **Rust**: Use `--release` for optimized builds
- **C++**: Use `-O3` for maximum optimization
- **Node.js**: Use `--production` for minimal dependencies

## Monitoring & Observability

### Build Metrics
- **Build Time**: Track build duration per component
- **Success Rate**: Monitor build success/failure rates
- **Artifact Size**: Track binary and library sizes
- **Dependency Count**: Monitor dependency growth

### Runtime Metrics
- **Startup Time**: Track component startup time
- **Memory Usage**: Monitor memory consumption
- **CPU Usage**: Track CPU utilization
- **Error Rates**: Monitor runtime errors

## Troubleshooting

### Common Build Issues
1. **Go Module Issues**: Run `go mod tidy` and `go mod download`
2. **Rust Compilation**: Run `cargo clean` and `cargo build`
3. **C++ Dependencies**: Install Envoy headers and Abseil
4. **Node.js Issues**: Run `npm ci` and `npm run build`
5. **Java Dependencies**: Run `mvn clean install`

### Debug Commands
```bash
# Verbose build output
make build-all VERBOSE=true

# Debug specific component
make build-rust DEBUG=true

# Check dependencies
make check-deps

# Validate configuration
make validate-config
```
