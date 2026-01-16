package models

import (
	"errors"
	"net/url"
	"regexp"
)

// Destination represents a webhook destination configuration
type Destination struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	TimeoutSecs  int               `json:"timeout_seconds"`
	Filters      []Filter          `json:"filters"`
	Enabled      bool              `json:"enabled"`
	RetryPolicy  RetryPolicy       `json:"retry_policy"`
}

// Filter represents a metadata filter for webhooks
type Filter struct {
	Field    string   `json:"field"`               // Metadata key to filter on
	Operator string   `json:"operator"`            // eq, ne, in, contains, regex
	Value    string   `json:"value,omitempty"`     // Single value for eq, ne, contains, regex
	Values   []string `json:"values,omitempty"`    // Multiple values for in operator
}

// RetryPolicy defines retry behavior for failed deliveries
type RetryPolicy struct {
	MaxAttempts     int   `json:"max_attempts"`
	BackoffSchedule []int `json:"backoff_schedule"` // Backoff delays in seconds
}

// Validate validates the destination configuration
func (d *Destination) Validate() error {
	if d.ID == "" {
		return errors.New("id is required")
	}
	if d.Name == "" {
		return errors.New("name is required")
	}
	if d.URL == "" {
		return errors.New("url is required")
	}

	// Validate URL
	if _, err := url.Parse(d.URL); err != nil {
		return errors.New("invalid url")
	}

	// Validate method
	if d.Method == "" {
		d.Method = "POST"
	}
	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true}
	if !validMethods[d.Method] {
		return errors.New("invalid method, must be GET, POST, PUT, or PATCH")
	}

	// Validate timeout
	if d.TimeoutSecs <= 0 {
		d.TimeoutSecs = 30
	}

	// Validate filters
	for i, filter := range d.Filters {
		if err := filter.Validate(); err != nil {
			return err
		}
		d.Filters[i] = filter
	}

	// Validate retry policy
	if d.RetryPolicy.MaxAttempts <= 0 {
		d.RetryPolicy.MaxAttempts = 10
	}
	if len(d.RetryPolicy.BackoffSchedule) == 0 {
		d.RetryPolicy.BackoffSchedule = []int{60, 300, 900, 3600, 21600, 86400}
	}

	return nil
}

// Validate validates the filter configuration
func (f *Filter) Validate() error {
	if f.Field == "" {
		return errors.New("filter field is required")
	}

	validOperators := map[string]bool{
		"eq":       true,
		"ne":       true,
		"in":       true,
		"contains": true,
		"regex":    true,
	}

	if !validOperators[f.Operator] {
		return errors.New("invalid operator, must be eq, ne, in, contains, or regex")
	}

	// Validate operator-specific requirements
	switch f.Operator {
	case "in":
		if len(f.Values) == 0 {
			return errors.New("'in' operator requires values array")
		}
	case "eq", "ne", "contains":
		if f.Value == "" {
			return errors.New("operator requires value field")
		}
	case "regex":
		if f.Value == "" {
			return errors.New("regex operator requires value field")
		}
		// Validate regex pattern
		if _, err := regexp.Compile(f.Value); err != nil {
			return errors.New("invalid regex pattern: " + err.Error())
		}
	}

	return nil
}

// Matches checks if a metadata value matches the filter
func (f *Filter) Matches(value string) bool {
	switch f.Operator {
	case "eq":
		return value == f.Value
	case "ne":
		return value != f.Value
	case "in":
		for _, v := range f.Values {
			if value == v {
				return true
			}
		}
		return false
	case "contains":
		return regexp.MustCompile(regexp.QuoteMeta(f.Value)).MatchString(value)
	case "regex":
		re, err := regexp.Compile(f.Value)
		if err != nil {
			return false
		}
		return re.MatchString(value)
	}
	return false
}
