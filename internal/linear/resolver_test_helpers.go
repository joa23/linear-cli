package linear

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupMockServer creates a test HTTP server and a Linear client configured to use it
// This is used for testing resolver functions without hitting the real Linear API
func setupMockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()

	server := httptest.NewServer(handler)

	// Create a base client pointing to the test server
	baseClient := NewTestBaseClient("test-token", server.URL, server.Client())

	// Create a full client with all sub-clients
	client := &Client{
		base:          baseClient,
		Issues:        NewIssueClient(baseClient),
		Comments:      NewCommentClient(baseClient),
		Teams:         NewTeamClient(baseClient),
		Projects:      NewProjectClient(baseClient),
		Notifications: NewNotificationClient(baseClient),
		Attachments:   NewAttachmentClient(baseClient),
	}

	return server, client
}
