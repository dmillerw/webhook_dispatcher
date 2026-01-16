package testutil

import (
	"net/http"
	"testing"
	"time"
)

// AssertReceived waits for a request on the channel or fails the test
func AssertReceived(t *testing.T, ch chan *http.Request, timeout time.Duration) *http.Request {
	t.Helper()

	select {
	case req := <-ch:
		return req
	case <-time.After(timeout):
		t.Fatal("Timeout waiting for request")
		return nil
	}
}

// AssertNotReceived ensures no request is received within the timeout
func AssertNotReceived(t *testing.T, ch chan *http.Request, timeout time.Duration) {
	t.Helper()

	select {
	case <-ch:
		t.Fatal("Unexpectedly received request")
	case <-time.After(timeout):
		// Success - no request received
	}
}
