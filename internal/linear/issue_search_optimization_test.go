package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestListAssignedIssuesWithProject(t *testing.T) {
	// Test that ListAssignedIssues includes project information
	mockResponseBody := `{
		"data": {
			"issues": {
				"nodes": [
					{
						"id": "issue-1",
						"identifier": "PROJ-1",
						"title": "First Issue",
						"state": {
							"name": "Todo"
						},
						"project": {
							"id": "project-123",
							"name": "Test Project"
						}
					},
					{
						"id": "issue-2",
						"identifier": "TASK-2",
						"title": "Second Issue",
						"state": {
							"name": "In Progress"
						},
						"project": null
					},
					{
						"id": "issue-3",
						"identifier": "PROJ-3",
						"title": "Third Issue",
						"state": {
							"name": "Done"
						},
						"project": {
							"id": "project-456",
							"name": "Another Project"
						}
					}
				]
			}
		}
	}`
	
	// Create a client with mocked HTTP transport
	client := NewClient("test-token")
	client.base.httpClient = &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			},
		},
	}
	
	// Get assigned issues
	issues, err := client.ListAssignedIssues("user-123")
	
	// Verify no error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	// Verify we got 3 issues
	if len(issues) != 3 {
		t.Errorf("Expected 3 issues, got %d", len(issues))
	}
	
	// Verify first issue has project
	if issues[0].Project.ID == "" {
		t.Error("Expected first issue to have project ID, got empty")
	} else {
		if issues[0].Project.ID != "project-123" {
			t.Errorf("Expected project ID 'project-123', got '%s'", issues[0].Project.ID)
		}
		if issues[0].Project.Name != "Test Project" {
			t.Errorf("Expected project name 'Test Project', got '%s'", issues[0].Project.Name)
		}
	}
	
	// Verify second issue has no project
	if issues[1].Project != nil && issues[1].Project.ID != "" {
		t.Errorf("Expected second issue to have no project, got ID: %s, Name: %s",
			issues[1].Project.ID, issues[1].Project.Name)
	}
	
	// Verify third issue has project
	if issues[2].Project.ID == "" {
		t.Error("Expected third issue to have project ID, got empty")
	} else {
		if issues[2].Project.ID != "project-456" {
			t.Errorf("Expected project ID 'project-456', got '%s'", issues[2].Project.ID)
		}
		if issues[2].Project.Name != "Another Project" {
			t.Errorf("Expected project name 'Another Project', got '%s'", issues[2].Project.Name)
		}
	}
}

func TestIssueProjectStructure(t *testing.T) {
	// Test that Issue struct properly supports basic project info
	issue := Issue{
		ID:         "issue-123",
		Identifier: "PROJ-123",
		Title:      "Test Issue",
		Project:    &Project{},
	}

	// Set project info
	issue.Project.ID = "project-789"
	issue.Project.Name = "My Project"
	
	// Verify project fields are accessible
	if issue.Project.ID != "project-789" {
		t.Errorf("Expected project ID 'project-789', got '%s'", issue.Project.ID)
	}
	
	if issue.Project.Name != "My Project" {
		t.Errorf("Expected project name 'My Project', got '%s'", issue.Project.Name)
	}
}