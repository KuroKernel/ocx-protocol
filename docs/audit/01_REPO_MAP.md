# OCX Protocol Repository Map

## Directory Structure

```
ocx-protocol/
├── 📁 adapters/                    # Integration adapters for different platforms
│   ├── 📁 ad3-envoy/              # C++ Envoy HTTP filter
│   │   ├── 📁 src/                # C++ source files
│   │   ├── 📁 build/              # CMake build artifacts
│   │   ├── CMakeLists.txt         # CMake configuration
│   │   └── envoy_filter.proto     # Protocol buffer definitions
│   ├── 📁 ad4-github/             # Node.js GitHub Action
│   │   ├── 📁 src/                # TypeScript source
│   │   ├── 📁 dist/               # Webpack build output
│   │   ├── package.json           # Node.js dependencies
│   │   └── webpack.config.js      # Bundling configuration
│   ├── 📁 ad5-terraform/          # Terraform provider
│   │   ├── 📁 internal/           # Go provider implementation
│   │   ├── go.mod                 # Go module definition
│   │   └── main_simple.go         # Provider entry point
│   └── 📁 ad6-kafka/              # Java Kafka interceptors
│       ├── 📁 src/main/java/      # Java source files
│       ├── pom.xml                # Maven configuration
│       └── 📁 target/             # Maven build artifacts
├── 📁 api/                        # OpenAPI specifications
├── 📁 archive/                    # Legacy/archived components
│   └── 📁 saas-components/        # Archived SaaS components
├── 📁 benchmarks/                 # Performance benchmarks
├── 📁 bin/                        # Built binaries
├── 📁 build/                      # Build artifacts
│   └── 📁 static/                 # Static web assets
├── 📁 cmd/                        # Go command-line applications
│   ├── 📁 api-server/             # Main API server
│   ├── 📁 ocx-server/             # OCX core server
│   ├── 📁 ocx-verifier/           # Standalone verifier CLI
│   ├── 📁 ocx-webhook/            # Kubernetes webhook
│   └── 📁 [20+ other commands]    # Various CLI tools
├── 📁 conformance/                # Test vectors and conformance testing
│   ├── 📁 receipts/               # Generated receipt test vectors
│   └── 📁 golden/                 # Golden test vectors
├── 📁 database/                   # Database schemas and migrations
│   ├── 📁 migrations/             # SQL migration files
│   └── 📁 schema/                 # Database schema definitions
├── 📁 deployment/                 # Deployment configurations
│   ├── 📁 kubernetes/             # K8s manifests
│   └── 📁 terraform/              # Infrastructure as code
├── 📁 docs/                       # Documentation
│   ├── 📁 audit/                  # This audit documentation
│   ├── 📁 spec/                   # Protocol specifications
│   └── 📁 webhook/                # Webhook documentation
├── 📁 examples/                   # Example applications
├── 📁 fixtures/                   # Test fixtures and sample data
├── 📁 go/                         # Go standard library (embedded)
├── 📁 helm/                       # Helm charts
│   └── 📁 ocx-webhook/            # Webhook Helm chart
├── 📁 internal/                   # Private Go packages
│   ├── 📁 ai/                     # AI/ML integration
│   ├── 📁 api/                    # API handlers
│   ├── 📁 consensus/              # Consensus mechanisms
│   ├── 📁 database/               # Database layer
│   ├── 📁 engine/                 # Execution engine
│   ├── 📁 gpu/                    # GPU integration
│   ├── 📁 security/               # Security utilities
│   ├── 📁 verification/           # Verification logic
│   └── 📁 [20+ other packages]    # Various internal components
├── 📁 k8s/                        # Kubernetes resources
│   └── 📁 webhook/                # Webhook K8s manifests
├── 📁 keys/                       # Cryptographic keys and certificates
├── 📁 libocx-verify/              # Rust verification library
│   ├── 📁 src/                    # Rust source code
│   ├── 📁 tests/                  # Rust test files
│   ├── 📁 target/                 # Cargo build artifacts
│   ├── Cargo.toml                 # Rust dependencies
│   └── build.rs                   # Build script
├── 📁 node_modules/               # Node.js dependencies (ignored)
├── 📁 pkg/                        # Public Go packages
│   ├── 📁 api/                    # API types and handlers
│   ├── 📁 cbor/                   # CBOR serialization
│   ├── 📁 deterministicvm/        # Deterministic VM implementation
│   ├── 📁 executor/               # Execution engine
│   ├── 📁 verify/                 # Verification logic
│   └── 📁 [10+ other packages]    # Various public packages
├── 📁 posters/                    # Architecture posters and diagrams
├── 📁 public/                     # Public web assets
│   └── 📁 assets/                 # Images, logos, icons
├── 📁 release/                    # Release artifacts
│   └── 📁 pilot-kit/              # Pilot deployment kit
├── 📁 scripts/                    # Build and utility scripts
├── 📁 src/                        # React frontend source
│   ├── 📁 components/             # React components
│   └── 📁 pages/                  # React pages
├── 📁 store/                      # State management
├── 📁 tests/                      # Test suites
│   ├── 📁 benchmark/              # Performance tests
│   ├── 📁 integration/            # Integration tests
│   ├── 📁 security/               # Security tests
│   └── 📁 unit/                   # Unit tests
├── 📁 .github/workflows/          # GitHub Actions CI/CD
├── 📄 Makefile                    # Main build system
├── 📄 go.mod                      # Go module definition
├── 📄 package.json                # Node.js dependencies
└── 📄 README.md                   # Project documentation
```

