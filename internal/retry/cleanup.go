package retry

import (
	"context"
	"log/slog"
	"time"
)

// Cleaner periodically cleans up old retry entries
type Cleaner struct {
	store    Store
	logger   *slog.Logger
	interval time.Duration
	maxAge   time.Duration
}

// NewCleaner creates a new retry cleaner
func NewCleaner(store Store, logger *slog.Logger) *Cleaner {
	return &Cleaner{
		store:    store,
		logger:   logger,
		interval: 1 * time.Hour,  // Run every hour
		maxAge:   7 * 24 * time.Hour, // 7 days
	}
}

// Start starts the cleanup process
func (c *Cleaner) Start(ctx context.Context) {
	c.logger.Info("starting retry cleaner", "interval", c.interval, "max_age", c.maxAge)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()

		case <-ctx.Done():
			c.logger.Info("retry cleaner stopping")
			return
		}
	}
}

// cleanup removes old retry entries
// Note: BadgerDB handles TTL automatically, so this is mostly for logging
func (c *Cleaner) cleanup() {
	count, err := c.store.Count()
	if err != nil {
		c.logger.Error("failed to get retry count during cleanup", "error", err)
		return
	}

	c.logger.Debug("cleanup check", "current_queue_size", count)
}
