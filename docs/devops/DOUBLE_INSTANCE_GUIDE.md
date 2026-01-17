# Algorave Deployment Guide

Deploy Algorave on two Lightsail instances: one gateway (Go server + Caddy), one for the Next.js frontend.

## Architecture

```
                         algorave.cc
                              │
                              ▼
┌──────────────────────────────────────────────────────────┐
│                     Lightsail VPC                        │
│                                                          │
│  ┌────────────────────────┐    ┌──────────────────────┐  │
│  │  Gateway ($5/mo)       │    │  Frontend ($3.50/mo) │  │
│  │  Public: 3.216.27.18   │    │  Private: 172.26.x.x │  │
│  │                        │    │                      │  │
│  │  Caddy ──┬─► Go :8080  │    │  Next.js :3000       │  │
│  │          │             │    │                      │  │
│  │          └─────────────┼────┼──►                   │  │
│  │                        │    │                      │  │
│  │  Redis :6379           │    │                      │  │
│  └────────────────────────┘    └──────────────────────┘  │
│                                                          │
└──────────────────────────────────────────────────────────┘

Routing (Caddy):
  /api/*  → Go server (localhost:8080)
  /*      → Frontend (172.26.x.x:3000)
```

## CI/CD Flow

Both repos use GitHub Actions to build and deploy:

```
Push to main → GitHub Actions → Build Docker image → Push to Docker Hub → SSH deploy
```

## Initial Setup

### 1. Create Instances

In Lightsail console, create two Ubuntu 22.04 instances in the same region:
- `algorave-gateway` — $5/month (1GB RAM)
- `algorave-frontend` — $3.50/month (512MB RAM)

Note the **private IP** of the frontend instance (Networking tab).

### 2. Setup Frontend Instance

```bash
ssh -i key.pem ubuntu@FRONTEND_PUBLIC_IP

# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker ubuntu
exit && ssh -i key.pem ubuntu@FRONTEND_PUBLIC_IP

# Create deployment directory
mkdir -p ~/algorave/frontend
cd ~/algorave/frontend

# Create docker-compose.yml
cat > docker-compose.yml << 'EOF'
services:
  frontend:
    image: kadetxx/algorave-frontend:latest
    ports:
      - '3000:3000'
    restart: unless-stopped
EOF

# Login to Docker Hub and start
docker login
docker compose up -d
```

### 3. Setup Gateway Instance

```bash
ssh -i key.pem ubuntu@GATEWAY_PUBLIC_IP

# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker ubuntu
exit && ssh -i key.pem ubuntu@GATEWAY_PUBLIC_IP

# Install Caddy
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install caddy

# Create Caddyfile (replace FRONTEND_PRIVATE_IP)
sudo tee /etc/caddy/Caddyfile << 'EOF'
algorave.cc {
    handle /api/* {
        reverse_proxy localhost:8080 {
            flush_interval -1
        }
    }

    handle {
        reverse_proxy FRONTEND_PRIVATE_IP:3000
    }
}

www.algorave.cc {
    redir https://algorave.cc{uri} permanent
}
EOF

sudo systemctl reload caddy

# Create Docker network
docker network create algorave-net

# Start Redis
docker run -d \
  --name algorave-redis \
  --network algorave-net \
  --restart unless-stopped \
  -v algorave_redis_data:/data \
  redis:7-alpine

# Create .env file
nano ~/.env  # Add production environment variables

# Pull and run server (first time)
docker pull kadetxx/algorave-server:latest
docker run -d \
  -p 8080:8080 \
  --name algorave \
  --network algorave-net \
  --restart unless-stopped \
  --env-file ~/.env \
  -e REDIS_URL=redis://algorave-redis:6379 \
  kadetxx/algorave-server:latest
```

### 4. Configure Firewalls

**Gateway** (Lightsail Networking tab):
- HTTP (80) — Anywhere
- HTTPS (443) — Anywhere
- SSH (22) — Your IP only

**Frontend** (Lightsail Networking tab):
- SSH (22) — Your IP only
- Delete HTTP/HTTPS rules (traffic goes through gateway)

### 5. Point DNS

Add A records pointing to gateway public IP:
- `algorave.cc` → Gateway IP
- `www.algorave.cc` → Gateway IP

### 6. Setup GitHub Actions Secrets

**Server repo** (`github.com/your-org/server`):
| Secret | Value |
|--------|-------|
| `DOCKERHUB_USERNAME` | `kadetxx` |
| `DOCKERHUB_TOKEN` | Docker Hub access token |
| `LIGHTSAIL_HOST` | Gateway public IP |
| `LIGHTSAIL_SSH_KEY` | Lightsail default private key |

**Frontend repo** (`github.com/your-org/frontend`):
| Secret | Value |
|--------|-------|
| `DOCKERHUB_USERNAME` | `kadetxx` |
| `DOCKERHUB_TOKEN` | Docker Hub access token |
| `LIGHTSAIL_HOST` | Frontend public IP |
| `LIGHTSAIL_SSH_KEY` | Lightsail default private key |

---

## Common Commands

### Gateway
```bash
# View server logs
docker logs algorave -f

# Restart server
docker restart algorave

# View Caddy logs
journalctl -u caddy -f

# Reload Caddy config
sudo systemctl reload caddy

# Check Redis
docker exec algorave-redis redis-cli PING
```

### Frontend
```bash
# View logs
docker compose logs -f

# Manual redeploy
cd ~/algorave/frontend
docker compose down
docker compose pull
docker compose up -d

# Check status
docker ps
```

---

## Environment Variables

Required in `~/.env` on gateway:

| Variable | Description |
|----------|-------------|
| `BASE_URL` | `https://algorave.cc` |
| `PORT` | `8080` |
| `JWT_SECRET` | `openssl rand -base64 64` |
| `SUPABASE_CONNECTION_STRING` | Database URL |
| `ANTHROPIC_API_KEY` | Claude API key |
| `OPENAI_API_KEY` | OpenAI API key (embeddings) |
| `GITHUB_CLIENT_ID` | OAuth client ID |
| `GITHUB_CLIENT_SECRET` | OAuth secret |
| `GOOGLE_CLIENT_ID` | OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | OAuth secret |

---

## Troubleshooting

### Bot defense trapping IPs
```bash
# List trapped IPs
docker exec algorave-redis redis-cli KEYS "botdefense:trapped:*"

# Clear specific IP
docker exec algorave-redis redis-cli DEL "botdefense:trapped:YOUR_IP" "botdefense:reason:YOUR_IP"

# Clear all traps
docker exec algorave-redis redis-cli KEYS "botdefense:*" | xargs -r docker exec -i algorave-redis redis-cli DEL
```

### Frontend not updating after deploy
```bash
# Check container age
docker ps

# Force restart
docker compose down && docker compose pull && docker compose up -d
```

### Port conflict on frontend
```bash
# Kill any container using port 3000
docker ps -q --filter "publish=3000" | xargs -r docker stop | xargs -r docker rm
docker compose up -d
```

---

## Cost

| Resource | Cost |
|----------|------|
| Gateway (1GB) | $5/mo |
| Frontend (512MB) | $3.50/mo |
| **Total** | **$8.50/mo** |
