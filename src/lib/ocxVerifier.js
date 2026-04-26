/**
 * ocxVerifier — browser-side OCX receipt verification.
 *
 * Loads `libocx-verify-wasm` (the canonical Rust verifier compiled to
 * WebAssembly, served from /wasm/) and exposes a small async API:
 *
 *     const v = await loadVerifier();
 *     const result = v.verifyHex(cborHex, pubkeyHex);
 *
 * The wasm module is loaded once and memoized — repeated calls reuse the
 * same instance. Loading is dynamic and `webpackIgnore`d so the React
 * bundler doesn't try to inline it; the wasm + glue ship as plain static
 * files under /wasm/ which Netlify serves with the right Content-Type.
 *
 * Returns a promise that resolves to:
 *
 *   {
 *     verifyHex(cborHex: string, pubkeyHex: string) => VerifyResult
 *     verifyBytes(cbor: Uint8Array, pubkey: Uint8Array) => VerifyResult
 *     version: string
 *   }
 *
 * VerifyResult shape (matches the Rust struct in libocx-verify-wasm):
 *
 *   {
 *     ok: boolean,
 *     status_code: 'SUCCESS' | 'INVALID_CBOR' | 'NON_CANONICAL_CBOR' |
 *                  'MISSING_FIELD' | 'INVALID_FIELD_VALUE' |
 *                  'INVALID_SIGNATURE' | 'HASH_MISMATCH' |
 *                  'INVALID_TIMESTAMP' | 'UNEXPECTED_EOF' |
 *                  'INTEGER_OVERFLOW' | 'INVALID_UTF8' |
 *                  'INVALID_PUBLIC_KEY_LENGTH' | 'INVALID_HEX_INPUT',
 *     message: string,
 *     receipt: ReceiptView | null,
 *     bytes_total: number,
 *     signed_message_bytes: number,
 *   }
 */

const WASM_GLUE_URL = "/wasm/libocx_verify_wasm.js";
const WASM_BINARY_URL = "/wasm/libocx_verify_wasm_bg.wasm";

let cachedLoadPromise = null;

export function loadVerifier() {
  if (cachedLoadPromise) return cachedLoadPromise;
  cachedLoadPromise = (async () => {
    // webpackIgnore: tell CRA's bundler to leave this dynamic import alone.
    // The browser does the runtime URL import directly.
    const mod = await import(/* webpackIgnore: true */ WASM_GLUE_URL);
    await mod.default(WASM_BINARY_URL);
    return {
      verifyHex: (cborHex, pubkeyHex) => mod.ocxVerifyHex(cborHex, pubkeyHex),
      verifyBytes: (cbor, pubkey) => mod.ocxVerifyBytes(cbor, pubkey),
      version: mod.ocxVerifierVersion(),
    };
  })().catch((err) => {
    // On failure, clear the cache so a retry can attempt to load again.
    cachedLoadPromise = null;
    throw err;
  });
  return cachedLoadPromise;
}

/**
 * Map an OCX_STATUS_CODE to a one-line human explanation. The wasm
 * module already returns a `message` field, but these are tighter and
 * suitable for the result panel's secondary line.
 */
export const STATUS_DESCRIPTIONS = {
  SUCCESS: "Ed25519 signature valid over the canonical receipt.",
  INVALID_CBOR: "Input bytes are not well-formed CBOR.",
  NON_CANONICAL_CBOR:
    "CBOR is well-formed but not in canonical form (RFC 8949 § 4.2).",
  MISSING_FIELD: "Receipt is missing a required field.",
  INVALID_FIELD_VALUE: "A field value is outside its valid range.",
  INVALID_SIGNATURE:
    "Signature did not verify against the supplied public key.",
  HASH_MISMATCH: "An embedded hash does not match.",
  INVALID_TIMESTAMP: "Receipt timestamp is outside acceptable bounds.",
  UNEXPECTED_EOF: "CBOR input was truncated.",
  INTEGER_OVERFLOW: "Integer overflow during parsing.",
  INVALID_UTF8: "Invalid UTF-8 in a CBOR text-string field.",
  INVALID_PUBLIC_KEY_LENGTH:
    "Public key must be exactly 32 bytes (64 hex characters).",
  INVALID_HEX_INPUT: "Input is not valid hex.",
};
