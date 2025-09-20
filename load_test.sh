#!/bin/bash

# OCX Protocol Load Test - Target SLO: p99 < 20ms at 200 RPS
# Usage: ./load_test.sh [RPS] [duration_seconds]

RPS=${1:-200}
DURATION=${2:-60}
BASE_URL="http://localhost:8080"
RESULTS_DIR="load_test_results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo "🚀 OCX Protocol Load Test"
echo "Target: ${RPS} RPS for ${DURATION} seconds"
echo "SLO: p99 < 20ms, 0% 5xx errors"
echo ""

# Create results directory
mkdir -p "$RESULTS_DIR"

# Test data
ARTIFACT="dGVzdA=="
INPUT="aW5wdXQ="
MAX_CYCLES=1000

echo "📊 Starting load test..."

# Function to make a single request and measure time
make_request() {
    local id=$1
    local start_time=$(date +%s.%N)
    
    response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/v1/execute" \
        -H "Content-Type: application/json" \
        -H "Idempotency-Key: load-test-$id" \
        -d "{\"artifact\":\"$ARTIFACT\",\"input\":\"$INPUT\",\"max_cycles\":$MAX_CYCLES}" \
        -o /tmp/response_$id.json)
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc -l)
    local http_code=${response: -3}
    
    echo "$duration,$http_code" >> "$RESULTS_DIR/load_test_$TIMESTAMP.csv"
}

# Calculate requests per second interval
interval=$(echo "scale=6; 1 / $RPS" | bc -l)

echo "⏱️  Sending $RPS requests per second for $DURATION seconds..."
echo "📝 Results will be saved to: $RESULTS_DIR/load_test_$TIMESTAMP.csv"

# Start load test
request_id=1
start_time=$(date +%s)
end_time=$((start_time + DURATION))

while [ $(date +%s) -lt $end_time ]; do
    make_request $request_id &
    request_id=$((request_id + 1))
    sleep $interval
done

# Wait for all requests to complete
wait

echo ""
echo "✅ Load test completed!"
echo "📊 Analyzing results..."

# Analyze results
if [ -f "$RESULTS_DIR/load_test_$TIMESTAMP.csv" ]; then
    # Calculate statistics
    total_requests=$(wc -l < "$RESULTS_DIR/load_test_$TIMESTAMP.csv")
    successful_requests=$(awk -F',' '$2 == "200" {count++} END {print count+0}' "$RESULTS_DIR/load_test_$TIMESTAMP.csv")
    error_requests=$((total_requests - successful_requests))
    error_rate=$(echo "scale=2; $error_requests * 100 / $total_requests" | bc -l)
    
    # Calculate latency percentiles
    p50=$(awk -F',' '$2 == "200" {print $1}' "$RESULTS_DIR/load_test_$TIMESTAMP.csv" | sort -n | awk 'NR==int(NR*0.5)')
    p95=$(awk -F',' '$2 == "200" {print $1}' "$RESULTS_DIR/load_test_$TIMESTAMP.csv" | sort -n | awk 'NR==int(NR*0.95)')
    p99=$(awk -F',' '$2 == "200" {print $1}' "$RESULTS_DIR/load_test_$TIMESTAMP.csv" | sort -n | awk 'NR==int(NR*0.99)')
    
    # Convert to milliseconds
    p50_ms=$(echo "$p50 * 1000" | bc -l)
    p95_ms=$(echo "$p95 * 1000" | bc -l)
    p99_ms=$(echo "$p99 * 1000" | bc -l)
    
    echo ""
    echo "📈 LOAD TEST RESULTS:"
    echo "===================="
    echo "Total Requests: $total_requests"
    echo "Successful: $successful_requests"
    echo "Errors: $error_requests"
    echo "Error Rate: ${error_rate}%"
    echo ""
    echo "LATENCY PERCENTILES:"
    echo "P50: ${p50_ms}ms"
    echo "P95: ${p95_ms}ms"
    echo "P99: ${p99_ms}ms"
    echo ""
    
    # Check SLO compliance
    p99_check=$(echo "$p99_ms < 20" | bc -l)
    error_check=$(echo "$error_rate == 0" | bc -l)
    
    if [ "$p99_check" -eq 1 ] && [ "$error_check" -eq 1 ]; then
        echo "✅ SLO COMPLIANCE: PASSED"
        echo "   - P99 latency: ${p99_ms}ms < 20ms ✓"
        echo "   - Error rate: ${error_rate}% = 0% ✓"
    else
        echo "❌ SLO COMPLIANCE: FAILED"
        echo "   - P99 latency: ${p99_ms}ms (target: <20ms)"
        echo "   - Error rate: ${error_rate}% (target: 0%)"
    fi
    
    echo ""
    echo "📁 Detailed results saved to: $RESULTS_DIR/load_test_$TIMESTAMP.csv"
else
    echo "❌ No results file found. Check if server is running."
fi
