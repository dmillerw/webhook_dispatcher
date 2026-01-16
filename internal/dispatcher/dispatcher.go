package dispatcher

import (
	"log/slog"

	"github.com/user/webhook-dispatch/internal/config"
	"github.com/user/webhook-dispatch/internal/models"
)

// Dispatcher routes webhooks to destinations
type Dispatcher struct {
	configLoader *config.Loader
	workerPool   *WorkerPool
	logger       *slog.Logger
}

// NewDispatcher creates a new dispatcher
func NewDispatcher(configLoader *config.Loader, workerPool *WorkerPool, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		configLoader: configLoader,
		workerPool:   workerPool,
		logger:       logger,
	}
}

// Dispatch dispatches a webhook to all matching destinations
func (d *Dispatcher) Dispatch(event *models.WebhookEvent) {
	config := d.configLoader.GetConfig()

	for i := range config.Destinations {
		dest := &config.Destinations[i]

		// Skip disabled destinations
		if !dest.Enabled {
			continue
		}

		// Check if webhook matches destination filters
		if !EvaluateFilters(event.Metadata, dest.Filters) {
			d.logger.Debug("webhook filtered out",
				"webhook_id", event.ID,
				"destination", dest.Name,
			)
			continue
		}

		// Submit dispatch job to worker pool
		d.logger.Debug("dispatching to destination",
			"webhook_id", event.ID,
			"destination", dest.Name,
		)

		d.workerPool.Submit(&DispatchJob{
			Event:       event,
			Destination: dest,
			RetryCount:  0,
		})
	}
}
