# OCX Reputation System - Quick Start

## 🚀 What You Need to Run

### 1. Install WABT (WebAssembly Tools)

**Run this command in your terminal:**
```bash
./install_wabt.sh
```

Or manually:
```bash
sudo apt-get update && sudo apt-get install -y wabt
```

### 2. Build the WASM Aggregator Module

```bash
cd modules/reputation-aggregator
make build
make optimize
make artifacts
```

This creates: `artifacts/reputation-aggregator.wasm`

### 3. Start the Server

```bash
# Option A: SQLite (no setup needed)
export DATABASE_URL="file:./ocx.db"
./cmd/server/server

# Option B: PostgreSQL (if you have it)
export DATABASE_URL="postgresql://user:pass@localhost:5432/ocx"
psql $DATABASE_URL < database/migrations/002_trustscore.sql
./cmd/server/server
```

### 4. Test the Endpoints

```bash
# Generate a reputation badge
curl http://localhost:8080/api/v1/reputation/badge/testuser

# Get reputation stats (needs API key)
curl -H "X-API-Key: your-key" http://localhost:8080/api/v1/reputation/stats
```

## 📋 What's Already Working (No WABT Needed)

1. **Server compiles and runs** ✅
2. **5 API endpoints ready** ✅
3. **Database schema created** ✅
4. **GitHub integration** ✅
5. **SVG badge generation** ✅

## 🎯 Next Steps

### If you want to test RIGHT NOW (without WABT):
```bash
# Just start the server
go run cmd/server/main.go cmd/server/reputation_handlers.go
```

### If you want the full WASM experience:
1. Run `./install_wabt.sh` (needs sudo password)
2. Run `make build-aggregator`
3. Complete D-MVM integration

## 💡 Recommendation

**Start the server without WASM first** to see what's working, then add WABT later when you want to test the aggregator module.

```bash
# Quick test (no installation needed):
go run ./cmd/server
```

The reputation system works with or without the WASM aggregator!
