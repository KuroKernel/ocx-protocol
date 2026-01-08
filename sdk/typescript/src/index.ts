/**
 * OCX Protocol SDK
 * Verifiable Computation Receipt Toolkit for Kitaab and Beyond
 */

import * as ed25519 from '@noble/ed25519';
import { sha256 } from '@noble/hashes/sha256';
import { bytesToHex, hexToBytes } from '@noble/hashes/utils';

// ============================================================================
// Types
// ============================================================================

export interface ReceiptCore {
  version: number;
  programHash: Uint8Array;
  inputHash: Uint8Array;
  outputHash: Uint8Array;
  gasUsed: bigint;
  timestamp: number;
  issuerId: string;
  floatMode: 'strict' | 'deterministic' | 'native';
  nonce?: Uint8Array;
}

export interface SignedReceipt {
  receipt: Uint8Array;
  signature: Uint8Array;
  publicKey: Uint8Array;
}

export interface VerificationResult {
  valid: boolean;
  error?: string;
  fields?: ReceiptFields;
}

export interface ReceiptFields {
  programHash: string;
  inputHash: string;
  outputHash: string;
  gasUsed: bigint;
  issuerId: string;
  floatMode: string;
}

export interface MerkleProof {
  root: Uint8Array;
  index: number;
  proof: Uint8Array[];
}

export interface BatchVerifyResult {
  results: { index: number; valid: boolean; error?: string }[];
  totalCount: number;
  validCount: number;
  invalidCount: number;
  duration: number;
}

export interface OCXClientConfig {
  serverUrl: string;
  apiKey?: string;
  timeout?: number;
  retries?: number;
}

// ============================================================================
// Receipt Builder
// ============================================================================

export class ReceiptBuilder {
  private receipt: Partial<ReceiptCore> = {
    version: 1,
    floatMode: 'strict',
    timestamp: Math.floor(Date.now() / 1000),
  };

  /**
   * Set the program hash (SHA256 of WASM/code)
   */
  programHash(hash: Uint8Array | string): this {
    this.receipt.programHash = typeof hash === 'string' ? hexToBytes(hash) : hash;
    return this;
  }

  /**
   * Set the input hash (SHA256 of input data)
   */
  inputHash(hash: Uint8Array | string): this {
    this.receipt.inputHash = typeof hash === 'string' ? hexToBytes(hash) : hash;
    return this;
  }

  /**
   * Set the output hash (SHA256 of output data)
   */
  outputHash(hash: Uint8Array | string): this {
    this.receipt.outputHash = typeof hash === 'string' ? hexToBytes(hash) : hash;
    return this;
  }

  /**
   * Set the gas used during execution
   */
  gasUsed(gas: bigint | number): this {
    this.receipt.gasUsed = BigInt(gas);
    return this;
  }

  /**
   * Set the issuer ID
   */
  issuerId(id: string): this {
    this.receipt.issuerId = id;
    return this;
  }

  /**
   * Set the float mode for deterministic computation
   */
  floatMode(mode: 'strict' | 'deterministic' | 'native'): this {
    this.receipt.floatMode = mode;
    return this;
  }

  /**
   * Set custom timestamp
   */
  timestamp(ts: number): this {
    this.receipt.timestamp = ts;
    return this;
  }

  /**
   * Build the receipt from program, input, and output data
   */
  fromExecution(program: Uint8Array, input: Uint8Array, output: Uint8Array): this {
    this.receipt.programHash = sha256(program);
    this.receipt.inputHash = sha256(input);
    this.receipt.outputHash = sha256(output);
    return this;
  }

  /**
   * Build the receipt
   */
  build(): ReceiptCore {
    if (!this.receipt.programHash) throw new Error('programHash is required');
    if (!this.receipt.inputHash) throw new Error('inputHash is required');
    if (!this.receipt.outputHash) throw new Error('outputHash is required');
    if (this.receipt.gasUsed === undefined) throw new Error('gasUsed is required');
    if (!this.receipt.issuerId) throw new Error('issuerId is required');

    return this.receipt as ReceiptCore;
  }
}

// ============================================================================
// Cryptographic Operations
// ============================================================================

export class OCXCrypto {
  /**
   * Generate Ed25519 keypair
   */
  static async generateKeypair(): Promise<{ privateKey: Uint8Array; publicKey: Uint8Array }> {
    const privateKey = ed25519.utils.randomPrivateKey();
    const publicKey = await ed25519.getPublicKeyAsync(privateKey);
    return { privateKey, publicKey };
  }

