package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/user/webhook-dispatch/internal/api"
	"github.com/user/webhook-dispatch/internal/config"
	"github.com/user/webhook-dispatch/internal/dispatcher"
	"github.com/user/webhook-dispatch/internal/metadata"
	"github.com/user/webhook-dispatch/internal/metrics"
	"github.com/user/webhook-dispatch/internal/retry"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "configs/destinations.json", "path to destinations config file")
	dataDir := flag.String("data-dir", "data/retries", "path to data directory")
	port := flag.Int("port", 8080, "HTTP port for webhook ingestion and admin UI")
	metricsPort := flag.Int("metrics-port", 9090, "HTTP port for Prometheus metrics")
	workerCount := flag.Int("workers", 100, "number of dispatch workers")
	queueSize := flag.Int("queue-size", 1000, "dispatch queue buffer size")
	flag.Parse()

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("starting webhook dispatch application",
		"config", *configPath,
		"data_dir", *dataDir,
		"port", *port,
		"metrics_port", *metricsPort,
		"workers", *workerCount,
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize metrics
	metricsCollector := metrics.NewCollector()

	// Initialize config loader
	configLoader := config.NewLoader(*configPath)
	if err := configLoader.Load(); err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	logger.Info("config loaded successfully")

	// Start config watcher for hot-reload
	configWatcher, err := config.NewWatcher(configLoader, logger)
	if err != nil {
		logger.Error("failed to create config watcher", "error", err)
		os.Exit(1)
	}
	go configWatcher.Start(ctx)

	// Initialize BadgerDB store
	retryStore, err := retry.NewBadgerStore(*dataDir)
	if err != nil {
		logger.Error("failed to initialize retry store", "error", err)
		os.Exit(1)
	}
	defer retryStore.Close()
	logger.Info("retry store initialized")

	// Initialize worker pool
	workerPool := dispatcher.NewWorkerPool(*workerCount, *queueSize, metricsCollector, retryStore, logger)
	workerPool.Start(ctx)
	logger.Info("worker pool started")

	// Initialize dispatcher
	dispatcherInstance := dispatcher.NewDispatcher(configLoader, workerPool, logger)

	// Initialize retry scheduler
	retryScheduler := retry.NewScheduler(retryStore, workerPool, metricsCollector, logger)
	go retryScheduler.Start(ctx)

	// Initialize retry cleaner
	retryCleaner := retry.NewCleaner(retryStore, logger)
	go retryCleaner.Start(ctx)

	// Initialize metadata registry
	metadataRegistry := metadata.NewRegistry()

	// Register webhook paths
	registerWebhookPaths(metadataRegistry)

	// Setup HTTP router
	r := setupRouter(metadataRegistry, dispatcherInstance, configLoader, metricsCollector, logger)

	// Start main HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("starting HTTP server", "port", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Start metrics server
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *metricsPort),
		Handler: promhttp.Handler(),
	}

	go func() {
		logger.Info("starting metrics server", "port", *metricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server error", "error", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutting down gracefully...")

	// Cancel context to stop background goroutines
	cancel()

	// Shutdown HTTP servers
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("metrics server shutdown error", "error", err)
	}

	// Wait for workers to finish
	workerPool.Wait()

	logger.Info("shutdown complete")
}

// setupRouter configures the HTTP router
func setupRouter(
	registry *metadata.Registry,
	dispatcher *dispatcher.Dispatcher,
	configLoader *config.Loader,
	metrics *metrics.Collector,
	logger *slog.Logger,
) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Recoverer)
	r.Use(api.RequestIDMiddleware)
	r.Use(api.LoggingMiddleware(logger))
	r.Use(api.BodyLimitMiddleware(10 * 1024 * 1024)) // 10MB limit

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Webhook ingestion
	webhookHandler := api.NewWebhookHandler(registry, dispatcher, metrics, logger)
	for _, path := range registry.GetPaths() {
		r.Post(path, webhookHandler.ServeHTTP)
		logger.Info("registered webhook path", "path", path)
	}

	// Admin API
	adminHandler := api.NewAdminHandler(configLoader, logger)
	r.Route("/api", func(r chi.Router) {
		r.Get("/destinations", adminHandler.ListDestinations)
		r.Post("/destinations", adminHandler.CreateDestination)
		r.Get("/destinations/{id}", adminHandler.GetDestination)
		r.Put("/destinations/{id}", adminHandler.UpdateDestination)
		r.Delete("/destinations/{id}", adminHandler.DeleteDestination)
		r.Patch("/destinations/{id}/toggle", adminHandler.ToggleDestination)
	})

	// Serve admin UI static files from disk
	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/admin/*", http.StripPrefix("/admin/", fileServer))
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
	})

	return r
}

// registerWebhookPaths registers webhook paths with metadata extraction rules
func registerWebhookPaths(registry *metadata.Registry) {
	// Field Nation webhooks
	registry.Register("/webhooks/fieldnation", []metadata.ExtractionRule{
		{Name: "event.name", JSONPath: "event.name"},
	})
}
