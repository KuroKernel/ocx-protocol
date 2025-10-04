#!/bin/bash
set -euo pipefail

# Load the private key from the keystore
PRIVATE_KEY_HEX=$(cat keys/8d42905855072bf4.key)
PUBLIC_KEY_HEX=$(cat keys/8d42905855072bf4.json | jq -r .public_key)

# Convert to base64 for the verifier
PUB_B64=$(echo "$PUBLIC_KEY_HEX" | xxd -r -p | base64 -w0)

echo "Private key: ${PRIVATE_KEY_HEX:0:16}..."
echo "Public key: ${PUBLIC_KEY_HEX:0:16}..."
echo "Public key (base64): ${PUB_B64:0:16}..."

# Generate receipt using the CLI (it will use a random key, but we'll work around this)
./ocx execute artifacts/hello.sh --env OCX_INPUT=arg1 --output out/r1.cbor >/dev/null 2>&1

# The CLI generates a random key, so we need to extract the public key from the receipt
# Let's use the API instead to get a receipt with a known key
echo "Generated receipt with CLI (random key)"
ls -la out/r1.cbor

# For now, let's test the standalone verifier with a mock receipt
# We'll create a simple test receipt manually
echo "Testing standalone verifier..."
