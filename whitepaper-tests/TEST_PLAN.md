# OCX Protocol — Test Plan

**Document purpose:** formal, citable list of OCX protocol claims and the tests that verify each claim. Designed to be referenced from the OCX whitepaper.

**Scope:** Layer 1 (canonical receipt schema), Layer 2 (signed receipt protocol), and Layer 2.5 (deterministic computation evidence). Does not cover Layer 3 (peer-witnessed consensus) — that layer is not yet implemented; see `STATUS.md` and `STRATEGY.md` for the design.

---

## Claim 1: Canonical encoding of a `ReceiptCore` is deterministic and byte-equal across language implementations

**What it means:** Given identical field values, the Go (`pkg/receipt.CanonicalizeCore`), Python (`canonical_receipt.py`), and Rust (`libocx-verify`) canonical CBOR encoders produce identical byte sequences.

**Why it matters:** if encodings differ across languages, signatures produced in one language cannot be verified in another. Cross-language portability requires byte-identity of the canonical form.

**Tests verifying this claim:**

| Test | Location | Languages |
|---|---|---|
| `TestCanonicalizeCore` | `pkg/receipt/canonical_test.go` | Go |
| `TestCanonicalizeFull` | `pkg/receipt/canonical_test.go` | Go |
| `TestCanonicalizationDeterminism` | `pkg/receipt/canonical_test.go` | Go |
| `TestCanonicalizationConsistency` | `pkg/receipt/canonical_test.go` | Go |
| `test_canonical_cbor*` | `libocx-verify/tests/test_canonical_cbor.rs` | Rust |
| `cross_language_roundtrip` | `whitepaper-tests/cross_language_roundtrip.{go,py}` | Go ↔ Python ↔ Rust |
| `parity/dump_canonical.{go,py}` | `examples/gpu-verifier/parity/` | Go ↔ Python fixture parity |

**Pass criterion:** all tests succeed; the cross-language test specifically asserts byte-equality of the canonical signing bytes between Go and Python over a fixed fixture.

---

## Claim 2: Signing is deterministic Ed25519 over `b"OCXv1|receipt|" + canonical_cbor(signed_map)`

**What it means:** The signing message is the OCX domain separator string concatenated with the canonical CBOR encoding of the signed-fields map (integer keys 1-7 plus optional 9-15). Ed25519 over this message is byte-deterministic by definition of the algorithm. Different domain separators produce different signatures over otherwise-identical canonical bytes.

**Why it matters:** the domain separator prevents cross-protocol signature reuse. Without it, an OCX signature could be replayed against a different signing scheme that happens to use the same canonical CBOR shape.

**Tests verifying this claim:**

| Test | Location |
|---|---|
| `TestReceiptSignature*` | `pkg/keystore/keystore_test.go`, `pkg/verify/wrapper_test.go` |
| `test_verify_receipt_success` | `libocx-verify/tests/test_verify.rs` |
| `test_verify_receipt_invalid_signature` | `libocx-verify/tests/test_verify.rs` |
| `test_verify_receipt_invalid_signature_length` | `libocx-verify/tests/test_verify.rs` |
| `cross_language_roundtrip` Test 2 | `whitepaper-tests/cross_language_roundtrip.py` |

**Pass criterion:** Go and Python both sign the same canonical CBOR + domain-separated message and produce byte-identical 64-byte signatures, and the Rust verifier accepts them both.

---

## Claim 3: Receipts produced in any language verify offline through the canonical Rust verifier

**What it means:** `libocx-verify.so` exposes `ocx_verify_receipt_detailed(cbor_data, len, public_key, error_code_out) -> bool`. A receipt is verified by parsing canonical CBOR, reconstructing the signing message, and verifying Ed25519 against the supplied public key. No network access, no trusted server, no time service.

**Why it matters:** verification must be possible by anyone, anywhere, without trusting a central party. Offline verification is the foundation of trust-minimization.

**Tests verifying this claim:**

| Test | Location |
|---|---|
| `test_verify_receipt_detailed` (and 14 sibling tests) | `libocx-verify/tests/test_verify.rs` |
| `test_ffi*` (FFI ABI) | `libocx-verify/tests/test_ffi.rs` |
| `cross_language_roundtrip` Tests 4-6 | `whitepaper-tests/cross_language_roundtrip.py` |
| `bench_verify` | `whitepaper-tests/bench_verify.py` |

