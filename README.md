# OCX Protocol - GPU Testing Framework

Production-ready testing framework for the Open Compute Exchange Protocol with real NVIDIA GPU hardware integration.

## Quick Start

```bash
# Run simple demo (recommended first)
make simple-demo

# Run killer applications demo
make demo

# Quick GPU verification
./scripts/test_rtx5060.sh quick

# Live GPU monitoring
./scripts/test_rtx5060.sh monitor

# Full end-to-end test (offer → order → provision → monitor → settle)
./scripts/test_rtx5060.sh full
```

## Killer Applications

OCX Protocol includes ready-to-run programs that demonstrate its power:

1. **AlphaFold Protein Folding** - Simulates protein folding energy calculations
2. **LLVM Compiler Testing** - Tests compiler optimization passes
3. **Bitcoin Difficulty Adjustment** - Implements mining difficulty algorithms
4. **Doom Physics Simulation** - Game engine physics with collision detection
5. **WebGL Benchmark** - GPU shader compilation and performance testing

Each program runs deterministically with cryptographic receipts, cycle-accurate metering, and verifiable results.

## Architecture

```
.
├── cmd/ocx-gpu-test/           # Single, clean binary
│   └── main.go
├── internal/
│   ├── gpu/                    # NVIDIA GPU adapter & metrics
│   │   ├── info.go
│   │   ├── monitor.go
│   │   └── runmodes.go
│   └── ocxstub/                # Drop-in OCX client stub
│       └── client.go
├── scripts/
│   └── test_rtx5060.sh
└── bin/
    └── ocx-gpu-test            # Built binary
```

## Features

- **Real Hardware Integration**: Works with actual NVIDIA GPUs via `nvidia-smi`
- **Complete Business Flow**: Order → Matching → Provisioning → Usage → Settlement
- **Live Monitoring**: Real-time GPU metrics (utilization, temperature, memory, power)
- **Production Ready**: Clean architecture, proper error handling, JSON logging
- **Drop-in Replacement**: Easy to swap `ocxstub` with real OCX client

## GPU Requirements

- NVIDIA GPU with `nvidia-smi` support
- Driver version 570+ recommended
- CUDA toolkit optional (for workload testing)

## Example Output

```bash
$ ./scripts/test_rtx5060.sh quick
GPU=NVIDIA Graphics Device, Mem=8151MB, Driver=570.153.02, Temp=56C, Util=84%

$ ./scripts/test_rtx5060.sh full
GPU=NVIDIA Graphics Device, Mem=8151MB, Driver=570.153.02, Temp=61C, Util=87%
offer=offer_1757963962616250243 $/h=2.50
order=order_1757963962616254021
matched order=order_1757963962616254021 provider=local-nvidia-provider
lease=lease_1757963964625396154 addr=192.168.150.102:22 ssh_user=kurokernel
util=96% temp=61C mem=851/8151MB power=0W
util=99% temp=60C mem=853/8151MB power=0W
full test complete
```

## Development

```bash
# Build the binary
go build -o ./bin/ocx-gpu-test ./cmd/ocx-gpu-test

# Run with custom options
./bin/ocx-gpu-test -test=monitor -duration=60s -server=http://localhost:8080
```

## Integration

To integrate with a real OCX server, replace `internal/ocxstub` with `internal/ocxclient` implementing:

```go
CreateOffer(price float64) (*Offer, error)
PlaceOrder(offerID string, gpus, hours int, budget float64) (*Order, error)
WaitMatch(orderID string, timeout time.Duration) error
Provision(orderID string) (*Lease, error)
Settle(orderID string, amount float64) error
Release(leaseID string) error
```

No other code changes required.
