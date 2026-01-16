package dispatcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/user/webhook-dispatch/internal/metrics"
	"github.com/user/webhook-dispatch/internal/models"
	"github.com/user/webhook-dispatch/internal/retry"
)

// DispatchJob represents a webhook dispatch job
type DispatchJob struct {
	Event       *models.WebhookEvent
	Destination *models.Destination
	RetryCount  int
}

// WorkerPool manages concurrent webhook dispatching
type WorkerPool struct {
	jobQueue    chan *DispatchJob
	workerCount int
	client      *http.Client
	metrics     *metrics.Collector
	retryStore  retry.Store
	logger      *slog.Logger
	wg          sync.WaitGroup
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workerCount, queueSize int, metrics *metrics.Collector, retryStore retry.Store, logger *slog.Logger) *WorkerPool {
	return &WorkerPool{
		jobQueue:    make(chan *DispatchJob, queueSize),
		workerCount: workerCount,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		metrics:    metrics,
		retryStore: retryStore,
		logger:     logger,
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start(ctx context.Context) {
	p.logger.Info("starting worker pool", "workers", p.workerCount)

	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

// Submit submits a dispatch job to the pool
func (p *WorkerPool) Submit(job *DispatchJob) {
	p.jobQueue <- job
}

// SubmitRetry submits a retry job to the pool (implements retry.JobSubmitter interface)
func (p *WorkerPool) SubmitRetry(event *models.WebhookEvent, dest *models.Destination, retryCount int) {
	p.Submit(&DispatchJob{
		Event:       event,
		Destination: dest,
		RetryCount:  retryCount,
	})
}

// Wait waits for all workers to finish
func (p *WorkerPool) Wait() {
	close(p.jobQueue)
	p.wg.Wait()
}

// worker processes dispatch jobs
func (p *WorkerPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	p.logger.Debug("worker started", "worker_id", id)

	for {
		select {
		case job, ok := <-p.jobQueue:
			if !ok {
				p.logger.Debug("worker stopping (queue closed)", "worker_id", id)
				return
			}
			p.processJob(job)

		case <-ctx.Done():
			p.logger.Debug("worker stopping (context cancelled)", "worker_id", id)
			return
		}
	}
}

// processJob dispatches a webhook to a destination
func (p *WorkerPool) processJob(job *DispatchJob) {
	start := time.Now()
	dest := job.Destination

	p.logger.Debug("dispatching webhook",
		"webhook_id", job.Event.ID,
		"destination", dest.Name,
		"attempt", job.RetryCount+1,
	)

	// Create HTTP request
	req, err := http.NewRequest(dest.Method, dest.URL, bytes.NewReader(job.Event.RawBody))
	if err != nil {
		p.handleFailure(job, fmt.Sprintf("failed to create request: %v", err))
		return
	}

	// Set headers from original webhook
	for key, value := range job.Event.Headers {
		req.Header.Set(key, value)
	}

	// Set custom headers from destination config
	for key, value := range dest.Headers {
		req.Header.Set(key, value)
	}

	// Set timeout if specified
	if dest.TimeoutSecs > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(dest.TimeoutSecs)*time.Second)
		defer cancel()
		req = req.WithContext(ctx)
	}

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		p.handleFailure(job, fmt.Sprintf("request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	// Drain response body to enable connection reuse
	io.Copy(io.Discard, resp.Body)

	// Record metrics
	duration := time.Since(start).Seconds()
	p.metrics.RecordDispatchDuration(dest.ID, duration)

	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		p.handleSuccess(job, duration)
	} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// Client error - don't retry
		p.logger.Warn("dispatch failed with client error",
			"webhook_id", job.Event.ID,
			"destination", dest.Name,
			"status_code", resp.StatusCode,
		)
		p.metrics.RecordDispatch(dest.ID, fmt.Sprintf("%d", resp.StatusCode))
	} else {
		// Server error - retry
		p.handleFailure(job, fmt.Sprintf("server error: %d", resp.StatusCode))
	}
}

// handleSuccess handles a successful dispatch
func (p *WorkerPool) handleSuccess(job *DispatchJob, duration float64) {
	p.logger.Info("dispatch successful",
		"webhook_id", job.Event.ID,
		"destination", job.Destination.Name,
		"duration_ms", int(duration*1000),
	)

	p.metrics.RecordDispatch(job.Destination.ID, "success")

	// Remove from retry queue if this was a retry
	if job.RetryCount > 0 {
		retryID := fmt.Sprintf("%s:%s", job.Event.ID, job.Destination.ID)
		if err := p.retryStore.Delete(retryID); err != nil {
			p.logger.Error("failed to delete retry entry", "error", err)
		}
	}
}

// handleFailure handles a failed dispatch
func (p *WorkerPool) handleFailure(job *DispatchJob, errorMsg string) {
	p.logger.Warn("dispatch failed",
		"webhook_id", job.Event.ID,
		"destination", job.Destination.Name,
		"attempt", job.RetryCount+1,
		"error", errorMsg,
	)

	p.metrics.RecordDispatch(job.Destination.ID, "failure")

	// Check if we should retry
	if job.RetryCount+1 >= job.Destination.RetryPolicy.MaxAttempts {
		p.logger.Error("max retry attempts reached, giving up",
			"webhook_id", job.Event.ID,
			"destination", job.Destination.Name,
		)
		return
	}

	// Calculate next retry time
	nextRetry := calculateNextRetry(job.RetryCount+1, job.Destination.RetryPolicy.BackoffSchedule)

	// Create retry entry
	retryEntry := &models.RetryEntry{
		ID:           fmt.Sprintf("%s:%s", job.Event.ID, job.Destination.ID),
		WebhookEvent: *job.Event,
		Destination:  *job.Destination,
		Attempt:      job.RetryCount + 1,
		NextRetryAt:  nextRetry,
		LastError:    errorMsg,
		CreatedAt:    time.Now(),
	}

	// Save to retry store
	if err := p.retryStore.Save(retryEntry); err != nil {
		p.logger.Error("failed to save retry entry", "error", err)
	} else {
		p.logger.Info("scheduled retry",
			"webhook_id", job.Event.ID,
			"destination", job.Destination.Name,
			"next_retry", nextRetry.Format(time.RFC3339),
		)
	}
}

// calculateNextRetry calculates the next retry time using exponential backoff
func calculateNextRetry(attempt int, backoffSchedule []int) time.Time {
	idx := attempt - 1
	if idx >= len(backoffSchedule) {
		idx = len(backoffSchedule) - 1
	}

	delay := time.Duration(backoffSchedule[idx]) * time.Second
	return time.Now().Add(delay)
}
