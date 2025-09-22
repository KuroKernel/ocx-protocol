#!/bin/bash
set -euo pipefail

echo "Fixing Docker permissions..."

# Add user to docker group
sudo usermod -aG docker $USER

# Set proper permissions
sudo chmod 666 /var/run/docker.sock

# Restart Docker service
sudo systemctl restart docker

echo "Docker permissions fixed. You may need to log out and back in."
