package linear

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestCreateIssue_WithTeamName(t *testing.T) {
	callCount := 0
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query string `json:"query"`
		}
		json.Unmarshal(body, &req)

		if callCount == 1 {
			// First call: list teams to resolve team name
			mockResponse := `{
				"data": {
					"teams": {
						"nodes": [
							{
								"id": "team-123",
								"name": "Engineering",
								"key": "ENG"
							}
						]
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		} else {
			// Second call: create issue with resolved team ID
			mockResponse := `{
				"data": {
					"issueCreate": {
						"success": true,
						"issue": {
							"id": "issue-123",
							"identifier": "ENG-456",
							"title": "Test Issue",
							"description": "Test description",
							"state": {
								"id": "state-123",
								"name": "Backlog"
							},
							"createdAt": "2024-01-01T00:00:00Z",
							"updatedAt": "2024-01-01T00:00:00Z",
							"url": "https://linear.app/test/issue/ENG-456"
						}
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		}
	})
	defer server.Close()

	resolver := NewResolver(client)
	client.resolver = resolver

	// Test: CreateIssue with team name should work
	issue, err := client.CreateIssue("Test Issue", "Test description", "Engineering")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if issue.Identifier != "ENG-456" {
		t.Errorf("Expected identifier ENG-456, got %s", issue.Identifier)
	}

	if issue.Title != "Test Issue" {
		t.Errorf("Expected title 'Test Issue', got %s", issue.Title)
	}
}

func TestCreateIssue_WithTeamKey(t *testing.T) {
	callCount := 0
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// First call: list teams to resolve team key
			mockResponse := `{
				"data": {
					"teams": {
						"nodes": [
							{
								"id": "team-123",
								"name": "Engineering",
								"key": "ENG"
							}
						]
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		} else {
			// Second call: create issue
			mockResponse := `{
				"data": {
					"issueCreate": {
						"success": true,
						"issue": {
							"id": "issue-123",
							"identifier": "ENG-456",
							"title": "Test Issue",
							"description": "Test description",
							"state": {
								"id": "state-123",
								"name": "Backlog"
							},
							"createdAt": "2024-01-01T00:00:00Z",
							"updatedAt": "2024-01-01T00:00:00Z",
							"url": "https://linear.app/test/issue/ENG-456"
						}
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		}
	})
	defer server.Close()

	resolver := NewResolver(client)
	client.resolver = resolver

	// Test: CreateIssue with team key should work
	issue, err := client.CreateIssue("Test Issue", "Test description", "ENG")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if issue.Identifier != "ENG-456" {
		t.Errorf("Expected identifier ENG-456, got %s", issue.Identifier)
	}
}

func TestCreateIssue_WithUUID(t *testing.T) {
	// Test that UUIDs still work (backward compatibility)
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{
			"data": {
				"issueCreate": {
					"success": true,
					"issue": {
						"id": "issue-123",
						"identifier": "ENG-456",
						"title": "Test Issue",
						"description": "Test description",
						"state": {
							"id": "state-123",
							"name": "Backlog"
						},
						"createdAt": "2024-01-01T00:00:00Z",
						"updatedAt": "2024-01-01T00:00:00Z",
						"url": "https://linear.app/test/issue/ENG-456"
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

	// Should work with UUID format (no resolution needed)
	// Use proper UUID format: 8-4-4-4-12
	issue, err := client.CreateIssue("Test Issue", "Test description", "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if issue.Identifier != "ENG-456" {
		t.Errorf("Expected identifier ENG-456, got %s", issue.Identifier)
	}
}

func TestCreateIssue_WithInvalidTeam(t *testing.T) {
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Return empty teams list
		mockResponse := `{
			"data": {
				"teams": {
					"nodes": []
				}
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	})
	defer server.Close()

	resolver := NewResolver(client)
	client.resolver = resolver

	_, err := client.CreateIssue("Test Issue", "Test description", "NonExistentTeam")
	if err == nil {
		t.Fatal("Expected error for non-existent team, got nil")
	}

	if !IsNotFoundError(err) {
		t.Errorf("Expected NotFoundError, got: %T", err)
	}
}
