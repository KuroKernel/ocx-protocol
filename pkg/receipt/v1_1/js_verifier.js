/**
 * OCX Protocol Receipt v1.1 JavaScript Verifier
 * 
 * This implementation provides cross-architecture compatibility by using
 * browser-native SubtleCrypto API for Ed25519 operations and canonical CBOR encoding.
 * 
 * Usage:
 *   const verifier = new OCXReceiptVerifier();
 *   const isValid = await verifier.verifyReceipt(receiptData, publicKey);
 */

class OCXReceiptVerifier {
    constructor() {
        this.domainSeparator = new TextEncoder().encode("OCXv1|receipt|");
    }

    /**
     * Verifies a receipt using Ed25519 signature verification
     * @param {ArrayBuffer} receiptData - The CBOR-encoded receipt
     * @param {ArrayBuffer} publicKey - The Ed25519 public key (32 bytes)
     * @returns {Promise<boolean>} - True if verification succeeds
     */
    async verifyReceipt(receiptData, publicKey) {
        try {
            // Parse the CBOR receipt
            const receipt = this.parseCBOR(receiptData);
            
            // Extract the core data
            const core = receipt.core;
            
            // Encode the core with canonical CBOR
            const coreData = this.encodeCanonicalCBOR(core);
            
            // Create the message that was signed
            const message = new Uint8Array(this.domainSeparator.length + coreData.length);
            message.set(this.domainSeparator, 0);
            message.set(coreData, this.domainSeparator.length);
            
            // Verify the signature
            const signature = new Uint8Array(receipt.signature);
            const isValid = await this.verifyEd25519Signature(publicKey, message, signature);
            
            return isValid;
        } catch (error) {
            console.error("Receipt verification failed:", error);
            return false;
        }
    }

    /**
     * Verifies Ed25519 signature using Web Crypto API
     * @param {ArrayBuffer} publicKey - Ed25519 public key
     * @param {Uint8Array} message - Message to verify
     * @param {Uint8Array} signature - Signature to verify
     * @returns {Promise<boolean>} - True if signature is valid
     */
    async verifyEd25519Signature(publicKey, message, signature) {
        try {
            // Import the public key
            const cryptoKey = await crypto.subtle.importKey(
                "raw",
                publicKey,
                {
                    name: "Ed25519",
                    namedCurve: "Ed25519"
                },
                false,
                ["verify"]
            );

            // Verify the signature
            const isValid = await crypto.subtle.verify(
                "Ed25519",
                cryptoKey,
                signature,
                message
            );

            return isValid;
        } catch (error) {
            console.error("Ed25519 verification failed:", error);
            return false;
        }
    }

    /**
     * Parses CBOR data into a JavaScript object
     * @param {ArrayBuffer} data - CBOR-encoded data
     * @returns {Object} - Parsed object
     */
    parseCBOR(data) {
        // This is a simplified CBOR parser for the specific receipt format
        // In production, you would use a proper CBOR library like cbor-js
        const view = new DataView(data);
        let offset = 0;

        // Parse the receipt structure
        const receipt = {};
        
        // Skip CBOR header and parse map
        offset = this.skipCBORHeader(view, offset);
        const mapLength = this.readCBORLength(view, offset);
        offset = this.advanceOffset(view, offset);

        // Parse map entries
        for (let i = 0; i < mapLength; i++) {
            const key = this.parseCBORValue(view, offset);
            offset = key.offset;
            
            const value = this.parseCBORValue(view, offset);
            offset = value.offset;
            
            receipt[key.value] = value.value;
        }

        return receipt;
    }

    /**
     * Encodes an object to canonical CBOR
     * @param {Object} obj - Object to encode
     * @returns {Uint8Array} - CBOR-encoded data
     */
    encodeCanonicalCBOR(obj) {
        // This is a simplified canonical CBOR encoder
        // In production, you would use a proper CBOR library with canonical encoding
        const chunks = [];
        
        // Start with map header
        chunks.push(this.encodeCBORMapHeader(obj));
        
        // Sort keys canonically and encode each entry
        const sortedKeys = Object.keys(obj).sort();
        for (const key of sortedKeys) {
            chunks.push(this.encodeCBORValue(key));
            chunks.push(this.encodeCBORValue(obj[key]));
        }
        
        // Combine all chunks
        const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
        const result = new Uint8Array(totalLength);
        let offset = 0;
        
        for (const chunk of chunks) {
            result.set(chunk, offset);
            offset += chunk.length;
        }
        
        return result;
    }

