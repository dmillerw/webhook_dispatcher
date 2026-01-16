# Webhook Dispatch

A lightweight, high-performance webhook dispatcher written in Go that receives webhooks, extracts metadata using JSONPath, and forwards them to multiple filtered destinations with automatic retry logic and hot-reload configuration.

## Features

- **High Performance**: Handles 1000+ webhooks/second with minimal resource usage
- **JSONPath Metadata Extraction**: Extract data from webhook payloads using simple JSONPath expressions
- **Intelligent Filtering**: Route webhooks to destinations based on metadata filters
- **Automatic Retries**: Failed deliveries are retried with exponential backoff for up to 7 days
- **Hot-Reload Configuration**: Update destinations without restarting the application
- **Admin UI**: Full CRUD interface to manage webhook destinations
- **Prometheus Metrics**: Built-in observability for monitoring webhook flows
- **Lightweight**: < 100MB memory footprint, no external database required

## Architecture

- **HTTP Router**: Chi (lightweight and fast)
- **Metadata Extraction**: gjson (zero-allocation JSONPath)
- **Retry Storage**: BadgerDB (embedded key-value store)
- **Metrics**: Prometheus
- **Hot-Reload**: fsnotify

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/user/webhook-dispatch.git
cd webhook-dispatch

# Start the application
docker-compose up -d

# View logs
docker-compose logs -f
```

The application will be available at:
- Webhook ingestion + Admin UI: http://localhost:8080
- Prometheus metrics: http://localhost:9090/metrics

### Using Go

```bash
# Install dependencies
go mod download

# Run the application
go run cmd/webhook-dispatch/main.go

# Or build and run
go build -o webhook-dispatch cmd/webhook-dispatch/main.go
./webhook-dispatch
```

### Configuration Flags

```bash
./webhook-dispatch \
  -config configs/destinations.json \
  -data-dir data/retries \
  -port 8080 \
  -metrics-port 9090 \
  -workers 100 \
  -queue-size 1000
