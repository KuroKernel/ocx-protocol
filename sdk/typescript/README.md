# OCX Protocol SDK

TypeScript/JavaScript SDK for creating and verifying OCX Protocol receipts.

## Installation

```bash
npm install @ocx-protocol/sdk
```

## Quick Start

```typescript
import {
  OCXClient,
  ReceiptBuilder,
  OCXCrypto,
  MerkleTree,
  Kitaab,
} from '@ocx-protocol/sdk';

// Create a client
const client = new OCXClient({
  serverUrl: 'https://ocx.example.com',
  apiKey: 'your-api-key',
});

// Build and sign a receipt
const receipt = new ReceiptBuilder()
  .programHash('abc123...') // 64-char hex
  .inputHash('def456...')
  .outputHash('789abc...')
  .gasUsed(1000)
  .issuerId('my-service')
  .floatMode('strict')
  .build();

const signed = await client.createReceipt(receipt);

// Verify a receipt
const result = await client.verifyReceipt(signed);
console.log(result.valid); // true
```

## Local Signing (No Server)

```typescript
import { OCXCrypto, ReceiptBuilder, ReceiptSerializer } from '@ocx-protocol/sdk';

// Generate keypair
const { privateKey, publicKey } = await OCXCrypto.generateKeypair();

// Build receipt
const receipt = new ReceiptBuilder()
  .fromExecution(programBytes, inputBytes, outputBytes)
  .gasUsed(1000n)
  .issuerId('local-signer')
  .build();

// Serialize and sign
const data = ReceiptSerializer.serialize(receipt);
const signature = await OCXCrypto.signReceipt(data, privateKey);

// Verify
const valid = await OCXCrypto.verifySignature(data, signature, publicKey);
```

## Merkle Trees

```typescript
import { MerkleTree } from '@ocx-protocol/sdk';

// Build tree from receipt hashes
const items = [receiptHash1, receiptHash2, receiptHash3];
const tree = new MerkleTree(items);

console.log(tree.rootHex); // Merkle root

// Generate proof for item at index 1
const proof = tree.generateProof(1);

// Verify proof
const valid = MerkleTree.verifyProof(items[1], proof);
```

## Kitaab Integration

The SDK includes helpers specifically for Kitaab integration:

```typescript
import { OCXClient, Kitaab } from '@ocx-protocol/sdk';

const client = new OCXClient({ serverUrl: 'https://ocx.kitaab.app' });

// Create invoice receipt
const invoiceReceipt = await Kitaab.createInvoiceReceipt(client, {
  invoiceId: 'INV-001',
  businessId: 'BIZ-123',
  amount: 10000,
  items: [{ name: 'Widget', qty: 10, price: 1000 }],
  gst: 1800,
});

// Create payment receipt
const paymentReceipt = await Kitaab.createPaymentReceipt(client, {
  paymentId: 'PAY-001',
  invoiceId: 'INV-001',
  amount: 11800,
  method: 'UPI',
  timestamp: Date.now() / 1000,
});

// Create credit score receipt
const creditReceipt = await Kitaab.createCreditScoreReceipt(client, {
  businessId: 'BIZ-123',
  score: 750,
  factors: { paymentHistory: 0.9, utilization: 0.3 },
  computedAt: Date.now() / 1000,
  inputDataHash: '...',
});

// Verify and identify receipt type
const result = await Kitaab.verifyKitaabReceipt(client, someReceipt);
console.log(result.type); // 'invoice', 'payment', 'credit-score', etc.
console.log(result.businessId);
```

## API Reference

### ReceiptBuilder

| Method | Description |
|--------|-------------|
| `programHash(hash)` | Set program/code hash (32 bytes or 64-char hex) |
| `inputHash(hash)` | Set input data hash |
| `outputHash(hash)` | Set output data hash |
| `gasUsed(gas)` | Set gas consumed |
| `issuerId(id)` | Set issuer identifier |
| `floatMode(mode)` | Set float handling: 'strict', 'deterministic', 'native' |
| `timestamp(ts)` | Set custom timestamp |
| `fromExecution(prog, in, out)` | Auto-hash program/input/output |
| `build()` | Build the receipt |

### OCXCrypto

| Method | Description |
|--------|-------------|
| `generateKeypair()` | Generate Ed25519 keypair |
| `signReceipt(data, privateKey)` | Sign receipt bytes |
| `verifySignature(data, sig, pubKey)` | Verify signature |
| `hash(data)` | SHA256 hash |
| `toHex(bytes)` | Convert to hex string |
| `fromHex(hex)` | Convert from hex string |

### OCXClient

| Method | Description |
|--------|-------------|
| `createReceipt(receipt)` | Create and sign receipt on server |
| `verifyReceipt(signed)` | Verify receipt signature |
| `batchVerify(receipts)` | Batch verify multiple receipts |
| `buildMerkleTree(items)` | Build Merkle tree |
| `verifyMerkleProof(...)` | Verify Merkle proof |
| `getPublicKey()` | Get server's public key |
| `health()` | Server health check |

## License

MIT
