#!/bin/bash

# OCX Protocol Pilot Kit Deployment Script
# This script automates the deployment of the OCX Protocol pilot kit

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$SCRIPT_DIR/.env.prod"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.prod.yml"

echo -e "${BLUE}🚀 OCX Protocol Pilot Kit Deployment${NC}"
echo "======================================"
echo "Project Root: $PROJECT_ROOT"
echo "Environment File: $ENV_FILE"
echo "Compose File: $COMPOSE_FILE"
echo ""

# Function to check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}🔍 Checking prerequisites...${NC}"
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ Docker is not installed${NC}"
        echo "   Please install Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        echo -e "${RED}❌ Docker Compose is not installed${NC}"
        echo "   Please install Docker Compose: https://docs.docker.com/compose/install/"
        exit 1
    fi
    
    # Check if Docker is running
    if ! docker info &> /dev/null; then
        echo -e "${RED}❌ Docker is not running${NC}"
        echo "   Please start Docker daemon"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Prerequisites check passed${NC}"
}

# Function to setup environment
setup_environment() {
    echo -e "${YELLOW}⚙️  Setting up environment...${NC}"
    
    if [ ! -f "$ENV_FILE" ]; then
        if [ -f "$SCRIPT_DIR/env.prod.example" ]; then
            cp "$SCRIPT_DIR/env.prod.example" "$ENV_FILE"
            echo -e "${YELLOW}📝 Created $ENV_FILE from template${NC}"
            echo -e "${YELLOW}   Please edit the file and set your passwords:${NC}"
            echo -e "${YELLOW}   nano $ENV_FILE${NC}"
            echo ""
            echo -e "${BLUE}Press Enter to continue after editing the environment file...${NC}"
            read -r
        else
            echo -e "${RED}❌ Environment template not found${NC}"
            exit 1
        fi
    fi
    
    # Validate required environment variables
    source "$ENV_FILE"
    
    if [ -z "$OCX_DB_PASSWORD" ] || [ "$OCX_DB_PASSWORD" = "your_secure_database_password_here" ]; then
        echo -e "${RED}❌ OCX_DB_PASSWORD not set in $ENV_FILE${NC}"
        exit 1
    fi
    
    if [ -z "$OCX_API_KEYS" ] || [ "$OCX_API_KEYS" = "admin:supersecretkey,user1:user1key,user2:user2key" ]; then
        echo -e "${RED}❌ OCX_API_KEYS not set in $ENV_FILE${NC}"
        exit 1
    fi
    
    if [ -z "$GRAFANA_PASSWORD" ] || [ "$GRAFANA_PASSWORD" = "your_grafana_admin_password_here" ]; then
        echo -e "${RED}❌ GRAFANA_PASSWORD not set in $ENV_FILE${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Environment configuration validated${NC}"
}

# Function to build and deploy
deploy_services() {
    echo -e "${YELLOW}🏗️  Building and deploying services...${NC}"
    
    # Build the OCX server image
    echo -e "${BLUE}Building OCX server image...${NC}"
    docker build -f "$PROJECT_ROOT/Dockerfile.complete" -t ocx-protocol:latest "$PROJECT_ROOT"
    
    # Start services
    echo -e "${BLUE}Starting services...${NC}"
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d
    
    echo -e "${GREEN}✅ Services deployed successfully${NC}"
}

# Function to wait for services
wait_for_services() {
    echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"
    
    # Wait for database
    echo -e "${BLUE}Waiting for database...${NC}"
    timeout 60 bash -c 'until docker-compose -f "'"$COMPOSE_FILE"'" --env-file "'"$ENV_FILE"'" exec -T postgres pg_isready -U ocx -d ocx; do sleep 2; done'
    
    # Wait for OCX server
    echo -e "${BLUE}Waiting for OCX server...${NC}"
    timeout 60 bash -c 'until curl -s -f http://localhost:8080/health > /dev/null; do sleep 2; done'
    
    # Wait for Prometheus
    echo -e "${BLUE}Waiting for Prometheus...${NC}"
    timeout 30 bash -c 'until curl -s -f http://localhost:9090/-/ready > /dev/null; do sleep 2; done'
    
    # Wait for Grafana
    echo -e "${BLUE}Waiting for Grafana...${NC}"
    timeout 30 bash -c 'until curl -s -f http://localhost:3000/api/health > /dev/null; do sleep 2; done'
    
    echo -e "${GREEN}✅ All services are ready${NC}"
}