    /**
     * Encodes a CBOR map header
     * @param {Object} obj - Object to encode
     * @returns {Uint8Array} - CBOR map header
     */
    encodeCBORMapHeader(obj) {
        const length = Object.keys(obj).length;
        if (length < 24) {
            return new Uint8Array([0xa0 | length]);
        } else if (length < 256) {
            return new Uint8Array([0xb8, length]);
        } else if (length < 65536) {
            const result = new Uint8Array(3);
            result[0] = 0xb9;
            result[1] = (length >> 8) & 0xff;
            result[2] = length & 0xff;
            return result;
        } else {
            throw new Error("Map too large for CBOR encoding");
        }
    }

    /**
     * Encodes a CBOR value
     * @param {*} value - Value to encode
     * @returns {Uint8Array} - CBOR-encoded value
     */
    encodeCBORValue(value) {
        if (typeof value === 'number') {
            return this.encodeCBORNumber(value);
        } else if (typeof value === 'string') {
            return this.encodeCBORString(value);
        } else if (value instanceof Uint8Array) {
            return this.encodeCBORBytes(value);
        } else if (Array.isArray(value)) {
            return this.encodeCBORArray(value);
        } else if (typeof value === 'object' && value !== null) {
            return this.encodeCBORMap(value);
        } else {
            throw new Error(`Unsupported CBOR value type: ${typeof value}`);
        }
    }

    /**
     * Encodes a CBOR number
     * @param {number} value - Number to encode
     * @returns {Uint8Array} - CBOR-encoded number
     */
    encodeCBORNumber(value) {
        if (Number.isInteger(value) && value >= 0) {
            if (value < 24) {
                return new Uint8Array([value]);
            } else if (value < 256) {
                return new Uint8Array([0x18, value]);
            } else if (value < 65536) {
                const result = new Uint8Array(3);
                result[0] = 0x19;
                result[1] = (value >> 8) & 0xff;
                result[2] = value & 0xff;
                return result;
            } else if (value < 4294967296) {
                const result = new Uint8Array(5);
                result[0] = 0x1a;
                result[1] = (value >> 24) & 0xff;
                result[2] = (value >> 16) & 0xff;
                result[3] = (value >> 8) & 0xff;
                result[4] = value & 0xff;
                return result;
            } else {
                const result = new Uint8Array(9);
                result[0] = 0x1b;
                for (let i = 0; i < 8; i++) {
                    result[8 - i] = (value >> (i * 8)) & 0xff;
                }
                return result;
            }
        } else {
            throw new Error("Negative numbers not supported in this implementation");
        }
    }

    /**
     * Encodes a CBOR string
     * @param {string} value - String to encode
     * @returns {Uint8Array} - CBOR-encoded string
     */
    encodeCBORString(value) {
        const bytes = new TextEncoder().encode(value);
        const length = bytes.length;
        
        let header;
        if (length < 24) {
            header = new Uint8Array([0x60 | length]);
        } else if (length < 256) {
            header = new Uint8Array([0x78, length]);
        } else if (length < 65536) {
            header = new Uint8Array(3);
            header[0] = 0x79;
            header[1] = (length >> 8) & 0xff;
            header[2] = length & 0xff;
        } else {
            throw new Error("String too long for CBOR encoding");
        }
        
        const result = new Uint8Array(header.length + bytes.length);
        result.set(header, 0);
        result.set(bytes, header.length);
        return result;
    }

    /**
     * Encodes CBOR bytes
     * @param {Uint8Array} value - Bytes to encode
     * @returns {Uint8Array} - CBOR-encoded bytes
     */
    encodeCBORBytes(value) {
        const length = value.length;
        
        let header;
        if (length < 24) {
            header = new Uint8Array([0x40 | length]);
        } else if (length < 256) {
            header = new Uint8Array([0x58, length]);
        } else if (length < 65536) {
            header = new Uint8Array(3);
            header[0] = 0x59;
            header[1] = (length >> 8) & 0xff;
            header[2] = length & 0xff;
        } else {
            throw new Error("Bytes too long for CBOR encoding");
        }
        
        const result = new Uint8Array(header.length + value.length);
        result.set(header, 0);
        result.set(value, header.length);
        return result;
    }

