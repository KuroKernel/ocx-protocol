import { describe, it, expect } from 'vitest';
import {
  ReceiptBuilder,
  ReceiptSerializer,
  OCXCrypto,
  MerkleTree,
  bytesToHex,
  hexToBytes,
} from './index';

describe('ReceiptBuilder', () => {
  it('should build a valid receipt', () => {
    const programHash = new Uint8Array(32).fill(1);
    const inputHash = new Uint8Array(32).fill(2);
    const outputHash = new Uint8Array(32).fill(3);

    const receipt = new ReceiptBuilder()
      .programHash(programHash)
      .inputHash(inputHash)
      .outputHash(outputHash)
      .gasUsed(1000n)
      .issuerId('test-issuer')
      .floatMode('strict')
      .build();

    expect(receipt.version).toBe(1);
    expect(receipt.programHash).toEqual(programHash);
    expect(receipt.inputHash).toEqual(inputHash);
    expect(receipt.outputHash).toEqual(outputHash);
    expect(receipt.gasUsed).toBe(1000n);
    expect(receipt.issuerId).toBe('test-issuer');
    expect(receipt.floatMode).toBe('strict');
  });

  it('should accept hex string hashes', () => {
    const hexHash = '0'.repeat(64);
    const receipt = new ReceiptBuilder()
      .programHash(hexHash)
      .inputHash(hexHash)
      .outputHash(hexHash)
      .gasUsed(100)
      .issuerId('test')
      .build();

    expect(bytesToHex(receipt.programHash)).toBe(hexHash);
  });

  it('should throw on missing fields', () => {
    expect(() => new ReceiptBuilder().build()).toThrow('programHash is required');
  });
});

describe('ReceiptSerializer', () => {
  it('should serialize and deserialize a receipt', () => {
    const receipt = new ReceiptBuilder()
      .programHash(new Uint8Array(32).fill(1))
      .inputHash(new Uint8Array(32).fill(2))
      .outputHash(new Uint8Array(32).fill(3))
      .gasUsed(12345n)
      .issuerId('test-issuer-123')
      .floatMode('deterministic')
      .timestamp(1704067200)
      .build();

    const serialized = ReceiptSerializer.serialize(receipt);
    const deserialized = ReceiptSerializer.deserialize(serialized);

    expect(deserialized.version).toBe(receipt.version);
    expect(deserialized.programHash).toEqual(receipt.programHash);
    expect(deserialized.inputHash).toEqual(receipt.inputHash);
    expect(deserialized.outputHash).toEqual(receipt.outputHash);
    expect(deserialized.gasUsed).toBe(receipt.gasUsed);
    expect(deserialized.issuerId).toBe(receipt.issuerId);
    expect(deserialized.floatMode).toBe(receipt.floatMode);
    expect(deserialized.timestamp).toBe(receipt.timestamp);
  });
});

describe('OCXCrypto', () => {
  it('should generate valid keypair', async () => {
    const { privateKey, publicKey } = await OCXCrypto.generateKeypair();

    expect(privateKey.length).toBe(32);
    expect(publicKey.length).toBe(32);
  });

  it('should sign and verify receipts', async () => {
    const { privateKey, publicKey } = await OCXCrypto.generateKeypair();

    const receipt = new ReceiptBuilder()
      .programHash(new Uint8Array(32).fill(1))
      .inputHash(new Uint8Array(32).fill(2))
      .outputHash(new Uint8Array(32).fill(3))
      .gasUsed(1000n)
      .issuerId('test')
      .build();

    const serialized = ReceiptSerializer.serialize(receipt);
    const signature = await OCXCrypto.signReceipt(serialized, privateKey);

    expect(signature.length).toBe(64);

    const valid = await OCXCrypto.verifySignature(serialized, signature, publicKey);
    expect(valid).toBe(true);

    // Tamper with data
    serialized[0] = 99;
    const invalid = await OCXCrypto.verifySignature(serialized, signature, publicKey);
    expect(invalid).toBe(false);
  });

  it('should hash data consistently', () => {
    const data = new Uint8Array([1, 2, 3, 4, 5]);
    const hash1 = OCXCrypto.hash(data);
    const hash2 = OCXCrypto.hash(data);

    expect(hash1).toEqual(hash2);
    expect(hash1.length).toBe(32);
  });

  it('should convert between hex and bytes', () => {
    const original = new Uint8Array([0, 255, 128, 64, 32]);
    const hex = OCXCrypto.toHex(original);
    const restored = OCXCrypto.fromHex(hex);

    expect(hex).toBe('00ff804020');
    expect(restored).toEqual(original);
  });
});

describe('MerkleTree', () => {
  it('should build a tree with single item', () => {
    const items = [new Uint8Array([1, 2, 3])];
    const tree = new MerkleTree(items);

    expect(tree.root.length).toBe(32);
    expect(tree.rootHex.length).toBe(64);
  });

  it('should build a tree with multiple items', () => {
    const items = [
      new Uint8Array([1]),
      new Uint8Array([2]),
      new Uint8Array([3]),
      new Uint8Array([4]),
    ];
    const tree = new MerkleTree(items);

    expect(tree.root.length).toBe(32);
  });

  it('should generate and verify proofs', () => {
    const items = [
      new Uint8Array([1]),
      new Uint8Array([2]),
      new Uint8Array([3]),
      new Uint8Array([4]),
      new Uint8Array([5]),
    ];
    const tree = new MerkleTree(items);

    for (let i = 0; i < items.length; i++) {
      const proof = tree.generateProof(i);
      const valid = MerkleTree.verifyProof(items[i], proof);
      expect(valid).toBe(true);
    }
  });

  it('should reject invalid proofs', () => {
    const items = [new Uint8Array([1]), new Uint8Array([2])];
    const tree = new MerkleTree(items);

    const proof = tree.generateProof(0);
    const valid = MerkleTree.verifyProof(new Uint8Array([99]), proof);

    expect(valid).toBe(false);
  });

  it('should produce deterministic roots', () => {
    const items = [
      new Uint8Array([1, 2, 3]),
      new Uint8Array([4, 5, 6]),
      new Uint8Array([7, 8, 9]),
    ];

    const tree1 = new MerkleTree(items);
    const tree2 = new MerkleTree(items);

    expect(tree1.rootHex).toBe(tree2.rootHex);
  });
});

describe('Hex utilities', () => {
  it('should convert bytes to hex', () => {
    const bytes = new Uint8Array([0, 1, 255, 128]);
    expect(bytesToHex(bytes)).toBe('0001ff80');
  });

  it('should convert hex to bytes', () => {
    const hex = '0001ff80';
    const bytes = hexToBytes(hex);
    expect(bytes).toEqual(new Uint8Array([0, 1, 255, 128]));
  });
});
