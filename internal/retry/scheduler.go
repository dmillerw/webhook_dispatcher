package retry

import (
	"context"
	"log/slog"
	"time"

	"github.com/user/webhook-dispatch/internal/metrics"
	"github.com/user/webhook-dispatch/internal/models"
)

// JobSubmitter defines the interface for submitting dispatch jobs
type JobSubmitter interface {
	SubmitRetry(event *models.WebhookEvent, dest *models.Destination, retryCount int)
}

// Scheduler periodically processes retry queue
type Scheduler struct {
	store      Store
	submitter  JobSubmitter
	metrics    *metrics.Collector
	logger     *slog.Logger
	interval   time.Duration
}

// NewScheduler creates a new retry scheduler
func NewScheduler(store Store, submitter JobSubmitter, metrics *metrics.Collector, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		store:     store,
		submitter: submitter,
		metrics:   metrics,
		logger:    logger,
		interval:  30 * time.Second,
	}
}

// Start starts the retry scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("starting retry scheduler", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Process immediately on start
	s.processRetries()

	for {
		select {
		case <-ticker.C:
			s.processRetries()

		case <-ctx.Done():
			s.logger.Info("retry scheduler stopping")
			return
		}
	}
}

// processRetries processes all due retry entries
func (s *Scheduler) processRetries() {
	now := time.Now()

	// Get all due retries
	entries, err := s.store.GetDue(now)
	if err != nil {
		s.logger.Error("failed to get due retries", "error", err)
		return
	}

	if len(entries) > 0 {
		s.logger.Info("processing retries", "count", len(entries))
	}

	// Submit each retry to worker pool
	for _, entry := range entries {
		s.logger.Debug("retrying webhook",
			"webhook_id", entry.WebhookEvent.ID,
			"destination", entry.Destination.Name,
			"attempt", entry.Attempt+1,
		)

		s.submitter.SubmitRetry(&entry.WebhookEvent, &entry.Destination, entry.Attempt)
	}

	// Update metrics
	count, err := s.store.Count()
	if err != nil {
		s.logger.Error("failed to get retry queue size", "error", err)
	} else {
		s.metrics.UpdateRetryQueueSize(count)
	}
}
