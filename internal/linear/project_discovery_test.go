package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestListAllProjects(t *testing.T) {
	// Test listing all projects in workspace
	mockResponseBody := `{
		"data": {
			"projects": {
				"nodes": [
					{
						"id": "project-1",
						"name": "Project Alpha",
						"description": "First project"
					},
					{
						"id": "project-2",
						"name": "Project Beta",
						"description": "Second project"
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
	
	// List all projects
	projects, err := client.ListAllProjects()
	
	// Verify no error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	// Verify we got 2 projects
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}
	
	// Verify first project
	if projects[0].ID != "project-1" {
		t.Errorf("Expected project ID 'project-1', got '%s'", projects[0].ID)
	}
	if projects[0].Name != "Project Alpha" {
		t.Errorf("Expected project name 'Project Alpha', got '%s'", projects[0].Name)
	}
	// Note: State field removed from projects query due to Linear API limitation
	// Use GetProject for individual project state information
}

func TestListUserProjects(t *testing.T) {
	// Test listing projects where user has assigned issues
	mockResponseBody := `{
		"data": {
			"projects": {
				"nodes": [
					{
						"id": "project-1",
						"name": "User Project 1",
						"description": "Project with user issues",
						"issues": {
							"nodes": [
								{
									"id": "issue-1",
									"assignee": {
										"id": "user-123"
									}
								},
								{
									"id": "issue-2",
									"assignee": {
										"id": "user-123"
									}
								}
							]
						}
					},
					{
						"id": "project-2",
						"name": "User Project 2",
						"description": "Another project with user issues",
						"issues": {
							"nodes": [
								{
									"id": "issue-3",
									"assignee": {
										"id": "user-123"
									}
								}
							]
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
	
	// List user projects
	projects, err := client.ListUserProjects("user-123")
	
	// Verify no error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	// Verify we got 2 projects
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}
	
	// Verify first project
	if projects[0].ID != "project-1" {
		t.Errorf("Expected project ID 'project-1', got '%s'", projects[0].ID)
	}
	if projects[0].Name != "User Project 1" {
		t.Errorf("Expected project name 'User Project 1', got '%s'", projects[0].Name)
	}
	
	// Verify second project
	if projects[1].ID != "project-2" {
		t.Errorf("Expected project ID 'project-2', got '%s'", projects[1].ID)
	}
	if projects[1].Name != "User Project 2" {
		t.Errorf("Expected project name 'User Project 2', got '%s'", projects[1].Name)
	}
}

func TestListUserProjects_EmptyUserID(t *testing.T) {
	// Test validation for empty user ID
	client := NewClient("test-token")
	
	// Try with empty user ID
	_, err := client.ListUserProjects("")
	
	// Should get validation error
	if err == nil {
		t.Error("Expected validation error for empty userID")
	}
	
	// Verify it's a validation error
	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("Expected ValidationError, got %T", err)
	}
}

func TestListUserProjects_OnlyReturnsProjectsWithUserIssues(t *testing.T) {
	// Test that only projects with user's assigned issues are returned
	mockResponseBody := `{
		"data": {
			"projects": {
				"nodes": [
					{
						"id": "project-1",
						"name": "Has User Issues",
						"description": "Project with user issues",
						"issues": {
							"nodes": [
								{
									"id": "issue-1",
									"assignee": {
										"id": "user-123"
									}
								}
							]
						}
					},
					{
						"id": "project-2",
						"name": "No User Issues",
						"description": "Project without user issues",
						"issues": {
							"nodes": []
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
	
	// List user projects
	projects, err := client.ListUserProjects("user-123")
	
	// Verify no error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	// Should only get 1 project (the one with user issues)
	if len(projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(projects))
	}
	
	// Verify it's the correct project
	if projects[0].ID != "project-1" {
		t.Errorf("Expected project ID 'project-1', got '%s'", projects[0].ID)
	}
}