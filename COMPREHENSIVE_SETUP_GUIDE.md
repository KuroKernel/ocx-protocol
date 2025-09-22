# OCX Protocol - Comprehensive Setup Guide for Non-Technical Users

## 🎯 **OVERVIEW**

This guide will help you set up the complete OCX Protocol infrastructure ecosystem on your Pop!_OS system. The OCX Protocol is an enterprise-grade cryptographic verification system that works across all infrastructure layers.

**What you'll get:**
- Complete multi-language build system
- 6 production-ready adapters
- Automated testing and deployment
- Performance monitoring
- Security scanning

## 📋 **PREREQUISITES SETUP**

### **Step 1: Update Your System**

```bash
# Update your Pop!_OS system
sudo apt update && sudo apt upgrade -y
```

### **Step 2: Install Required Software**

```bash
# Install development tools
sudo apt install -y curl wget git build-essential cmake pkg-config

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
newgrp docker

# Install Docker Compose
sudo apt install -y docker-compose

# Install Node.js 20 LTS
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Install Java 17
sudo apt install -y openjdk-17-jdk maven

# Install Rust (if not already installed)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source $HOME/.cargo/env

# Install Go (if not already installed)
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Terraform
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform

# Install Python and pip
sudo apt install -y python3 python3-pip python3-venv

# Install jq for JSON processing
sudo apt install -y jq
```

### **Step 3: Verify Installations**

```bash
# Check all installations
echo "=== Installation Verification ==="
docker --version          # Should show Docker version
node --version            # Should show Node.js v20.x
java --version            # Should show Java 17
rustc --version           # Should show Rust version
go version               # Should show Go 1.21.x
terraform --version      # Should show Terraform version
python3 --version        # Should show Python 3.x
jq --version             # Should show jq version
echo "=== All installations verified! ==="
```

## 🚀 **PROJECT SETUP**

### **Step 4: Navigate to Project Directory**

```bash
# Navigate to your OCX project directory
cd /home/kurokernel/Desktop/AXIS/ocx-protocol

# Verify you're in the right directory
pwd
ls -la
```

### **Step 5: Install Dependencies**

```bash
# Install all language dependencies
make install-deps
```

This command will:
- Install Rust dependencies
- Download Go modules
- Install Node.js packages
- Resolve Java Maven dependencies
- Install Python packages

### **Step 6: Build Everything**

```bash
# Build all components in the correct order
make build-all
```

This will build:
- Rust verifier (high-performance cryptographic verification)
- Go server (main OCX protocol server)
- C++ Envoy filter (network traffic management)
- Node.js GitHub Action (CI/CD integration)
- Go Terraform provider (infrastructure as code)
- Java Kafka interceptor (message streaming)

### **Step 7: Run All Tests**

```bash
# Run comprehensive test suite
make test-all
```

This will:
- Run unit tests for all components
- Run integration tests
- Run performance benchmarks
- Validate security

## 🏃 **STARTING THE SYSTEM**

### **Step 8: Start Development Environment**

```bash
# Start all services with Docker Compose
make start-dev-env
```

This will start:
- OCX Server (port 8080)
- PostgreSQL database (port 5432)
- Envoy proxy (ports 8000, 8001)
- Kafka message broker (port 9092)
- Zookeeper (port 2181)

### **Step 9: Verify System Health**

```bash
# Check that all services are running
make health-check
```

You should see:
- ✅ OCX server responding
- ✅ Envoy proxy responding
- ✅ Kafka responding

### **Step 10: Test Individual Components**

```bash
# Test OCX Server
curl http://localhost:8080/status
# Should return: {"status":"healthy","verifier":"0.1.0"}

# Test Envoy Proxy
curl http://localhost:8000/health
# Should return health status

# Test Kafka
docker exec -it deployment_kafka_1 kafka-topics --list --bootstrap-server localhost:9092
# Should list available topics
```

## 📊 **PERFORMANCE MONITORING**

### **Step 11: Run Performance Tests**

```bash
# Run comprehensive performance benchmarks
make benchmark
```

This will test:
- OCX Server response time (target: <1ms)
- Envoy filter overhead (target: <10ms)
- GitHub Action execution (target: <30s)
- Terraform operations (target: <5s)
- Kafka message processing (target: <1ms)

### **Step 12: Monitor Performance**

```bash
# Monitor real-time performance
make monitor-performance
```

## 🔒 **SECURITY VALIDATION**

### **Step 13: Run Security Scan**

```bash
# Run comprehensive security scan
make security-scan
```

This will:
- Scan Docker images for vulnerabilities
- Audit Rust dependencies
- Audit Node.js packages
- Audit Java Maven dependencies
- Check for security issues

## 🧪 **INTEGRATION TESTING**

### **Step 14: Run Integration Tests**

```bash
# Run cross-adapter integration tests
make integration-test
```

This will:
- Test communication between all adapters
- Validate end-to-end workflows
- Test failure scenarios
- Verify performance under load

## 📋 **DAILY DEVELOPMENT WORKFLOW**

