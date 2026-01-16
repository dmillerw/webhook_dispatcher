package testutil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockServer represents a mock HTTP server for testing
type MockServer struct {
	*httptest.Server
	Received chan *http.Request
}

// NewMockServer creates a new mock server that records all requests
func NewMockServer(t *testing.T, received chan *http.Request) *MockServer {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clone request body for testing
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Send to channel
		received <- r

		// Return success
		w.WriteHeader(http.StatusOK)
	})

	return &MockServer{
		Server:   httptest.NewServer(handler),
		Received: received,
	}
}

// NewMockServerWithHandler creates a mock server with a custom handler
func NewMockServerWithHandler(t *testing.T, handler http.HandlerFunc) *MockServer {
	return &MockServer{
		Server: httptest.NewServer(handler),
	}
}

// NewMockServerWithStatus creates a mock server that always returns the given status
func NewMockServerWithStatus(t *testing.T, statusCode int) *MockServer {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	})

	return &MockServer{
		Server: httptest.NewServer(handler),
	}
}
