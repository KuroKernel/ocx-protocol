# OCX Master Architecture Diagram - A3 Poster

## OCX Protocol v1.0.0-rc.1
**Production-Ready Deterministic Computation Infrastructure**

---

## 🎯 **CORE PRINCIPLES**

### ① **Proof, not promises**
OCX receipts contain mathematical proof of execution - artifact hash, input hash, output hash, cycle count, and cryptographic signature. No identity data, just verifiable facts.

### ② **Offline verify**
Anyone can validate receipts without network access using only the receipt blob and public key. No dependency on external services or registries.

### ③ **Cross-arch determinism**
The same computation produces identical receipt hashes on x86 and ARM architectures, proving true deterministic execution.

### ④ **Frozen spec + profiles**
The v1-min specification never changes. New capabilities get new profile IDs (v1-fp, v1-gpu), ensuring backward compatibility forever.

### ⑤ **Policy out-of-band**
OCX-EXT envelope carries optional metadata like auditor signatures or KYC data, keeping the base receipt identity-free and pure.

### ⑥ **Fair economics**
Revenue comes from convenience (hosted APIs) and assurance (audit services), never from lock-in or extraction. Users can always self-host and export data.

---

## 🏗️ **ARCHITECTURE LAYERS**

### 🎭 **Actors & Stakeholders**
- **Developers/Integrators**: Build applications using OCX
- **Operators/FinOps/Compliance**: Manage and monitor OCX deployments  
- **Auditors/Insurers/Regulators**: Verify and validate computations
- **End Users/Customers**: Consume verified computation results

### 🔌 **Adapters (Drop-ins)**
- **GitHub Action** (`ocx-verify-action`): CI/CD integration
- **Kubernetes Webhook** (`label: ocx=on`): Container orchestration
- **Airflow Operator** (`@ocx.task`): Workflow automation
- **FFmpeg Filter** (`-vf ocx=emit=1`): Media processing
- **PyTorch Wrapper** (`ocx.exec(...)`): ML/AI integration
- **CLI Wrapper** (`ocx run -- cmd`): Command-line integration

### ⚡ **Ingress (CLI / SDK / API)**
- **minimal-cli**: Command-line interface
- **SDKs**: Go/Python/Rust client libraries
- **REST API**: HTTP endpoints for execution and verification

### 🔥 **OCX Core (v1-min)**
- **Spec v1-min (FROZEN)**: Immutable specification
- **Deterministic VM**: No clock/syscalls/threads/FP
- **Cycle Meter**: Precise resource measurement (alpha, beta, gamma)
- **Transcript Builder**: Hash chain to Merkle root
- **CBOR Serializer**: Canonical, strict encoding
- **Ed25519 Signer**: Cryptographic signatures
- **Receipt Emitter**: Proof of computation

### ✅ **Truth & Conformance**
- **CRI-lite**: Executable specification reference
- **Conformance Suite**: Golden receipts and test vectors
- **Cross-Arch Determinism**: Identical results across architectures

### 🔍 **Verify & Storage**
- **Offline Verifier**: Constant-time verification
- **Hosted Verify API**: Stateless verification service ($5/M checks)
- **Receipts Store**: Immutable storage with search ($3/M receipts-mo)
- **Exporters**: Data export utilities (CSV/Parquet)

### 📋 **Policy Layer (Optional)**
- **OCX-EXT Envelope**: Optional metadata
- **Auditor Quorum**: Multi-signature validation
- **Billing/Chargeback**: Cost management
- **Compliance Mapping**: Regulatory compliance

### 📊 **Analytics & Bench**
- **Benchmarks**: Performance metrics (exec/verify p50/p99)
- **Determinism Lab**: Cross-platform testing matrix
- **Atlas**: Privacy-preserving analytics

### 🛡️ **Governance / Security**
- **Profiles**: Version management (v1-min, v1-fp, v1-gpu)
- **Disputes**: Drift resolution procedures
- **Fairness**: Economic principles
- **Security**: Strict validation and rate limiting

---

## 🔄 **DATA FLOWS**

### **Execute Path**
1. **Adapters** → **CLI/SDK/REST** → **OCX_EXEC**
2. **Deterministic VM** → **Cycle Meter** → **Transcript Builder**
3. **CBOR Serializer** → **Ed25519 Signer** → **Receipt Emitter**
4. **Receipt** → **Verifier/Storage/Policy**

### **Verify Paths**
1. **Offline Verifier**: Local validation without network
2. **Hosted Verify API**: Remote validation service
3. **Auditor Verification**: Multi-signature validation

### **Conformance Loop**
1. **Golden Vectors** → **OCX Core** → **Computed Receipts**
2. **Cross-Arch Testing** → **Identical Hashes** → **Determinism Proof**
3. **CRI-lite** → **Drift Detection** → **Golden Updates**

---

## 📁 **REPOSITORY STRUCTURE**

- **[/conformance](/conformance)**: Conformance testing and reference implementation
- **[/cmd/minimal-cli](/cmd/minimal-cli)**: Command-line interface
- **[/gateway.go](/gateway.go)**: HTTP API endpoints
- **[/pkg/ocx](/pkg/ocx)**: Core protocol implementation
- **[/pkg/receipt](/pkg/receipt)**: Receipt generation and verification
- **[/store](/store)**: Database layer and persistence
- **[/scripts](/scripts)**: Build and deployment scripts
- **[/docs](/docs)**: Technical documentation

---

## 🚀 **PRODUCTION STATUS**

### **✅ OCX Protocol v1.0.0-rc.1 - PRODUCTION READY**

- **Specification**: Frozen and immutable
- **Determinism**: Cross-architecture verified
- **Conformance**: 100% test pass rate
- **CI/CD**: Automated validation pipeline
- **Documentation**: Complete and comprehensive

### **Key Features**
- **Deterministic Execution**: Identical results across platforms
- **Cryptographic Proofs**: Ed25519 signatures and CBOR receipts
- **Offline Verification**: No network dependency
- **Fair Economics**: Convenience + assurance only
- **Frozen Spec**: Backward compatibility forever

---

## 📞 **CONTACT & RESOURCES**

- **Repository**: [OCX Protocol](https://github.com/ocx-protocol/ocx)
- **Documentation**: [/docs](/docs)
- **Specification**: [v1-min (Frozen)](/docs/spec-v1.md)
- **Conformance**: [Test Vectors](/conformance)
- **CI/CD**: [GitHub Actions](/.github/workflows)

---

**OCX Protocol v1.0.0-rc.1**  
*Production-Ready Deterministic Computation Infrastructure*

*This poster is updated only when profiles or adapters change. The core v1-min specification remains frozen.*
