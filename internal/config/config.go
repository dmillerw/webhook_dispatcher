package config

import (
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/user/webhook-dispatch/internal/models"
)

// Config represents the application configuration
type Config struct {
	Destinations []models.Destination `json:"destinations"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Destinations) == 0 {
		return errors.New("at least one destination is required")
	}

	// Validate each destination
	ids := make(map[string]bool)
	for i := range c.Destinations {
		if err := c.Destinations[i].Validate(); err != nil {
			return err
		}

		// Check for duplicate IDs
		if ids[c.Destinations[i].ID] {
			return errors.New("duplicate destination ID: " + c.Destinations[i].ID)
		}
		ids[c.Destinations[i].ID] = true

		// Expand environment variables in headers
		for key, value := range c.Destinations[i].Headers {
			c.Destinations[i].Headers[key] = expandEnv(value)
		}
	}

	return nil
}

// expandEnv expands environment variables in the format ${VAR_NAME}
func expandEnv(s string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name
		varName := strings.TrimPrefix(strings.TrimSuffix(match, "}"), "${")
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match // Return original if env var not found
	})
}
