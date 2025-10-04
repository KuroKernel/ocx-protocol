# OCX Protocol Deployment Guide for ocx.world

## Overview

This guide covers deploying OCX Protocol to your domain **ocx.world** with a clean, professional setup.

## Architecture

```
ocx.world                    → Frontend (static React app)
api.ocx.world                → Backend API server
```

## Prerequisites

- Domain: **ocx.world** (GoDaddy)
- Server: VPS or cloud instance (recommended: 2 CPU, 4GB RAM minimum)
- OS: Ubuntu 22.04 LTS or similar
- Docker installed (recommended) OR Go 1.24+

---

## Part 1: Backend Deployment (api.ocx.world)

### Option A: Docker Deployment (Recommended)

#### 1. Build the server

```bash
# On your local machine or CI/CD
docker build -t ocx-api:latest .
docker save ocx-api:latest | gzip > ocx-api.tar.gz
```

#### 2. Transfer and load on server

```bash
# On your server
scp ocx-api.tar.gz user@your-server:/opt/ocx/
ssh user@your-server
cd /opt/ocx
docker load < ocx-api.tar.gz
```

#### 3. Generate signing keys

```bash
mkdir -p /opt/ocx/keys
cd /opt/ocx/keys

# Generate Ed25519 signing key
openssl genpkey -algorithm ed25519 -out ocx_signing.pem
openssl pkey -in ocx_signing.pem -pubout -outform DER | tail -c 32 | base64 -w0 > ocx_public.b64

# Secure the private key
chmod 600 ocx_signing.pem
```

#### 4. Create environment file

```bash
cat > /opt/ocx/.env << 'EOF'
OCX_API_KEYS=your-random-api-key-here
OCX_SIGNING_KEY_PEM=/keys/ocx_signing.pem
OCX_LOG_LEVEL=info
OCX_PORT=8080
OCX_DISABLE_DB=true
EOF
```

#### 5. Run the container

```bash
docker run -d \
  --name ocx-api \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v /opt/ocx/keys:/keys:ro \
  --env-file /opt/ocx/.env \
  ocx-api:latest
```

#### 6. Set up Caddy reverse proxy

```bash
# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy

# Configure Caddy
sudo cat > /etc/caddy/Caddyfile << 'EOF'
api.ocx.world {
    reverse_proxy localhost:8080

    # Security headers
    header {
        Strict-Transport-Security "max-age=31536000;"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    # CORS for frontend
    @cors_preflight {
        method OPTIONS
    }
    handle @cors_preflight {
        header Access-Control-Allow-Origin "https://ocx.world"
        header Access-Control-Allow-Methods "GET, POST, OPTIONS"
        header Access-Control-Allow-Headers "Content-Type, X-API-Key"
        respond 204
    }

    header Access-Control-Allow-Origin "https://ocx.world"
}
EOF

# Reload Caddy
sudo systemctl reload caddy
```

### Option B: Direct Binary Deployment

```bash
# Build on server or upload binary
go build -o /opt/ocx/server ./cmd/server

# Create systemd service
sudo cat > /etc/systemd/system/ocx-api.service << 'EOF'
[Unit]
Description=OCX Protocol API Server
After=network.target

[Service]
Type=simple
User=ocx
WorkingDirectory=/opt/ocx
Environment=OCX_API_KEYS=your-api-key
Environment=OCX_SIGNING_KEY_PEM=/opt/ocx/keys/ocx_signing.pem
Environment=OCX_LOG_LEVEL=info
Environment=OCX_PORT=8080
Environment=OCX_DISABLE_DB=true
ExecStart=/opt/ocx/server
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable ocx-api
sudo systemctl start ocx-api
```

---

## Part 2: Frontend Deployment (ocx.world)

### Option A: Static Hosting on Same Server

#### 1. Upload built website

```bash
# From your local machine
cd /home/kurokernel/Desktop/AXIS/ocx-protocol
scp -r build/* user@your-server:/var/www/ocx-world/
```

#### 2. Configure Caddy for frontend

```bash
sudo cat > /etc/caddy/Caddyfile << 'EOF'
ocx.world {
    root * /var/www/ocx-world
    file_server

    # SPA fallback
    try_files {path} /index.html

    # Security headers
    header {
        Strict-Transport-Security "max-age=31536000;"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        Referrer-Policy "strict-origin-when-cross-origin"
    }
}

api.ocx.world {
    reverse_proxy localhost:8080

    header {
        Strict-Transport-Security "max-age=31536000;"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    @cors_preflight method OPTIONS
    handle @cors_preflight {
        header Access-Control-Allow-Origin "https://ocx.world"
        header Access-Control-Allow-Methods "GET, POST, OPTIONS"
        header Access-Control-Allow-Headers "Content-Type, X-API-Key"
        respond 204
    }

    header Access-Control-Allow-Origin "https://ocx.world"
}
EOF

sudo systemctl reload caddy
```

