package models

import (
	"time"
)

// WebhookEvent represents an incoming webhook
type WebhookEvent struct {
	ID         string            `json:"id"`
	Path       string            `json:"path"`
	RawBody    []byte            `json:"raw_body"`
	Headers    map[string]string `json:"headers"`
	Metadata   map[string]string `json:"metadata"`
	ReceivedAt time.Time         `json:"received_at"`
}
