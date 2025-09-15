# RTX 5060 GPU Testing for OCX Protocol

This directory contains a complete local GPU provider implementation that exposes your RTX 5060 as available compute capacity for OCX Protocol testing.

## 🎯 What This Does

- **Exposes your RTX 5060** as a compute provider in the OCX Protocol
- **Tests complete flow**: Order → Matching → Provisioning → Usage → Settlement
- **Real hardware validation**: Uses actual GPU hardware, not mocks
- **Performance monitoring**: Tracks GPU utilization, temperature, memory usage
- **End-to-end testing**: Validates the entire OCX Protocol stack

## 🚀 Quick Start

### Prerequisites
```bash
# NVIDIA drivers with nvidia-smi
nvidia-smi

# Go 1.22+
go version

# OCX server running
cd ../../cmd/server && go run main.go
```

### Run Tests
```bash
# Quick test (GPU verification)
./test_rtx5060.sh quick

# Full test (complete flow)
./test_rtx5060.sh full

# Monitor mode (real-time GPU stats)
./test_rtx5060.sh monitor
```

## 📊 Test Scenarios

### 1. Quick Test (`quick`)
- ✅ Verifies RTX 5060 is detected and healthy
- ✅ Checks GPU availability for provisioning
- ✅ Tests basic monitoring functionality
- ✅ Validates provider initialization

### 2. Full Test (`full`)
- ✅ Creates GPU offer ($2.50/hour)
- ✅ Places order for 1 GPU, 1 hour
- ✅ Waits for order matching
- ✅ Provisions RTX 5060 with real access details
- ✅ Monitors GPU usage during lease
- ✅ Runs test CUDA workload
- ✅ Releases GPU lease
- ✅ Verifies settlement ($2.50 → $2.25 provider + $0.25 protocol fee)

### 3. Monitor Mode (`monitor`)
- ✅ Real-time GPU metrics (utilization, temperature, memory)
- ✅ Process monitoring (what's using the GPU)
- ✅ Health status checking
- ✅ Performance scoring
- ✅ Continuous monitoring for specified duration

## 🔧 Technical Details

### GPU Provider Features
- **Hardware Detection**: Automatically detects RTX 5060
- **Health Monitoring**: Temperature, utilization, memory checks
- **Provisioning**: Creates real SSH access to your machine
- **Usage Tracking**: Monitors GPU usage during lease
- **Settlement**: Calculates real costs based on usage time

### Access Details Provided
When GPU is provisioned, you get:
```json
{
  "instance_id": "local-rtx5060-lease_123",
  "address": "192.168.1.100:22",
  "ssh_user": "kurokernel",
  "gpu_device": "0",
  "cuda_visible": "0",
  "nvidia_visible": "0",
  "hostname": "pop-os-desktop",
  "os": "Linux",
  "arch": "x86_64"
}
```

### GPU Metrics Tracked
- **Utilization**: GPU compute usage percentage
- **Temperature**: GPU temperature in Celsius
- **Memory**: Used/total VRAM in MB
- **Power**: Power consumption in watts
- **Clocks**: Graphics and memory clock speeds
- **Fan Speed**: GPU fan speed percentage
- **Processes**: Running GPU processes with resource usage

## 🎮 Test Workloads

### CUDA Test Program
The full test includes a simple CUDA program that:
- Allocates memory on GPU
- Runs vector addition kernel
- Validates GPU compute functionality
- Measures performance

### Real Usage Simulation
- Creates actual SSH access to your machine
- Provides GPU device information
- Tracks real resource usage
- Calculates actual costs

## 📈 Performance Validation

### What Gets Tested
- **GPU Detection**: RTX 5060 properly identified
- **Health Checks**: Temperature, utilization, memory limits
- **Provisioning Speed**: Time to create lease and access
- **Monitoring Accuracy**: Real-time metrics collection
- **Settlement Accuracy**: Correct cost calculation
- **Release Process**: Clean GPU release

### Success Criteria
- ✅ GPU detected and healthy
- ✅ Provisioning completes in < 5 seconds
- ✅ Monitoring updates every 5 seconds
- ✅ CUDA test runs successfully
- ✅ Settlement calculates correctly
- ✅ GPU released cleanly

## 🔍 Troubleshooting

### Common Issues

**RTX 5060 not detected**
```bash
# Check NVIDIA drivers
nvidia-smi

# Install drivers if needed
sudo apt install nvidia-driver-525
```

**OCX server not running**
```bash
# Start OCX server
cd ../../cmd/server
go run main.go
```

**CUDA compilation fails**
```bash
# Install CUDA toolkit
sudo apt install nvidia-cuda-toolkit

# Or skip CUDA test (it's optional)
```

**Permission denied on SSH**
```bash
# Ensure SSH is enabled
sudo systemctl enable ssh
sudo systemctl start ssh
```

### Debug Mode
```bash
# Run with verbose logging
go run main.go -test=quick -v

# Check GPU status manually
nvidia-smi --query-gpu=name,memory.total,temperature.gpu,utilization.gpu --format=csv
```

## 🎯 Production Readiness

This testing validates that OCX Protocol can:

✅ **Manage Real Hardware**: Not just theoretical compute  
✅ **Handle Real Provisioning**: Actual SSH access and GPU allocation  
✅ **Track Real Usage**: Live monitoring of GPU resources  
✅ **Calculate Real Costs**: Based on actual usage time  
✅ **Process Real Settlements**: Money flows between parties  
✅ **Scale to Real Providers**: Same pattern works for cloud providers  

## 🚀 Next Steps

Once this works with your RTX 5060:

1. **Approach Cloud Providers**: Show them working local implementation
2. **Scale Testing**: Test with multiple local GPUs
3. **Integration Testing**: Connect to real cloud APIs
4. **Performance Testing**: Load test with multiple concurrent leases
5. **Production Deployment**: Deploy to real infrastructure

## 💡 Key Benefits

- **Proof of Concept**: Real hardware, real money, real provisioning
- **Provider Confidence**: Shows OCX can manage actual compute resources
- **Technical Validation**: End-to-end system works correctly
- **Cost Validation**: Real settlement calculations
- **User Experience**: Complete workflow from order to access

**This gives you concrete evidence that OCX Protocol works with real hardware, not just mocks.**
