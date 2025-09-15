#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-quick}"

# build
go build -o ./bin/ocx-gpu-test ./cmd/ocx-gpu-test

case "$MODE" in
  quick)   ./bin/ocx-gpu-test -test=quick ;;
  monitor) ./bin/ocx-gpu-test -test=monitor -duration=30s ;;
  full)    ./bin/ocx-gpu-test -test=full -server="${OCX_SERVER_URL:-http://localhost:8080}" ;;
  *) echo "usage: $0 {quick|monitor|full}"; exit 2 ;;
esac
