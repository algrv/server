# Deployment Strategy

**Decision:** AWS Lightsail (VM)
**Monthly Cost:** $5-10 starting

---

## Requirements

| Requirement | Priority |
|-------------|----------|
| Docker container support | Must have |
| WebSocket support (no timeout limits) | Must have |
| Low cost for starting out | Must have |
| Easy to set up and manage | Must have |
| Transferable AWS skills | Nice to have |
| Auto-scaling | Not needed initially |

---

## Options Evaluated

### 1. AWS App Runner

Serverless container platform with auto-scaling.

| Pros | Cons |
|------|------|
| Simple: push container, get URL | 30-minute WebSocket timeout |
| Auto-scaling (even to zero) | Limited control |
| Pay per use | |

**Verdict:** WebSocket timeout is a dealbreaker for real-time features.

---

### 2. AWS ECS Fargate

Managed container orchestration without managing servers.

| Pros | Cons |
|------|------|
| Industry standard (good for resume) | Requires ALB (~$16/month fixed) |
| Full WebSocket support | Minimum ~$30/month |
| Auto-scaling | More complex setup |
| Production-grade | Overkill for starting out |

**Verdict:** Great for scale, but too expensive for a small project starting out.

---

### 3. AWS Lightsail (VM)

Simple virtual private server with predictable pricing.

| Pros | Cons |
|------|------|
| Cheap ($5/month) | Manual scaling only |
| Full WebSocket support | You manage the server |
| Direct public IP (no ALB needed) | Less "cloud-native" |
| Docker support | |
| Can add load balancer later ($18) | |
| Still teaches AWS fundamentals | |

**Verdict:** Best balance of cost, simplicity, and features for starting out.

---

### 4. AWS Lightsail Containers

Managed containers on Lightsail.

| Pros | Cons |
|------|------|
| Simpler than ECS | Starts at $7/month |
| Basic auto-scaling | Less common in job market |
| No server management | |

**Verdict:** Good middle ground, but VM gives more control at lower cost.

---

### 5. Non-AWS Alternatives (Considered)

| Provider | Cost | Notes |
|----------|------|-------|
| Fly.io | $0-10 | Excellent for WebSockets, but not AWS |
| Railway | $5+ | Simple, but not AWS |
| Hetzner | $4 | Cheapest, but not AWS |

**Verdict:** Skipped because AWS provides more scaling options + additional services needed when app grows.

---

## Cost Comparison

| Option | Monthly Cost | WebSockets | Complexity |
|--------|-------------|------------|------------|
| Lightsail VM | $5-10 | Full support | Low |
| Lightsail Containers | $7-25 | Full support | Low |
| ECS Fargate + ALB | $30+ | Full support | Medium |
| App Runner | $10-20 | 30-min limit | Low |

---

## Scaling Path

```
Phase 1 (Now)           Phase 2 (Growth)           Phase 3 (Scale)
      |                       |                          |
      v                       v                          v
Lightsail $5  ────────>  Lightsail + LB  ────────>  ECS Fargate
(single VM)              (multiple VMs)             (auto-scaling)
```

### When to Scale

| Trigger | Action |
|---------|--------|
| CPU consistently > 80% | Upgrade to larger Lightsail plan |
| Single instance can't handle load | Add instances + load balancer |
| Need auto-scaling | Migrate to ECS Fargate |

---

## Lightsail Plan Options

| Plan | RAM | vCPU | Storage | Transfer | Price |
|------|-----|------|---------|----------|-------|
| Nano | 512MB | 1 | 20GB | 1TB | $3.50 |
| Micro | 1GB | 1 | 40GB | 2TB | $5 |
| Small | 2GB | 1 | 60GB | 3TB | $10 |
| Medium | 4GB | 2 | 80GB | 4TB | $20 |

**Starting plan:** Micro ($5) - sufficient for initial deployment.

---

## Deployment Architecture

```
                Internet
                    |
                    v
    ┌───────────────────────────────┐
    │     Lightsail VM ($5/mo)      │
    │  ┌─────────────────────────┐  │
    │  │   Caddy (reverse proxy) │  │  <- Handles HTTPS (free Let's Encrypt)
    │  │   Port 443              │  │
    │  └───────────┬─────────────┘  │
    │              │                │
    │  ┌───────────v─────────────┐  │
    │  │   Docker Container      │  │  <- Algorave server
    │  │   Port 8080             │  │
    │  └─────────────────────────┘  │
    └───────────────────────────────┘
                    │
                    v
              Supabase (DB)         <- Existing managed database
```

---

## Why Lightsail Over Fargate

1. **Cost:** $5/month vs $30/month minimum
2. **Simplicity:** Direct server access, no ALB complexity
3. **WebSockets:** No timeout limits
4. **Flexibility:** Full control over the environment
5. **Growth path:** Can migrate to ECS when needed
6. **Still AWS:** Teaches fundamentals (EC2, networking, Docker)

---

## Next Steps

1. Create Dockerfile for the application
2. Set up Lightsail instance
3. Configure Caddy for HTTPS
4. Deploy container
5. Set up CI/CD (GitHub Actions)

---

## Future Considerations

- **When traffic grows:** Add Lightsail load balancer + more instances
- **When auto-scaling needed:** Migrate to ECS Fargate
- **Cost optimization:** Monitor usage, right-size instance
