#!/bin/bash

# OCX Protocol Pilot Kit Packaging Script
# This script creates a complete pilot kit package for distribution

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
PACKAGE_NAME="ocx-pilot-kit-$(date +%Y%m%d_%H%M%S)"
PACKAGE_DIR="/tmp/$PACKAGE_NAME"

echo -e "${BLUE}📦 OCX Protocol Pilot Kit Packaging${NC}"
echo "====================================="
echo "Project Root: $PROJECT_ROOT"
echo "Package Name: $PACKAGE_NAME"
echo "Package Directory: $PACKAGE_DIR"
echo ""

# Function to create package directory
create_package_dir() {
    echo -e "${YELLOW}📁 Creating package directory...${NC}"
    rm -rf "$PACKAGE_DIR"
    mkdir -p "$PACKAGE_DIR"
    echo -e "${GREEN}✅ Package directory created${NC}"
}

# Function to copy pilot kit files
copy_pilot_kit_files() {
    echo -e "${YELLOW}📋 Copying pilot kit files...${NC}"
    
    # Copy pilot kit files
    cp -r "$SCRIPT_DIR"/* "$PACKAGE_DIR/"
    
    # Copy Docker Compose file
    cp "$PROJECT_ROOT/docker-compose.prod.yml" "$PACKAGE_DIR/"
    
    # Copy environment template
    cp "$PROJECT_ROOT/env.prod.example" "$PACKAGE_DIR/"
    
    # Copy scripts
    cp -r "$PROJECT_ROOT/scripts" "$PACKAGE_DIR/"
    
    # Copy ops directory
    cp -r "$PROJECT_ROOT/ops" "$PACKAGE_DIR/"
    
    echo -e "${GREEN}✅ Pilot kit files copied${NC}"
}

# Function to create Docker image
build_docker_image() {
    echo -e "${YELLOW}🐳 Building Docker image...${NC}"
    
    # Build the OCX server image
    docker build -f "$PROJECT_ROOT/Dockerfile.complete" -t ocx-protocol:latest "$PROJECT_ROOT"
    
    # Save the image to a tar file
    docker save ocx-protocol:latest | gzip > "$PACKAGE_DIR/ocx-protocol-image.tar.gz"
    
    echo -e "${GREEN}✅ Docker image built and saved${NC}"
}

# Function to create documentation
create_documentation() {
    echo -e "${YELLOW}📚 Creating documentation...${NC}"
    
    # Create a comprehensive README
    cat > "$PACKAGE_DIR/README.md" << 'EOF'
# OCX Protocol Pilot Kit

## Quick Start

1. **Extract the package:**
   ```bash
   tar -xzf ocx-pilot-kit-*.tar.gz
   cd ocx-pilot-kit-*
   ```

2. **Configure environment:**
   ```bash
   cp env.prod.example .env.prod
   nano .env.prod  # Set your passwords
   ```

3. **Deploy:**
   ```bash
   ./deploy.sh
   ```

4. **Access services:**
   - API: http://localhost:8080
   - Swagger UI: http://localhost:8080/swagger/
   - Grafana: http://localhost:3000
   - Prometheus: http://localhost:9090

## What's Included

- **OCX Protocol Server** - Production-ready API server
- **PostgreSQL Database** - Receipt storage and persistence
- **Nginx Load Balancer** - SSL termination and rate limiting
- **Prometheus** - Metrics collection and monitoring
- **Grafana** - Monitoring dashboards and visualization
- **Load Testing Scripts** - Performance validation tools
- **Deployment Scripts** - Automated deployment and management

## Features

- ✅ **Production Ready** - Enterprise-grade deployment
- ✅ **Secure** - API key authentication and rate limiting
- ✅ **Monitored** - Real-time metrics and health checks
- ✅ **Scalable** - Horizontal scaling support
- ✅ **Tested** - Comprehensive load testing suite
- ✅ **Documented** - Complete API documentation

## Support

- **Documentation**: https://docs.ocx.dev
- **API Reference**: http://localhost:8080/swagger/
- **GitHub**: https://github.com/your-org/ocx-protocol
- **Email**: support@ocx.dev

## License

MIT License - see LICENSE file for details.
EOF

    # Create a quick reference card
    cat > "$PACKAGE_DIR/QUICK_REFERENCE.md" << 'EOF'
# OCX Protocol Quick Reference

## Essential Commands

```bash
# Deploy
./deploy.sh

# Start services
make start

# Stop services
make stop

# Check health
make health

# View logs
make logs

# Run tests
make test

# Scale server
make scale REPLICAS=3
```

## Service URLs

- **API**: http://localhost:8080
- **Swagger**: http://localhost:8080/swagger/
- **Grafana**: http://localhost:3000
- **Prometheus**: http://localhost:9090

## Default Credentials

- **Grafana**: admin / (check .env.prod)
- **API Key**: supersecretkey

## Health Checks

```bash
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

## Troubleshooting

```bash
# Check service status
make status

# View service logs
make logs

# Restart services
make restart

# Clean up
make clean
```
EOF

    echo -e "${GREEN}✅ Documentation created${NC}"
}

# Function to create package archive
create_package_archive() {
    echo -e "${YELLOW}📦 Creating package archive...${NC}"
    
    cd /tmp
    tar -czf "$PACKAGE_NAME.tar.gz" "$PACKAGE_NAME"
    
    # Move to project root
    mv "$PACKAGE_NAME.tar.gz" "$PROJECT_ROOT/"
    
    echo -e "${GREEN}✅ Package archive created: $PROJECT_ROOT/$PACKAGE_NAME.tar.gz${NC}"
}

# Function to display package info
show_package_info() {
    echo ""
    echo -e "${GREEN}🎉 Pilot Kit Package Created Successfully!${NC}"
    echo "============================================="
    echo ""
    echo -e "${BLUE}📦 Package Details:${NC}"
    echo "  • Name: $PACKAGE_NAME.tar.gz"
    echo "  • Location: $PROJECT_ROOT/$PACKAGE_NAME.tar.gz"
    echo "  • Size: $(du -h "$PROJECT_ROOT/$PACKAGE_NAME.tar.gz" | cut -f1)"
    echo ""
    echo -e "${BLUE}📋 Package Contents:${NC}"
    echo "  • OCX Protocol Server (Docker image)"
    echo "  • Production Docker Compose configuration"
    echo "  • Environment configuration templates"
    echo "  • Deployment and management scripts"
    echo "  • Load testing suite (k6 scripts)"
    echo "  • Monitoring configuration (Prometheus/Grafana)"
    echo "  • Complete documentation and quick reference"
    echo ""
    echo -e "${BLUE}🚀 Distribution:${NC}"
    echo "  • Upload to GitHub Releases"
    echo "  • Share with enterprise customers"
    echo "  • Use for pilot deployments"
    echo ""
    echo -e "${YELLOW}💡 Next Steps:${NC}"
    echo "  1. Test the package on a clean system"
    echo "  2. Upload to GitHub Releases"
    echo "  3. Share with pilot customers"
    echo "  4. Collect feedback and iterate"
    echo ""
    echo -e "${GREEN}Ready for enterprise pilots! 🚀${NC}"
}

# Function to cleanup
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up temporary files...${NC}"
    rm -rf "$PACKAGE_DIR"
    echo -e "${GREEN}✅ Cleanup completed${NC}"
}

# Main packaging flow
main() {
    create_package_dir
    copy_pilot_kit_files
    build_docker_image
    create_documentation
    create_package_archive
    show_package_info
    cleanup
}

# Run main function
main "$@"
