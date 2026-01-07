# OCX Protocol Phase 1: Determinism Foundation Fixes

**Goal:** Make OCX receipts truly deterministic - identical artifact + identical input = identical receipt (except nonce)

**Status:** PLANNING
**Branch:** `integrate-dmvm-execution`

---

## Overview

Four critical fixes required:

| # | Issue | Current State | Target State |
|---|-------|---------------|--------------|
| 1 | Gas Model | Host-cycle dependent | Opcode-counting only |
| 2 | Nonce/Replay | Implemented but disconnected | Unified and enforced |
| 3 | FP Validator | Opcode presence only | Control flow analysis |
| 4 | Sandboxing | Fails open | Fails closed (strict) |

---

## Fix 1: Hardware-Independent Gas Model

### Problem

`pkg/deterministicvm/vm.go:253-271` calculates gas from `hostCycles`:

```go
func calculateDeterministicGas(cycles uint64, stdoutLen, stderrLen int) uint64 {
    baseGas := cycles / 1000  // <-- cycles is host-dependent!
    ...
}
```

And `vm.go:319-364` shows cycle calculation uses hardware-specific methods:
- `perf_event_open()` - varies by CPU
- `/proc/stat jiffies` - varies by kernel tick rate
- `cgroup cpu.stat` - varies by cgroup configuration
- Wall-clock fallback - varies by load

### Solution

**Use instruction-counting gas model exclusively.** The `gas.go` opcode table is already correct - we just need to USE it.

### Files to Modify

1. **`pkg/deterministicvm/vm.go`**
   - Remove `calculateDeterministicGas()` function that uses host cycles
   - Create new `InstrumentedExecutor` that counts WASM instructions or shell commands
   - For WASM: Use wazero's fuel metering (already partially implemented)
   - For shell: Parse script and sum opcode costs from `gas.go:GasCosts`