  /**
   * Sign receipt with Ed25519 private key
   */
  static async signReceipt(receipt: Uint8Array, privateKey: Uint8Array): Promise<Uint8Array> {
    return await ed25519.signAsync(receipt, privateKey);
  }

  /**
   * Verify receipt signature
   */
  static async verifySignature(
    receipt: Uint8Array,
    signature: Uint8Array,
    publicKey: Uint8Array
  ): Promise<boolean> {
    try {
      return await ed25519.verifyAsync(signature, receipt, publicKey);
    } catch {
      return false;
    }
  }

  /**
   * Hash data with SHA256
   */
  static hash(data: Uint8Array): Uint8Array {
    return sha256(data);
  }

  /**
   * Convert bytes to hex string
   */
  static toHex(bytes: Uint8Array): string {
    return bytesToHex(bytes);
  }

  /**
   * Convert hex string to bytes
   */
  static fromHex(hex: string): Uint8Array {
    return hexToBytes(hex);
  }
}

// ============================================================================
// Receipt Serialization (CBOR-compatible format)
// ============================================================================

export class ReceiptSerializer {
  /**
   * Serialize receipt to bytes (simplified CBOR-like format)
   */
  static serialize(receipt: ReceiptCore): Uint8Array {
    const encoder = new TextEncoder();
    const parts: Uint8Array[] = [];

    // Version (1 byte)
    parts.push(new Uint8Array([receipt.version]));

    // Program hash (32 bytes)
    parts.push(receipt.programHash);

    // Input hash (32 bytes)
    parts.push(receipt.inputHash);

    // Output hash (32 bytes)
    parts.push(receipt.outputHash);

    // Gas used (8 bytes, big-endian)
    const gasBuffer = new ArrayBuffer(8);
    const gasView = new DataView(gasBuffer);
    gasView.setBigUint64(0, receipt.gasUsed, false);
    parts.push(new Uint8Array(gasBuffer));

    // Timestamp (8 bytes, big-endian)
    const tsBuffer = new ArrayBuffer(8);
    const tsView = new DataView(tsBuffer);
    tsView.setBigUint64(0, BigInt(receipt.timestamp), false);
    parts.push(new Uint8Array(tsBuffer));

    // Issuer ID (length-prefixed string)
    const issuerBytes = encoder.encode(receipt.issuerId);
    const issuerLen = new Uint8Array([issuerBytes.length]);
    parts.push(issuerLen);
    parts.push(issuerBytes);

    // Float mode (1 byte: 0=strict, 1=deterministic, 2=native)
    const floatModeMap = { strict: 0, deterministic: 1, native: 2 };
    parts.push(new Uint8Array([floatModeMap[receipt.floatMode]]));

    // Concatenate all parts
    const totalLength = parts.reduce((sum, p) => sum + p.length, 0);
    const result = new Uint8Array(totalLength);
    let offset = 0;
    for (const part of parts) {
      result.set(part, offset);
      offset += part.length;
    }

    return result;
  }

  /**
   * Deserialize receipt from bytes
   */
  static deserialize(data: Uint8Array): ReceiptCore {
    const decoder = new TextDecoder();
    let offset = 0;

    // Version
    const version = data[offset++];

    // Program hash
    const programHash = data.slice(offset, offset + 32);
    offset += 32;

    // Input hash
    const inputHash = data.slice(offset, offset + 32);
    offset += 32;

    // Output hash
    const outputHash = data.slice(offset, offset + 32);
    offset += 32;

    // Gas used
    const gasView = new DataView(data.buffer, data.byteOffset + offset, 8);
    const gasUsed = gasView.getBigUint64(0, false);
    offset += 8;

    // Timestamp
    const tsView = new DataView(data.buffer, data.byteOffset + offset, 8);
    const timestamp = Number(tsView.getBigUint64(0, false));
    offset += 8;

    // Issuer ID
    const issuerLen = data[offset++];
    const issuerId = decoder.decode(data.slice(offset, offset + issuerLen));
    offset += issuerLen;

    // Float mode
    const floatModeMap = ['strict', 'deterministic', 'native'] as const;
    const floatMode = floatModeMap[data[offset++]];

    return {
      version,
      programHash,
      inputHash,
      outputHash,
      gasUsed,
      timestamp,
      issuerId,
      floatMode,
    };
  }
}

// ============================================================================
// Merkle Tree
// ============================================================================

export class MerkleTree {
  private leaves: Uint8Array[];
  private layers: Uint8Array[][];

  constructor(items: Uint8Array[]) {
    if (items.length === 0) {
      throw new Error('Cannot create empty Merkle tree');
    }

    // Hash all leaves
    this.leaves = items.map((item) => sha256(item));
    this.layers = [this.leaves];
    this.buildTree();
  }

