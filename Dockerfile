FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -a -installsuffix cgo \
    -o webhook-dispatch \
    ./cmd/webhook-dispatch

# Final stage - minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -g 1000 webhook && \
    adduser -D -u 1000 -G webhook webhook

WORKDIR /app

# Create data directory first
RUN mkdir -p data/retries && \
    chown -R webhook:webhook /app

# Copy binary from builder
COPY --from=builder --chown=webhook:webhook /build/webhook-dispatch .

# Copy web static files
COPY --from=builder --chown=webhook:webhook /build/web ./web

# Copy default config (can be overridden with volume)
COPY --from=builder --chown=webhook:webhook /build/configs ./configs

# Switch to non-root user
USER webhook

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["./webhook-dispatch"]
CMD []