**Pass criterion:** Rust verifier returns `OCX_SUCCESS` for valid receipts, returns the specific error code for each invalidity class (`OCX_INVALID_CBOR`, `OCX_NON_CANONICAL_CBOR`, `OCX_MISSING_FIELD`, `OCX_INVALID_FIELD_VALUE`, `OCX_INVALID_SIGNATURE`, `OCX_HASH_MISMATCH`, `OCX_INVALID_TIMESTAMP`).

---

## Claim 4: Verification is fast enough for hot-path use (sub-millisecond, single-threaded)

**What it means:** verifying one receipt completes in less than one millisecond on commodity x86_64 hardware in a single Python ctypes-bound thread.

**Why it matters:** any application using OCX receipts must verify on every read, every dispute, every audit. If verification is slow, the protocol can't be embedded in time-sensitive workflows (real-time fraud detection, on-the-fly audit logs, etc.).

**Tests verifying this claim:**

| Test | Location |
|---|---|
| `bench_verify.py` | `whitepaper-tests/bench_verify.py` |
| `performance_test` | `libocx-verify/tests/performance_test.rs` |
| `BenchmarkCanonicalizeCore`, `BenchmarkCanonicalizeFull` | `pkg/receipt/canonical_test.go` |

**Pass criterion:** median verification < 1ms, p99 < 5ms, p999 < 10ms over ≥10,000 distinct receipts.

---

## Claim 5: Tampering with any signed field is detected by the verifier

**What it means:** any single-bit modification to a transmitted CBOR receipt, after signing, causes verification to fail with `OCX_INVALID_SIGNATURE`.

**Why it matters:** the protocol's anti-forgery guarantee. If a verifier can't distinguish authentic from modified receipts, the receipt is worthless.

**Tests verifying this claim:**

| Test | Location |
|---|---|
| `cross_language_roundtrip` Test 5 (signature byte flip) | `whitepaper-tests/cross_language_roundtrip.py` |
| `test_verify_receipt_invalid_signature` | `libocx-verify/tests/test_verify.rs` |
| `bench_verify` tamper test (used in earlier sessions) | `examples/gpu-verifier/test_determinism_gpu.py` |
| `TestReceiptValidation*` | `pkg/receipt/conformance_test.go`, `pkg/verify/property_test.go` |

**Pass criterion:** flipping any single byte in a valid receipt (signature, signed payload, or framing) yields `OCX_INVALID_SIGNATURE` or a related error from the verifier.

---

## Claim 6: Verification fails for the wrong public key

**What it means:** a receipt signed by issuer A's key cannot be verified using issuer B's public key.

**Tests verifying this claim:**

| Test | Location |
|---|---|
| `cross_language_roundtrip` Test 6 (wrong-key check) | `whitepaper-tests/cross_language_roundtrip.py` |
| `test_verify_receipt_invalid_public_key_length` | `libocx-verify/tests/test_verify.rs` |

**Pass criterion:** verifying with an unrelated public key returns `OCX_INVALID_SIGNATURE`.

---

## Claim 7: Receipt schema enforces semantic constraints

**What it means:** the verifier rejects logically inconsistent receipts even if the cryptographic signature is valid:

- `cycles_used == 0` → `OCX_INVALID_FIELD_VALUE`
- `cycles_used > 1_000_000_000` → `OCX_INVALID_FIELD_VALUE`
- `cycles_used < (finished_at - started_at)` (less than 1 cycle/sec implied) → `OCX_INVALID_FIELD_VALUE`
- any of `program_hash`, `input_hash`, `output_hash` is all zeros → `OCX_HASH_MISMATCH`
- `program_hash == input_hash`, `program_hash == output_hash`, or `input_hash == output_hash` → `OCX_HASH_MISMATCH`
- `finished_at < started_at` → `OCX_INVALID_TIMESTAMP`
- duration outside [1 second, 24 hours] → `OCX_INVALID_TIMESTAMP`
- `issuer_key_id` empty or > 256 chars or contains control chars → `OCX_INVALID_FIELD_VALUE`
- `signature.len() != 64` → `OCX_INVALID_SIGNATURE`

**Tests verifying this claim:**

| Test | Location |
|---|---|
| `test_verify_receipt_zero_cycles` | `libocx-verify/tests/test_verify.rs` |
| `test_verify_receipt_invalid_hash_constraints` | `libocx-verify/tests/test_verify.rs` |
| `test_verify_receipt_duplicate_hashes` | `libocx-verify/tests/test_verify.rs` |
| `test_verify_receipt_invalid_timestamps` | `libocx-verify/tests/test_verify.rs` |
| `test_verify_receipt_empty_key_id` | `libocx-verify/tests/test_verify.rs` |
| `test_verify_receipt_invalid_signature_length` | `libocx-verify/tests/test_verify.rs` |

