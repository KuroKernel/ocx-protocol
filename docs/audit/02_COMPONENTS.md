# OCX Protocol Components Deep Dive

## 1. API Server (`cmd/api-server/`)

**Purpose**: Main HTTP API server providing REST endpoints for OCX operations.

**Public Surface**:
- `GET /api/v1/status` - Health check
- `POST /api/v1/execute` - Execute artifact with OCX verification
- `POST /api/v1/verify` - Verify receipt signature
- `GET /api/v1/receipts/{id}` - Retrieve receipt by ID
- `POST /api/v1/batch-verify` - Batch verify multiple receipts

**Inbound Dependencies**: HTTP clients, CLI tools, webhook
**Outbound Dependencies**: `pkg/verify`, `pkg/deterministicvm`, `pkg/cbor`

**Data Contracts**: OCX Receipt (canonical CBOR + Ed25519 signature)
**Startup Concerns**: Port binding, TLS configuration, database connection
**Tests**: Basic HTTP endpoint tests, integration tests
**Gaps**: No rate limiting, no authentication, no metrics

## 2. Rust Verifier Library (`libocx-verify/`)

**Purpose**: High-performance cryptographic verification of OCX receipts.

**Public Surface**:
- `verify_receipt(cbor_data, public_key)` - Verify single receipt
- `verify_receipts_batch(receipts, public_keys)` - Batch verification
- `extract_receipt_fields(cbor_data)` - Parse receipt fields
- C FFI interface for Go/Java/C++ integration

**Inbound Dependencies**: Go applications via FFI, C++ applications
**Outbound Dependencies**: `ring` (crypto), `hex` (encoding)

**Data Contracts**: Canonical CBOR with integer keys (1-8), Ed25519 signatures
**Startup Concerns**: FFI library loading, crypto initialization
**Tests**: Unit tests, golden vector tests, FFI tests
**Gaps**: Placeholder signatures in tests, no real Ed25519 implementation

## 3. Deterministic VM (`pkg/deterministicvm/`)

**Purpose**: Isolated execution environment ensuring deterministic computation.

**Public Surface**:
- `ExecuteArtifact(artifact, input, config)` - Execute with isolation
- `GetArtifactInfo(artifact)` - Analyze artifact metadata
- `VMConfig` - Configuration for execution limits

**Inbound Dependencies**: API server, CLI tools
**Outbound Dependencies**: OS process isolation, resource monitoring

**Data Contracts**: Artifact hash, input/output hashes, cycle counts
**Startup Concerns**: Process isolation setup, resource limits
**Tests**: Unit tests, integration tests with real artifacts
**Gaps**: Limited OS support, no GPU isolation

## 4. Kubernetes Webhook (`cmd/ocx-webhook/`)

**Purpose**: Mutating admission controller for automatic OCX injection.

**Public Surface**:
- Kubernetes admission webhook endpoint
- Pod mutation based on `ocx-inject: "true"` annotation
- Init container injection for OCX binary

**Inbound Dependencies**: Kubernetes API server
**Outbound Dependencies**: `pkg/verify`, container runtime

**Data Contracts**: Kubernetes PodSpec, OCX annotations
**Startup Concerns**: TLS certificate management, RBAC configuration
**Tests**: Unit tests, integration tests with test clusters
**Gaps**: No certificate auto-rotation, limited error handling

## 5. Envoy HTTP Filter (`adapters/ad3-envoy/`)

**Purpose**: Service mesh integration for transparent OCX verification.

**Public Surface**:
- HTTP filter for Envoy proxy
- Request/response verification
- Configurable verification policies

**Inbound Dependencies**: Envoy proxy runtime
**Outbound Dependencies**: `libocx-verify` FFI, HTTP client

**Data Contracts**: HTTP headers, OCX receipts in responses
**Startup Concerns**: Envoy filter registration, FFI library loading
**Tests**: Unit tests, Envoy integration tests
**Gaps**: Incomplete implementation, missing error handling

## 6. GitHub Action (`adapters/ad4-github/`)

**Purpose**: CI/CD integration for automated OCX verification.

**Public Surface**:
- GitHub Action with inputs: `artifact`, `input`, `expected-output`
- Action outputs: `receipt`, `verified`, `cycles-used`

**Inbound Dependencies**: GitHub Actions runtime
**Outbound Dependencies**: OCX CLI, file system

