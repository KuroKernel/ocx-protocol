#!/bin/bash

# OCX Protocol Load Testing Script
# This script runs comprehensive load tests using k6

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL=${API_BASE_URL:-"http://localhost:8080"}
API_KEY=${API_KEY:-"supersecretkey"}
RESULTS_DIR="./ops/loadtest-results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create results directory
mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}🚀 OCX Protocol Load Testing Suite${NC}"
echo "=================================="
echo "API Base URL: $API_BASE_URL"
echo "Results Directory: $RESULTS_DIR"
echo "Timestamp: $TIMESTAMP"
echo ""

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}❌ k6 is not installed. Please install k6 first:${NC}"
    echo "   https://k6.io/docs/getting-started/installation/"
    exit 1
fi

# Check if API is accessible
echo -e "${YELLOW}🔍 Checking API accessibility...${NC}"
if ! curl -s -f "$API_BASE_URL/health" > /dev/null; then
    echo -e "${RED}❌ API is not accessible at $API_BASE_URL${NC}"
    echo "   Please ensure the OCX server is running"
    exit 1
fi
echo -e "${GREEN}✅ API is accessible${NC}"
echo ""

# Function to run a load test
run_loadtest() {
    local test_name="$1"
    local test_file="$2"
    local description="$3"
    
    echo -e "${BLUE}🧪 Running $test_name${NC}"
    echo "Description: $description"
    echo "Test File: $test_file"
    echo ""
    
    # Run the test
    if k6 run \
        --env API_BASE_URL="$API_BASE_URL" \
        --env API_KEY="$API_KEY" \
        --out json="$RESULTS_DIR/${test_name}_${TIMESTAMP}.json" \
        "$test_file"; then
        echo -e "${GREEN}✅ $test_name completed successfully${NC}"
    else
        echo -e "${RED}❌ $test_name failed${NC}"
        return 1
    fi
    echo ""
}

# Run load tests
echo -e "${YELLOW}📊 Starting Load Tests...${NC}"
echo ""

# 1. Standard Load Test
run_loadtest "loadtest" "ops/loadtest.js" "Standard load test with gradual ramp-up to 100 users"

# 2. Stress Test
run_loadtest "stress-test" "ops/loadtest-stress.js" "Stress test with ramp-up to 500 users"

# Generate summary report
echo -e "${BLUE}📋 Generating Summary Report...${NC}"
cat > "$RESULTS_DIR/summary_${TIMESTAMP}.md" << EOF
# OCX Protocol Load Test Results

**Test Date:** $(date)
**API Base URL:** $API_BASE_URL
**Test Duration:** $(date -d @$(($(date +%s) - $(date -d "$TIMESTAMP" +%s))) -u +%H:%M:%S)

## Test Results

### 1. Standard Load Test
- **File:** loadtest_${TIMESTAMP}.json
- **Description:** Gradual ramp-up to 100 users over 21 minutes
- **Thresholds:** 95th percentile < 200ms, Error rate < 1%

### 2. Stress Test
- **File:** stress-test_${TIMESTAMP}.json
- **Description:** Ramp-up to 500 users over 12 minutes
- **Thresholds:** 95th percentile < 1000ms, Error rate < 5%

## Performance Analysis

To analyze the results in detail:

1. **View JSON Results:**
   \`\`\`bash
   cat $RESULTS_DIR/loadtest_${TIMESTAMP}.json | jq '.'
   \`\`\`

2. **Generate HTML Report (if k6-to-junit is installed):**
   \`\`\`bash
   k6-to-junit $RESULTS_DIR/loadtest_${TIMESTAMP}.json > $RESULTS_DIR/loadtest_${TIMESTAMP}.xml
   \`\`\`

3. **Import to Grafana:**
   - Use the JSON files to import metrics into Grafana
   - Create dashboards for performance monitoring

## Recommendations

Based on the test results:

- **If error rate > 1%:** Check server resources and database performance
- **If 95th percentile > 200ms:** Optimize database queries and caching
- **If stress test fails:** Consider horizontal scaling or load balancing

## Next Steps

1. Review the detailed JSON results
2. Identify performance bottlenecks
3. Optimize based on findings
4. Re-run tests to validate improvements
EOF

echo -e "${GREEN}✅ Load testing completed successfully!${NC}"
echo ""
echo -e "${BLUE}📁 Results saved to: $RESULTS_DIR${NC}"
echo -e "${BLUE}📋 Summary report: $RESULTS_DIR/summary_${TIMESTAMP}.md${NC}"
echo ""
echo -e "${YELLOW}💡 To view results:${NC}"
echo "   cat $RESULTS_DIR/summary_${TIMESTAMP}.md"
echo "   cat $RESULTS_DIR/loadtest_${TIMESTAMP}.json | jq '.'"
echo ""
echo -e "${GREEN}🎉 Load testing suite completed!${NC}"
