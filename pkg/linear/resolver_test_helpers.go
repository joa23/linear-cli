package linear

import (
	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/comments"
	"github.com/joa23/linear-cli/pkg/linear/attachments"
	"github.com/joa23/linear-cli/pkg/linear/issues"
	"github.com/joa23/linear-cli/pkg/linear/projects"
	"github.com/joa23/linear-cli/pkg/linear/teams"
	"github.com/joa23/linear-cli/pkg/linear/users"
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
	baseClient := core.NewTestBaseClient("test-token", server.URL, server.Client())

	// Create a full client with all sub-clients
	client := &Client{
		base:          baseClient,
		Issues:        issues.NewClient(baseClient),
		Comments:      comments.NewClient(baseClient),
		Teams:         teams.NewClient(baseClient),
		Projects:      projects.NewClient(baseClient),
		Notifications: users.NewNotificationClient(baseClient),
		Attachments:   attachments.NewClient(baseClient),
	}

	return server, client
}
