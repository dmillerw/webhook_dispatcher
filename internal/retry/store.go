package retry

import (
	"time"

	"github.com/user/webhook-dispatch/internal/models"
)

// Store defines the interface for retry storage
type Store interface {
	// Save stores a retry entry
	Save(entry *models.RetryEntry) error

	// Get retrieves a retry entry by ID
	Get(id string) (*models.RetryEntry, error)

	// GetDue retrieves all retry entries that are due for retry
	GetDue(before time.Time) ([]*models.RetryEntry, error)

	// Delete removes a retry entry
	Delete(id string) error

	// Count returns the total number of retry entries
	Count() (int, error)

	// Close closes the store
	Close() error
}