    /**
     * Encodes a CBOR array
     * @param {Array} value - Array to encode
     * @returns {Uint8Array} - CBOR-encoded array
     */
    encodeCBORArray(value) {
        const length = value.length;
        
        let header;
        if (length < 24) {
            header = new Uint8Array([0x80 | length]);
        } else if (length < 256) {
            header = new Uint8Array([0x98, length]);
        } else {
            throw new Error("Array too long for CBOR encoding");
        }
        
        const chunks = [header];
        for (const item of value) {
            chunks.push(this.encodeCBORValue(item));
        }
        
        const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
        const result = new Uint8Array(totalLength);
        let offset = 0;
        
        for (const chunk of chunks) {
            result.set(chunk, offset);
            offset += chunk.length;
        }
        
        return result;
    }

    /**
     * Encodes a CBOR map
     * @param {Object} value - Map to encode
     * @returns {Uint8Array} - CBOR-encoded map
     */
    encodeCBORMap(value) {
        const length = Object.keys(value).length;
        
        let header;
        if (length < 24) {
            header = new Uint8Array([0xa0 | length]);
        } else if (length < 256) {
            header = new Uint8Array([0xb8, length]);
        } else {
            throw new Error("Map too long for CBOR encoding");
        }
        
        const chunks = [header];
        const sortedKeys = Object.keys(value).sort();
        
        for (const key of sortedKeys) {
            chunks.push(this.encodeCBORValue(key));
            chunks.push(this.encodeCBORValue(value[key]));
        }
        
        const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
        const result = new Uint8Array(totalLength);
        let offset = 0;
        
        for (const chunk of chunks) {
            result.set(chunk, offset);
            offset += chunk.length;
        }
        
        return result;
    }

    // Helper methods for CBOR parsing (simplified)
    skipCBORHeader(view, offset) {
        const firstByte = view.getUint8(offset);
        if ((firstByte & 0xe0) === 0xa0) { // Map
            return offset + 1;
        } else if (firstByte === 0xb8) { // Map with 1-byte length
            return offset + 2;
        } else if (firstByte === 0xb9) { // Map with 2-byte length
            return offset + 3;
        }
        return offset;
    }

    readCBORLength(view, offset) {
        const firstByte = view.getUint8(offset);
        if ((firstByte & 0xe0) === 0xa0) {
            return firstByte & 0x1f;
        } else if (firstByte === 0xb8) {
            return view.getUint8(offset + 1);
        } else if (firstByte === 0xb9) {
            return view.getUint16(offset + 1);
        }
        return 0;
    }

    advanceOffset(view, offset) {
        const firstByte = view.getUint8(offset);
        if ((firstByte & 0xe0) === 0xa0) {
            return offset + 1;
        } else if (firstByte === 0xb8) {
            return offset + 2;
        } else if (firstByte === 0xb9) {
            return offset + 3;
        }
        return offset;
    }

    parseCBORValue(view, offset) {
        // Simplified CBOR value parser
        // In production, you would use a proper CBOR library
        const firstByte = view.getUint8(offset);
        
        if ((firstByte & 0xe0) === 0x00) { // Unsigned integer
            const value = firstByte & 0x1f;
            return { value, offset: offset + 1 };
        } else if ((firstByte & 0xe0) === 0x60) { // Text string
            const length = firstByte & 0x1f;
            const bytes = new Uint8Array(view.buffer, offset + 1, length);
            const value = new TextDecoder().decode(bytes);
            return { value, offset: offset + 1 + length };
        } else if ((firstByte & 0xe0) === 0x40) { // Byte string
            const length = firstByte & 0x1f;
            const value = new Uint8Array(view.buffer, offset + 1, length);
            return { value, offset: offset + 1 + length };
        }
        
        throw new Error(`Unsupported CBOR value at offset ${offset}`);
    }
}

// Export for use in Node.js or browser
if (typeof module !== 'undefined' && module.exports) {
    module.exports = OCXReceiptVerifier;
} else if (typeof window !== 'undefined') {
    window.OCXReceiptVerifier = OCXReceiptVerifier;
}
