# Deployment Guide

## Prerequisites

- Docker installed
- AWS account (for Lightsail deployment)
- Domain name (optional, for HTTPS)

---

## Local Development

### Build and Run

```bash
# Build image
docker build -t algorave-server .

# Run with env file
docker run -p 8080:8080 --env-file .env algorave-server

# Test
curl http://localhost:8080/health
```

---

## Production Deployment (AWS Lightsail)

### 1. Create Lightsail Instance

1. Go to https://lightsail.aws.amazon.com
2. Create instance: Linux/Unix > Ubuntu 24.04 LTS
3. Select $5/month plan
4. Name it `algorave-server`
5. Create and wait for running status

### 2. Open Firewall Ports

1. Click instance > Networking tab
2. Add rules:
   - TCP 80 (HTTP)
   - TCP 443 (HTTPS)
   - TCP 8080 (optional, for direct access)

### 3. Install Docker on Server

SSH into Lightsail (click terminal icon), then:

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install docker.io -y
sudo systemctl start docker && sudo systemctl enable docker
sudo usermod -aG docker ubuntu
newgrp docker
```

### 4. Push Image to Docker Hub

On your local machine:

```bash
# Login
docker login

# Build for AMD64 (required if on Apple Silicon)
docker buildx build --platform linux/amd64 -t YOUR_DOCKERHUB_USERNAME/algorave-server:latest --push .
```

### 5. Deploy on Lightsail

SSH into Lightsail:

```bash
# Create env file
nano ~/.env
# Paste your environment variables, save with Ctrl+O, exit with Ctrl+X

# Pull and run
docker pull YOUR_DOCKERHUB_USERNAME/algorave-server:latest
docker run -d -p 8080:8080 --name algorave --restart unless-stopped --env-file ~/.env YOUR_DOCKERHUB_USERNAME/algorave-server:latest

# Verify
docker ps
docker logs algorave
curl http://localhost:8080/health
```

---

## HTTPS Setup (Caddy)

### 1. Point DNS to Lightsail

Add A records in your domain registrar:
- `@` -> Lightsail public IP
- `www` -> Lightsail public IP

### 2. Install Caddy

```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy -y
```

### 3. Configure Caddy

```bash
sudo nano /etc/caddy/Caddyfile
```

Replace contents with:

```
yourdomain.com {
    reverse_proxy localhost:8080
}

www.yourdomain.com {
    redir https://yourdomain.com{uri} permanent
}
```

Restart:

```bash
sudo systemctl restart caddy
sudo systemctl status caddy
```

Caddy automatically obtains SSL certificates from Let's Encrypt.

---

## Common Operations

```bash
# View logs
docker logs algorave

# Restart app
docker restart algorave

# Redeploy new version
docker pull YOUR_DOCKERHUB_USERNAME/algorave-server:latest
docker stop algorave && docker rm algorave
docker run -d -p 8080:8080 --name algorave --restart unless-stopped --env-file ~/.env YOUR_DOCKERHUB_USERNAME/algorave-server:latest

# Check Caddy status
sudo systemctl status caddy

# View Caddy logs
sudo journalctl -u caddy --no-pager | tail -50
```

---

## Environment Variables

Required variables in `.env`:

| Variable | Description |
|----------|-------------|
| `BASE_URL` | Production URL (e.g., https://yourdomain.com) |
| `PORT` | Server port (default: 8080) |
| `JWT_SECRET` | Generate with: `openssl rand -base64 64` |
| `SUPABASE_CONNECTION_STRING` | Database connection string |
| `ANTHROPIC_API_KEY` | Claude API key |
| `OPENAI_API_KEY` | OpenAI API key (for embeddings) |
| `GITHUB_CLIENT_ID` | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth client secret |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret |

---

## OAuth Callback URLs

When setting up OAuth providers, use these callback URLs:

| Provider | Callback URL |
|----------|-------------|
| GitHub | `https://yourdomain.com/api/v1/auth/github/callback` |
| Google | `https://yourdomain.com/api/v1/auth/google/callback` |
