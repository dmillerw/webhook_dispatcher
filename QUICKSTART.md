# Quick Start Guide

## Prerequisites

Before running the application, you need to install Go 1.22 or later.

### Install Go

**macOS:**
```bash
brew install go
```

**Linux:**
```bash
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

## Running the Application

### Step 1: Install Dependencies

```bash
cd /Users/dylan/Documents/Projects/webhook_dispatch
go mod download
```

### Step 2: Run the Application

```bash
go run cmd/webhook-dispatch/main.go
```

The application will start on:
- **Webhooks & Admin UI**: http://localhost:8080
- **Metrics**: http://localhost:9090

### Step 3: Test It Out

#### Send a Test Webhook

```bash
curl -X POST http://localhost:8080/webhooks/github \
  -H "Content-Type: application/json" \
  -d '{
    "action": "push",
    "ref": "refs/heads/main",
    "repository": {"full_name": "test/repo"}
  }'
```

Expected response:
```json
{"status":"accepted"}
```

#### Access the Admin UI

Open your browser to: http://localhost:8080/admin/

You can:
- View all configured destinations
- Add new destinations
- Edit existing destinations
- Enable/disable destinations
- Delete destinations

#### View Metrics

```bash
curl http://localhost:9090/metrics | grep webhook
```

You should see metrics like:
```
webhooks_received_total{path="/webhooks/github"} 1
dispatches_total{destination="example-slack",status="success"} 0
retry_queue_size 0
```

## Building for Production

### Build Binary

```bash
go build -o webhook-dispatch cmd/webhook-dispatch/main.go
./webhook-dispatch
```

### Using Docker

```bash
# Build image
docker build -t webhook-dispatch .

# Run container
docker run -d \
  -p 8080:8080 \
  -p 9090:9090 \
  -v $(pwd)/configs:/app/configs:ro \
  -v webhook-data:/app/data \
  --name webhook-dispatch \
  webhook-dispatch
```

### Using Docker Compose

```bash
docker-compose up -d
docker-compose logs -f
```

## Configuration

### Edit Destinations

Edit `configs/destinations.json`:

```json
{
  "destinations": [
    {
      "id": "my-webhook",
      "name": "My Webhook Endpoint",
      "url": "https://your-server.com/webhook",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json",
        "Authorization": "Bearer YOUR_TOKEN"
      },
      "timeout_seconds": 30,
      "filters": [
        {
          "field": "event.type",
          "operator": "eq",
          "value": "push"
        }
      ],
      "enabled": true,
      "retry_policy": {
        "max_attempts": 10,
        "backoff_schedule": [60, 300, 900, 3600, 21600, 86400]
      }
    }
  ]
}
```

The configuration will automatically reload when you save the file!

### Environment Variables

For sensitive data, use environment variables:

```bash
export API_TOKEN="your-secret-token"
export SLACK_TOKEN="your-slack-token"
```

Then in your config:
```json
{
  "headers": {
    "Authorization": "Bearer ${API_TOKEN}"
  }
}
```

## Testing

### Run Tests

```bash
# All tests
go test ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# E2E tests
go test ./test/e2e/...
```

### Test Webhook Filtering

1. Configure a destination with a filter:
```json
{
  "filters": [
    {"field": "event.type", "operator": "eq", "value": "push"}
  ]
}
```

2. Send matching webhook:
```bash
curl -X POST http://localhost:8080/webhooks/github \
  -d '{"action":"push"}'
```

3. Send non-matching webhook:
```bash
curl -X POST http://localhost:8080/webhooks/github \
  -d '{"action":"opened"}'
```

4. Check metrics to see only matching webhooks were dispatched:
```bash
curl http://localhost:9090/metrics | grep dispatches_total
```

## Next Steps

1. **Add Your Webhook Paths**: Edit `cmd/webhook-dispatch/main.go` to register your webhook paths
2. **Configure Destinations**: Update `configs/destinations.json` with your actual endpoints
3. **Set Up Monitoring**: Integrate with Prometheus/Grafana
4. **Production Deployment**: Use Docker with reverse proxy (nginx/Caddy) for HTTPS
5. **Complete E2E Tests**: Implement full test suite based on `test/e2e/` structure

## Troubleshooting

### "command not found: go"

Go is not installed. See "Install Go" section above.

### Port already in use

Change the port with flags:
```bash
go run cmd/webhook-dispatch/main.go -port 8081 -metrics-port 9091
```

### Config not reloading

Check file permissions and JSON syntax:
```bash
cat configs/destinations.json | jq .
```

### High memory usage

Check retry queue size:
```bash
curl http://localhost:9090/metrics | grep retry_queue_size
```

## Support

- **Documentation**: See [README.md](README.md)
- **Issues**: GitHub issue tracker
- **Architecture**: See implementation plan at `/Users/dylan/.claude/plans/iterative-napping-sunbeam.md`
