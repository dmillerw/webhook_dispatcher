package models

import (
	"time"
)

// RetryEntry represents a failed webhook delivery pending retry
type RetryEntry struct {
	ID           string       `json:"id"`
	WebhookEvent WebhookEvent `json:"webhook_event"`
	Destination  Destination  `json:"destination"`
	Attempt      int          `json:"attempt"`
	NextRetryAt  time.Time    `json:"next_retry_at"`
	LastError    string       `json:"last_error"`
	CreatedAt    time.Time    `json:"created_at"`
}
