# TrustScore WASM Module Build System

.PHONY: all build test clean help install-tools build-wasm build-release optimize

# Default target
all: build

# Install required tools
install-tools:
	@echo "🔧 Installing WASM build tools..."
	rustup target add wasm32-unknown-unknown
	cargo install wasm-opt || echo "wasm-opt already installed"
	@echo "✅ Tools installed"

# Build WASM module (debug)
build:
	@echo "🔨 Building TrustScore WASM module (debug)..."
	cargo build --target wasm32-unknown-unknown
	@echo "✅ Debug build complete"
	@ls -lh target/wasm32-unknown-unknown/debug/trustscore_wasm.wasm

# Build WASM module (release)
build-release:
	@echo "🔨 Building TrustScore WASM module (release)..."
	cargo build --target wasm32-unknown-unknown --release
	@echo "✅ Release build complete"
	@ls -lh target/wasm32-unknown-unknown/release/trustscore_wasm.wasm

# Optimize WASM with wasm-opt
optimize: build-release
	@echo "⚡ Optimizing WASM binary..."
	mkdir -p ../../artifacts
	wasm-opt -Oz \
		target/wasm32-unknown-unknown/release/trustscore_wasm.wasm \
		-o ../../artifacts/trustscore.wasm
	@echo "✅ Optimized WASM created"
	@echo "📊 Size comparison:"
	@echo "   Release: $$(du -h target/wasm32-unknown-unknown/release/trustscore_wasm.wasm | cut -f1)"
	@echo "   Optimized: $$(du -h ../../artifacts/trustscore.wasm | cut -f1)"

# Build optimized WASM (convenience target)
build-wasm: optimize
	@echo "✅ Production WASM ready: ../../artifacts/trustscore.wasm"

# Run tests
test:
	@echo "🧪 Running tests..."
	cargo test --verbose
	@echo "✅ All tests passed"

# Run determinism tests specifically
test-determinism:
	@echo "🔍 Running determinism tests..."
	cargo test --test determinism_test -- --nocapture
	@echo "✅ Determinism tests passed"

# Run tests multiple times to verify determinism
test-determinism-stress:
	@echo "🔄 Running determinism stress tests (100 iterations)..."
	@for i in $$(seq 1 100); do \
		cargo test --test determinism_test --quiet || exit 1; \
	done
	@echo "✅ Determinism verified across 100 runs"

# Benchmark performance
bench:
	@echo "⚡ Running benchmarks..."
	cargo bench
	@echo "✅ Benchmarks complete"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	cargo clean
	rm -f ../../artifacts/trustscore.wasm
	@echo "✅ Clean complete"

# Check code quality
check:
	@echo "🔍 Running cargo check..."
	cargo check
	@echo "✅ Check complete"

# Format code
fmt:
	@echo "📝 Formatting code..."
	cargo fmt
	@echo "✅ Format complete"

# Lint code
lint:
	@echo "🔍 Running clippy..."
	cargo clippy -- -D warnings
	@echo "✅ Lint complete"

# Full CI pipeline
ci: check test lint build-wasm
	@echo "✅ CI pipeline complete"

# Show help
help:
	@echo "TrustScore WASM Module - Build Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  install-tools        Install Rust WASM toolchain"
	@echo "  build                Build debug WASM"
	@echo "  build-release        Build release WASM"
	@echo "  optimize             Optimize WASM with wasm-opt"
	@echo "  build-wasm           Build optimized production WASM (default)"
	@echo "  test                 Run all tests"
	@echo "  test-determinism     Run determinism tests"
	@echo "  test-determinism-stress  Stress test determinism (100 runs)"
	@echo "  bench                Run performance benchmarks"
	@echo "  clean                Remove build artifacts"
	@echo "  check                Run cargo check"
	@echo "  fmt                  Format code"
	@echo "  lint                 Run clippy linter"
	@echo "  ci                   Full CI pipeline"
	@echo "  help                 Show this help"
