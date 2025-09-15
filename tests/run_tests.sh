#!/bin/bash
# run_tests.sh - Comprehensive Test Runner for OCX Protocol
# Runs all test suites with proper reporting and documentation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
BASE_URL="http://localhost:8080"
TEST_RESULTS_DIR="test_results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create results directory
mkdir -p "$TEST_RESULTS_DIR"

echo -e "${BLUE}OCX Protocol Comprehensive Test Suite${NC}"
echo -e "${BLUE}=====================================${NC}"
echo "Timestamp: $TIMESTAMP"
echo "Base URL: $BASE_URL"
echo ""

# Function to run test suite
run_test_suite() {
    local suite_name="$1"
    local test_dir="$2"
    local description="$3"
    
    echo -e "${YELLOW}Running $suite_name Tests...${NC}"
    echo "Description: $description"
    echo "Directory: $test_dir"
    echo ""
    
    # Change to test directory
    cd "$test_dir"
    
    # Run tests with verbose output
    if go test -v -timeout 30m -race -coverprofile=coverage.out . 2>&1 | tee "../$TEST_RESULTS_DIR/${suite_name}_${TIMESTAMP}.log"; then
        echo -e "${GREEN}✓ $suite_name tests PASSED${NC}"
        return 0
    else
        echo -e "${RED}✗ $suite_name tests FAILED${NC}"
        return 1
    fi
}

# Function to check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking Prerequisites...${NC}"
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is not installed${NC}"
        exit 1
    fi
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Error: Docker is not installed${NC}"
        exit 1
    fi
    
    # Check if server is running
    if ! curl -s "$BASE_URL/health" > /dev/null; then
        echo -e "${YELLOW}Warning: Server not running at $BASE_URL${NC}"
        echo "Please start the OCX server before running tests"
        echo ""
    fi
    
    echo -e "${GREEN}✓ Prerequisites check passed${NC}"
    echo ""
}

# Function to generate test report
generate_report() {
    local total_tests=$1
    local passed_tests=$2
    local failed_tests=$3
    
    echo -e "${BLUE}Test Report${NC}"
    echo -e "${BLUE}===========${NC}"
    echo "Total Test Suites: $total_tests"
    echo -e "Passed: ${GREEN}$passed_tests${NC}"
    echo -e "Failed: ${RED}$failed_tests${NC}"
    echo ""
    
    # Generate HTML report
    cat > "$TEST_RESULTS_DIR/test_report_${TIMESTAMP}.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>OCX Protocol Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .summary { margin: 20px 0; }
        .passed { color: green; }
        .failed { color: red; }
        .test-suite { margin: 10px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>OCX Protocol Test Report</h1>
        <p>Generated: $(date)</p>
        <p>Total Suites: $total_tests | Passed: <span class="passed">$passed_tests</span> | Failed: <span class="failed">$failed_tests</span></p>
    </div>
    
    <div class="summary">
        <h2>Test Summary</h2>
        <p>This report covers comprehensive testing of the OCX Protocol including:</p>
        <ul>
            <li>Load Testing - 1000+ concurrent API calls, database performance, HMAC authentication</li>
            <li>Integration Testing - Provider failures, partial provisioning, network interruptions, dispute resolution</li>
            <li>Security Testing - Penetration testing, cryptographic validation, SQL injection, rate limiting</li>
            <li>Business Logic Testing - Matching algorithm, double-entry ledger, fee calculation, idempotency</li>
            <li>Deployment Testing - Docker reliability, database migrations, backup/recovery, monitoring</li>
            <li>User Experience Testing - CLI reliability, API response times, error messages, documentation</li>
        </ul>
    </div>
    
    <div class="test-suite">
        <h3>Load Testing</h3>
        <p>Tests system performance under high load conditions</p>
    </div>
    
    <div class="test-suite">
        <h3>Integration Testing</h3>
        <p>Tests system behavior under various failure scenarios</p>
    </div>
    
    <div class="test-suite">
        <h3>Security Testing</h3>
        <p>Tests system security and vulnerability resistance</p>
    </div>
    
    <div class="test-suite">
        <h3>Business Logic Testing</h3>
        <p>Tests core business logic and financial calculations</p>
    </div>
    
    <div class="test-suite">
        <h3>Deployment Testing</h3>
        <p>Tests deployment and operational procedures</p>
    </div>
    
    <div class="test-suite">
        <h3>User Experience Testing</h3>
        <p>Tests user-facing functionality and experience</p>
    </div>
</body>
</html>
