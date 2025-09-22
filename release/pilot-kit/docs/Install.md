# OCX Protocol Installation Guide

## Prerequisites

### System Requirements
- **OS**: Linux x86_64 (Ubuntu 20.04+, CentOS 8+, RHEL 8+)
- **Memory**: 2GB RAM minimum, 4GB recommended
- **CPU**: 2 cores minimum, 4 cores recommended
- **Storage**: 10GB free space minimum
- **Network**: Internet access for Docker images

### Software Requirements
- **Docker**: 20.10+ (for containerized deployment)
- **Docker Compose**: 2.0+ (for multi-container deployment)
- **PostgreSQL**: 15+ (or use provided compose service)
- **curl**: For health checks and testing

### Optional Requirements
- **Kubernetes**: 1.20+ (for Helm deployment)
- **Helm**: 3.0+ (for Kubernetes deployment)
- **Prometheus**: For monitoring and alerting

## Installation Methods

### Method 1: Docker Compose (Recommended)

#### Step 1: Download Pilot Kit
```bash
# Download and extract pilot kit
wget https://github.com/ocx-protocol/ocx/releases/download/v1.0.0-rc.1-pilot1/ocx-pilot-kit.tar.gz
tar -xzf ocx-pilot-kit.tar.gz
cd ocx-pilot-kit
```

#### Step 2: Set Environment Variables
```bash
# Required environment variables
export OCX_DB_URL="postgres://ocx:ocx_password@postgres:5432/ocx?sslmode=disable"
export OCX_MAX_BODY_BYTES=$((1024*1024))  # 1MB
export OCX_RATE_LIMIT_RPS="100"
export OCX_LOG_LEVEL="info"
export OCX_METRICS_ENABLED="true"
```

#### Step 3: Start Services
```bash
# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f ocx-api
```

#### Step 4: Verify Installation
```bash
# Health check
curl -s http://localhost:8080/health

# Readiness check
curl -s http://localhost:8080/readyz

# Smoke test
bash scripts/smoke.sh
```

### Method 2: Kubernetes with Helm

#### Step 1: Install Helm
```bash
# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Verify installation
helm version
```

#### Step 2: Add OCX Helm Repository
```bash
# Add repository
helm repo add ocx https://charts.ocx-protocol.com
helm repo update

# Verify repository
helm search repo ocx
```

#### Step 3: Create Values File
```yaml
# values.yaml
api:
  image:
    repository: ocx-protocol
    tag: v1.0.0-rc.1-pilot1
  service:
    port: 8080
  resources:
    requests:
      memory: "512Mi"
      cpu: "250m"
    limits:
      memory: "1Gi"
      cpu: "500m"
  env:
    DATABASE_URL: "postgres://ocx:ocx_password@postgres:5432/ocx?sslmode=disable"
    OCX_MAX_BODY_BYTES: "1048576"
    RATE_LIMIT_RPS: "100"
    LOG_LEVEL: "info"
    METRICS_ENABLED: "true"

postgres:
  enabled: true
  auth:
    postgresPassword: "ocx_password"
    username: "ocx"
    password: "ocx_password"
    database: "ocx"
  persistence:
    size: "10Gi"
```

#### Step 4: Install OCX
```bash
# Install OCX
helm install ocx ocx/ocx -f values.yaml

# Check installation
helm status ocx
kubectl get pods
```

#### Step 5: Verify Installation
```bash
# Port forward to access service
kubectl port-forward svc/ocx-api 8080:8080

# Health check
curl -s http://localhost:8080/health

# Smoke test
bash scripts/smoke.sh
```

### Method 3: Manual Installation

#### Step 1: Install Dependencies
```bash
# Install Go 1.21+
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install PostgreSQL
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib

# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

#### Step 2: Setup Database
```bash
# Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE ocx;
CREATE USER ocx WITH PASSWORD 'ocx_password';
GRANT ALL PRIVILEGES ON DATABASE ocx TO ocx;
\q
EOF

# Create tables
psql -h localhost -U ocx -d ocx -f scripts/schema.sql
```

#### Step 3: Build and Install
```bash
# Clone repository
git clone https://github.com/ocx-protocol/ocx.git
cd ocx

# Build binary
go build -o ocx-api ./cmd/api-server

# Install binary
sudo cp ocx-api /usr/local/bin/
sudo chmod +x /usr/local/bin/ocx-api
```

#### Step 4: Create Systemd Service
```ini
# /etc/systemd/system/ocx.service
[Unit]
Description=OCX Protocol API Server
After=network.target postgresql.service

[Service]
Type=simple
User=ocx
Group=ocx
WorkingDirectory=/opt/ocx
ExecStart=/usr/local/bin/ocx-api
Environment=DATABASE_URL=postgres://ocx:ocx_password@localhost:5432/ocx?sslmode=disable
Environment=PORT=8080
Environment=LOG_LEVEL=info
Environment=METRICS_ENABLED=true
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

#### Step 5: Start Service
```bash
# Create user and directory
sudo useradd -r -s /bin/false ocx
sudo mkdir -p /opt/ocx
sudo chown ocx:ocx /opt/ocx

# Start service
sudo systemctl daemon-reload
sudo systemctl start ocx
sudo systemctl enable ocx

# Check status
sudo systemctl status ocx
```

## Configuration

### Environment Variables

#### Required Variables
```bash
# Database connection
export DATABASE_URL="postgres://ocx:ocx_password@postgres:5432/ocx?sslmode=disable"

# Server configuration
export PORT="8080"
export LOG_LEVEL="info"
export METRICS_ENABLED="true"
```