## Key Directories with Source Code

### `/cmd/` - Command Line Applications
**Purpose**: Go command-line tools and servers
**Key Files**: 
- `api-server/main.go` - Main API server
- `ocx-server/main.go` - OCX core server  
- `ocx-verifier/main.go` - Standalone verifier
- `ocx-webhook/main.go` - Kubernetes webhook

**Dependencies**: Imports from `/pkg/` and `/internal/`
**Called By**: Docker containers, Kubernetes pods, CLI users

### `/pkg/` - Public Go Packages
**Purpose**: Reusable Go libraries for OCX functionality
**Key Files**:
- `verify/` - Receipt verification logic
- `cbor/` - CBOR serialization/deserialization
- `deterministicvm/` - Deterministic execution environment
- `executor/` - Program execution engine

**Dependencies**: Standard library, external Go modules
**Called By**: `/cmd/` applications, `/internal/` packages

### `/internal/` - Private Go Packages
**Purpose**: Internal implementation details not exposed outside module
**Key Files**:
- `api/` - HTTP API handlers
- `database/` - Database layer
- `security/` - Security utilities
- `verification/` - Core verification logic

**Dependencies**: `/pkg/` packages, external modules
**Called By**: `/cmd/` applications only

### `/libocx-verify/` - Rust Verification Library
**Purpose**: High-performance cryptographic verification in Rust
**Key Files**:
- `src/lib.rs` - Main library entry point
- `src/receipt.rs` - Receipt parsing and validation
- `src/ffi.rs` - C FFI interface
- `src/canonical_cbor.rs` - CBOR canonicalization

**Dependencies**: `ring` (crypto), `hex` (encoding)
**Called By**: Go applications via FFI, C++ applications

### `/adapters/` - Platform Integration Adapters
**Purpose**: Drop-in integrations for different platforms
**Key Files**:
- `ad3-envoy/src/` - C++ Envoy filter
- `ad4-github/src/` - Node.js GitHub Action
- `ad5-terraform/internal/` - Terraform provider
- `ad6-kafka/src/` - Java Kafka interceptors

**Dependencies**: Platform-specific SDKs
**Called By**: Platform runtimes (Envoy, GitHub Actions, etc.)

## Test Directories

### `/tests/` - Test Suites
- `integration/` - End-to-end integration tests
- `unit/` - Unit tests for individual components
- `benchmark/` - Performance benchmarks
- `security/` - Security-focused tests

### `/conformance/` - Conformance Testing
- `receipts/v1/` - Generated test vectors
- `golden/` - Reference test vectors

### `/libocx-verify/tests/` - Rust Tests
- `test_receipt.rs` - Receipt parsing tests
- `test_ffi.rs` - FFI interface tests
- `golden_vectors.rs` - Cross-language test vectors

## Build Artifacts

### `/bin/` - Built Binaries
- Go executables built from `/cmd/`
- Generated by `go build` commands

### `/build/` - Build Output
- Static web assets
- Generated documentation

### `/libocx-verify/target/` - Rust Build Artifacts
- `debug/` - Debug builds
- `release/` - Optimized builds
- Generated by `cargo build`

### `/adapters/*/target/` - Java Build Artifacts
- Maven build output
- JAR files for Kafka interceptors

### `/adapters/*/dist/` - Node.js Build Artifacts
- Webpack bundled output
- Single-file GitHub Actions

## Configuration Files

- `Makefile` - Main build system (400+ lines)
- `go.mod` - Go module dependencies
- `Cargo.toml` - Rust dependencies
- `package.json` - Node.js dependencies
- `pom.xml` - Maven dependencies
- `CMakeLists.txt` - CMake configuration
