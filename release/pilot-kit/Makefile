# OCX Protocol Pilot Kit Makefile
# This Makefile provides convenient commands for managing the pilot kit

.PHONY: help deploy start stop restart status logs health test clean

# Default target
help:
	@echo "OCX Protocol Pilot Kit - Available Commands"
	@echo "==========================================="
	@echo ""
	@echo "Deployment:"
	@echo "  deploy     - Deploy the complete pilot kit"
	@echo "  start      - Start all services"
	@echo "  stop       - Stop all services"
	@echo "  restart    - Restart all services"
	@echo ""
	@echo "Monitoring:"
	@echo "  status     - Show service status"
	@echo "  logs       - Show service logs"
	@echo "  health     - Run health checks"
	@echo ""
	@echo "Testing:"
	@echo "  test       - Run load tests"
	@echo "  test-quick - Run quick health tests"
	@echo ""
	@echo "Maintenance:"
	@echo "  clean      - Clean up containers and volumes"
	@echo "  backup     - Backup database"
	@echo "  restore    - Restore database from backup"
	@echo ""

# Deployment commands
deploy:
	@echo "🚀 Deploying OCX Protocol Pilot Kit..."
	./deploy.sh

start:
	@echo "▶️  Starting OCX Protocol services..."
	docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d

stop:
	@echo "⏹️  Stopping OCX Protocol services..."
	docker-compose -f docker-compose.prod.yml --env-file .env.prod down

restart:
	@echo "🔄 Restarting OCX Protocol services..."
	docker-compose -f docker-compose.prod.yml --env-file .env.prod restart

# Monitoring commands
status:
	@echo "📊 OCX Protocol Service Status"
	@echo "=============================="
	docker-compose -f docker-compose.prod.yml --env-file .env.prod ps

logs:
	@echo "📋 OCX Protocol Service Logs"
	@echo "============================"
	docker-compose -f docker-compose.prod.yml --env-file .env.prod logs -f

health:
	@echo "🏥 Running Health Checks"
	@echo "========================"
	@echo "Checking OCX API..."
	@curl -s -f http://localhost:8080/health > /dev/null && echo "✅ OCX API: Healthy" || echo "❌ OCX API: Unhealthy"
	@echo "Checking Database..."
	@docker-compose -f docker-compose.prod.yml --env-file .env.prod exec -T postgres pg_isready -U ocx -d ocx > /dev/null && echo "✅ Database: Healthy" || echo "❌ Database: Unhealthy"
	@echo "Checking Prometheus..."
	@curl -s -f http://localhost:9090/-/ready > /dev/null && echo "✅ Prometheus: Healthy" || echo "❌ Prometheus: Unhealthy"
	@echo "Checking Grafana..."
	@curl -s -f http://localhost:3000/api/health > /dev/null && echo "✅ Grafana: Healthy" || echo "❌ Grafana: Unhealthy"

# Testing commands
test:
	@echo "🧪 Running Load Tests"
	@echo "====================="
	./ops/run-loadtests.sh

test-quick:
	@echo "⚡ Running Quick Health Tests"
	@echo "============================="
	@echo "Testing API endpoints..."
	@curl -s -f http://localhost:8080/health > /dev/null && echo "✅ Health endpoint: OK" || echo "❌ Health endpoint: FAIL"
	@curl -s -f http://localhost:8080/swagger/ > /dev/null && echo "✅ Swagger UI: OK" || echo "❌ Swagger UI: FAIL"
	@curl -s -f http://localhost:8080/metrics > /dev/null && echo "✅ Metrics endpoint: OK" || echo "❌ Metrics endpoint: FAIL"
	@echo "Testing database connection..."
	@docker-compose -f docker-compose.prod.yml --env-file .env.prod exec -T postgres psql -U ocx -d ocx -c "SELECT 1;" > /dev/null && echo "✅ Database connection: OK" || echo "❌ Database connection: FAIL"

# Maintenance commands
clean:
	@echo "🧹 Cleaning up OCX Protocol resources..."
	docker-compose -f docker-compose.prod.yml --env-file .env.prod down -v
	docker system prune -f
	docker volume prune -f

backup:
	@echo "💾 Creating database backup..."
	@mkdir -p backups
	docker-compose -f docker-compose.prod.yml --env-file .env.prod exec -T postgres pg_dump -U ocx ocx > backups/ocx_backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "✅ Backup created in backups/ directory"

restore:
	@echo "📥 Restoring database from backup..."
	@if [ -z "$(BACKUP_FILE)" ]; then \
		echo "❌ Please specify backup file: make restore BACKUP_FILE=backups/your_backup.sql"; \
		exit 1; \
	fi
	@if [ ! -f "$(BACKUP_FILE)" ]; then \
		echo "❌ Backup file not found: $(BACKUP_FILE)"; \
		exit 1; \
	fi
	docker-compose -f docker-compose.prod.yml --env-file .env.prod exec -T postgres psql -U ocx -d ocx < $(BACKUP_FILE)
	@echo "✅ Database restored from $(BACKUP_FILE)"

# Development commands
dev:
	@echo "🔧 Starting development environment..."
	docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d postgres prometheus grafana
	@echo "✅ Development services started"
	@echo "💡 Run 'go run cmd/server/main.go' to start the OCX server locally"

scale:
	@echo "📈 Scaling OCX server..."
	@if [ -z "$(REPLICAS)" ]; then \
		echo "❌ Please specify number of replicas: make scale REPLICAS=3"; \
		exit 1; \
	fi
	docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d --scale ocx-server=$(REPLICAS)
	@echo "✅ OCX server scaled to $(REPLICAS) replicas"

# Information commands
info:
	@echo "ℹ️  OCX Protocol Pilot Kit Information"
	@echo "====================================="
	@echo "Service URLs:"
	@echo "  • OCX API:        http://localhost:8080"
	@echo "  • Swagger UI:     http://localhost:8080/swagger/"
	@echo "  • Grafana:        http://localhost:3000"
	@echo "  • Prometheus:     http://localhost:9090"
	@echo ""
	@echo "Default Credentials:"
	@echo "  • Grafana:        admin / (check .env.prod)"
	@echo "  • API Key:        supersecretkey"
	@echo ""
	@echo "Quick Commands:"
	@echo "  • Health Check:   make health"
	@echo "  • View Logs:      make logs"
	@echo "  • Run Tests:      make test"
	@echo "  • Stop All:       make stop"
