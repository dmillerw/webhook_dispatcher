package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector holds all Prometheus metrics
type Collector struct {
	WebhooksReceived   *prometheus.CounterVec
	WebhookDuration    *prometheus.HistogramVec
	DispatchesTotal    *prometheus.CounterVec
	DispatchDuration   *prometheus.HistogramVec
	RetryQueueSize     prometheus.Gauge
}

// NewCollector creates and registers all Prometheus metrics
func NewCollector() *Collector {
	return &Collector{
		WebhooksReceived: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "webhooks_received_total",
				Help: "Total number of webhooks received",
			},
			[]string{"path"},
		),

		WebhookDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "webhook_processing_duration_seconds",
				Help:    "Time spent processing webhooks",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"path"},
		),

		DispatchesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dispatches_total",
				Help: "Total number of webhook dispatches",
			},
			[]string{"destination", "status"},
		),

		DispatchDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dispatch_duration_seconds",
				Help:    "Time spent dispatching webhooks",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"destination"},
		),

		RetryQueueSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "retry_queue_size",
				Help: "Current number of entries in retry queue",
			},
		),
	}
}

// RecordWebhookReceived increments the webhook received counter
func (c *Collector) RecordWebhookReceived(path string) {
	c.WebhooksReceived.WithLabelValues(path).Inc()
}

// RecordWebhookDuration records the webhook processing duration
func (c *Collector) RecordWebhookDuration(path string, duration float64) {
	c.WebhookDuration.WithLabelValues(path).Observe(duration)
}

// RecordDispatch records a dispatch attempt
func (c *Collector) RecordDispatch(destination, status string) {
	c.DispatchesTotal.WithLabelValues(destination, status).Inc()
}

// RecordDispatchDuration records the dispatch duration
func (c *Collector) RecordDispatchDuration(destination string, duration float64) {
	c.DispatchDuration.WithLabelValues(destination).Observe(duration)
}

// UpdateRetryQueueSize updates the retry queue size gauge
func (c *Collector) UpdateRetryQueueSize(size int) {
	c.RetryQueueSize.Set(float64(size))
}
