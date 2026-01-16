package metadata

import (
	"github.com/tidwall/gjson"
)

// ExtractionRule defines how to extract a metadata field from JSON
type ExtractionRule struct {
	Name     string `json:"name"`      // Metadata key name
	JSONPath string `json:"json_path"` // gjson path expression
}

// Extractor extracts metadata from webhook JSON bodies
type Extractor struct {
	rules []ExtractionRule
}

// NewExtractor creates a new metadata extractor
func NewExtractor(rules []ExtractionRule) *Extractor {
	return &Extractor{
		rules: rules,
	}
}

// Extract extracts metadata from a JSON body
func (e *Extractor) Extract(jsonBody []byte) map[string]string {
	metadata := make(map[string]string)

	for _, rule := range e.rules {
		result := gjson.GetBytes(jsonBody, rule.JSONPath)
		if result.Exists() {
			metadata[rule.Name] = result.String()
		}
	}

	return metadata
}
