# Quick Reference

## Local Development

```bash
# Build and run
docker build -t algopatterns-server .
docker run -p 8080:8080 --env-file .env algopatterns-server

# Test
curl http://localhost:8080/health
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BASE_URL` | Production URL |
| `PORT` | Server port (default: 8080) |
| `JWT_SECRET` | Generate: `openssl rand -base64 64` |
| `SUPABASE_CONNECTION_STRING` | Database URL |
| `ANTHROPIC_API_KEY` | Claude API key |
| `OPENAI_API_KEY` | OpenAI API key (embeddings) |
| `GITHUB_CLIENT_ID` | GitHub OAuth ID |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth secret |
| `GOOGLE_CLIENT_ID` | Google OAuth ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth secret |

## OAuth Callback URLs

| Provider | URL |
|----------|-----|
| GitHub | `https://algopatterns.cc/api/v1/auth/github/callback` |
| Google | `https://algopatterns.cc/api/v1/auth/google/callback` |

## Common Commands

```bash
# View logs
docker logs algopatterns

# Restart
docker restart algopatterns

# Rebuild and deploy
docker compose up -d --build
```

## Production Deployment

See [DOUBLE_INSTANCE_GUIDE.md](./DOUBLE_INSTANCE_GUIDE.md) for full production setup.
