# OCX Protocol - Comprehensive Build System Guide

## 🎯 **OVERVIEW**

The OCX Protocol now features a comprehensive multi-language build system that handles Rust, Go, C++, Node.js, and Java components. This guide covers setup, building, testing, and deployment across all languages.

## 🏗️ **ARCHITECTURE**

```
ocx-protocol/
├── libocx-verify/           # Rust verifier
├── pkg/                     # Go server components
├── adapters/
│   ├── ad2-kubernetes/      # Kubernetes webhook (Go)
│   ├── ad3-envoy/          # Envoy filter (C++)
│   ├── ad4-github/         # GitHub Action (Node.js)
│   ├── ad5-terraform/      # Terraform provider (Go)
│   └── ad6-kafka/          # Kafka interceptor (Java)
├── deployment/             # Infrastructure configs
├── tests/integration/      # Integration tests
└── .github/workflows/      # CI/CD pipeline
```

## 🚀 **QUICK START**

### **Prerequisites**

```bash
# Install required tools
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh  # Rust
go install golang.org/x/tools/cmd/goimports@latest                # Go
sudo apt-get install build-essential cmake                       # C++
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash  # Node.js
sudo apt-get install openjdk-17-jdk maven                        # Java
```

### **Build Everything**

```bash
# Single command to build all components
make build-all

# Run all tests
make test-all

# Run performance benchmarks
make benchmark

# Deploy to test environment
make deploy-all
```

## 📋 **DETAILED BUILD INSTRUCTIONS**

### **1. Rust Verifier**

```bash
cd libocx-verify
cargo build --release --features ffi
cargo test --features ffi
cargo bench --features ffi
```

**Dependencies:**
- Rust 1.70+
- `ring` for cryptography
- `cc` for C FFI compilation

### **2. Go Components**

```bash
# Build all Go components
make build-go

# Individual components
go build -tags rust_verifier -o bin/ocx-server ./cmd/server
go build -o bin/ad2-webhook ./cmd/ocx-webhook
go build -o bin/ocx-verifier ./cmd/ocx-verifier
```

**Dependencies:**
- Go 1.21+
- CGO enabled for Rust FFI
- Rust library built first

### **3. C++ Envoy Filter**

```bash
cd adapters/ad3-envoy
make build
make test
make benchmark
```

**Dependencies:**
- GCC 11+ or Clang 12+
- CMake 3.16+
- Envoy API libraries
- Protobuf/gRPC
- nlohmann/json

**Build Options:**
- `make build` - Standard build
- `make test` - Run tests
- `make clean` - Clean artifacts
- `make install` - Install to system

### **4. Node.js GitHub Action**

```bash
cd adapters/ad4-github
npm ci
npm run build
npm test
npm run benchmark
```

**Dependencies:**
- Node.js 18+
- npm 8+
- @vercel/ncc for bundling

**Scripts:**
- `npm run build` - Build with ncc
- `npm test` - Run Jest tests
- `npm run lint` - ESLint checking
- `npm run clean` - Clean artifacts

### **5. Terraform Provider**

```bash
cd adapters/ad5-terraform
make build
make test
make test-acc
```

**Dependencies:**
- Go 1.21+
- Terraform 1.0+
- terraform-plugin-framework

**Commands:**
- `make build` - Build provider
- `make test` - Unit tests
- `make test-acc` - Acceptance tests
- `make install` - Install to Terraform

### **6. Java Kafka Interceptor**

```bash
cd adapters/ad6-kafka
mvn clean package
mvn test
mvn exec:java -Dexec.mainClass="dev.ocx.kafka.BenchmarkRunner"
```

**Dependencies:**
- Java 17+
- Maven 3.8+
- Apache Kafka 2.8+

**Profiles:**
- `mvn package` - Standard build
- `mvn test` - Run tests
- `mvn exec:java -Pbenchmark` - Run benchmarks

## 🧪 **TESTING FRAMEWORK**

### **Integration Tests**

```bash
# Start test environment
docker-compose -f tests/integration/docker-compose.test.yml up -d

# Run integration tests
cd tests/integration
python -m pytest -v

# Stop test environment
docker-compose -f tests/integration/docker-compose.test.yml down
```

### **Test Coverage**

