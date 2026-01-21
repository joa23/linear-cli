package linear

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestUpdateIssue_WithAssigneeName(t *testing.T) {
	callCount := 0
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query string `json:"query"`
		}
		json.Unmarshal(body, &req)

		if callCount == 1 {
			// First call: list users to resolve assignee name
			mockResponse := `{
				"data": {
					"users": {
						"nodes": [
							{
								"id": "user-123",
								"name": "John Doe",
								"email": "john@company.com",
								"displayName": "John Doe"
							}
						]
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		} else {
			// Second call: update issue with resolved assignee ID
			mockResponse := `{
				"data": {
					"issueUpdate": {
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
							"assignee": {
								"id": "user-123",
								"name": "John Doe",
								"email": "john@company.com"
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

	assigneeName := "John Doe"
	issue, err := client.UpdateIssue("issue-123", UpdateIssueInput{
		AssigneeID: &assigneeName,
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if issue.Assignee == nil {
		t.Fatal("Expected assignee to be set")
	}

	if issue.Assignee.Name != "John Doe" {
		t.Errorf("Expected assignee name 'John Doe', got %s", issue.Assignee.Name)
	}
}

func TestUpdateIssue_WithAssigneeEmail(t *testing.T) {
	callCount := 0
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// First call: get user by email
			mockResponse := `{
				"data": {
					"users": {
						"nodes": [
							{
								"id": "user-123",
								"name": "John Doe",
								"email": "john@company.com",
								"displayName": "John"
							}
						]
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		} else {
			// Second call: update issue
			mockResponse := `{
				"data": {
					"issueUpdate": {
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
							"assignee": {
								"id": "user-123",
								"name": "John Doe",
								"email": "john@company.com"
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

	assigneeEmail := "john@company.com"
	issue, err := client.UpdateIssue("issue-123", UpdateIssueInput{
		AssigneeID: &assigneeEmail,
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if issue.Assignee == nil {
		t.Fatal("Expected assignee to be set")
	}

	if issue.Assignee.Email != "john@company.com" {
		t.Errorf("Expected assignee email 'john@company.com', got %s", issue.Assignee.Email)
	}
}

func TestUpdateIssue_WithParentIdentifier(t *testing.T) {
	callCount := 0
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// First call: resolve parent issue identifier using issue(id:) query
			mockResponse := `{
				"data": {
					"issue": {
						"id": "parent-issue-uuid",
						"identifier": "ENG-123",
						"title": "Parent Issue",
						"description": "",
						"state": {
							"id": "state-1",
							"name": "Backlog"
						},
						"createdAt": "2024-01-01T00:00:00Z",
						"updatedAt": "2024-01-01T00:00:00Z",
						"url": "https://linear.app/test/issue/ENG-123",
						"children": { "nodes": [] },
						"attachments": { "nodes": [] }
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		} else {
			// Second call: update issue with resolved parent ID
			mockResponse := `{
				"data": {
					"issueUpdate": {
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
							"parent": {
								"id": "parent-issue-uuid",
								"identifier": "ENG-123",
								"title": "Parent Issue"
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

	parentIdentifier := "ENG-123"
	issue, err := client.UpdateIssue("issue-123", UpdateIssueInput{
		ParentID: &parentIdentifier,
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if issue.Parent == nil {
		t.Fatal("Expected parent to be set")
	}

	if issue.Parent.Identifier != "ENG-123" {
		t.Errorf("Expected parent identifier 'ENG-123', got %s", issue.Parent.Identifier)
	}
}

func TestUpdateIssue_WithTeamName(t *testing.T) {
	callCount := 0
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// First call: list teams to resolve team name
			mockResponse := `{
				"data": {
					"teams": {
						"nodes": [
							{
								"id": "team-456",
								"name": "Product",
								"key": "PROD"
							}
						]
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		} else {
			// Second call: update issue with resolved team ID
			mockResponse := `{
				"data": {
					"issueUpdate": {
						"success": true,
						"issue": {
							"id": "issue-123",
							"identifier": "PROD-789",
							"title": "Test Issue",
							"description": "Test description",
							"state": {
								"id": "state-123",
								"name": "Backlog"
							},
							"createdAt": "2024-01-01T00:00:00Z",
							"updatedAt": "2024-01-01T00:00:00Z",
							"url": "https://linear.app/test/issue/PROD-789"
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

	teamName := "Product"
	issue, err := client.UpdateIssue("issue-123", UpdateIssueInput{
		TeamID: &teamName,
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if issue.Identifier != "PROD-789" {
		t.Errorf("Expected identifier to change to PROD-789, got %s", issue.Identifier)
	}
}

func TestUpdateIssue_NoResolutionNeeded(t *testing.T) {
	// Test that UUIDs work without resolution
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{
			"data": {
				"issueUpdate": {
					"success": true,
					"issue": {
						"id": "issue-123",
						"identifier": "ENG-456",
						"title": "Updated Title",
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

	title := "Updated Title"
	issue, err := client.UpdateIssue("issue-123", UpdateIssueInput{
		Title: &title,
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if issue.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %s", issue.Title)
	}
}