  private buildTree(): void {
    let currentLayer = this.leaves;

    while (currentLayer.length > 1) {
      const nextLayer: Uint8Array[] = [];

      for (let i = 0; i < currentLayer.length; i += 2) {
        const left = currentLayer[i];
        const right = i + 1 < currentLayer.length ? currentLayer[i + 1] : left;

        // Combine and hash (sort to ensure determinism)
        const combined = new Uint8Array(64);
        if (this.compareBytes(left, right) <= 0) {
          combined.set(left, 0);
          combined.set(right, 32);
        } else {
          combined.set(right, 0);
          combined.set(left, 32);
        }
        nextLayer.push(sha256(combined));
      }

      this.layers.push(nextLayer);
      currentLayer = nextLayer;
    }
  }

  private compareBytes(a: Uint8Array, b: Uint8Array): number {
    for (let i = 0; i < Math.min(a.length, b.length); i++) {
      if (a[i] !== b[i]) return a[i] - b[i];
    }
    return a.length - b.length;
  }

  /**
   * Get the Merkle root
   */
  get root(): Uint8Array {
    return this.layers[this.layers.length - 1][0];
  }

  /**
   * Get the root as hex string
   */
  get rootHex(): string {
    return bytesToHex(this.root);
  }

  /**
   * Generate proof for item at index
   */
  generateProof(index: number): MerkleProof {
    if (index < 0 || index >= this.leaves.length) {
      throw new Error('Index out of bounds');
    }

    const proof: Uint8Array[] = [];
    let currentIndex = index;

    for (let i = 0; i < this.layers.length - 1; i++) {
      const layer = this.layers[i];
      const siblingIndex = currentIndex % 2 === 0 ? currentIndex + 1 : currentIndex - 1;

      if (siblingIndex < layer.length) {
        proof.push(layer[siblingIndex]);
      }

      currentIndex = Math.floor(currentIndex / 2);
    }

    return {
      root: this.root,
      index,
      proof,
    };
  }

  /**
   * Verify a Merkle proof
   */
  static verifyProof(item: Uint8Array, proof: MerkleProof): boolean {
    let hash = sha256(item);
    let index = proof.index;

    for (const sibling of proof.proof) {
      const combined = new Uint8Array(64);

      // Sort for determinism
      if (index % 2 === 0) {
        combined.set(hash, 0);
        combined.set(sibling, 32);
      } else {
        combined.set(sibling, 0);
        combined.set(hash, 32);
      }

      hash = sha256(combined);
      index = Math.floor(index / 2);
    }

    // Compare with root
    return bytesToHex(hash) === bytesToHex(proof.root);
  }
}

// ============================================================================
// OCX Client (API Client)
// ============================================================================

export class OCXClient {
  private config: Required<OCXClientConfig>;

  constructor(config: OCXClientConfig) {
    this.config = {
      serverUrl: config.serverUrl.replace(/\/$/, ''),
      apiKey: config.apiKey || '',
      timeout: config.timeout || 30000,
      retries: config.retries || 3,
    };
  }

  private async request<T>(
    method: string,
    path: string,
    body?: object
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.config.apiKey) {
      headers['X-API-Key'] = this.config.apiKey;
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const response = await fetch(`${this.config.serverUrl}${path}`, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      if (!response.ok) {
        const error = await response.json().catch(() => ({ error: response.statusText }));
        throw new Error(error.error || `HTTP ${response.status}`);
      }

      return await response.json();
    } finally {
      clearTimeout(timeout);
    }
  }

  /**
   * Create and sign a receipt
   */
  async createReceipt(receipt: ReceiptCore): Promise<SignedReceipt> {
    const response = await this.request<{
      receipt: string;
      signature: string;
      public_key: string;
    }>('POST', '/api/v1/receipts', {
      program_hash: bytesToHex(receipt.programHash),
      input_hash: bytesToHex(receipt.inputHash),
      output_hash: bytesToHex(receipt.outputHash),
      gas_used: receipt.gasUsed.toString(),
      issuer_id: receipt.issuerId,
      float_mode: receipt.floatMode,
    });

    return {
      receipt: hexToBytes(response.receipt),
      signature: hexToBytes(response.signature),
      publicKey: hexToBytes(response.public_key),
    };
  }

