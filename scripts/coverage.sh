#!/bin/bash
# coverage.sh - Comprehensive coverage analysis script

set -e

echo "📊 OCX Protocol Coverage Analysis"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_THRESHOLD=80
COVERAGE_DIR="tests/coverage"
COVERAGE_FILE="$COVERAGE_DIR/coverage.out"
COVERAGE_HTML="$COVERAGE_DIR/coverage.html"

# Create coverage directory
mkdir -p "$COVERAGE_DIR"

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install required tools if not present
install_tools() {
    if ! command_exists go; then
        print_status $RED "❌ Go is not installed"
        exit 1
    fi
    
    if ! command_exists bc; then
        print_status $YELLOW "⚠️  Installing bc for calculations..."
        sudo apt-get update && sudo apt-get install -y bc
    fi
}

# Run tests with coverage
run_coverage() {
    print_status $YELLOW "🧪 Running tests with coverage..."
    
    # Run tests with coverage
    go test -v -coverprofile="$COVERAGE_FILE" -covermode=atomic ./...
    
    if [ ! -f "$COVERAGE_FILE" ]; then
        print_status $RED "❌ Coverage file not generated"
        exit 1
    fi
    
    print_status $GREEN "✅ Coverage data collected"
}

# Generate HTML coverage report
generate_html_report() {
    print_status $YELLOW "📄 Generating HTML coverage report..."
    
    go tool cover -html="$COVERAGE_FILE" -o="$COVERAGE_HTML"
    
    if [ -f "$COVERAGE_HTML" ]; then
        print_status $GREEN "✅ HTML report generated: $COVERAGE_HTML"
    else
        print_status $RED "❌ Failed to generate HTML report"
        exit 1
    fi
}

# Analyze coverage by package
analyze_package_coverage() {
    print_status $YELLOW "📦 Analyzing coverage by package..."
    
    echo ""
    echo "Package Coverage:"
    echo "================="
    
    # Get coverage by package
    go tool cover -func="$COVERAGE_FILE" | grep -E "^\w+.*\s+[0-9]+\.[0-9]+%" | while read line; do
        package=$(echo "$line" | awk '{print $1}')
        coverage=$(echo "$line" | awk '{print $3}' | sed 's/%//')
        
        if (( $(echo "$coverage >= $COVERAGE_THRESHOLD" | bc -l) )); then
            print_status $GREEN "✅ $package: ${coverage}%"
        else
            print_status $RED "❌ $package: ${coverage}% (below threshold)"
        fi
    done
}

# Check overall coverage threshold
check_threshold() {
    print_status $YELLOW "🎯 Checking coverage threshold..."
    
    # Get total coverage
    total_coverage=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}' | sed 's/%//')
    
    echo ""
    echo "Overall Coverage: ${total_coverage}%"
    echo "Threshold: ${COVERAGE_THRESHOLD}%"
    
    if (( $(echo "$total_coverage >= $COVERAGE_THRESHOLD" | bc -l) )); then
        print_status $GREEN "✅ Coverage threshold met!"
        return 0
    else
        print_status $RED "❌ Coverage threshold not met!"
        return 1
    fi
}

# Generate coverage report
generate_report() {
    print_status $YELLOW "📋 Generating coverage report..."
    
    local report_file="$COVERAGE_DIR/coverage-report.md"
    
    cat > "$report_file" << EOF
# OCX Protocol Coverage Report

Generated: $(date)

## Summary

- **Overall Coverage**: $(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}')
- **Threshold**: ${COVERAGE_THRESHOLD}%
- **Status**: $([ $? -eq 0 ] && echo "✅ PASSED" || echo "❌ FAILED")

## Package Coverage

\`\`\`
$(go tool cover -func="$COVERAGE_FILE" | grep -E "^\w+.*\s+[0-9]+\.[0-9]+%")
\`\`\`

## Files with Low Coverage

$(go tool cover -func="$COVERAGE_FILE" | grep -E "^\w+.*\s+[0-9]+\.[0-9]+%" | awk '$3+0 < '$COVERAGE_THRESHOLD' {print $1 " " $3}')

## Recommendations

1. Focus on files with coverage below ${COVERAGE_THRESHOLD}%
2. Add more test cases for critical functions
3. Consider refactoring complex functions for better testability

