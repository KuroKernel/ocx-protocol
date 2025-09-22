#!/bin/bash
set -euo pipefail

echo "Installing complete Envoy development headers..."

# Remove incomplete installation
sudo rm -rf /usr/local/include/envoy

# Install Envoy from official repository with full headers
cd /tmp
wget https://github.com/envoyproxy/envoy/archive/refs/tags/v1.27.7.tar.gz
tar -xzf v1.27.7.tar.gz
cd envoy-1.27.7

# Install required build dependencies
sudo apt install -y \
    libc++-dev \
    libc++abi-dev \
    pkg-config \
    zip \
    g++ \
    zlib1g-dev \
    unzip \
    python3

# Create proper header structure
sudo mkdir -p /usr/local/include/envoy
sudo cp -r include/* /usr/local/include/envoy/
sudo cp -r source /usr/local/include/envoy/

# Install Abseil (required by Envoy)
cd /tmp
git clone https://github.com/abseil/abseil-cpp.git --depth=1 -b 20230802.1
cd abseil-cpp
mkdir build && cd build
cmake .. -DCMAKE_POSITION_INDEPENDENT_CODE=TRUE
make -j$(nproc)
sudo make install

echo "Complete Envoy headers installed"
