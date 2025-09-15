# OCX Protocol - Open Compute Exchange

The OCX Protocol is a neutral compute rail that becomes the default way the world discovers, books, runs, verifies, and settles AI compute. Think Visa + SWIFT for GPU/AI, with Linux-grade openness and Kubernetes-grade orchestration.

## Architecture

### Core Components

- **Protocol Schemas** (`types.go`) - The foundational data structures that define how all compute gets quoted globally
- **Identity System** (`id.go`) - Ed25519 key management, DID-style documents, KYC integration
- **HTTP Gateway** (`internal/gateway.go`) - REST API for offers and orders
- **CLI Tool** (`cmd/ocxctl/`) - Command-line interface for interacting with the protocol

### Key Features

- **Cryptographic Security** - All messages are signed with Ed25519 signatures
- **Identity Management** - KYC integration and party verification
- **Market Mechanics** - Offer/Order/Lease flow for discovery → booking → execution
- **Metering & Settlement** - Usage tracking and billing with dispute resolution
- **Compliance** - Built-in support for GDPR, HIPAA, and other regulatory requirements

## Quick Start

### 1. Start the Server

```bash
go run main.go
```

The server will start on port 8080 by default.

### 2. Use the CLI

```bash
# Build the CLI
go build -o ocxctl cmd/ocxctl/main.go

# Make an offer
./ocxctl -command make-offer

# List offers
./ocxctl -command list-offers

# Place an order (replace with actual offer ID)
./ocxctl -command place-order -offer-id "01J8Z3TF6X9H3W1M6A6J1KSTQH"

# List orders
./ocxctl -command list-orders
```

### 3. API Endpoints

- `POST /offers` - Publish an offer
- `GET /offers` - List all offers
- `POST /orders` - Place an order
- `GET /orders` - List all orders
- `GET /health` - Health check

## Protocol Flow

1. **Offer** - Provider publishes compute capacity with pricing
2. **Order** - Buyer places an order for specific resources
3. **Lease** - System creates a lease with access credentials
4. **Meter** - Usage is tracked and reported
5. **Invoice** - Billing is generated and settled

## Example JSON

See `fixtures/end-to-end-example.json` for a complete example of the protocol flow.

## Development

This is a minimal implementation focused on the core protocol. The codebase is designed to be:

- **Minimal** - Under 10,000 lines of code
- **Secure** - Cryptographic signatures on all messages
- **Extensible** - Protocol versioning and schema evolution
- **Production-ready** - No stubs or demos

## License

MIT License - see LICENSE file for details.
