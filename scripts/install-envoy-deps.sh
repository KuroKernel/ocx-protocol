#!/bin/bash
set -euo pipefail

echo "Installing Envoy development dependencies..."

# Install Bazel
curl -fsSL https://bazel.build/bazel-release.pub.gpg | gpg --dearmor > bazel.gpg
sudo mv bazel.gpg /etc/apt/trusted.gpg.d/
echo "deb [arch=amd64] https://storage.googleapis.com/bazel-apt stable jdk1.8" | sudo tee /etc/apt/sources.list.d/bazel.list
sudo apt update && sudo apt install bazel

# Install Envoy build dependencies
sudo apt install -y \
    build-essential \
    libtool \
    cmake \
    automake \
    autoconf \
    ninja-build \
    curl \
    unzip \
    virtualenv \
    python3-dev \
    python3-pip

# Clone Envoy source for headers
cd /tmp
git clone https://github.com/envoyproxy/envoy.git --depth=1 -b v1.27.0
sudo mkdir -p /usr/local/include/envoy
sudo cp -r envoy/include/* /usr/local/include/envoy/
sudo cp -r envoy/source/* /usr/local/include/envoy/

echo "Envoy dependencies installed successfully"