### Option B: Netlify/Vercel (Easiest)

#### Netlify

```bash
# Install Netlify CLI
npm install -g netlify-cli

# Deploy
cd /home/kurokernel/Desktop/AXIS/ocx-protocol
netlify deploy --prod --dir=build
```

#### Vercel

```bash
# Install Vercel CLI
npm install -g vercel

# Deploy
cd /home/kurokernel/Desktop/AXIS/ocx-protocol
vercel --prod
```

---

## Part 3: DNS Configuration (GoDaddy)

Log into GoDaddy and set these DNS records:

### For Same-Server Hosting

```
Type    Name    Value                   TTL
A       @       your.server.ip.address  600
A       api     your.server.ip.address  600
```

### For Split Hosting (Netlify/Vercel + VPS)

```
Type    Name    Value                   TTL
CNAME   @       your-netlify-site.netlify.app    600
A       api     your.server.ip.address           600
```

---

## Part 4: Testing

### Test Backend

```bash
# Health check
curl https://api.ocx.world/livez
curl https://api.ocx.world/readyz

# Test execution (replace with your API key)
curl -X POST https://api.ocx.world/api/v1/execute \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"artifact_hash":"test","input":"74657374"}'
```

### Test Frontend

```bash
# Should load the website
curl https://ocx.world

# Check in browser
open https://ocx.world
```

---

## Recommended Production Setup

### Best Practice Architecture

```
┌─────────────────┐
│   CloudFlare    │ ← DNS + CDN + DDoS protection
└────────┬────────┘
         │
    ┌────┴─────┐
    │          │
┌───┴───┐  ┌──┴──────┐
│Netlify│  │ VPS     │
│(www)  │  │(api)    │
└───────┘  │         │
           │ Caddy   │
           │  ↓      │
           │ Docker  │
           │  ↓      │
           │ OCX API │
           └─────────┘
```

1. **Frontend**: Netlify or Vercel (free tier works)
   - Automatic HTTPS
   - Global CDN
   - Zero config

2. **Backend**: VPS with Docker + Caddy
   - DigitalOcean: $6/month droplet
   - Hetzner: €4/month VPS
   - Automatic HTTPS via Caddy

3. **DNS**: GoDaddy → CloudFlare (recommended)
   - Better DDoS protection
   - Free CDN
   - Better DNS management

### Monitoring

```bash
# Set up basic monitoring
cat > /opt/ocx/monitor.sh << 'EOF'
#!/bin/bash
if ! curl -f https://api.ocx.world/livez > /dev/null 2>&1; then
    echo "OCX API is down!" | mail -s "OCX Alert" contact@ocx.world
    docker restart ocx-api
fi
EOF

chmod +x /opt/ocx/monitor.sh

# Add to crontab
(crontab -l 2>/dev/null; echo "*/5 * * * * /opt/ocx/monitor.sh") | crontab -
```

---

## Security Checklist

- [ ] Change default API keys
- [ ] Secure private keys (chmod 600)
- [ ] Enable firewall (ufw/iptables)
- [ ] Set up fail2ban
- [ ] Enable automatic security updates
- [ ] Regular backups of keys directory
- [ ] Monitor logs for suspicious activity
- [ ] Set up SSL/TLS (Caddy does this automatically)

---

## Quick Start Commands (Copy-Paste Ready)

### Full Stack on Single VPS

```bash
# 1. Install dependencies
sudo apt update
sudo apt install -y docker.io git curl

# 2. Clone and build
git clone https://github.com/your-org/ocx-protocol
cd ocx-protocol
docker build -t ocx-api .

# 3. Generate keys
mkdir -p /opt/ocx/keys
openssl genpkey -algorithm ed25519 -out /opt/ocx/keys/ocx_signing.pem
chmod 600 /opt/ocx/keys/ocx_signing.pem

# 4. Run API
docker run -d --name ocx-api --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v /opt/ocx/keys:/keys:ro \
  -e OCX_API_KEYS=changeme \
  -e OCX_SIGNING_KEY_PEM=/keys/ocx_signing.pem \
  -e OCX_LOG_LEVEL=info \
  -e OCX_DISABLE_DB=true \
  ocx-api

# 5. Install Caddy (see above for full commands)

# 6. Upload frontend to /var/www/ocx-world/

# 7. Configure DNS at GoDaddy
```

---

## Troubleshooting

### API not responding

```bash
# Check if container is running
docker ps -a

# Check logs
docker logs ocx-api

# Restart
docker restart ocx-api
```

### Certificate errors

```bash
# Check Caddy logs
sudo journalctl -u caddy -f

# Reload Caddy
sudo systemctl reload caddy
```

### CORS errors

Make sure your Caddyfile has correct CORS headers for `https://ocx.world`

---

## Support

- Email: contact@ocx.world
- GitHub: https://github.com/your-org/ocx-protocol

---

**Last updated**: 2025-01-04
