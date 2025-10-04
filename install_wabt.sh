#!/bin/bash
# Install WebAssembly Binary Toolkit (WABT)

echo "Installing WABT tools for WebAssembly compilation..."

# Option 1: Install from apt (easiest)
echo "Option 1: Installing from apt repository..."
sudo apt-get update
sudo apt-get install -y wabt

# Verify installation
if command -v wat2wasm &> /dev/null; then
    echo "✓ WABT installed successfully!"
    wat2wasm --version
    exit 0
fi

# Option 2: Build from source (if apt doesn't have it)
echo "Option 2: Building from source..."
cd /tmp
git clone --depth 1 --recursive https://github.com/WebAssembly/wabt
cd wabt
mkdir -p build && cd build
cmake .. -DBUILD_TESTS=OFF
cmake --build . --parallel $(nproc)
sudo cmake --install .

# Verify installation
if command -v wat2wasm &> /dev/null; then
    echo "✓ WABT built and installed successfully!"
    wat2wasm --version
else
    echo "✗ Installation failed. Please install manually."
    exit 1
fi
