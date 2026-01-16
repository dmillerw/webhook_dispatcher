package config

import (
	"context"
	"log/slog"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches the config file for changes and triggers reloads
type Watcher struct {
	loader  *Loader
	watcher *fsnotify.Watcher
	logger  *slog.Logger
}

// NewWatcher creates a new config file watcher
func NewWatcher(loader *Loader, logger *slog.Logger) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Add config file to watch list
	if err := w.Add(loader.configPath); err != nil {
		w.Close()
		return nil, err
	}

	return &Watcher{
		loader:  loader,
		watcher: w,
		logger:  logger,
	}, nil
}

// Start starts watching for config file changes
func (w *Watcher) Start(ctx context.Context) {
	w.logger.Info("config watcher started", "path", w.loader.configPath)

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only reload on write events
			if event.Op&fsnotify.Write == fsnotify.Write {
				w.logger.Info("config file changed, reloading", "path", event.Name)

				if err := w.loader.Reload(); err != nil {
					w.logger.Error("failed to reload config", "error", err)
				} else {
					w.logger.Info("config reloaded successfully")
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("config watcher error", "error", err)

		case <-ctx.Done():
			w.logger.Info("config watcher stopping")
			w.watcher.Close()
			return
		}
	}
}