2. **`pkg/deterministicvm/gas.go`**
   - Rename `CalculateDeterministicGas()` to `CalculateScriptGas()` (it's script-specific)
   - Add `CalculateWASMGas(wasmBytes []byte) uint64` that statically analyzes WASM
   - Remove string-contains heuristics, use proper AST parsing

3. **`pkg/deterministicvm/wasm_engine.go`**
   - Enable fuel consumption in `WASMEngine.Run()`
   - Return fuel consumed as `GasUsed`

4. **`pkg/deterministicvm/fuel_meter.go`**
   - Ensure `FuelMeteredWASMEngine` actually consumes fuel during execution
   - Map WASM opcodes to fuel costs

### Implementation Steps

```
1.1 [ ] Create pkg/deterministicvm/gas_model.go
    - Define GasModel interface
    - Implement WASMGasModel (instruction counting)
    - Implement ScriptGasModel (command counting)

1.2 [ ] Modify wasm_engine.go
    - Add fuel configuration to wazero runtime
    - Track fuel consumption during execution
    - Return consumed fuel as GasUsed

1.3 [ ] Modify vm.go
    - Replace calculateDeterministicGas() with GasModel.Calculate()
    - Remove all host-cycle based gas calculations
    - Keep HostCycles in ExecutionResult but mark as "diagnostic only"

1.4 [ ] Update types.go OCXReceipt
    - Add comment clarifying GasUsed is deterministic
    - Add comment clarifying HostCycles is diagnostic/non-signed

1.5 [ ] Add tests
    - Test same WASM on different machines = same gas
    - Test same script on different machines = same gas
```

### Acceptance Criteria
- [ ] Running same artifact+input on x86 and ARM64 produces identical GasUsed
- [ ] GasUsed is derived purely from instruction/command counting
- [ ] HostCycles remains available for diagnostics but is NOT in signed receipt core

---

## Fix 2: Nonce in Signed Receipts (Unified)

### Problem

Two separate receipt structures exist:

1. `pkg/deterministicvm/types.go:186-205` - `OCXReceipt` (no nonce in SignedCore)
2. `pkg/receipt/v1_1/types.go:9-21` - `ReceiptCore` (HAS nonce at field 9)

The v1_1 implementation is correct but disconnected from the main DMVM flow.

### Solution

**Unify on v1_1 structure.** The receipt/v1_1 package has proper:
- Nonce generation (`crypto.go:82-90`)
- Replay protection (`replay.go`)
- Signature verification with clock skew (`crypto.go:159-187`)

### Files to Modify

1. **`pkg/deterministicvm/types.go`**
   - Remove duplicate `OCXReceipt` and `SignedCore` structs
   - Import and use `pkg/receipt/v1_1.ReceiptCore` and `ReceiptFull`
   - Or: Add Nonce field to existing SignedCore (field 8)

2. **`pkg/deterministicvm/executor.go`** (or wherever receipts are created)
   - Use `v1_1.CryptoManager.CreateReceipt()` instead of manual construction
   - Ensure nonce is generated and included

3. **`cmd/server/main.go`** (API handlers)
   - Ensure verification endpoint calls `ReplayProtection.CheckAndRecordNonce()`
   - Reject receipts with reused nonces

### Implementation Steps

```
2.1 [ ] Audit current receipt creation flow
    - Find all places where OCXReceipt is constructed
    - Map data flow from execution to receipt

2.2 [ ] Unify receipt types
    - Option A: Migrate deterministicvm to use v1_1 types
    - Option B: Add Nonce to deterministicvm.SignedCore
    - Choose based on which causes fewer breaking changes

2.3 [ ] Wire up replay protection
    - Initialize ReplayProtection in server startup
    - Call CheckAndRecordNonce() in verification handler
    - Add database migration for ocx_replay_protection table

2.4 [ ] Update verify-standalone binary
    - Add --check-replay flag
    - Connect to database for nonce verification
    - Validate clock skew

2.5 [ ] Add tests
    - Test replay attack detection
    - Test clock skew rejection
    - Test nonce uniqueness across receipts
```

### Acceptance Criteria
- [ ] Every receipt contains a unique 16-byte nonce
- [ ] Nonce is part of signed core data
- [ ] Server rejects receipts with previously-seen nonces
- [ ] verify-standalone validates nonce freshness

---

## Fix 3: Complete FP Validator with Control Flow Analysis

### Problem

`pkg/deterministicvm/fp_validator.go:167-177` only checks if FP opcodes EXIST:

```go
func (v *FPValidator) checkFloatingPointOps(module *WASMModule) error {
    for _, section := range module.Sections {
        if section.ID == 10 { // Code section
            if err := v.checkCodeSection(section.Data); err != nil {
                return err  // Just checks opcode presence
            }
        }
    }
    return nil
}
```

This misses:
- FP operations in unreachable code paths
- Conditional FP usage (`if debug { use_float() }`)
- FP operations behind function pointers

### Solution

**Implement proper control flow graph (CFG) analysis.** For MVP, we can:
1. Be conservative: reject ANY module containing FP opcodes (current)
2. Add CFG analysis to prove FP paths are unreachable

Option 1 is safer for now. Option 2 is future enhancement.

### Files to Modify

1. **`pkg/deterministicvm/fp_validator.go`**
   - Add `StrictMode` that rejects ANY FP opcode presence (safest)
   - Add `AnalyzeMode` that builds CFG (future enhancement)
   - Improve opcode scanning to catch all FP opcodes (current list incomplete)

2. **New: `pkg/deterministicvm/cfg_analyzer.go`** (future)
   - Build control flow graph from WASM
   - Identify reachable basic blocks
   - Prove FP blocks unreachable from entry points

### Implementation Steps

```
3.1 [ ] Audit FP opcode coverage
    - Compare current isFloatOpcode() against WASM spec
    - Add any missing FP opcodes
    - Include SIMD float operations

3.2 [ ] Add strict validation mode
    - New function: ValidateStrictNoFloat(wasmBytes) error
    - Rejects module if ANY FP opcode present anywhere
    - No control flow analysis needed (conservative)

3.3 [ ] Add ELF validation improvements
    - Current containsFloatInstructions() uses pattern matching
    - Add disassembly for accurate detection (use capstone or similar)
    - Or: For MVP, just reject all ELF with FP sections

3.4 [ ] Wire into execution flow
    - Call ValidateStrictNoFloat() before WASM execution
    - Call validateELFNoFloat() before ELF execution
    - Make validation mandatory in StrictMode

3.5 [ ] Add tests
    - Test module with FP in unreachable code = rejected (strict)
    - Test module with no FP = accepted
    - Test module with FP constants only (no ops) = configurable
```

### Acceptance Criteria
- [ ] No WASM module with FP opcodes passes strict validation
- [ ] No ELF with FP instructions passes strict validation
- [ ] Validation is mandatory in production (StrictMode=true)
- [ ] Clear error messages indicating which FP operation was found

---

## Fix 4: Fail-Hard Sandboxing

### Problem

`pkg/deterministicvm/seccomp.go:54-61` fails open:

```go
if !DetectSeccompAvailability() {
    if cfg.StrictSandbox {
        return ErrSeccompUnavailable  // Only fails if strict
    }
    logger.Printf("Seccomp not available, continuing without sandbox")
    return nil  // <-- CONTINUES WITHOUT SANDBOX!
}
```

And `vm.go:292-294` has namespaces commented out:

```go
// Note: Namespaces disabled for testing - enable in production
// Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNET | syscall.CLONE_NEWUTS,
```

### Solution

**In production mode, sandbox failures must be fatal.**

### Files to Modify

1. **`pkg/deterministicvm/seccomp.go`**
   - Remove the "continue without sandbox" path
   - If StrictSandbox=true and seccomp unavailable, return error
   - Add logging for security audit trail

2. **`pkg/deterministicvm/vm.go`**
   - Enable namespace isolation for StrictMode
   - Add CLONE_NEWPID, CLONE_NEWNET, CLONE_NEWUTS, CLONE_NEWIPC
   - Make cgroup application synchronous (before execution, not after)

3. **`pkg/deterministicvm/cgroup_manager.go`**
   - Apply cgroup BEFORE process starts (not after)
   - Fail execution if cgroup application fails in StrictMode

4. **Configuration**
   - Add `OCX_STRICT_MODE=true` environment variable
   - Default to strict in production, permissive in development

### Implementation Steps

```
4.1 [ ] Make seccomp mandatory in strict mode
    - Remove "continue without sandbox" code path
    - Return ErrSeccompUnavailable if unavailable AND strict

4.2 [ ] Enable namespace isolation
    - Uncomment CLONE_* flags in configureLinuxIsolation()
    - Add CLONE_NEWIPC for IPC isolation
    - Test that namespaces work on target deployment platforms

4.3 [ ] Fix cgroup timing race
    - Move applyCgroups() before cmd.Start()
    - Use cgroup v2 unified hierarchy
    - Fail if cgroup setup fails in StrictMode

4.4 [ ] Add production mode configuration
    - OCX_STRICT_MODE env var (default: true in production)
    - Log security configuration at startup
    - Warn if running in permissive mode

4.5 [ ] Add security tests
    - Test that network syscalls are blocked
    - Test that fork/exec is blocked
    - Test that ptrace is blocked
    - Test that file access outside workdir is blocked
```

### Acceptance Criteria
- [ ] Server refuses to start if seccomp unavailable in strict mode
- [ ] Process runs in isolated namespaces (PID, NET, UTS, IPC)
- [ ] Cgroups applied before execution begins
- [ ] Forbidden syscalls cause immediate process termination
- [ ] Security configuration logged at startup

---

## Execution Order

Recommended order (dependencies):

1. **Fix 4: Sandboxing** (independent, security critical)
2. **Fix 1: Gas Model** (independent, determinism critical)
3. **Fix 3: FP Validator** (independent, determinism critical)
4. **Fix 2: Nonce/Replay** (depends on understanding receipt flow)

Can parallelize: Fix 4 + Fix 1 in parallel, then Fix 3 + Fix 2.

---

## Testing Strategy

### Unit Tests
- [ ] `gas_model_test.go` - Same input = same gas across architectures
- [ ] `fp_validator_test.go` - FP detection coverage
- [ ] `seccomp_test.go` - Syscall blocking verification
- [ ] `replay_test.go` - Nonce uniqueness and rejection

### Integration Tests
- [ ] Full receipt generation on x86 and ARM64 - compare GasUsed
- [ ] Receipt verification with replay attack - should fail
- [ ] Execution with network syscall - should terminate
- [ ] Execution with FP WASM - should reject in strict mode

### Cross-Architecture Tests
- [ ] Generate receipt on x86_64, verify on ARM64
- [ ] Generate receipt on ARM64, verify on x86_64
- [ ] Same artifact + input = identical signed core

---

## Success Metrics

After Phase 1 completion:

1. **Determinism**: `sha256(signed_core_A) == sha256(signed_core_B)` for identical executions
2. **Replay Protection**: Second submission of same receipt = rejection
3. **FP Safety**: Zero FP modules pass strict validation
4. **Sandbox Enforcement**: Zero syscall violations in production

---

## Notes

- Don't break Railway deployment during changes
- Keep HostCycles for diagnostics (just exclude from signature)
- v1_1 receipt package is more mature - prefer it
- Test on Vultr (x86) and local ARM if available
