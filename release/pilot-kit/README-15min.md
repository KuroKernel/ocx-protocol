# OCX Protocol Pilot Kit - 15 Minute Setup

## 🚀 Quick Start Guide

This pilot kit provides everything you need to deploy and test the OCX Protocol in production within 15 minutes.

## 📋 Prerequisites

- Docker and Docker Compose installed
- 4GB RAM and 2 CPU cores minimum
- Ports 80, 443, 8080, 3000, 9090 available

## ⚡ 15-Minute Deployment

### Step 1: Download and Extract (2 minutes)

```bash
# Download the pilot kit
wget https://github.com/your-org/ocx-protocol/releases/latest/download/ocx-pilot-kit.tar.gz

# Extract
tar -xzf ocx-pilot-kit.tar.gz
cd ocx-pilot-kit
```

### Step 2: Configure Environment (3 minutes)

```bash
# Copy environment template
cp env.prod.example .env.prod

# Edit configuration
nano .env.prod
```

**Required Configuration:**
```bash
# Database
OCX_DB_PASSWORD=your_secure_password_here

# API Keys
OCX_API_KEYS=admin:supersecretkey,youruser:yourkey

# Grafana
GRAFANA_PASSWORD=your_grafana_password_here
```

### Step 3: Deploy with Docker Compose (5 minutes)

```bash
# Start all services
docker compose -f docker-compose.prod.yml up -d

# Wait for services to be ready
docker compose -f docker-compose.prod.yml logs -f ocx-server
```

### Step 4: Verify Deployment (3 minutes)

```bash
# Check health
curl http://localhost:8080/health

# Check API documentation
open http://localhost:8080/swagger/

# Check monitoring
open http://localhost:3000  # Grafana (admin/your_grafana_password)
open http://localhost:9090  # Prometheus
```

### Step 5: Run Load Tests (2 minutes)

```bash
# Install k6 (if not installed)
# Ubuntu/Debian: sudo apt-get install k6
# macOS: brew install k6
# Or download from: https://k6.io/docs/getting-started/installation/

# Run load tests
./ops/run-loadtests.sh
```

## 🎯 What You Get

### Core Services
- **OCX API Server** (Port 8080) - Main API with Swagger UI
- **PostgreSQL Database** (Port 5432) - Receipt storage
- **Nginx Load Balancer** (Ports 80/443) - SSL termination and rate limiting
- **Prometheus** (Port 9090) - Metrics collection
- **Grafana** (Port 3000) - Monitoring dashboards

### API Endpoints
- `GET /health` - Health check
- `GET /swagger/` - Interactive API documentation
- `POST /api/v1/execute` - Execute artifacts and generate receipts
- `POST /verify` - Verify receipt signatures
- `GET /api/v1/receipts/` - List stored receipts
- `GET /metrics` - Prometheus metrics

### Monitoring & Observability
- **Real-time Metrics** - Request rates, response times, error rates
- **Health Dashboards** - System status and performance
- **Alert Rules** - Automated monitoring and notifications
- **Load Testing** - Performance validation scripts

## 🔧 Configuration Options

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OCX_DB_PASSWORD` | Database password | Required |
| `OCX_API_KEYS` | API keys (comma-separated) | Required |
| `GRAFANA_PASSWORD` | Grafana admin password | Required |
| `OCX_RATE_LIMIT_RPS` | Rate limit (requests/second) | 1000 |
| `OCX_REQUEST_TIMEOUT` | Request timeout | 30s |
| `OCX_BODY_SIZE_LIMIT` | Max request body size | 1MB |

### Scaling Options

```bash
# Scale OCX server
docker compose -f docker-compose.prod.yml up -d --scale ocx-server=3

# Scale with resource limits
docker compose -f docker-compose.prod.yml up -d --scale ocx-server=5
```

## 📊 Performance Expectations

### Load Testing Results
- **Standard Load**: 100 concurrent users
- **Response Time**: P95 < 200ms
- **Error Rate**: < 1%
- **Throughput**: 1000+ RPS

### Resource Usage
- **Memory**: 2GB total (1GB OCX + 512MB DB + 256MB monitoring)
- **CPU**: 2 cores (1 core OCX + 0.5 core DB + 0.5 core monitoring)
- **Storage**: 10GB (database + logs + metrics)

## 🔒 Security Features

- **API Key Authentication** - Secure API access
- **Rate Limiting** - DDoS protection
- **SSL/TLS Support** - Encrypted communication
- **Input Validation** - Request sanitization
- **Audit Logging** - Complete request tracking

## 🚨 Troubleshooting

### Common Issues

**1. Port Conflicts**
```bash
# Check port usage
netstat -tulpn | grep :8080
# Kill conflicting processes
sudo kill -9 <PID>
```

**2. Database Connection Issues**
```bash
# Check database logs
docker compose -f docker-compose.prod.yml logs postgres
# Restart database
docker compose -f docker-compose.prod.yml restart postgres
```

**3. High Memory Usage**
```bash
# Check resource usage
docker stats
# Scale down if needed
docker compose -f docker-compose.prod.yml up -d --scale ocx-server=1
```

### Health Checks

```bash
# API Health
curl http://localhost:8080/health

# Database Health
docker compose -f docker-compose.prod.yml exec postgres pg_isready

# All Services Status
docker compose -f docker-compose.prod.yml ps
```

## 📈 Next Steps

### Production Deployment
1. **SSL Certificates** - Configure real SSL certificates
2. **Domain Setup** - Point your domain to the server
3. **Backup Strategy** - Set up automated database backups
4. **Monitoring** - Configure alerting and notifications
5. **Scaling** - Add load balancers and multiple instances

### Enterprise Features
1. **High Availability** - Multi-region deployment
2. **Advanced Security** - WAF, DDoS protection
3. **Compliance** - Audit trails, data retention
4. **Integration** - CI/CD pipelines, monitoring tools

## 📞 Support

- **Documentation**: https://docs.ocx.dev
- **API Reference**: http://localhost:8080/swagger/
- **GitHub Issues**: https://github.com/your-org/ocx-protocol/issues
- **Email Support**: support@ocx.dev

## 🎉 Success!

You now have a fully functional OCX Protocol deployment with:
- ✅ Production-ready API server
- ✅ Secure database with receipts
- ✅ Real-time monitoring and metrics
- ✅ Load testing and performance validation
- ✅ Complete documentation and support

**Ready for enterprise pilots and production workloads!**