  /**
   * Verify a receipt
   */
  async verifyReceipt(signedReceipt: SignedReceipt): Promise<VerificationResult> {
    const response = await this.request<{
      valid: boolean;
      error?: string;
      fields?: {
        program_hash: string;
        input_hash: string;
        output_hash: string;
        gas_used: string;
        issuer_id: string;
        float_mode: string;
      };
    }>('POST', '/api/v1/receipts/verify', {
      receipt: bytesToHex(signedReceipt.receipt),
      signature: bytesToHex(signedReceipt.signature),
      public_key: bytesToHex(signedReceipt.publicKey),
    });

    const result: VerificationResult = {
      valid: response.valid,
      error: response.error,
    };

    if (response.fields) {
      result.fields = {
        programHash: response.fields.program_hash,
        inputHash: response.fields.input_hash,
        outputHash: response.fields.output_hash,
        gasUsed: BigInt(response.fields.gas_used),
        issuerId: response.fields.issuer_id,
        floatMode: response.fields.float_mode,
      };
    }

    return result;
  }

  /**
   * Batch verify multiple receipts
   */
  async batchVerify(receipts: SignedReceipt[]): Promise<BatchVerifyResult> {
    const response = await this.request<{
      results: { index: number; valid: boolean; error?: string }[];
      total_count: number;
      valid_count: number;
      invalid_count: number;
      duration: string;
    }>('POST', '/api/v1/receipts/batch-verify', {
      receipts: receipts.map((r) => ({
        receipt: bytesToHex(r.receipt),
        signature: bytesToHex(r.signature),
        public_key: bytesToHex(r.publicKey),
      })),
    });

    return {
      results: response.results,
      totalCount: response.total_count,
      validCount: response.valid_count,
      invalidCount: response.invalid_count,
      duration: parseFloat(response.duration),
    };
  }

  /**
   * Build Merkle tree from items
   */
  async buildMerkleTree(items: string[]): Promise<{
    root: string;
    leafCount: number;
    treeHeight: number;
    proofs: Record<number, string[]>;
  }> {
    return await this.request('POST', '/api/v1/merkle/tree', { items });
  }

  /**
   * Verify Merkle proof
   */
  async verifyMerkleProof(
    root: string,
    item: string,
    index: number,
    proof: string[]
  ): Promise<boolean> {
    const response = await this.request<{ valid: boolean }>('POST', '/api/v1/merkle/verify', {
      root,
      item,
      index,
      proof,
    });
    return response.valid;
  }

  /**
   * Get server public key
   */
  async getPublicKey(): Promise<{ publicKey: string; algorithm: string }> {
    const response = await this.request<{ public_key: string; algorithm: string }>(
      'GET',
      '/api/v1/keys/public'
    );
    return {
      publicKey: response.public_key,
      algorithm: response.algorithm,
    };
  }

  /**
   * Health check
   */
  async health(): Promise<{
    status: string;
    version: string;
    uptime: string;
    requests: number;
  }> {
    return await this.request('GET', '/health');
  }
}

// ============================================================================
// Kitaab Integration Helpers
// ============================================================================

export namespace Kitaab {
  /**
   * Create receipt for invoice finalization
   */
  export async function createInvoiceReceipt(
    client: OCXClient,
    invoiceData: {
      invoiceId: string;
      businessId: string;
      amount: number;
      items: object[];
      gst: number;
    }
  ): Promise<SignedReceipt> {
    const encoder = new TextEncoder();
    const invoiceBytes = encoder.encode(JSON.stringify(invoiceData));

    const receipt = new ReceiptBuilder()
      .programHash(sha256(encoder.encode('kitaab-invoice-v1')))
      .inputHash(sha256(invoiceBytes))
      .outputHash(sha256(invoiceBytes)) // Invoice data is the output
      .gasUsed(1000)
      .issuerId(`kitaab:${invoiceData.businessId}`)
      .floatMode('strict')
      .build();

    return await client.createReceipt(receipt);
  }

  /**
   * Create receipt for payment recording
   */
  export async function createPaymentReceipt(
    client: OCXClient,
    paymentData: {
      paymentId: string;
      invoiceId: string;
      amount: number;
      method: string;
      timestamp: number;
    }
  ): Promise<SignedReceipt> {
    const encoder = new TextEncoder();
    const paymentBytes = encoder.encode(JSON.stringify(paymentData));

    const receipt = new ReceiptBuilder()
      .programHash(sha256(encoder.encode('kitaab-payment-v1')))
      .inputHash(sha256(encoder.encode(paymentData.invoiceId)))
      .outputHash(sha256(paymentBytes))
      .gasUsed(500)
      .issuerId(`kitaab:payment:${paymentData.paymentId}`)
      .floatMode('strict')
      .timestamp(paymentData.timestamp)
      .build();

    return await client.createReceipt(receipt);
  }

