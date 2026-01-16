package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/user/webhook-dispatch/internal/config"
	"github.com/user/webhook-dispatch/internal/models"
)

// AdminHandler handles admin API requests
type AdminHandler struct {
	configLoader *config.Loader
	logger       *slog.Logger
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(configLoader *config.Loader, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		configLoader: configLoader,
		logger:       logger,
	}
}

// ListDestinations returns all destinations
func (h *AdminHandler) ListDestinations(w http.ResponseWriter, r *http.Request) {
	config := h.configLoader.GetConfig()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config.Destinations)
}

// GetDestination returns a specific destination
func (h *AdminHandler) GetDestination(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	config := h.configLoader.GetConfig()

	for _, dest := range config.Destinations {
		if dest.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(dest)
			return
		}
	}

	http.Error(w, "Destination not found", http.StatusNotFound)
}

// CreateDestination creates a new destination
func (h *AdminHandler) CreateDestination(w http.ResponseWriter, r *http.Request) {
	var dest models.Destination

	if err := json.NewDecoder(r.Body).Decode(&dest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate ID if not provided
	if dest.ID == "" {
		dest.ID = uuid.New().String()
	}

	// Validate
	if err := dest.Validate(); err != nil {
		h.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get current config
	currentConfig := h.configLoader.GetConfig()

	// Check for duplicate ID
	for _, existing := range currentConfig.Destinations {
		if existing.ID == dest.ID {
			h.writeError(w, "Destination ID already exists", http.StatusConflict)
			return
		}
	}

	// Add new destination
	newConfig := &config.Config{
		Destinations: append(currentConfig.Destinations, dest),
	}

	// Save config
	if err := h.configLoader.SaveConfig(newConfig); err != nil {
		h.logger.Error("failed to save config", "error", err)
		h.writeError(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	h.logger.Info("destination created", "id", dest.ID, "name", dest.Name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dest)
}

// UpdateDestination updates an existing destination
func (h *AdminHandler) UpdateDestination(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var dest models.Destination
	if err := json.NewDecoder(r.Body).Decode(&dest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure ID matches
	dest.ID = id

	// Validate
	if err := dest.Validate(); err != nil {
		h.writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get current config
	currentConfig := h.configLoader.GetConfig()

	// Find and update destination
	found := false
	newDestinations := make([]models.Destination, 0, len(currentConfig.Destinations))
	for _, existing := range currentConfig.Destinations {
		if existing.ID == id {
			newDestinations = append(newDestinations, dest)
			found = true
		} else {
			newDestinations = append(newDestinations, existing)
		}
	}

	if !found {
		http.Error(w, "Destination not found", http.StatusNotFound)
		return
	}

	// Save config
	newConfig := &config.Config{Destinations: newDestinations}
	if err := h.configLoader.SaveConfig(newConfig); err != nil {
		h.logger.Error("failed to save config", "error", err)
		h.writeError(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	h.logger.Info("destination updated", "id", dest.ID, "name", dest.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dest)
}

// DeleteDestination deletes a destination
func (h *AdminHandler) DeleteDestination(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get current config
	currentConfig := h.configLoader.GetConfig()

	// Find and remove destination
	found := false
	newDestinations := make([]models.Destination, 0, len(currentConfig.Destinations))
	for _, existing := range currentConfig.Destinations {
		if existing.ID == id {
			found = true
		} else {
			newDestinations = append(newDestinations, existing)
		}
	}

	if !found {
		http.Error(w, "Destination not found", http.StatusNotFound)
		return
	}

	// Save config
	newConfig := &config.Config{Destinations: newDestinations}
	if err := h.configLoader.SaveConfig(newConfig); err != nil {
		h.logger.Error("failed to save config", "error", err)
		h.writeError(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	h.logger.Info("destination deleted", "id", id)

	w.WriteHeader(http.StatusNoContent)
}

// ToggleDestination enables or disables a destination
func (h *AdminHandler) ToggleDestination(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get current config
	currentConfig := h.configLoader.GetConfig()

	// Find and toggle destination
	found := false
	newDestinations := make([]models.Destination, 0, len(currentConfig.Destinations))
	for _, existing := range currentConfig.Destinations {
		if existing.ID == id {
			existing.Enabled = !existing.Enabled
			found = true
		}
		newDestinations = append(newDestinations, existing)
	}

	if !found {
		http.Error(w, "Destination not found", http.StatusNotFound)
		return
	}

	// Save config
	newConfig := &config.Config{Destinations: newDestinations}
	if err := h.configLoader.SaveConfig(newConfig); err != nil {
		h.logger.Error("failed to save config", "error", err)
		h.writeError(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	// Find updated destination to return
	for _, dest := range newDestinations {
		if dest.ID == id {
			h.logger.Info("destination toggled", "id", id, "enabled", dest.Enabled)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(dest)
			return
		}
	}
}

// writeError writes an error response
func (h *AdminHandler) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
