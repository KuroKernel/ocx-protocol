#!/bin/bash
# OCX Protocol - Automated Setup Script for Pop!_OS
# This script automates the complete setup process

set -e  # Exit on any error

echo "🚀 OCX Protocol - Automated Setup Script"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install package if not exists
install_if_missing() {
    local package=$1
    local install_cmd=$2
    
    if command_exists "$package"; then
        print_success "$package is already installed"
    else
        print_status "Installing $package..."
        eval "$install_cmd"
        print_success "$package installed successfully"
    fi
}

# Check if running on Pop!_OS
if ! grep -q "Pop!_OS" /etc/os-release; then
    print_warning "This script is optimized for Pop!_OS. Other distributions may work but are not guaranteed."
fi

print_status "Starting OCX Protocol setup..."

# Step 1: Update system
print_status "Updating system packages..."
sudo apt update && sudo apt upgrade -y
print_success "System updated"

# Step 2: Install basic dependencies
print_status "Installing basic dependencies..."
sudo apt install -y curl wget git build-essential cmake pkg-config jq python3 python3-pip python3-venv
print_success "Basic dependencies installed"

# Step 3: Install Docker
print_status "Installing Docker..."
if command_exists docker; then
    print_success "Docker is already installed"
else
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker $USER
    rm get-docker.sh
    print_success "Docker installed"
fi

# Step 4: Install Docker Compose
install_if_missing docker-compose "sudo apt install -y docker-compose"

# Step 5: Install Node.js 20 LTS
print_status "Installing Node.js 20 LTS..."
if command_exists node; then
    NODE_VERSION=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
    if [ "$NODE_VERSION" -ge 20 ]; then
        print_success "Node.js $NODE_VERSION is already installed"
    else
        print_status "Upgrading Node.js to version 20..."
        curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
        sudo apt install -y nodejs
        print_success "Node.js 20 installed"
    fi
else
    curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    sudo apt install -y nodejs
    print_success "Node.js 20 installed"
fi

# Step 6: Install Java 17
print_status "Installing Java 17..."
if command_exists java; then
    JAVA_VERSION=$(java -version 2>&1 | head -n 1 | cut -d'"' -f2 | cut -d'.' -f1)
    if [ "$JAVA_VERSION" -ge 17 ]; then
        print_success "Java $JAVA_VERSION is already installed"
    else
        print_status "Installing Java 17..."
        sudo apt install -y openjdk-17-jdk maven
        print_success "Java 17 installed"
    fi
else
    sudo apt install -y openjdk-17-jdk maven
    print_success "Java 17 installed"
fi

# Step 7: Install Rust
print_status "Installing Rust..."
if command_exists rustc; then
    print_success "Rust is already installed"
else
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
    source $HOME/.cargo/env
    print_success "Rust installed"
fi

# Step 8: Install Go
print_status "Installing Go..."
if command_exists go; then
    GO_VERSION=$(go version | cut -d' ' -f3 | cut -d'o' -f2 | cut -d'.' -f1-2)
    if [ "$GO_VERSION" = "go1.21" ]; then
        print_success "Go $GO_VERSION is already installed"
    else
        print_status "Installing Go 1.21..."
        wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
        sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        export PATH=$PATH:/usr/local/go/bin
        rm go1.21.5.linux-amd64.tar.gz
        print_success "Go 1.21 installed"
    fi
else
    wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
    rm go1.21.5.linux-amd64.tar.gz
    print_success "Go 1.21 installed"
fi

# Step 9: Install Terraform
install_if_missing terraform "wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg && echo 'deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main' | sudo tee /etc/apt/sources.list.d/hashicorp.list && sudo apt update && sudo apt install terraform"

# Step 10: Verify installations
print_status "Verifying installations..."
echo "=== Installation Verification ==="
docker --version
node --version
java --version
rustc --version
go version
terraform --version
python3 --version
jq --version
echo "=== All installations verified! ==="

# Step 11: Install project dependencies
print_status "Installing project dependencies..."
if [ -f "Makefile" ]; then
    make install-deps
    print_success "Project dependencies installed"
else
    print_error "Makefile not found. Please run this script from the OCX protocol directory."
    exit 1
fi

# Step 12: Build the project
print_status "Building OCX Protocol..."
make build-all
print_success "OCX Protocol built successfully"

# Step 13: Run tests
print_status "Running tests..."
make test-all
print_success "All tests passed"

# Step 14: Start development environment
print_status "Starting development environment..."
make start-dev-env
print_success "Development environment started"

# Step 15: Wait for services and verify
print_status "Waiting for services to be ready..."
sleep 30

print_status "Verifying system health..."
make health-check

print_success "🎉 OCX Protocol setup completed successfully!"
echo ""
echo "Your OCX Protocol infrastructure is now running with:"
echo "  - OCX Server: http://localhost:8080"
echo "  - Envoy Proxy: http://localhost:8000"
echo "  - Kafka: localhost:9092"
echo "  - PostgreSQL: localhost:5432"
echo ""
echo "Common commands:"
echo "  make health-check     - Check system health"
echo "  make logs            - View system logs"
echo "  make stop-dev-env    - Stop all services"
echo "  make start-dev-env   - Start all services"
echo "  make monitor-performance - Check performance"
echo ""
echo "For detailed usage, see COMPREHENSIVE_SETUP_GUIDE.md"
echo ""
print_success "Setup complete! 🚀"
