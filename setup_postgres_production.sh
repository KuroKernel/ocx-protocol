#!/bin/bash

echo "=== POSTGRESQL PRODUCTION SETUP ==="
echo ""

# Check if running as root or with sudo
if [ "$EUID" -eq 0 ]; then
    echo "Running as root - setting up PostgreSQL..."
    
    # Start and enable PostgreSQL
    systemctl start postgresql
    systemctl enable postgresql
    
    # Create database and user
    sudo -u postgres psql -c "CREATE DATABASE ocx;" 2>/dev/null || echo "Database 'ocx' may already exist"
    sudo -u postgres psql -c "CREATE USER ocx WITH PASSWORD 'ocxpass';" 2>/dev/null || echo "User 'ocx' may already exist"
    sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE ocx TO ocx;" 2>/dev/null || echo "Privileges may already be granted"
    
    # Test connection
    sudo -u postgres psql -c "SELECT version();" 2>/dev/null && echo "PostgreSQL is working!"
    
else
    echo "Please run this script with sudo:"
    echo "sudo ./setup_postgres_production.sh"
    echo ""
    echo "Or run these commands manually:"
    echo "sudo systemctl start postgresql"
    echo "sudo systemctl enable postgresql"
    echo "sudo -u postgres psql -c \"CREATE DATABASE ocx;\""
    echo "sudo -u postgres psql -c \"CREATE USER ocx WITH PASSWORD 'ocxpass';\""
    echo "sudo -u postgres psql -c \"GRANT ALL PRIVILEGES ON DATABASE ocx TO ocx;\""
    echo "sudo -u postgres psql -c \"SELECT version();\""
fi

echo ""
echo "After PostgreSQL is set up, test with:"
echo "DATABASE_URL=\"postgres://ocx:ocxpass@localhost:5432/ocx?sslmode=disable\" ./server"
