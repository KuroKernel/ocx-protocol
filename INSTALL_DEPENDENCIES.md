# OCX Protocol - Dependency Installation Guide

## 🔧 **MANUAL INSTALLATION REQUIRED**

Since sudo privileges are required for system-level installations, please run these commands manually in your terminal.

## **Step 1: Install Maven (Java Build Tool)**

```bash
# Update package list
sudo apt update

# Install Maven
sudo apt install -y maven

# Verify installation
mvn --version
```

## **Step 2: Install Envoy Dependencies**

```bash
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
```

## **Step 3: Fix Docker Permissions**

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Set proper permissions
sudo chmod 666 /var/run/docker.sock

# Restart Docker service
sudo systemctl restart docker

# Note: You may need to log out and back in for group changes to take effect
```

## **Step 4: Install Go 1.21 (if needed)**

If you need a newer Go version for the Terraform provider:

```bash
# Download and install Go 1.21
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

## **Step 5: Test All Components**

After installing dependencies, run these commands to test everything:

```bash
cd /home/kurokernel/Desktop/AXIS/ocx-protocol

# Fix all components
make fix-all

# Build everything
make build-all

# Test everything
make test-all

# Check system health
make health-check
```

## **Alternative: Use Docker for Development**

If you prefer not to install system dependencies, you can use Docker for development:

```bash
# Build the complete Docker image
docker build -f Dockerfile.complete -t ocx-protocol:latest .

# Run the complete environment
docker-compose -f deployment/docker-compose.yml up -d
```

## **Verification Commands**

After installation, verify each component:

```bash
# Check Maven
mvn --version

# Check Bazel
bazel --version

# Check Docker
docker --version
docker ps

# Check Go (if updated)
go version

# Test OCX components
make build-rust
make build-go
make build-github
make build-envoy
make build-terraform
make build-kafka
```

## **Troubleshooting**

If you encounter issues:

1. **Docker permissions**: Log out and back in after adding user to docker group
2. **Go version conflicts**: Use `go env GOROOT` to check current Go installation
3. **Maven not found**: Ensure `/usr/bin/mvn` is in your PATH
4. **Envoy headers missing**: Check that `/usr/local/include/envoy` exists

## **Quick Test**

Once all dependencies are installed, run this single command to test everything:

```bash
make fix-all && make build-all && make test-all && make health-check
```

This will install, build, test, and verify all OCX Protocol components.