**Pass criterion:** each invalid-field condition produces the documented error code.

---

## Claim 8: Deterministic computation produces byte-identical OCX receipts across fresh process invocations

**What it means:** running the same program (model + flags) on the same input under the same execution environment produces the same `output_hash` (and where applicable `logits_hash`) across cold-started, independent Python processes.

**Why it matters:** OCX's value proposition depends on the underlying computation being reproducible. If a verifier re-runs the same input and gets a different output, they cannot tell whether the prover lied or the computation simply isn't deterministic.

**Tests verifying this claim** (all results committed in `examples/gpu-verifier/results/h100/`):

| Configuration | Runs | Status |
|---|---|---|
| Qwen 2.5-72B-Instruct, 1×H100 + CPU offload, bf16, 128 tokens | 3 fresh processes | byte-identical |
| Qwen 2.5-72B-Instruct, 2×H100 pipeline parallel, bf16, 128 tokens | 1 fresh process | matches CPU-offload byte-for-byte |
| Qwen 2.5-72B-Instruct, 2×H100 tensor parallel, bf16, 32 tokens | 3 fresh torchruns | byte-identical |
| Qwen 2.5-72B-Instruct, 2×H100 tensor parallel, bf16, 128 tokens | 3 fresh torchruns | byte-identical |
| Meta Llama-3.1-70B-Instruct, 2×H100 tensor parallel, bf16, 32 tokens | 3 fresh torchruns | byte-identical |
| Meta Llama-3.1-70B-Instruct, 2×H100 tensor parallel, bf16, 128 tokens | 3 fresh torchruns | byte-identical |

Earlier-session evidence (CPU; `examples/ai-verifier/`):
- Qwen 2.5-0.5B q4_k_m on x86 CPU (5 in-process runs + 3 fresh processes) — byte-identical
- Qwen 2.5-7B q4_k_m on x86 CPU (3 fresh processes, short and 96-token long-gen) — byte-identical
- Qwen 2.5-0.5B q4_k_m on aarch64 CPU (Raspberry Pi 5, 3 fresh processes) — byte-identical within architecture
- Cross-architecture (x86_64 vs aarch64): texts diverge at token ~55 due to SIMD reduction-order differences (DOCUMENTED LIMITATION, not a protocol failure)

**Pass criterion:** every committed receipt JSON in a given test group has identical `output_hash` and `logits_hash` across runs in that group; aggregator script is `whitepaper-tests/aggregate_determinism_evidence.py`.

---

## Out-of-scope for this test plan

- **Deterministic VM tests** (`pkg/deterministicvm/`) require Linux namespaces, cgroups v2, and seccomp privileges. They fail in user-mode environments without those capabilities. They are protocol-adjacent infrastructure tests, not Layer 1 schema tests, and are not run as part of this test plan.
- **CLI conformance tests** (`conformance/`) require building `minimal-cli`. That binary is not committed; build separately if running them.
- **End-to-end stress / load / business / UX tests** (`tests/load`, `tests/business`, `tests/ux`) are application-level tests for the OCX server product, not protocol-level claims. Out of scope.
- **Layer 3 (peer-witnessed consensus, VDF temporal proofs):** designed in `STRATEGY.md` and the vision docs, not yet implemented, and therefore has no tests in this plan.

---

## How to run the tests in this plan

```bash
cd /home/kurokernel/Desktop/AXIS/ocx-protocol

# Layer 1 + 2 (Go side)
go test ./pkg/receipt/... ./pkg/verify/... ./pkg/keystore/... ./pkg/chain/... ./pkg/executor/... -count=1 -timeout=120s -v

# Layer 1 + 2 (Rust side)
cd libocx-verify && cargo test --release && cd ..

# Cross-language end-to-end
go run whitepaper-tests/cross_language_roundtrip.go
examples/ai-verifier/venv/bin/python whitepaper-tests/cross_language_roundtrip.py

# Verification latency benchmark
examples/ai-verifier/venv/bin/python whitepaper-tests/bench_verify.py 10000

# Determinism evidence aggregation (reads committed receipt JSONs)
python3 whitepaper-tests/aggregate_determinism_evidence.py
```
