# OCX Protocol - Quick Reference Card

## 🚀 **One-Command Setup**

```bash
# Automated setup (run once)
./setup.sh

# Manual setup
make install-deps && make build-all && make start-dev-env
```

## 📋 **Daily Commands**

### **Starting Work**
```bash
make start-dev-env    # Start all services
make health-check     # Verify everything is working
```

### **During Development**
```bash
make build-all        # Build everything
make test-all         # Run all tests
make logs            # View system logs
```

### **End of Day**
```bash
make stop-dev-env     # Stop all services
make clean           # Clean temporary files
```

## 🔧 **Component-Specific Commands**

### **Build Individual Components**
```bash
make build-rust       # Rust verifier
make build-go         # Go server
make build-envoy      # Envoy filter
make build-github     # GitHub action
make build-terraform  # Terraform provider
make build-kafka      # Kafka interceptor
```

### **Test Individual Components**
```bash
make test-rust        # Test Rust verifier
make test-go          # Test Go components
make test-envoy       # Test Envoy filter
make test-github      # Test GitHub action
make test-terraform   # Test Terraform provider
make test-kafka       # Test Kafka interceptor
```

## 🏥 **Health & Monitoring**

### **System Health**
```bash
make health-check     # Check all services
make logs            # View logs
make monitor-performance  # Performance metrics
```

### **Service URLs**
- **OCX Server**: http://localhost:8080/status
- **Envoy Proxy**: http://localhost:8000/health
- **Kafka**: localhost:9092
- **PostgreSQL**: localhost:5432

## 🔒 **Security & Testing**

### **Security**
```bash
make security-scan    # Run security scan
```

### **Testing**
```bash
make test-all         # All tests
make integration-test # Integration tests
make benchmark        # Performance tests
```

## 🚀 **Deployment**

### **Deployment Commands**
```bash
make deploy-local     # Local deployment
make deploy-staging   # Staging deployment
make deploy-prod      # Production deployment
```

## 🛠️ **Troubleshooting**

### **Common Issues**
```bash
# If builds fail
make clean-all && make build-all

# If services won't start
make stop-dev-env && make start-dev-env

# If tests fail
make logs  # Check specific service logs
```

### **Reset Everything**
```bash
make stop-dev-env
make clean-all
make install-deps
make build-all
make start-dev-env
```

## 📊 **Performance Targets**

| Component | Target | Command to Check |
|-----------|--------|------------------|
| OCX Server | <1ms | `make monitor-performance` |
| Envoy Filter | <10ms | `make monitor-performance` |
| GitHub Action | <30s | `make benchmark` |
| Terraform | <5s | `make benchmark` |
| Kafka | <1ms | `make benchmark` |

## 🎯 **Success Indicators**

✅ **System is healthy when:**
- `make health-check` shows all green
- All services respond to their health endpoints
- Performance metrics are within targets
- No critical security vulnerabilities

## 🆘 **Emergency Commands**

### **Quick Restart**
```bash
make stop-dev-env && make start-dev-env
```

### **Full Reset**
```bash
make stop-dev-env
docker system prune -f
make clean-all
make install-deps
make build-all
make start-dev-env
```

### **Check Resources**
```bash
htop                 # System resources
docker stats         # Container resources
make logs           # Service logs
```

## 📚 **Documentation**

- **Complete Setup Guide**: `COMPREHENSIVE_SETUP_GUIDE.md`
- **Build System Guide**: `docs/BUILD_SYSTEM_GUIDE.md`
- **API Documentation**: `docs/API_REFERENCE.md`
- **Troubleshooting**: `docs/TROUBLESHOOTING.md`

---

**Remember: The system is designed to "just work" after setup. Use `make health-check` to verify everything is running correctly! 🚀**
