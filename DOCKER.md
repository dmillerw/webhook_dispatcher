# Docker Quick Reference

## Quick Commands

### Build and Run

```bash
# Build image
docker build -t webhook-dispatch:latest .

# Run container
docker run -d \
  -p 8080:8080 \
  -p 9090:9090 \
  -v $(pwd)/configs:/app/configs:ro \
  -v webhook-data:/app/data \
  --name webhook-dispatch \
  webhook-dispatch:latest

# Check status
docker ps
docker logs webhook-dispatch

# Stop and remove
docker stop webhook-dispatch
docker rm webhook-dispatch
```

### Using Docker Compose

```bash
# Start (basic)
docker-compose up -d

# Start (production with monitoring)
docker-compose -f docker-compose.prod.yml up -d

# Start (development)
docker-compose -f docker-compose.dev.yml up -d

# View logs
docker-compose logs -f webhook-dispatch

# Restart
docker-compose restart

# Stop
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## Docker Compose Files

| File | Purpose | Use Case |
|------|---------|----------|
| `docker-compose.yml` | Basic deployment | Local testing, simple deployments |
| `docker-compose.prod.yml` | Production stack | Full production with nginx, Prometheus, Grafana |
| `docker-compose.dev.yml` | Development | Hot-reload, debugging, testing |

## GitHub Actions Workflows

### 1. `.github/workflows/docker-build.yml`

**Triggers:**
- Push to `main` or `develop` branch
- Pull requests to `main`
- Git tags matching `v*`

**Actions:**
- Builds multi-arch image (amd64, arm64)
- Pushes to GitHub Container Registry
- Creates tags: `latest`, `develop`, `v1.2.3`, `sha-abc123`
- Runs Trivy security scan

**Image location:** `ghcr.io/user/webhook-dispatch:latest`

### 2. `.github/workflows/docker-publish-dockerhub.yml`

**Triggers:**
- Git tags matching `v*`

**Actions:**
- Builds multi-arch image (amd64, arm64, arm/v7)
- Pushes to Docker Hub
- Updates Docker Hub description

**Required secrets:**
- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`

**Image location:** `yourusername/webhook-dispatch:latest`

## Environment Variables

Create `.env` file from template:

```bash
cp .env.example .env
```

Edit with your values:

```env
API_TOKEN=your-api-token
SLACK_TOKEN=your-slack-webhook
GRAFANA_PASSWORD=secure-password
```

## Image Tags

### GitHub Container Registry

```bash
# Latest stable
ghcr.io/user/webhook-dispatch:latest

# Development branch
ghcr.io/user/webhook-dispatch:develop

# Specific version
ghcr.io/user/webhook-dispatch:v1.2.3
ghcr.io/user/webhook-dispatch:1.2
ghcr.io/user/webhook-dispatch:1

# By commit SHA
ghcr.io/user/webhook-dispatch:main-abc1234
```

### Docker Hub

```bash
# Latest stable
yourusername/webhook-dispatch:latest

# Specific version
yourusername/webhook-dispatch:1.2.3
```

## Multi-Architecture Support

Built for:
- `linux/amd64` (x86_64)
- `linux/arm64` (ARM 64-bit)
- `linux/arm/v7` (ARM 32-bit, Docker Hub only)

Docker automatically pulls the correct architecture for your system.

## Volume Mounts

### Configuration (Read-only)

```bash
-v $(pwd)/configs:/app/configs:ro
```

Mount your `destinations.json` configuration file.

### Data (Persistent)

```bash
-v webhook-data:/app/data
```

Stores BadgerDB retry queue. Survives container restarts.

## Port Mapping

| Container Port | Host Port | Purpose |
|---------------|-----------|---------|
| 8080 | 8080 | Webhook ingestion, Admin UI |
| 9090 | 9090 | Prometheus metrics |

## Health Checks

Built-in Docker health check:

```bash
# Check container health
docker inspect webhook-dispatch | grep -A 5 Health

# Manual health check
curl http://localhost:8080/health
```

## Security Features

### Non-root User

Container runs as user `webhook` (UID 1000, GID 1000).

### Read-only Config

Configuration mounted read-only to prevent accidental modification.

### Network Isolation (Production)

Services communicate on internal networks not exposed to host.

## Troubleshooting

### Build Failures

```bash
# Clean build cache
docker builder prune -a

# Build with no cache
docker build --no-cache -t webhook-dispatch:latest .
```

### Permission Issues

```bash
# Fix data directory permissions
docker run --rm -v webhook-data:/data alpine chown -R 1000:1000 /data
```

### Port Conflicts

```bash
# Check what's using port 8080
sudo lsof -i :8080

# Use different ports
docker run -p 8081:8080 -p 9091:9090 webhook-dispatch:latest
```

### View Container Logs

```bash
# Last 100 lines
docker logs --tail 100 webhook-dispatch

# Follow logs
docker logs -f webhook-dispatch

# With timestamps
docker logs -t webhook-dispatch
```

### Exec Into Container

```bash
# Get shell
docker exec -it webhook-dispatch sh

# Run command
docker exec webhook-dispatch ls -la /app
```

## Performance Tuning

### Resource Limits

```bash
docker run -d \
  --memory="512m" \
  --cpus="2" \
  webhook-dispatch:latest
```

Or in docker-compose.yml:

```yaml
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 512M
    reservations:
      cpus: '0.5'
      memory: 256M
```

### Worker Pool Size

```bash
docker run -d webhook-dispatch:latest -workers 200 -queue-size 2000
```

## Development Tips

### Live Reload

Use dev compose file with mounted source:

```bash
docker-compose -f docker-compose.dev.yml up
```

### Debug Mode

```bash
docker run -d \
  -e LOG_LEVEL=debug \
  webhook-dispatch:latest
```

### Build Local Image

```bash
# Build
docker build -t webhook-dispatch:dev .

# Tag for testing
docker tag webhook-dispatch:dev webhook-dispatch:test

# Run test image
docker run -p 8080:8080 webhook-dispatch:test
```

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Deploy to Production
  run: |
    ssh user@server "cd /opt/webhook-dispatch && \
      docker-compose pull && \
      docker-compose up -d"
```

### GitLab CI Example

```yaml
deploy:
  stage: deploy
  script:
    - docker pull ghcr.io/user/webhook-dispatch:latest
    - docker-compose up -d
  only:
    - main
```

## Cleanup

```bash
# Remove container
docker rm -f webhook-dispatch

# Remove image
docker rmi webhook-dispatch:latest

# Remove volumes
docker volume rm webhook-data

# Clean everything (careful!)
docker-compose down -v
docker system prune -a
```

## Getting Help

```bash
# See available flags
docker run --rm webhook-dispatch:latest -help

# View image info
docker inspect webhook-dispatch:latest

# Check image size
docker images webhook-dispatch:latest
```
