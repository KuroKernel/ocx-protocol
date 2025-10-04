#!/bin/bash
# Simple deterministic program
echo "Hello from OCX Protocol!"
echo "Input: ${OCX_INPUT:-default}"
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Random: 42"  # Fixed for determinism
