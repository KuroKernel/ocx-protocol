#!/bin/bash
# Cross-architecture determinism verification script for OCX Protocol v1.0.0-rc.1

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_ROOT/build"
RESULTS_DIR="$PROJECT_ROOT/determinism-results"
TIMESTAMP=$(date -u +"%Y%m%d_%H%M%S")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary files..."
    rm -rf "$BUILD_DIR"
    if [ -d "$RESULTS_DIR" ]; then
        rm -rf "$RESULTS_DIR"
    fi
}

# Set up trap for cleanup
trap cleanup EXIT

# Create directories
mkdir -p "$BUILD_DIR" "$RESULTS_DIR"

log_info "Starting OCX Protocol v1.0.0-rc.1 Cross-Architecture Determinism Test"
log_info "Project root: $PROJECT_ROOT"
log_info "Build directory: $BUILD_DIR"
log_info "Results directory: $RESULTS_DIR"

# Test vectors for determinism verification
declare -A TEST_VECTORS=(
    ["test1"]="artifact_hex=48656c6c6f20576f726c64 input_hex=74657374 max_cycles=1000"
    ["test2"]="artifact_hex=48656c6c6f20576f726c64 input_hex=74657374 max_cycles=5000"
    ["test3"]="artifact_hex=48656c6c6f20576f726c64 input_hex=646966666572656e74 max_cycles=1000"
    ["test4"]="artifact_hex=646966666572656e74 input_hex=74657374 max_cycles=1000"
    ["test5"]="artifact_hex=48656c6c6f20576f726c64 input_hex=74657374 max_cycles=10000"
)

# Architectures to test
ARCHITECTURES=("linux/amd64" "linux/arm64")

# Function to build CLI for specific architecture
build_cli() {
    local arch="$1"
    local os="${arch%%/*}"
    local arch_name="${arch##*/}"
    local output_name="minimal-cli-${os}-${arch_name}"
    
    log_info "Building CLI for $arch..."
    
    # Set environment variables for cross-compilation
    export GOOS="$os"
    export GOARCH="$arch_name"
    export CGO_ENABLED=0
    
    # Build with version information
    go build \
        -ldflags "-X ocx.local/internal/version.SpecHash=abc123def456 -X ocx.local/internal/version.Build=$(date -u +"%Y-%m-%dT%H:%M:%SZ") -X ocx.local/internal/version.GitCommit=abc123 -X ocx.local/internal/version.GitBranch=main" \
        -o "$BUILD_DIR/$output_name" \
        "$PROJECT_ROOT/cmd/minimal-cli"
    
    if [ $? -eq 0 ]; then
        log_success "Built $output_name successfully"
    else
        log_error "Failed to build $output_name"
        return 1
    fi
}

# Function to execute test vector and capture receipt hash
execute_test() {
    local cli_path="$1"
    local test_name="$2"
    local test_params="$3"
    local output_file="$4"
    
    log_info "Executing $test_name with $cli_path..."
    
    # Parse test parameters
    local artifact_hex input_hex max_cycles
    eval "$test_params"
    
    # Execute CLI and capture output
    local result
    result=$("$cli_path" -command execute \
        -server "http://localhost:9000" \
        -artifact "$(echo "$artifact_hex" | xxd -r -p)" \
        -input "$(echo "$input_hex" | xxd -r -p)" \
        -max-cycles "$max_cycles" \
        -lease-id "test-$test_name" 2>&1)
    
    if [ $? -eq 0 ]; then
        # Extract receipt hash from JSON output
        local receipt_hash
        receipt_hash=$(echo "$result" | grep -o '"receipt_hash":"[^"]*"' | cut -d'"' -f4)
        
        if [ -n "$receipt_hash" ]; then
            echo "$receipt_hash" > "$output_file"
            log_success "Captured receipt hash for $test_name: ${receipt_hash:0:16}..."
            return 0
        else
            log_error "Failed to extract receipt hash from $test_name"
            return 1
        fi
    else
        log_error "Failed to execute $test_name: $result"
        return 1
    fi
}