  /**
   * Create receipt for credit score computation
   */
  export async function createCreditScoreReceipt(
    client: OCXClient,
    scoreData: {
      businessId: string;
      score: number;
      factors: object;
      computedAt: number;
      inputDataHash: string;
    }
  ): Promise<SignedReceipt> {
    const encoder = new TextEncoder();
    const scoreBytes = encoder.encode(
      JSON.stringify({ score: scoreData.score, factors: scoreData.factors })
    );

    const receipt = new ReceiptBuilder()
      .programHash(sha256(encoder.encode('kitaab-credit-score-v1')))
      .inputHash(hexToBytes(scoreData.inputDataHash))
      .outputHash(sha256(scoreBytes))
      .gasUsed(5000)
      .issuerId(`kitaab:credit:${scoreData.businessId}`)
      .floatMode('deterministic')
      .timestamp(scoreData.computedAt)
      .build();

    return await client.createReceipt(receipt);
  }

  /**
   * Create receipt for daily business snapshot
   */
  export async function createDailySnapshotReceipt(
    client: OCXClient,
    snapshotData: {
      businessId: string;
      date: string;
      revenue: number;
      expenses: number;
      invoiceCount: number;
      paymentCount: number;
    }
  ): Promise<SignedReceipt> {
    const encoder = new TextEncoder();
    const snapshotBytes = encoder.encode(JSON.stringify(snapshotData));

    const receipt = new ReceiptBuilder()
      .programHash(sha256(encoder.encode('kitaab-daily-snapshot-v1')))
      .inputHash(sha256(encoder.encode(`${snapshotData.businessId}:${snapshotData.date}`)))
      .outputHash(sha256(snapshotBytes))
      .gasUsed(2000)
      .issuerId(`kitaab:snapshot:${snapshotData.date}`)
      .floatMode('strict')
      .build();

    return await client.createReceipt(receipt);
  }

  /**
   * Create receipt for GST computation
   */
  export async function createGSTReceipt(
    client: OCXClient,
    gstData: {
      businessId: string;
      period: string;
      gstIn: number;
      gstOut: number;
      netGst: number;
      transactions: string[];
    }
  ): Promise<SignedReceipt> {
    const encoder = new TextEncoder();
    const gstBytes = encoder.encode(
      JSON.stringify({
        gstIn: gstData.gstIn,
        gstOut: gstData.gstOut,
        netGst: gstData.netGst,
      })
    );

    const txHash = sha256(encoder.encode(gstData.transactions.join(',')));

    const receipt = new ReceiptBuilder()
      .programHash(sha256(encoder.encode('kitaab-gst-v1')))
      .inputHash(txHash)
      .outputHash(sha256(gstBytes))
      .gasUsed(3000)
      .issuerId(`kitaab:gst:${gstData.businessId}:${gstData.period}`)
      .floatMode('strict')
      .build();

    return await client.createReceipt(receipt);
  }

  /**
   * Verify a Kitaab receipt and extract business context
   */
  export async function verifyKitaabReceipt(
    client: OCXClient,
    signedReceipt: SignedReceipt
  ): Promise<{
    valid: boolean;
    type?: 'invoice' | 'payment' | 'credit-score' | 'snapshot' | 'gst' | 'unknown';
    businessId?: string;
    error?: string;
  }> {
    const result = await client.verifyReceipt(signedReceipt);

    if (!result.valid) {
      return { valid: false, error: result.error };
    }

    // Parse issuer ID to determine type and business
    const issuerId = result.fields?.issuerId || '';
    let type: 'invoice' | 'payment' | 'credit-score' | 'snapshot' | 'gst' | 'unknown' = 'unknown';
    let businessId: string | undefined;

    if (issuerId.startsWith('kitaab:payment:')) {
      type = 'payment';
    } else if (issuerId.startsWith('kitaab:credit:')) {
      type = 'credit-score';
      businessId = issuerId.replace('kitaab:credit:', '');
    } else if (issuerId.startsWith('kitaab:snapshot:')) {
      type = 'snapshot';
    } else if (issuerId.startsWith('kitaab:gst:')) {
      type = 'gst';
      const parts = issuerId.split(':');
      businessId = parts[2];
    } else if (issuerId.startsWith('kitaab:')) {
      type = 'invoice';
      businessId = issuerId.replace('kitaab:', '');
    }

    return { valid: true, type, businessId };
  }
}

// ============================================================================
// Export all
// ============================================================================

export {
  sha256,
  bytesToHex,
  hexToBytes,
};