# Function to run health checks
run_health_checks() {
    echo -e "${YELLOW}🏥 Running health checks...${NC}"
    
    # Check OCX API
    echo -e "${BLUE}Checking OCX API...${NC}"
    if curl -s -f http://localhost:8080/health > /dev/null; then
        echo -e "${GREEN}✅ OCX API is healthy${NC}"
    else
        echo -e "${RED}❌ OCX API health check failed${NC}"
        return 1
    fi
    
    # Check database
    echo -e "${BLUE}Checking database...${NC}"
    if docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T postgres pg_isready -U ocx -d ocx > /dev/null; then
        echo -e "${GREEN}✅ Database is healthy${NC}"
    else
        echo -e "${RED}❌ Database health check failed${NC}"
        return 1
    fi
    
    # Check Prometheus
    echo -e "${BLUE}Checking Prometheus...${NC}"
    if curl -s -f http://localhost:9090/-/ready > /dev/null; then
        echo -e "${GREEN}✅ Prometheus is healthy${NC}"
    else
        echo -e "${RED}❌ Prometheus health check failed${NC}"
        return 1
    fi
    
    # Check Grafana
    echo -e "${BLUE}Checking Grafana...${NC}"
    if curl -s -f http://localhost:3000/api/health > /dev/null; then
        echo -e "${GREEN}✅ Grafana is healthy${NC}"
    else
        echo -e "${RED}❌ Grafana health check failed${NC}"
        return 1
    fi
    
    echo -e "${GREEN}✅ All health checks passed${NC}"
}

# Function to display deployment info
show_deployment_info() {
    echo ""
    echo -e "${GREEN}🎉 OCX Protocol Pilot Kit Deployed Successfully!${NC}"
    echo "=================================================="
    echo ""
    echo -e "${BLUE}📊 Service URLs:${NC}"
    echo "  • OCX API:        http://localhost:8080"
    echo "  • Swagger UI:     http://localhost:8080/swagger/"
    echo "  • Grafana:        http://localhost:3000"
    echo "  • Prometheus:     http://localhost:9090"
    echo ""
    echo -e "${BLUE}🔑 Default Credentials:${NC}"
    echo "  • Grafana:        admin / $GRAFANA_PASSWORD"
    echo "  • API Key:        supersecretkey"
    echo ""
    echo -e "${BLUE}🧪 Quick Tests:${NC}"
    echo "  • Health Check:   curl http://localhost:8080/health"
    echo "  • API Docs:       open http://localhost:8080/swagger/"
    echo "  • Load Tests:     ./ops/run-loadtests.sh"
    echo ""
    echo -e "${BLUE}📋 Management Commands:${NC}"
    echo "  • View Logs:      docker-compose -f $COMPOSE_FILE logs -f"
    echo "  • Stop Services:  docker-compose -f $COMPOSE_FILE down"
    echo "  • Restart:        docker-compose -f $COMPOSE_FILE restart"
    echo "  • Scale:          docker-compose -f $COMPOSE_FILE up -d --scale ocx-server=3"
    echo ""
    echo -e "${YELLOW}💡 Next Steps:${NC}"
    echo "  1. Test the API using Swagger UI"
    echo "  2. Run load tests to validate performance"
    echo "  3. Configure monitoring dashboards in Grafana"
    echo "  4. Set up SSL certificates for production"
    echo ""
    echo -e "${GREEN}Ready for enterprise pilots! 🚀${NC}"
}

# Main deployment flow
main() {
    check_prerequisites
    setup_environment
    deploy_services
    wait_for_services
    run_health_checks
    show_deployment_info
}

# Run main function
main "$@"