### **Starting Work Each Day**

```bash
# Start all services
make start-dev-env

# Check system health
make health-check

# View system logs if needed
make logs
```

### **During Development**

```bash
# Build specific component
make build-rust         # Rust verifier
make build-go           # Go server
make build-envoy        # Envoy filter
make build-github       # GitHub action
make build-terraform    # Terraform provider
make build-kafka        # Kafka interceptor

# Run specific tests
make test-rust          # Test Rust verifier
make test-go            # Test Go components
make test-envoy         # Test Envoy filter
make test-github        # Test GitHub action
make test-terraform     # Test Terraform provider
make test-kafka         # Test Kafka interceptor

# Run integration tests
make integration-test
```

### **End of Day Cleanup**

```bash
# Stop all services cleanly
make stop-dev-env

# Clean temporary files
make clean
```

## 🚀 **DEPLOYMENT COMMANDS**

### **Local Deployment**

```bash
# Deploy to local environment
make deploy-local
```

### **Staging Deployment**

```bash
# Deploy to staging environment
make deploy-staging
```

### **Production Deployment**

```bash
# Deploy to production environment
make deploy-prod
```

## 🔧 **TROUBLESHOOTING**

### **If Builds Fail**

```bash
# Clean everything and rebuild
make clean-all
make install-deps
make build-all
```

### **If Services Won't Start**

```bash
# Check Docker status
docker system prune
docker-compose -f deployment/docker-compose.yml down
docker-compose -f deployment/docker-compose.yml up -d

# Check logs
make logs
```

### **If Tests Fail**

```bash
# Check specific service logs
docker-compose -f deployment/docker-compose.yml logs ocx-server
docker-compose -f deployment/docker-compose.yml logs envoy
docker-compose -f deployment/docker-compose.yml logs kafka
```

### **If Performance is Slow**

```bash
# Check system resources
htop
docker stats

# Restart services
make stop-dev-env
make start-dev-env
```

## 📚 **COMMON COMMANDS REFERENCE**

### **Build Commands**
- `make build-all` - Build everything
- `make build-rust` - Build Rust verifier
- `make build-go` - Build Go components
- `make build-envoy` - Build Envoy filter
- `make build-github` - Build GitHub action
- `make build-terraform` - Build Terraform provider
- `make build-kafka` - Build Kafka interceptor

### **Test Commands**
- `make test-all` - Run all tests
- `make test-rust` - Test Rust verifier
- `make test-go` - Test Go components
- `make test-envoy` - Test Envoy filter
- `make test-github` - Test GitHub action
- `make test-terraform` - Test Terraform provider
- `make test-kafka` - Test Kafka interceptor
- `make integration-test` - Run integration tests

### **Development Commands**
- `make install-deps` - Install all dependencies
- `make start-dev-env` - Start development environment
- `make stop-dev-env` - Stop development environment
- `make health-check` - Check system health
- `make logs` - Show system logs
- `make monitor-performance` - Monitor performance

### **Deployment Commands**
- `make deploy-local` - Deploy locally
- `make deploy-staging` - Deploy to staging
- `make deploy-prod` - Deploy to production

### **Maintenance Commands**
- `make clean-all` - Clean all build artifacts
- `make security-scan` - Run security scan
- `make benchmark` - Run performance benchmarks

## 🎯 **SUCCESS CRITERIA**

Your setup is successful when:

✅ **All components build without errors**
```bash
make build-all  # Should complete successfully
```

✅ **All tests pass**
```bash
make test-all  # Should show 100% pass rate
```

✅ **All services start and respond**
```bash
make health-check  # Should show all green checkmarks
```

✅ **Performance targets are met**
```bash
make benchmark  # Should show performance within targets
```

✅ **Security scan passes**
```bash
make security-scan  # Should show no critical vulnerabilities
```

## 🚀 **NEXT STEPS**

Once everything is running successfully:

1. **Development**: Use specific build commands for your work
2. **Testing**: Run tests regularly during development
3. **Monitoring**: Check performance metrics regularly
4. **Deployment**: Use staging environment before production
5. **Security**: Run security scans regularly

## 💡 **TIPS FOR SUCCESS**

1. **Always run `make health-check`** after starting services
2. **Use `make logs`** to troubleshoot issues
3. **Run `make clean-all`** if you encounter build issues
4. **Check `make monitor-performance`** to ensure good performance
5. **Use `make security-scan`** regularly for security validation

## 🆘 **GETTING HELP**

If you encounter issues:

1. **Check the logs**: `make logs`
2. **Verify health**: `make health-check`
3. **Clean and rebuild**: `make clean-all && make build-all`
4. **Check system resources**: `htop` and `docker stats`
5. **Restart services**: `make stop-dev-env && make start-dev-env`

## 🎉 **CONGRATULATIONS!**

You now have a complete, enterprise-grade OCX Protocol infrastructure ecosystem running on your system. The system is designed to "just work" after initial setup, with all complexity handled by the Makefile commands.

**Your OCX Protocol is ready for production use! 🚀**
