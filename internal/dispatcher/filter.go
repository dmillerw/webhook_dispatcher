package dispatcher

import (
	"github.com/user/webhook-dispatch/internal/models"
)

// EvaluateFilters checks if a webhook matches all filters for a destination
func EvaluateFilters(metadata map[string]string, filters []models.Filter) bool {
	// No filters means match all
	if len(filters) == 0 {
		return true
	}

	// All filters must match (AND logic)
	for _, filter := range filters {
		value, exists := metadata[filter.Field]

		// If field doesn't exist in metadata, filter fails
		if !exists {
			return false
		}

		// Check if value matches filter
		if !filter.Matches(value) {
			return false
		}
	}

	return true
}
