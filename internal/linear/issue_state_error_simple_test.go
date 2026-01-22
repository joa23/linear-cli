package linear

import (
	"github.com/joa23/linear-cli/internal/token"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateIssueStateInvalidStateID(t *testing.T) {
	// Create test server that returns an error for invalid state ID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		// Return error response similar to what Linear returns
		response := `{
			"data": {
				"issueUpdate": null
			},
			"errors": [{
				"message": "Entity not found in validateAccess: stateId",
				"path": ["issueUpdate"],
				"extensions": {
					"userPresentableMessage": "The specified state ID does not exist or you don't have access to it"
				}
			}]
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	// Create base client and issue client
	base := &BaseClient{
		tokenProvider: token.NewStaticProvider("test-token"),
		httpClient: http.DefaultClient,
		baseURL: server.URL,
	}
	issueClient := NewIssueClient(base)

	// Test updating with invalid state ID
	err := issueClient.UpdateIssueState("test-issue-id", "c3a1e220-24fc-45ba-a5da-27d3bb0dd7f5")

	// Verify we get an error
	if err == nil {
		t.Fatal("Expected error for invalid state ID, got nil")
	}

	// Check that error message contains helpful information
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected error to contain 'does not exist', got: %v", err)
	}
	
	// Check for helpful guidance
	if !strings.Contains(err.Error(), "linear_get_workflow_states") {
		t.Errorf("Expected error to suggest using linear_get_workflow_states, got: %v", err)
	}
	
	// Check that the invalid state ID is included
	if !strings.Contains(err.Error(), "c3a1e220-24fc-45ba-a5da-27d3bb0dd7f5") {
		t.Errorf("Expected error to include the invalid state ID, got: %v", err)
	}
}