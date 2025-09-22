# OCX Protocol Dependency Graphs

## System Context Diagram

```mermaid
graph TB
    %% Actors
    subgraph "Actors"
        DEV[Developers]
        OPS[Operators]
        AUD[Auditors]
        USR[End Users]
    end
    
    %% Adapters
    subgraph "Adapters"
        GH[GitHub Action]
        K8S[K8s Webhook]
        ENV[Envoy Filter]
        KAF[Kafka Interceptor]
        TF[Terraform Provider]
        CLI[CLI Tools]
    end
    
    %% API Layer
    subgraph "API Layer"
        REST[REST API Server]
        SDK[Go SDK]
    end
    
    %% Core Libraries
    subgraph "Core Libraries"
        VER[Go Verifier]
        RUST[Rust Verifier]
        VM[Deterministic VM]
        CBOR[CBOR Library]
    end
    
    %% Storage
    subgraph "Storage"
        DB[(Database)]
        FS[File System]
        CACHE[Memory Cache]
    end
    
    %% External
    subgraph "External"
        K8S_API[Kubernetes API]
        GITHUB[GitHub Actions]
        ENVOY[Envoy Proxy]
        KAFKA[Kafka Cluster]
    end
    
    %% Connections
    DEV --> GH
    DEV --> CLI
    OPS --> K8S
    OPS --> TF
    AUD --> CLI
    USR --> REST
    
    GH --> REST
    K8S --> K8S_API
    ENV --> ENVOY
    KAF --> KAFKA
    TF --> REST
    CLI --> REST
    
    REST --> VER
    REST --> VM
    SDK --> VER
    SDK --> VM
    
    VER --> RUST
    VER --> CBOR
    VM --> CBOR
    
    VER --> DB
    VER --> FS
    REST --> CACHE
    
    %% Data Flow
    RUST -.->|"Receipt CBOR"| VER
    VER -.->|"Receipt CBOR"| REST
    REST -.->|"Receipt CBOR"| DB
```

## Go Package Dependency Graph

```mermaid
graph TB
    %% Main commands
    subgraph "Commands"
        API[cmd/api-server]
        WEBHOOK[cmd/ocx-webhook]
        VERIFIER[cmd/ocx-verifier]
        CLI[cmd/ocx]
    end
    
    %% Public packages
    subgraph "Public Packages (pkg/)"
        PKG_VERIFY[pkg/verify]
        PKG_CBOR[pkg/cbor]
        PKG_VM[pkg/deterministicvm]
        PKG_EXEC[pkg/executor]
        PKG_API[pkg/api]
    end
    
    %% Internal packages
    subgraph "Internal Packages (internal/)"
        INT_API[internal/api]
        INT_DB[internal/database]
        INT_SEC[internal/security]
        INT_VER[internal/verification]
    end
    
    %% External dependencies
    subgraph "External Dependencies"
        EXT_CBOR[github.com/fxamacker/cbor/v2]
        EXT_UUID[github.com/google/uuid]
        EXT_PROM[github.com/prometheus/client_golang]
        EXT_SQL[github.com/lib/pq]
    end
    
    %% Command dependencies
    API --> PKG_VERIFY
    API --> PKG_VM
    API --> PKG_API
    API --> INT_API
    
    WEBHOOK --> PKG_VERIFY
    WEBHOOK --> INT_API
    
    VERIFIER --> PKG_VERIFY
    VERIFIER --> PKG_CBOR
    
    CLI --> PKG_VERIFY
    CLI --> PKG_VM
    
    %% Package dependencies
    PKG_VERIFY --> PKG_CBOR
    PKG_VERIFY --> RUST_FFI[Rust FFI]
    
    PKG_VM --> PKG_CBOR
    PKG_VM --> PKG_EXEC
    
    PKG_API --> PKG_VERIFY
    
    INT_API --> PKG_VERIFY
    INT_API --> INT_DB
    
    INT_DB --> EXT_SQL
    INT_VER --> PKG_VERIFY
    
    %% External dependencies
    PKG_CBOR --> EXT_CBOR
    PKG_API --> EXT_UUID
    INT_API --> EXT_PROM
```

