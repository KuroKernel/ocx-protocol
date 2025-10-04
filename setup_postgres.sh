#!/bin/bash
set -euo pipefail

echo "=== OCX PostgreSQL Setup ==="
echo ""

# Check if PostgreSQL is installed
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL is not installed"
    echo "Please install PostgreSQL first:"
    echo "  sudo apt update && sudo apt install postgresql postgresql-contrib"
    exit 1
fi

# Check if PostgreSQL service is running
if ! systemctl is-active --quiet postgresql; then
    echo "⚠️  PostgreSQL service is not running"
    echo "Please start PostgreSQL:"
    echo "  sudo systemctl start postgresql"
    echo "  sudo systemctl enable postgresql"
    exit 1
fi

echo "✅ PostgreSQL is running"

# Create database and user (requires sudo)
echo ""
echo "Creating database and user..."
echo "You'll need to run these commands with sudo:"
echo ""
echo "sudo -u postgres psql -c \"CREATE DATABASE ocx;\""
echo "sudo -u postgres psql -c \"CREATE USER ocx WITH PASSWORD 'ocx';\""
echo "sudo -u postgres psql -c \"GRANT ALL PRIVILEGES ON DATABASE ocx TO ocx;\""
echo ""

# Test connection
echo "Testing connection..."
if PGPASSWORD=ocx psql -h localhost -U ocx -d ocx -c "SELECT version();" &>/dev/null; then
    echo "✅ Database connection successful"
    
    # Run migrations
    echo ""
    echo "Running database migrations..."
    PGPASSWORD=ocx psql -h localhost -U ocx -d ocx -f database/migrations/0001_init.sql
    PGPASSWORD=ocx psql -h localhost -U ocx -d ocx -f database/migrations/0002_receipt_v1_1.sql
    echo "✅ Database schema created"
    
    echo ""
    echo "🎉 PostgreSQL setup complete!"
    echo ""
    echo "You can now run the server with:"
    echo "  OCX_DB_URL=\"postgres://ocx:ocx@localhost:5432/ocx\" ./server"
    
else
    echo "❌ Database connection failed"
    echo "Please run the setup commands above first"
    exit 1
fi