# Function to compare receipt hashes
compare_hashes() {
    local test_name="$1"
    local arch1="$2"
    local arch2="$3"
    local hash1_file="$4"
    local hash2_file="$5"
    
    if [ ! -f "$hash1_file" ] || [ ! -f "$hash2_file" ]; then
        log_error "Missing hash files for $test_name"
        return 1
    fi
    
    local hash1 hash2
    hash1=$(cat "$hash1_file")
    hash2=$(cat "$hash2_file")
    
    if [ "$hash1" = "$hash2" ]; then
        log_success "Hashes match for $test_name: $hash1"
        return 0
    else
        log_error "Hash mismatch for $test_name:"
        log_error "  $arch1: $hash1"
        log_error "  $arch2: $hash2"
        return 1
    fi
}

# Main execution
main() {
    log_info "Building CLI for all architectures..."
    
    # Build CLI for each architecture
    for arch in "${ARCHITECTURES[@]}"; do
        build_cli "$arch" || exit 1
    done
    
    log_info "Starting determinism verification..."
    
    # Start test server
    log_info "Starting test server..."
    cd "$PROJECT_ROOT"
    ./test-port > /dev/null 2>&1 &
    local server_pid=$!
    
    # Wait for server to start
    sleep 3
    
    # Test if server is running
    if ! curl -s http://localhost:9000/health > /dev/null; then
        log_error "Test server failed to start"
        kill $server_pid 2>/dev/null || true
        exit 1
    fi
    
    log_success "Test server started (PID: $server_pid)"
    
    # Initialize results
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    
    # Execute tests for each architecture
    for arch in "${ARCHITECTURES[@]}"; do
        local os="${arch%%/*}"
        local arch_name="${arch##*/}"
        local cli_name="minimal-cli-${os}-${arch_name}"
        local cli_path="$BUILD_DIR/$cli_name"
        
        log_info "Testing architecture: $arch"
        
        for test_name in "${!TEST_VECTORS[@]}"; do
            local test_params="${TEST_VECTORS[$test_name]}"
            local hash_file="$RESULTS_DIR/${test_name}-${arch_name}.hash"
            
            if execute_test "$cli_path" "$test_name" "$test_params" "$hash_file"; then
                ((total_tests++))
            else
                ((failed_tests++))
            fi
        done
    done
    
    # Compare hashes between architectures
    log_info "Comparing receipt hashes between architectures..."
    
    local arch1="${ARCHITECTURES[0]##*/}"
    local arch2="${ARCHITECTURES[1]##*/}"
    
    for test_name in "${!TEST_VECTORS[@]}"; do
        local hash1_file="$RESULTS_DIR/${test_name}-${arch1}.hash"
        local hash2_file="$RESULTS_DIR/${test_name}-${arch2}.hash"
        
        if compare_hashes "$test_name" "$arch1" "$arch2" "$hash1_file" "$hash2_file"; then
            ((passed_tests++))
        else
            ((failed_tests++))
        fi
    done
    
    # Stop test server
    log_info "Stopping test server..."
    kill $server_pid 2>/dev/null || true
    wait $server_pid 2>/dev/null || true
    
    # Generate results summary
    local results_file="$RESULTS_DIR/determinism-results-${TIMESTAMP}.json"
    cat > "$results_file" << EOF
{
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "version": "OCX Protocol v1.0.0-rc.1",
    "architectures": ${ARCHITECTURES[@]},
    "total_tests": $total_tests,
    "passed_tests": $passed_tests,
    "failed_tests": $failed_tests,
    "success_rate": "$(( passed_tests * 100 / total_tests ))%",
    "deterministic": $([ $failed_tests -eq 0 ] && echo "true" || echo "false")
}
EOF
    
    # Print final results
    log_info "=== DETERMINISM TEST RESULTS ==="
    log_info "Total tests: $total_tests"
    log_info "Passed: $passed_tests"
    log_info "Failed: $failed_tests"
    log_info "Success rate: $(( passed_tests * 100 / total_tests ))%"
    
    if [ $failed_tests -eq 0 ]; then
        log_success "All tests passed! OCX Protocol is deterministic across architectures."
        log_success "Results saved to: $results_file"
        exit 0
    else
        log_error "Some tests failed. OCX Protocol is not deterministic."
        log_error "Results saved to: $results_file"
        exit 1
    fi
}

# Run main function
main "$@"
