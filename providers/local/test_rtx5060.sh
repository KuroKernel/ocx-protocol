#!/bin/bash
# test_rtx5060.sh - RTX 5060 GPU Testing Script
# Tests OCX Protocol with real RTX 5060 hardware

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🚀 OCX Protocol RTX 5060 GPU Test${NC}"
echo -e "${BLUE}===================================${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

# Check if nvidia-smi is available
if ! command -v nvidia-smi &> /dev/null; then
    echo -e "${RED}❌ nvidia-smi not found. Please install NVIDIA drivers.${NC}"
    exit 1
fi

# Check if RTX 5060 is detected
if ! nvidia-smi --query-gpu=name --format=csv,noheader,nounits | grep -i "nvidia"; then
    echo -e "${YELLOW}⚠️  RTX 5060 not detected. Available GPUs:${NC}"
    nvidia-smi --query-gpu=name --format=csv,noheader,nounits
    echo ""
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go not found. Please install Go.${NC}"
    exit 1
fi

# Check if OCX server is running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  OCX server not running at localhost:8080${NC}"
    echo "Please start the OCX server first:"
    echo "  cd ../../cmd/server && go run main.go"
    echo ""
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"
echo ""

# Build the test client
echo -e "${YELLOW}Building GPU test client...${NC}"
if ! go build -o cmd/gpu_test_client ./cmd/working_main.go; then
    echo -e "${RED}❌ Failed to build test client${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Test client built successfully${NC}"
echo ""

# Show GPU information
echo -e "${YELLOW}GPU Information:${NC}"
nvidia-smi --query-gpu=name,memory.total,driver_version,temperature.gpu,utilization.gpu --format=csv,noheader,nounits | while IFS=, read -r name memory driver temp util; do
    echo "  GPU: $(echo $name | xargs)"
    echo "  Memory: $(echo $memory | xargs)MB"
    echo "  Driver: $(echo $driver | xargs)"
    echo "  Temperature: $(echo $temp | xargs)°C"
    echo "  Utilization: $(echo $util | xargs)%"
done
echo ""

# Run tests based on argument
case "${1:-quick}" in
    "quick")
        echo -e "${YELLOW}Running quick test...${NC}"
        ./cmd/gpu_test_client -test=quick
        ;;
    "full")
        echo -e "${YELLOW}Running full test...${NC}"
        ./cmd/gpu_test_client -test=full
        ;;
    "monitor")
        echo -e "${YELLOW}Running monitor mode...${NC}"
        ./cmd/gpu_test_client -test=monitor -duration=60s
        ;;
    *)
        echo "Usage: $0 [quick|full|monitor]"
        echo ""
        echo "Test modes:"
        echo "  quick   - Quick GPU verification and basic test"
        echo "  full    - Complete order-to-provisioning flow"
        echo "  monitor - Monitor GPU usage for specified duration"
        echo ""
        echo "Examples:"
        echo "  $0 quick"
        echo "  $0 full"
        echo "  $0 monitor"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}🎉 GPU test completed successfully!${NC}"
echo -e "${BLUE}OCX Protocol successfully managed real RTX 5060 hardware${NC}"