```

## Configuration

### Destination Configuration

Destinations are defined in `configs/destinations.json`:

```json
{
  "destinations": [
    {
      "id": "slack-notifications",
      "name": "Slack Webhook",
      "url": "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json"
      },
      "timeout_seconds": 30,
      "filters": [
        {
          "field": "event.type",
          "operator": "in",
          "values": ["push", "pull_request"]
        },
        {
          "field": "repository",
          "operator": "contains",
          "value": "myproject"
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

### Filter Operators

- **eq**: Exact match (`"operator": "eq", "value": "push"`)
- **ne**: Not equal (`"operator": "ne", "value": "disabled"`)
- **in**: Value in list (`"operator": "in", "values": ["push", "pr"]`)
- **contains**: Substring match (`"operator": "contains", "value": "main"`)
- **regex**: Regular expression (`"operator": "regex", "value": "^feature-.*"`)

### Environment Variables

Headers support environment variable substitution using `${VAR_NAME}`:

```json
{
  "headers": {
    "Authorization": "Bearer ${API_TOKEN}",
    "X-Custom-Header": "${CUSTOM_VALUE}"
  }
}
```

## Webhook Path Registration

Webhook paths are registered in code (see `cmd/webhook-dispatch/main.go`):

```go
// Register GitHub webhooks
registry.Register("/webhooks/github", []metadata.ExtractionRule{
    {Name: "event.type", JSONPath: "action"},
    {Name: "event.branch", JSONPath: "ref"},
    {Name: "repository", JSONPath: "repository.full_name"},
})

// Register GitLab webhooks
registry.Register("/webhooks/gitlab", []metadata.ExtractionRule{
    {Name: "event.type", JSONPath: "object_kind"},
    {Name: "event.branch", JSONPath: "project.default_branch"},
})
```

## Usage

### Sending Webhooks

Send webhooks to registered paths:

```bash
curl -X POST http://localhost:8080/webhooks/github \
  -H "Content-Type: application/json" \
  -d '{
    "action": "push",
    "ref": "refs/heads/main",
    "repository": {
      "full_name": "user/myproject"
    }
  }'
```

### Admin UI

Access the admin interface at http://localhost:8080/admin/

Features:
- View all destinations
- Add new destinations
- Edit existing destinations
- Delete destinations
- Enable/disable destinations

### Admin API

Manage destinations programmatically:

```bash
# List all destinations
curl http://localhost:8080/api/destinations

# Get specific destination
curl http://localhost:8080/api/destinations/{id}

# Create destination
curl -X POST http://localhost:8080/api/destinations \
  -H "Content-Type: application/json" \
  -d @new-destination.json

# Update destination
curl -X PUT http://localhost:8080/api/destinations/{id} \
  -H "Content-Type: application/json" \
  -d @updated-destination.json

# Delete destination
curl -X DELETE http://localhost:8080/api/destinations/{id}

# Toggle enabled/disabled
curl -X PATCH http://localhost:8080/api/destinations/{id}/toggle
```

### Metrics

Access Prometheus metrics at http://localhost:9090/metrics

Key metrics:
- `webhooks_received_total{path}` - Total webhooks received per path
- `webhook_processing_duration_seconds{path}` - Processing time per path
- `dispatches_total{destination, status}` - Total dispatches per destination and status
- `dispatch_duration_seconds{destination}` - Dispatch time per destination
- `retry_queue_size` - Current number of entries in retry queue

Example Prometheus queries:

```promql
# Webhook ingestion rate
rate(webhooks_received_total[5m])

# P95 processing latency
histogram_quantile(0.95, rate(webhook_processing_duration_seconds_bucket[5m]))

# Dispatch success rate
rate(dispatches_total{status="success"}[5m]) / rate(dispatches_total[5m])

# Failed deliveries in retry queue
retry_queue_size
```

## How It Works

1. **Webhook Ingestion**: Incoming webhooks are received at registered paths
2. **Metadata Extraction**: JSONPath rules extract relevant data from the webhook body
3. **Filtering**: Each destination's filters are evaluated against the extracted metadata
4. **Dispatching**: Matching webhooks are sent to a worker pool for concurrent delivery
5. **Retry Handling**: Failed deliveries are persisted to BadgerDB and retried with exponential backoff
6. **Cleanup**: Retry entries older than 7 days are automatically removed

## Retry Behavior

- Failed deliveries are retried up to 10 times (configurable)
- Exponential backoff schedule: 1m, 5m, 15m, 1h, 6h, 24h (configurable)
- Retry entries are persisted to disk and survive restarts
- 4xx client errors are not retried (considered permanent failures)
- 5xx server errors and network failures trigger retries

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run with race detector
go test -race ./...

# Run E2E tests
go test ./test/e2e/...
```

### Project Structure

```
webhook_dispatch/
├── cmd/webhook-dispatch/     # Application entry point
├── internal/
│   ├── api/                  # HTTP handlers and middleware
│   ├── config/               # Configuration loading and hot-reload
│   ├── dispatcher/           # Webhook routing and worker pool
│   ├── metadata/             # JSONPath metadata extraction
│   ├── metrics/              # Prometheus metrics
│   ├── models/               # Data structures
│   └── retry/                # Retry storage and scheduling
├── web/static/               # Admin UI (HTML/CSS/JS)
├── configs/                  # Configuration files
├── test/                     # E2E tests
└── data/                     # Runtime data (BadgerDB)
```

## Performance

Benchmarks on modest hardware (4 CPU cores, 8GB RAM):

- **Ingestion**: 1000+ webhooks/second
- **Dispatch**: 500+ concurrent dispatches
- **Memory**: < 100MB baseline
- **Latency**: P95 < 100ms (excluding downstream)

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

### Logs

Structured JSON logs are written to stdout:

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "webhook received",
  "webhook_id": "123e4567-e89b-12d3-a456-426614174000",
  "path": "/webhooks/github",
  "metadata": {"event.type": "push"}
}
```

## Security

- **Input Validation**: Request body size limited to 10MB
- **HTTPS**: Configure reverse proxy (nginx, Caddy) with TLS
- **Secrets**: Use environment variables for sensitive data
- **Authentication**: Add basic auth or API key middleware for admin endpoints (TODO)

## Troubleshooting

### High Retry Queue Size

- Check downstream service health
- Review dispatch error logs
- Consider increasing worker pool size

### Config Not Reloading

- Check file permissions
- Verify JSON syntax
- Review validation errors in logs

### High Memory Usage

- Check retry queue size: `retry_queue_size` metric
- Review BadgerDB settings
- Consider archiving old retry entries

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Support

For issues and feature requests, please use the GitHub issue tracker.
