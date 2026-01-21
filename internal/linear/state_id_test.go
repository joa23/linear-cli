package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestGetIssueIncludesStateID(t *testing.T) {
	// Test that GetIssue returns state ID
	mockResponseBody := `{
		"data": {
			"issue": {
				"id": "issue-123",
				"identifier": "PROJ-123",
				"title": "Test issue",
				"description": "Test description",
				"state": {
					"id": "state-todo-123",
					"name": "Todo"
				},
				"project": {
					"name": "Test Project",
					"description": "Project description"
				},
				"creator": {
					"id": "user-1",
					"name": "Test User"
				},
				"children": {
					"nodes": []
				},
				"comments": {
					"nodes": []
				}
			}
		}
	}`
	
	client := NewClient("test-token")
	client.base.httpClient = &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			},
		},
	}
	
	issue, err := client.GetIssue("issue-123")
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	
	// Verify state ID is populated
	if issue.State.ID == "" {
		t.Error("Expected state ID to be populated, but it was empty")
	}
	
	if issue.State.ID != "state-todo-123" {
		t.Errorf("Expected state ID 'state-todo-123', got '%s'", issue.State.ID)
	}
	
	if issue.State.Name != "Todo" {
		t.Errorf("Expected state name 'Todo', got '%s'", issue.State.Name)
	}
}

func TestListAssignedIssuesIncludesStateID(t *testing.T) {
	// Test that ListAssignedIssues returns state IDs
	mockResponseBody := `{
		"data": {
			"issues": {
				"nodes": [
					{
						"id": "issue-1",
						"identifier": "PROJ-1",
						"title": "First issue",
						"state": {
							"id": "state-inprogress-456",
							"name": "In Progress"
						},
						"project": null
					},
					{
						"id": "issue-2", 
						"identifier": "PROJ-2",
						"title": "Second issue",
						"state": {
							"id": "state-done-789",
							"name": "Done"
						},
						"project": {
							"id": "project-1",
							"name": "Test Project"
						}
					}
				]
			}
		}
	}`
	
	client := NewClient("test-token")
	client.base.httpClient = &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			},
		},
	}
	
	issues, err := client.ListAssignedIssues("user-123")
	if err != nil {
		t.Fatalf("ListAssignedIssues failed: %v", err)
	}
	
	if len(issues) != 2 {
		t.Fatalf("Expected 2 issues, got %d", len(issues))
	}
	
	// Check first issue
	if issues[0].State.ID == "" {
		t.Error("Expected first issue state ID to be populated, but it was empty")
	}
	if issues[0].State.ID != "state-inprogress-456" {
		t.Errorf("Expected first issue state ID 'state-inprogress-456', got '%s'", issues[0].State.ID)
	}
	
	// Check second issue
	if issues[1].State.ID != "state-done-789" {
		t.Errorf("Expected second issue state ID 'state-done-789', got '%s'", issues[1].State.ID)
	}
}

func TestUpdateIssueStateUsesStateID(t *testing.T) {
	// Test that UpdateIssueState can use the state ID we now provide
	mockResponseBody := `{
		"data": {
			"issueUpdate": {
				"success": true,
				"issue": {
					"id": "issue-123",
					"state": {
						"id": "state-inreview-999",
						"name": "In Review"
					}
				}
			}
		}
	}`
	
	client := NewClient("test-token")
	client.base.httpClient = &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			},
		},
	}
	
	// Now we can use actual state IDs instead of guessing
	err := client.UpdateIssueState("issue-123", "state-inreview-999")
	if err != nil {
		t.Fatalf("UpdateIssueState failed: %v", err)
	}
}