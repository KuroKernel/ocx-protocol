# Makefile for OCX Reputation Aggregator WASM Module

.PHONY: all build test clean optimize verify golden-vectors

# Tool paths (use system-installed tools)
WAT2WASM = wat2wasm
WASM2WAT = wasm2wat
WASM_OPT = wasm-opt
WASM_VALIDATE = wasm-validate

# Build artifacts
WAT_SOURCE = aggregator.wat
WASM_OUTPUT = aggregator.wasm
WASM_OPTIMIZED = aggregator.opt.wasm

# Build flags
WAT2WASM_FLAGS = --debug-names
WASM_OPT_FLAGS = -O3 --strip-debug

all: build optimize

# ============================================================
# BUILD TARGETS
# ============================================================

build: $(WASM_OUTPUT)
	@echo "✓ Built reputation aggregator WASM module"

$(WASM_OUTPUT): $(WAT_SOURCE)
	$(WAT2WASM) $(WAT2WASM_FLAGS) -o $@ $<
	@echo "  → Generated: $@"
	@ls -lh $@

optimize: $(WASM_OPTIMIZED)

$(WASM_OPTIMIZED): $(WASM_OUTPUT)
	$(WASM_OPT) $(WASM_OPT_FLAGS) -o $@ $<
	@echo "  → Optimized: $@"
	@ls -lh $@
	@echo "  → Size reduction: $$(echo "scale=1; 100 - (100 * $$(stat -c%s $@) / $$(stat -c%s $<))" | bc)%"

# ============================================================
# VALIDATION TARGETS
# ============================================================

validate: $(WASM_OUTPUT)
	@echo "Validating WASM module..."
	$(WASM_VALIDATE) $<
	@echo "✓ WASM module is valid"

disassemble: $(WASM_OUTPUT)
	$(WASM2WAT) $< -o aggregator.disasm.wat
	@echo "  → Disassembled to: aggregator.disasm.wat"

inspect: $(WASM_OUTPUT)
	@echo "=== WASM Module Inspection ==="
	@echo "File size: $$(stat -c%s $<) bytes"
	@echo ""
	@echo "Imports:"
	@wasm-objdump -x $< | grep -A10 "Import\[" || true
	@echo ""
	@echo "Exports:"
	@wasm-objdump -x $< | grep -A10 "Export\[" || true
	@echo ""
	@echo "Functions:"
	@wasm-objdump -x $< | grep -A20 "Function\[" || true

# ============================================================
# TESTING TARGETS
# ============================================================

test: build
	@echo "Running determinism tests..."
	@go test -v ../../../pkg/deterministicvm/reputation_test.go -run TestReputationDeterminism
	@echo "✓ Determinism tests passed"

test-gas: build
	@echo "Testing gas usage..."
	@go test -v ../../../pkg/deterministicvm/reputation_test.go -run TestReputationGasUsage
	@echo "✓ Gas usage within target (238 ± 5%)"

test-cross-platform: build
	@echo "Cross-platform determinism test..."
	@echo "  Platform: $$(uname -sm)"
	@go run test_determinism.go
	@echo "✓ 1000 iterations produced identical output"

golden-vectors: build
	@echo "Running golden vector tests..."
	@go test -v ../../../pkg/deterministicvm/reputation_test.go -run TestReputationGoldenVectors
	@echo "✓ All golden vectors passed"

benchmark: build optimize
	@echo "Benchmarking execution performance..."
	@go test -bench=. -benchmem ../../../pkg/deterministicvm/reputation_bench_test.go
	@echo "Target: <5ms execution, <1ms verification"

# ============================================================
# INSTALLATION TARGETS
# ============================================================

install-tools:
	@echo "Installing WABT tools..."
	@if ! command -v wat2wasm &> /dev/null; then \
		echo "Installing wabt..."; \
		cd /tmp && \
		git clone --recursive https://github.com/WebAssembly/wabt && \
		cd wabt && \
		mkdir -p build && cd build && \
		cmake .. && \
		cmake --build . && \
		sudo cmake --install .; \
	else \
		echo "✓ wat2wasm already installed"; \
	fi
	@if ! command -v wasm-opt &> /dev/null; then \
		echo "Installing binaryen (wasm-opt)..."; \
		cd /tmp && \
		git clone https://github.com/WebAssembly/binaryen && \
		cd binaryen && \
		cmake . && make && \
		sudo cmake --install .; \
	else \
		echo "✓ wasm-opt already installed"; \
	fi

# ============================================================
# INTEGRATION WITH OCX BUILD SYSTEM
# ============================================================

artifacts: optimize
	@echo "Copying artifacts to OCX artifacts directory..."
	@mkdir -p ../../../artifacts
	@cp $(WASM_OPTIMIZED) ../../../artifacts/reputation-aggregator.wasm
	@echo "✓ Installed: artifacts/reputation-aggregator.wasm"
	@sha256sum ../../../artifacts/reputation-aggregator.wasm

# ============================================================
# CLEANUP
# ============================================================

clean:
	rm -f $(WASM_OUTPUT) $(WASM_OPTIMIZED) aggregator.disasm.wat
	@echo "✓ Cleaned build artifacts"

distclean: clean
	rm -rf ../../../artifacts/reputation-aggregator.wasm
	@echo "✓ Removed all artifacts"

# ============================================================
# HELP
# ============================================================

help:
	@echo "OCX Reputation Aggregator Build System"
	@echo ""
	@echo "Targets:"
	@echo "  make build           - Compile WAT to WASM"
	@echo "  make optimize        - Optimize WASM with wasm-opt"
	@echo "  make validate        - Validate WASM module"
	@echo "  make test            - Run determinism tests"
	@echo "  make test-gas        - Test gas usage"
	@echo "  make golden-vectors  - Run golden vector tests"
	@echo "  make benchmark       - Benchmark performance"
	@echo "  make artifacts       - Copy to OCX artifacts directory"
	@echo "  make inspect         - Inspect WASM module structure"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make install-tools   - Install WABT and binaryen"