#### Optional Variables
```bash
# Security configuration
export OCX_MAX_BODY_BYTES="1048576"  # 1MB
export RATE_LIMIT_RPS="100"
export KEYSTORE_DIR="/app/keys"

# Performance configuration
export READ_HEADER_TIMEOUT="3s"
export READ_TIMEOUT="5s"
export WRITE_TIMEOUT="15s"
export IDLE_TIMEOUT="60s"

# Database configuration
export DB_TYPE="postgres"
export MAX_RECEIPT_SIZE="1048576"  # 1MB
```

### Configuration Files

#### Docker Compose
```yaml
# docker-compose.yml
version: '3.8'
services:
  ocx-api:
    image: ocx-protocol:v1.0.0-rc.1-pilot1
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://ocx:ocx_password@postgres:5432/ocx?sslmode=disable
      - PORT=8080
      - LOG_LEVEL=info
      - METRICS_ENABLED=true
    depends_on:
      - postgres
    restart: unless-stopped

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=ocx
      - POSTGRES_USER=ocx
      - POSTGRES_PASSWORD=ocx_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  postgres_data:
```

#### Helm Values
```yaml
# helm/values.yaml
api:
  image:
    repository: ocx-protocol
    tag: v1.0.0-rc.1-pilot1
  service:
    type: ClusterIP
    port: 8080
  env:
    DATABASE_URL: "postgres://ocx:ocx_password@postgres:5432/ocx?sslmode=disable"
    PORT: "8080"
    LOG_LEVEL: "info"
    METRICS_ENABLED: "true"
  resources:
    requests:
      memory: "512Mi"
      cpu: "250m"
    limits:
      memory: "1Gi"
      cpu: "500m"

postgres:
  enabled: true
  auth:
    postgresPassword: "ocx_password"
    username: "ocx"
    password: "ocx_password"
    database: "ocx"
  persistence:
    enabled: true
    size: "10Gi"
```

## Verification

### Health Checks
```bash
# Basic health check
curl -s http://localhost:8080/health | jq

# Readiness check
curl -s http://localhost:8080/readyz | jq

# Liveness check
curl -s http://localhost:8080/livez | jq

# Metrics endpoint
curl -s http://localhost:8080/metrics
```

### Smoke Test
```bash
# Run smoke test
bash scripts/smoke.sh

# Expected output:
# ✅ Health: healthy
# ✅ Execute: Got receipt blob
# ✅ Verify: Receipt validation working
# ✅ Readiness: ready
# ✅ Liveness: alive
# ✅ Metrics: X lines of data
# ✅ Idempotency: Identical responses for same key
# ✅ Idempotency Mismatch: Correctly rejected with E007
# ✅ Resource Limits: Correctly rejected with E001
```

### Load Test
```bash
# Run load test
bash scripts/load_test.sh 200 60

# Expected output:
# Total Requests: 12000
# Successful: 12000
# Errors: 0
# Error Rate: 0.00%
# P50: 5.2ms
# P95: 12.8ms
# P99: 18.5ms
# ✅ SLO COMPLIANCE: PASSED
```

## Troubleshooting

### Common Issues

#### Service Won't Start
```bash
# Check logs
docker-compose logs ocx-api
# or
journalctl -u ocx -f

# Check database connectivity
psql $DATABASE_URL -c "SELECT 1;"

# Check port availability
netstat -tlnp | grep 8080
```

#### Database Connection Issues
```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Check database exists
psql -h localhost -U ocx -d ocx -c "SELECT 1;"

# Check connection string
echo $DATABASE_URL
```

#### Performance Issues
```bash
# Check resource usage
docker stats ocx-api
# or
top -p $(pgrep -f ocx-api)

# Check database performance
psql $DATABASE_URL -c "SELECT * FROM pg_stat_activity;"

# Run load test
bash scripts/load_test.sh 100 30
```

#### Verification Failures
```bash
# Check key store
ls -la /app/keys/

# Check logs for signature errors
docker-compose logs ocx-api | grep -i signature

# Test with known good receipt
bash scripts/smoke.sh
```

### Debug Commands
```bash
# Check service status
docker-compose ps
# or
kubectl get pods
# or
sudo systemctl status ocx

# Check logs
docker-compose logs -f ocx-api
# or
kubectl logs -f deployment/ocx-api
# or
journalctl -u ocx -f

# Check metrics
curl -s http://localhost:8080/metrics | grep ocx_

# Check database
psql $DATABASE_URL -c "SELECT COUNT(*) FROM receipts;"
```

## Next Steps

### Monitoring Setup
1. **Prometheus**: Configure scraping from `/metrics` endpoint
2. **Grafana**: Import provided dashboards
3. **Alerting**: Configure alert rules from `docs/SLOs.md`

### Production Hardening
1. **SSL/TLS**: Configure HTTPS with valid certificates
2. **Firewall**: Restrict access to necessary ports
3. **Backups**: Configure automated backup procedures
4. **Logging**: Set up centralized logging

### Integration
1. **API Integration**: Use provided OpenAPI specification
2. **Client SDKs**: Use provided client libraries
3. **CI/CD**: Integrate verification into build pipelines
4. **Monitoring**: Add custom metrics and alerts

## Support

### Documentation
- **API Reference**: `/api/openapi.yaml`
- **Runbooks**: `docs/Runbooks.md`
- **Security**: `docs/Security.md`
- **Operations**: `docs/Operations.md`

### Getting Help
- **GitHub Issues**: https://github.com/ocx-protocol/ocx/issues
- **Documentation**: https://docs.ocx-protocol.com
- **Support**: support@ocx-protocol.com

### Response Times
- **Critical Issues**: 15 minutes
- **High Priority**: 1 hour
- **Medium Priority**: 4 hours
- **Low Priority**: 24 hours
