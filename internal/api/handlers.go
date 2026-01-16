package api

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/user/webhook-dispatch/internal/dispatcher"
	"github.com/user/webhook-dispatch/internal/metadata"
	"github.com/user/webhook-dispatch/internal/metrics"
	"github.com/user/webhook-dispatch/internal/models"
)

// WebhookHandler handles incoming webhook requests
type WebhookHandler struct {
	registry   *metadata.Registry
	dispatcher *dispatcher.Dispatcher
	metrics    *metrics.Collector
	logger     *slog.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(
	registry *metadata.Registry,
	dispatcher *dispatcher.Dispatcher,
	metrics *metrics.Collector,
	logger *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		registry:   registry,
		dispatcher: dispatcher,
		metrics:    metrics,
		logger:     logger,
	}
}

// ServeHTTP handles webhook ingestion
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	path := r.URL.Path

	// Check if path is registered
	if !h.registry.IsRegistered(path) {
		h.logger.Warn("unknown webhook path", "path", path)
		http.Error(w, "Unknown webhook path", http.StatusNotFound)
		return
	}

	// Record metrics
	h.metrics.RecordWebhookReceived(path)

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", "error", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Extract metadata
	webhookMetadata, _ := h.registry.Extract(path, body)

	// Copy headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Create webhook event
	event := &models.WebhookEvent{
		ID:         uuid.New().String(),
		Path:       path,
		RawBody:    body,
		Headers:    headers,
		Metadata:   webhookMetadata,
		ReceivedAt: time.Now(),
	}

	h.logger.Info("webhook received",
		"webhook_id", event.ID,
		"path", path,
		"metadata", webhookMetadata,
	)

	// Dispatch to destinations
	h.dispatcher.Dispatch(event)

	// Record processing duration
	duration := time.Since(start).Seconds()
	h.metrics.RecordWebhookDuration(path, duration)

	// Return success
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"accepted"}`))
}
