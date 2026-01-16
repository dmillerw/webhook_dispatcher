# Deployment Guide

This guide covers different deployment scenarios for the Webhook Dispatch application.

## Table of Contents

- [Docker Compose Deployment](#docker-compose-deployment)
- [GitHub Container Registry](#github-container-registry)
- [Docker Hub](#docker-hub)
- [Production Setup with Monitoring](#production-setup-with-monitoring)
- [Kubernetes Deployment](#kubernetes-deployment)

## Docker Compose Deployment

### Quick Start

1. **Build and run locally:**
```bash
docker-compose up -d
```

2. **Access the application:**
- Webhooks & Admin UI: http://localhost:8080
- Metrics: http://localhost:9090

3. **View logs:**
```bash
docker-compose logs -f
```

4. **Stop the application:**
```bash
docker-compose down
```

### Using Pre-built Image

Edit `docker-compose.yml` to use a pre-built image:

```yaml
services:
  webhook-dispatch:
    image: ghcr.io/user/webhook-dispatch:latest
    # Remove the 'build' section
```

Then run:
```bash
docker-compose pull
docker-compose up -d
```

## GitHub Container Registry

### Automatic Builds

The GitHub Actions workflow automatically builds and pushes images on:
- Every push to `main` branch → `latest` tag
- Every push to `develop` branch → `develop` tag
- Every tag `v*` → semver tags (`v1.2.3`, `1.2`, `1`, etc.)

### Pull and Run

```bash
# Pull latest version
docker pull ghcr.io/user/webhook-dispatch:latest

# Run container
docker run -d \
  -p 8080:8080 \
  -p 9090:9090 \
  -v $(pwd)/configs:/app/configs:ro \
  -v webhook-data:/app/data \
  --name webhook-dispatch \
  ghcr.io/user/webhook-dispatch:latest
```

### Setup GitHub Actions

1. **Enable GitHub Packages** in your repository settings

2. **Workflow runs automatically** - no additional secrets needed (uses `GITHUB_TOKEN`)

3. **For Docker Hub publishing**, add secrets:
   - `DOCKERHUB_USERNAME`
   - `DOCKERHUB_TOKEN`

## Docker Hub

### Manual Build and Push

```bash
# Build for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 \
  -t yourusername/webhook-dispatch:latest \
  -t yourusername/webhook-dispatch:v1.0.0 \
  --push .
```

### Automated Publishing

Tag your code to trigger automated publishing:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The GitHub Actions workflow will:
1. Build multi-architecture image (amd64, arm64, arm/v7)
2. Push to Docker Hub with version tags
3. Update Docker Hub description from README

## Production Setup with Monitoring

### Prerequisites

- Domain name pointed to your server
- SSL certificates (use Let's Encrypt)

### Setup Steps

1. **Copy environment file:**
```bash
cp .env.example .env
# Edit .env with your secrets
```

2. **Generate SSL certificates (using certbot):**
```bash
# Install certbot
sudo apt-get update
sudo apt-get install certbot

# Generate certificate
sudo certbot certonly --standalone -d webhook.example.com
```

3. **Copy certificates:**
```bash
sudo mkdir -p certs
sudo cp /etc/letsencrypt/live/webhook.example.com/fullchain.pem certs/
sudo cp /etc/letsencrypt/live/webhook.example.com/privkey.pem certs/
sudo chown -R $USER:$USER certs/
```

4. **Update nginx config:**
Edit `nginx/conf.d/webhook-dispatch.conf` and replace `webhook.example.com` with your domain.

5. **Start services:**
```bash
docker-compose -f docker-compose.prod.yml up -d
```

6. **Access services:**
- Application: https://webhook.example.com
- Grafana: http://your-ip:3000 (default: admin/admin)
- Prometheus: http://your-ip:9091

### Monitoring Setup

1. **Login to Grafana** (http://your-ip:3000)
   - Default username: `admin`
   - Password: set in `.env` or default `admin`

2. **Prometheus datasource** is automatically configured

3. **Import dashboard:**
   - Go to Dashboards → Import
   - Upload a Grafana dashboard JSON or create custom dashboards

### Health Checks

```bash
# Check all services
docker-compose -f docker-compose.prod.yml ps

# Check application health
curl https://webhook.example.com/health

# Check metrics
curl http://localhost:9091/api/v1/targets
```

## Development Setup

### Start development environment:
```bash
docker-compose -f docker-compose.dev.yml up -d
```

This includes:
- Hot-reload with Air
- Mock HTTP server (port 1080)
- Webhook testing tool (port 8081)
- Delve debugger (port 2345)

## Security Best Practices

### 1. Use Environment Variables for Secrets

Never commit secrets. Use `.env` file:

```bash
# .env
API_TOKEN=your-secret-token
SLACK_TOKEN=your-slack-webhook-url
```

### 2. Enable Authentication for Admin UI

Add basic auth to nginx config:

```bash
# Create password file
sudo apt-get install apache2-utils
htpasswd -c nginx/.htpasswd admin

# Uncomment auth lines in nginx/conf.d/webhook-dispatch.conf
```

### 3. Restrict Metrics Endpoint

The nginx config blocks public access to `/metrics`. Access metrics via:
- Internal network only
- Prometheus scraping
- VPN connection

### 4. Use HTTPS

Always use SSL certificates in production. Free options:
- Let's Encrypt (recommended)
- Cloudflare SSL

### 5. Network Isolation

The production setup uses separate networks:
- `internal` - Application communication (not exposed)
- `monitoring` - Monitoring stack (limited exposure)

## Updating the Application

### Zero-Downtime Update

```bash
# Pull latest image
docker-compose pull

# Recreate containers
docker-compose up -d

# Old containers are removed after new ones are healthy
```

### With docker-compose.prod.yml

```bash
docker-compose -f docker-compose.prod.yml pull
docker-compose -f docker-compose.prod.yml up -d
```

## Backup and Restore

### Backup Retry Queue

```bash
# Create backup
docker run --rm \
  -v webhook_dispatch_webhook-data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/webhook-data-$(date +%Y%m%d).tar.gz -C /data .
```

### Restore from Backup

```bash
# Stop application
docker-compose down

# Restore data
docker run --rm \
  -v webhook_dispatch_webhook-data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar xzf /backup/webhook-data-20240115.tar.gz -C /data

# Start application
docker-compose up -d
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker-compose logs webhook-dispatch

# Check if ports are available
sudo lsof -i :8080
sudo lsof -i :9090
```

### Database Lock Issues

```bash
# Remove locks
docker-compose down
docker volume rm webhook_dispatch_webhook-data
docker-compose up -d
```

### High Memory Usage

```bash
# Check resource usage
docker stats webhook-dispatch

# Adjust memory limits in docker-compose.prod.yml
```

### Configuration Not Reloading

```bash
# Check file permissions
ls -la configs/

# Force reload by restarting
docker-compose restart webhook-dispatch
```

## Scaling

### Horizontal Scaling

To scale across multiple instances:

1. **Use external load balancer** (nginx, HAProxy, AWS ALB)
2. **Keep separate retry storage** per instance (acceptable for this use case)
3. **Share config** via mounted volume or config management system

Example with nginx load balancer:

```nginx
upstream webhook_backend {
    least_conn;
    server webhook-1:8080;
    server webhook-2:8080;
    server webhook-3:8080;
}
```

### Vertical Scaling

Adjust resource limits in `docker-compose.prod.yml`:

```yaml
deploy:
  resources:
    limits:
      cpus: '4'
      memory: 2G
```

## Monitoring Alerts

### Prometheus Alert Rules

Create `prometheus/alerts/webhook-dispatch.yml`:

```yaml
groups:
  - name: webhook_dispatch
    interval: 30s
    rules:
      - alert: HighRetryQueueSize
        expr: retry_queue_size > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High retry queue size"

      - alert: LowDispatchSuccessRate
        expr: rate(dispatches_total{status="success"}[5m]) / rate(dispatches_total[5m]) < 0.9
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Low dispatch success rate"
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/user/webhook-dispatch/issues
- Documentation: See README.md