**Data Contracts**: GitHub Action inputs/outputs, OCX receipts
**Startup Concerns**: Action packaging, dependency resolution
**Tests**: Jest unit tests, GitHub Actions integration tests
**Gaps**: Limited error reporting, no caching

## 7. Terraform Provider (`adapters/ad5-terraform/`)

**Purpose**: Infrastructure as code integration for OCX verification.

**Public Surface**:
- `ocx_provenance` resource for artifact verification
- Data sources for receipt validation
- Provider configuration

**Inbound Dependencies**: Terraform runtime
**Outbound Dependencies**: OCX API server, HTTP client

**Data Contracts**: Terraform resource schema, OCX receipts
**Startup Concerns**: Provider registration, API client setup
**Tests**: Unit tests, Terraform acceptance tests
**Gaps**: Minimal implementation, no real API integration

## 8. Kafka Interceptor (`adapters/ad6-kafka/`)

**Purpose**: Message queue integration for OCX verification.

**Public Surface**:
- Producer interceptor for message signing
- Consumer interceptor for message verification
- Configurable verification policies

**Inbound Dependencies**: Kafka client runtime
**Outbound Dependencies**: `libocx-verify` FFI, HTTP client

**Data Contracts**: Kafka message headers, OCX receipts
**Startup Concerns**: Interceptor registration, FFI library loading
**Tests**: Unit tests, Kafka integration tests
**Gaps**: Incomplete implementation, missing error handling

## 9. CLI Tools (`cmd/ocx-*`)

**Purpose**: Command-line interfaces for OCX operations.

**Public Surface**:
- `ocx execute` - Execute artifact with verification
- `ocx verify` - Verify receipt signature
- `ocxctl` - Administrative commands
- `ocx-verifier` - Standalone verification tool

**Inbound Dependencies**: Terminal users, scripts
**Outbound Dependencies**: `pkg/verify`, `pkg/deterministicvm`

**Data Contracts**: Command-line arguments, OCX receipts
**Startup Concerns**: Argument parsing, configuration loading
**Tests**: Unit tests, integration tests
**Gaps**: Limited error messages, no configuration management

## 10. Conformance Suite (`conformance/`)

**Purpose**: Cross-language test vectors and conformance validation.

**Public Surface**:
- Golden test vectors for all languages
- Conformance test runners
- Reference implementations

**Inbound Dependencies**: All language implementations
**Outbound Dependencies**: OCX specification, test frameworks

**Data Contracts**: Canonical CBOR test vectors, expected results
**Startup Concerns**: Test vector generation, cross-platform compatibility
**Tests**: Cross-language conformance tests
**Gaps**: Limited test coverage, no automated generation

## 11. Helm Chart (`helm/ocx-webhook/`)

**Purpose**: Kubernetes deployment package for OCX webhook.

**Public Surface**:
- Helm chart with configurable values
- Kubernetes manifests for webhook deployment
- RBAC and service account configuration

**Inbound Dependencies**: Helm package manager
**Outbound Dependencies**: Kubernetes cluster, container registry

**Data Contracts**: Helm values, Kubernetes manifests
**Startup Concerns**: Chart installation, resource creation
**Tests**: Helm template tests, deployment tests
**Gaps**: No upgrade strategy, limited configuration options

## 12. Frontend (`src/`)

**Purpose**: React-based web interface for OCX operations.

**Public Surface**:
- Web UI for receipt verification
- Dashboard for monitoring
- Configuration interface

**Inbound Dependencies**: Web browsers
**Outbound Dependencies**: API server, static assets

**Data Contracts**: REST API calls, JSON responses
**Startup Concerns**: Build process, asset bundling
**Tests**: Jest unit tests, React testing library
**Gaps**: Limited functionality, no real-time updates

## Common Patterns Across Components

### Data Flow
1. **Input**: Artifact + input data + configuration
2. **Execution**: Deterministic VM with cycle counting
3. **Receipt Generation**: Canonical CBOR + Ed25519 signature
4. **Verification**: Cryptographic signature validation
5. **Storage**: Receipt persistence and retrieval

### Error Handling
- Consistent error codes across languages
- Detailed error messages with context
- Graceful degradation for non-critical failures

### Configuration
- Environment variable based configuration
- Default values for development
- Production-ready security settings

### Testing Strategy
- Unit tests for individual components
- Integration tests for cross-component workflows
- Conformance tests for specification compliance
- Performance tests for critical paths
