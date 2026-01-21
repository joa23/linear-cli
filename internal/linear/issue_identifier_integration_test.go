package linear

import (
	"net/http"
	"testing"
)

func TestGetIssue_WithIdentifier(t *testing.T) {
	// Setup mock server that handles issue(id:) query with identifier
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Linear's issue(id:) query accepts identifiers directly
		mockResponse := `{
			"data": {
				"issue": {
					"id": "issue-uuid-123",
					"identifier": "CEN-123",
					"title": "Test Issue",
					"description": "Test description",
					"state": {
						"id": "state-123",
						"name": "In Progress"
					},
					"createdAt": "2024-01-01T00:00:00Z",
					"updatedAt": "2024-01-01T00:00:00Z",
					"url": "https://linear.app/test/issue/CEN-123",
					"children": { "nodes": [] },
					"attachments": { "nodes": [] }
				}
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	})
	defer server.Close()

	// Create resolver and integrate with client
	resolver := NewResolver(client)
	client.resolver = resolver

	// Test: GetIssue with identifier should work
	issue, err := client.GetIssue("CEN-123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if issue.Identifier != "CEN-123" {
		t.Errorf("Expected identifier CEN-123, got %s", issue.Identifier)
	}

	if issue.Title != "Test Issue" {
		t.Errorf("Expected title 'Test Issue', got %s", issue.Title)
	}
}

func TestGetIssue_WithInvalidIdentifier(t *testing.T) {
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Return empty search results
		mockResponse := `{
			"data": {
				"issues": {
					"nodes": [],
					"pageInfo": {
						"hasNextPage": false,
						"endCursor": null
					}
				}
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	})
	defer server.Close()

	resolver := NewResolver(client)
	client.resolver = resolver

	_, err := client.GetIssue("CEN-999")
	if err == nil {
		t.Fatal("Expected error for non-existent issue, got nil")
	}

	if !IsNotFoundError(err) {
		t.Errorf("Expected NotFoundError, got: %T", err)
	}
}

func TestGetIssue_WithUUID(t *testing.T) {
	// Test that UUIDs still work (backward compatibility)
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{
			"data": {
				"issue": {
					"id": "issue-uuid-123",
					"identifier": "CEN-123",
					"title": "Test Issue"
				}
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	})
	defer server.Close()

	resolver := NewResolver(client)
	client.resolver = resolver

	// Should work with UUID format (no resolution needed)
	issue, err := client.GetIssue("issue-uuid-123")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if issue.Identifier != "CEN-123" {
		t.Errorf("Expected identifier CEN-123, got %s", issue.Identifier)
	}
}
