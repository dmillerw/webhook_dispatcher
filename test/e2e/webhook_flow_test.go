package e2e_test

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/user/webhook-dispatch/test/testutil"
)

// TestWebhookIngestionAndDispatch tests the end-to-end webhook flow
func TestWebhookIngestionAndDispatch(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Setup mock destination server
	received := make(chan *http.Request, 10)
	mockServer := testutil.NewMockServer(t, received)
	defer mockServer.Close()

	t.Logf("Mock server URL: %s", mockServer.URL)

	// TODO: Setup test app with mock destination
	// This would involve:
	// 1. Creating a test config with the mock server URL
	// 2. Starting the application in test mode
	// 3. Waiting for it to be ready

	// For now, this is a placeholder showing the test structure
	t.Skip("Test infrastructure setup needed - see full E2E test plan")

	// Send webhook payload
	payload := []byte(`{
		"action": "push",
		"ref": "refs/heads/main",
		"repository": {
			"full_name": "test/repo"
		}
	}`)

	resp, err := http.Post("http://localhost:8080/webhooks/github", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, resp.StatusCode)
	}

	// Verify destination received webhook
	req := testutil.AssertReceived(t, received, 5*time.Second)

	if req.Method != "POST" {
		t.Errorf("Expected POST, got %s", req.Method)
	}

	t.Log("Webhook flow test passed!")
}
