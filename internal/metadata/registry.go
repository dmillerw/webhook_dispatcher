package metadata

import (
	"sync"
)

// Registry manages webhook path registrations and their metadata extraction rules
type Registry struct {
	mu      sync.RWMutex
	paths   map[string]*Extractor
}

// NewRegistry creates a new metadata registry
func NewRegistry() *Registry {
	return &Registry{
		paths: make(map[string]*Extractor),
	}
}

// Register registers a webhook path with metadata extraction rules
func (r *Registry) Register(path string, rules []ExtractionRule) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.paths[path] = NewExtractor(rules)
}

// Extract extracts metadata for a registered path
func (r *Registry) Extract(path string, jsonBody []byte) (map[string]string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	extractor, ok := r.paths[path]
	if !ok {
		return nil, false
	}

	return extractor.Extract(jsonBody), true
}

// IsRegistered checks if a path is registered
func (r *Registry) IsRegistered(path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.paths[path]
	return ok
}

// GetPaths returns all registered paths
func (r *Registry) GetPaths() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make([]string, 0, len(r.paths))
	for path := range r.paths {
		paths = append(paths, path)
	}
	return paths
}