## Rust Crate Dependency Graph

```mermaid
graph TB
    %% Main crate
    subgraph "libocx-verify"
        LIB[lib.rs]
        RECEIPT[receipt.rs]
        FFI[ffi.rs]
        CBOR[canonical_cbor.rs]
        ERROR[error.rs]
        SPEC[spec.rs]
        DEBUG[debug.rs]
    end
    
    %% External dependencies
    subgraph "External Dependencies"
        RING[ring]
        HEX[hex]
        TEMPFILE[tempfile]
        CC[cc]
    end
    
    %% Test crates
    subgraph "Tests"
        TEST_RECEIPT[test_receipt.rs]
        TEST_FFI[test_ffi.rs]
        TEST_GOLDEN[golden_vectors.rs]
        TEST_DEMO[demo.rs]
    end
    
    %% Internal dependencies
    LIB --> RECEIPT
    LIB --> FFI
    LIB --> CBOR
    LIB --> ERROR
    LIB --> SPEC
    LIB --> DEBUG
    
    RECEIPT --> CBOR
    RECEIPT --> ERROR
    RECEIPT --> SPEC
    
    FFI --> RECEIPT
    FFI --> ERROR
    
    CBOR --> ERROR
    
    DEBUG --> RECEIPT
    DEBUG --> ERROR
    
    %% External dependencies
    RECEIPT --> RING
    DEBUG --> HEX
    FFI --> CC
    
    %% Test dependencies
    TEST_RECEIPT --> RECEIPT
    TEST_FFI --> FFI
    TEST_GOLDEN --> RECEIPT
    TEST_DEMO --> RECEIPT
```

## Data Flow: Execute → Receipt → Verify

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant VM
    participant CBOR
    participant RUST
    participant DB
    
    %% Execution Phase
    Client->>API: POST /execute {artifact, input}
    API->>VM: ExecuteArtifact(artifact, input)
    VM->>VM: Run with cycle counting
    VM-->>API: {output, cycles, timestamps}
    
    %% Receipt Generation Phase
    API->>CBOR: CreateReceipt(artifact_hash, input_hash, output_hash, cycles)
    CBOR->>CBOR: Serialize to canonical CBOR
    CBOR->>RUST: Sign with Ed25519
    RUST-->>CBOR: Ed25519 signature
    CBOR-->>API: Complete receipt CBOR
    
    %% Storage Phase
    API->>DB: Store receipt
    API-->>Client: {receipt, output, cycles}
    
    %% Verification Phase
    Client->>API: POST /verify {receipt, public_key}
    API->>RUST: VerifyReceipt(receipt, public_key)
    RUST->>RUST: Parse CBOR
    RUST->>RUST: Extract signature
    RUST->>RUST: Reconstruct message
    RUST->>RUST: Verify Ed25519
    RUST-->>API: {verified, fields}
    API-->>Client: {verified, receipt_fields}