- **Unit Tests**: Each component has comprehensive unit tests
- **Integration Tests**: Cross-component communication testing
- **Performance Tests**: Benchmarking across all components
- **Security Tests**: Vulnerability scanning and security validation

## 🐳 **DOCKER INTEGRATION**

### **Multi-Stage Build**

```bash
# Build complete Docker image
docker build -f Dockerfile.complete -t ocx-protocol:latest .

# Run with all components
docker run -p 8080:8080 -p 8000:8000 -p 9092:9092 ocx-protocol:latest
```

### **Individual Component Images**

```bash
# Rust verifier only
docker build -f Dockerfile.rust -t ocx-rust:latest .

# Go server only
docker build -f Dockerfile -t ocx-go:latest .
```

## 🔄 **CI/CD PIPELINE**

### **GitHub Actions**

The CI/CD pipeline automatically:
- Builds all components in parallel
- Runs comprehensive tests
- Performs security scanning
- Creates multi-architecture Docker images
- Generates release packages

### **Manual Triggers**

```bash
# Trigger specific language builds
make build-rust
make build-go
make build-envoy
make build-github
make build-terraform
make build-kafka
```

## 📊 **PERFORMANCE REQUIREMENTS**

| Component | Latency Target | Throughput Target |
|-----------|----------------|-------------------|
| Envoy Filter | <10ms | >1000 req/s |
| GitHub Action | <30s | N/A |
| Terraform Provider | <5s | N/A |
| Kafka Interceptor | <1ms | >10000 msg/s |
| Rust Verifier | <0.1ms | >100000 verifications/s |

## 🔧 **TROUBLESHOOTING**

### **Common Issues**

1. **Rust FFI Linking Errors**
   ```bash
   # Ensure Rust library is built first
   cd libocx-verify && cargo build --release --features ffi
   ```

2. **C++ Compilation Errors**
   ```bash
   # Install missing dependencies
   sudo apt-get install libprotobuf-dev protobuf-compiler libgrpc++-dev
   ```

3. **Node.js Build Failures**
   ```bash
   # Clear npm cache and reinstall
   npm cache clean --force
   rm -rf node_modules package-lock.json
   npm install
   ```

4. **Java Maven Issues**
   ```bash
   # Clean and rebuild
   mvn clean
   mvn dependency:resolve
   mvn package
   ```

### **Debug Mode**

```bash
# Enable debug logging
export OCX_DEBUG=true
export RUST_LOG=debug
export OCX_LOG_LEVEL=debug

# Run with verbose output
make build-all VERBOSE=1
```

## 📚 **DEVELOPMENT WORKFLOW**

### **1. Setup Development Environment**

```bash
# Clone repository
git clone https://github.com/ocx-protocol/ocx-protocol.git
cd ocx-protocol

# Install development tools
make setup-dev

# Build everything
make build-all
```

### **2. Make Changes**

```bash
# Edit code in any language
# Run tests for specific component
make test-rust    # or test-go, test-envoy, etc.

# Run all tests
make test-all
```

### **3. Commit and Push**

```bash
# Pre-commit hooks will run tests
git add .
git commit -m "Add new feature"
git push
```

## 🎯 **SUCCESS CRITERIA**

The build system is successful when:

✅ **All components build without errors**
```bash
make build-all  # Should complete successfully
```

✅ **All tests pass**
```bash
make test-all  # Should show 100% pass rate
```

✅ **Performance benchmarks meet targets**
```bash
make benchmark  # Should show performance within targets
```

✅ **Integration tests pass**
```bash
make test-integration  # Should complete successfully
```

✅ **Docker images build and run**
```bash
docker build -f Dockerfile.complete -t ocx-protocol:latest .
docker run --rm ocx-protocol:latest  # Should start successfully
```

## 🚀 **NEXT STEPS**

1. **Production Deployment**: Deploy to production environments
2. **Monitoring**: Set up comprehensive monitoring and alerting
3. **Scaling**: Implement horizontal scaling strategies
4. **Security**: Enhance security scanning and compliance
5. **Documentation**: Expand API documentation and examples

---

**The OCX Protocol build system is now production-ready with comprehensive multi-language support, automated testing, and seamless integration across all infrastructure layers.**