```

## Cross-Language Integration

```mermaid
graph TB
    %% Go Layer
    subgraph "Go Layer"
        GO_VERIFY[Go Verifier]
        GO_FFI[FFI Bridge]
    end
    
    %% Rust Layer
    subgraph "Rust Layer"
        RUST_VERIFY[Rust Verifier]
        RUST_FFI[C FFI Interface]
    end
    
    %% C++ Layer
    subgraph "C++ Layer"
        CPP_FILTER[Envoy Filter]
        CPP_FFI[C++ FFI Wrapper]
    end
    
    %% Java Layer
    subgraph "Java Layer"
        JAVA_INTERCEPTOR[Kafka Interceptor]
        JAVA_FFI[JNI Wrapper]
    end
    
    %% Node.js Layer
    subgraph "Node.js Layer"
        NODE_ACTION[GitHub Action]
        NODE_FFI[Node.js FFI]
    end
    
    %% Data Flow
    GO_VERIFY --> GO_FFI
    GO_FFI --> RUST_FFI
    RUST_FFI --> RUST_VERIFY
    
    CPP_FILTER --> CPP_FFI
    CPP_FFI --> RUST_FFI
    
    JAVA_INTERCEPTOR --> JAVA_FFI
    JAVA_FFI --> RUST_FFI
    
    NODE_ACTION --> NODE_FFI
    NODE_FFI --> RUST_FFI
    
    %% Shared Data
    RUST_VERIFY -.->|"Receipt CBOR"| GO_VERIFY
    RUST_VERIFY -.->|"Receipt CBOR"| CPP_FILTER
    RUST_VERIFY -.->|"Receipt CBOR"| JAVA_INTERCEPTOR
    RUST_VERIFY -.->|"Receipt CBOR"| NODE_ACTION
```

## Build System Dependencies

```mermaid
graph TB
    %% Build Tools
    subgraph "Build Tools"
        MAKE[Makefile]
        DOCKER[Docker]
        GITHUB[GitHub Actions]
    end
    
    %% Language Builders
    subgraph "Language Builders"
        GO_BUILD[go build]
        CARGO[cargo build]
        CMAKE[cmake]
        NPM[npm build]
        MAVEN[mvn package]
    end
    
    %% Artifacts
    subgraph "Artifacts"
        GO_BIN[Go Binaries]
        RUST_LIB[Rust Library]
        CPP_FILTER[C++ Filter]
        NODE_ACTION[Node.js Action]
        JAVA_JAR[Java JARs]
    end
    
    %% Dependencies
    MAKE --> GO_BUILD
    MAKE --> CARGO
    MAKE --> CMAKE
    MAKE --> NPM
    MAKE --> MAVEN
    
    GO_BUILD --> GO_BIN
    CARGO --> RUST_LIB
    CMAKE --> CPP_FILTER
    NPM --> NODE_ACTION
    MAVEN --> JAVA_JAR
    
    DOCKER --> GO_BIN
    DOCKER --> RUST_LIB
    DOCKER --> CPP_FILTER
    DOCKER --> NODE_ACTION
    DOCKER --> JAVA_JAR
    
    GITHUB --> MAKE
    GITHUB --> DOCKER
```

## Runtime Dependencies

```mermaid
graph TB
    %% Runtime Environment
    subgraph "Runtime Environment"
        K8S[Kubernetes]
        ENVOY[Envoy Proxy]
        KAFKA[Kafka Cluster]
        GITHUB[GitHub Actions]
        TERRAFORM[Terraform]
    end
    
    %% OCX Components
    subgraph "OCX Components"
        API[API Server]
        WEBHOOK[Webhook]
        FILTER[Envoy Filter]
        INTERCEPTOR[Kafka Interceptor]
        ACTION[GitHub Action]
        PROVIDER[Terraform Provider]
    end
    
    %% External Services
    subgraph "External Services"
        DB[(Database)]
        REGISTRY[Container Registry]
        SECRETS[Secret Management]
        MONITORING[Monitoring]
    end
    
    %% Dependencies
    K8S --> WEBHOOK
    K8S --> API
    ENVOY --> FILTER
    KAFKA --> INTERCEPTOR
    GITHUB --> ACTION
    TERRAFORM --> PROVIDER
    
    API --> DB
    WEBHOOK --> SECRETS
    FILTER --> API
    INTERCEPTOR --> API
    ACTION --> API
    PROVIDER --> API
    
    API --> MONITORING
    WEBHOOK --> MONITORING
    FILTER --> MONITORING
    INTERCEPTOR --> MONITORING
    ACTION --> MONITORING
    PROVIDER --> MONITORING
```
