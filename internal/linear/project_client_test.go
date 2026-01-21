package linear

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestProjectIssuesUnmarshaling verifies that projects with nested issues are properly unmarshaled
func TestProjectIssuesUnmarshaling(t *testing.T) {
	tests := []struct {
		name           string
		graphqlResponse string
		wantIssueCount int
		wantFirstIssue *ProjectIssue
	}{
		{
			name: "project with nested issues structure",
			graphqlResponse: `{
				"data": {
					"project": {
						"id": "proj-123",
						"name": "Test Project",
						"description": "Test Description",
						"state": "started",
						"createdAt": "2024-01-01T00:00:00Z",
						"updatedAt": "2024-01-01T00:00:00Z",
						"issues": {
							"nodes": [
								{
									"id": "issue-1",
									"identifier": "TEST-1",
									"title": "First Issue",
									"state": {
										"id": "state-1",
										"name": "In Progress"
									},
									"assignee": {
										"id": "user-1",
										"name": "Test User",
										"email": "test@example.com"
									}
								},
								{
									"id": "issue-2",
									"identifier": "TEST-2",
									"title": "Second Issue",
									"state": {
										"id": "state-2",
										"name": "Done"
									},
									"assignee": null
								}
							]
						}
					}
				}
			}`,
			wantIssueCount: 2,
			wantFirstIssue: &ProjectIssue{
				ID:         "issue-1",
				Identifier: "TEST-1",
				Title:      "First Issue",
				State: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					ID:   "state-1",
					Name: "In Progress",
				},
				Assignee: &User{
					ID:    "user-1",
					Name:  "Test User",
					Email: "test@example.com",
				},
			},
		},
		{
			name: "project without issues",
			graphqlResponse: `{
				"data": {
					"project": {
						"id": "proj-456",
						"name": "Empty Project",
						"description": "No issues",
						"state": "planned",
						"createdAt": "2024-01-01T00:00:00Z",
						"updatedAt": "2024-01-01T00:00:00Z",
						"issues": {
							"nodes": []
						}
					}
				}
			}`,
			wantIssueCount: 0,
			wantFirstIssue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that returns our mock response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.graphqlResponse))
			}))
			defer server.Close()

			// Create base client with test server
			base := &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				apiToken:   "test-token",
			}
			pc := NewProjectClient(base)

			// Test GetProject
			project, err := pc.GetProject("test-id")
			if err != nil {
				t.Fatalf("GetProject failed: %v", err)
			}

			// Verify project basic fields
			if project.ID == "" {
				t.Error("Expected project ID to be set")
			}

			// Verify issues are properly unmarshaled
			issues := project.GetIssues()
			if len(issues) != tt.wantIssueCount {
				t.Errorf("Expected %d issues, got %d", tt.wantIssueCount, len(issues))
			}

			// Check first issue details if expected
			if tt.wantFirstIssue != nil && len(issues) > 0 {
				firstIssue := issues[0]
				if firstIssue.ID != tt.wantFirstIssue.ID {
					t.Errorf("Expected first issue ID %s, got %s", tt.wantFirstIssue.ID, firstIssue.ID)
				}
				if firstIssue.Identifier != tt.wantFirstIssue.Identifier {
					t.Errorf("Expected first issue identifier %s, got %s", tt.wantFirstIssue.Identifier, firstIssue.Identifier)
				}
				if firstIssue.Title != tt.wantFirstIssue.Title {
					t.Errorf("Expected first issue title %s, got %s", tt.wantFirstIssue.Title, firstIssue.Title)
				}
				if firstIssue.State.ID != tt.wantFirstIssue.State.ID {
					t.Errorf("Expected first issue state ID %s, got %s", tt.wantFirstIssue.State.ID, firstIssue.State.ID)
				}
				if firstIssue.State.Name != tt.wantFirstIssue.State.Name {
					t.Errorf("Expected first issue state name %s, got %s", tt.wantFirstIssue.State.Name, firstIssue.State.Name)
				}
				
				// Check assignee
				if tt.wantFirstIssue.Assignee != nil {
					if firstIssue.Assignee == nil {
						t.Error("Expected first issue to have assignee")
					} else {
						if firstIssue.Assignee.ID != tt.wantFirstIssue.Assignee.ID {
							t.Errorf("Expected assignee ID %s, got %s", tt.wantFirstIssue.Assignee.ID, firstIssue.Assignee.ID)
						}
					}
				}
			}
		})
	}
}

// TestListUserProjectsUnmarshaling tests that ListUserProjects properly handles nested issues
func TestListUserProjectsUnmarshaling(t *testing.T) {
	graphqlResponse := `{
		"data": {
			"projects": {
				"nodes": [
					{
						"id": "proj-1",
						"name": "Project 1",
						"description": "First project",
						"state": "started",
						"createdAt": "2024-01-01T00:00:00Z",
						"updatedAt": "2024-01-01T00:00:00Z",
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
						"id": "proj-2",
						"name": "Project 2",
						"description": "Second project",
						"state": "planned",
						"createdAt": "2024-01-02T00:00:00Z",
						"updatedAt": "2024-01-02T00:00:00Z",
						"issues": {
							"nodes": []
						}
					}
				]
			}
		}
	}`

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request body contains expected filter
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		
		variables, ok := reqBody["variables"].(map[string]interface{})
		if !ok {
			t.Error("Expected variables in request")
		}
		
		filter, ok := variables["filter"].(map[string]interface{})
		if !ok {
			t.Error("Expected filter in variables")
		}
		
		// Check that the filter is looking for the right user
		if issues, ok := filter["issues"].(map[string]interface{}); ok {
			if assignee, ok := issues["assignee"].(map[string]interface{}); ok {
				if id, ok := assignee["id"].(map[string]interface{}); ok {
					if eq, ok := id["eq"].(string); ok && eq != "user-123" {
						t.Errorf("Expected filter for user-123, got %s", eq)
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(graphqlResponse))
	}))
	defer server.Close()

	// Create base client
	base := &BaseClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
		apiToken:   "test-token",
	}
	pc := NewProjectClient(base)

	// Test ListUserProjects
	projects, err := pc.ListUserProjects("user-123", 10)
	if err != nil {
		t.Fatalf("ListUserProjects failed: %v", err)
	}

	// Verify we got 1 project (only projects with user issues are returned)
	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}

	// Check first project has issues
	firstProject := projects[0]
	if firstProject.Issues == nil {
		t.Error("Expected first project to have issues field")
	} else {
		issues := firstProject.GetIssues()
		if len(issues) != 1 {
			t.Errorf("Expected first project to have 1 issue, got %d", len(issues))
		}
		if len(issues) > 0 && issues[0].ID != "issue-1" {
			t.Errorf("Expected issue ID 'issue-1', got %s", issues[0].ID)
		}
	}
}

// TestCreateProjectUnmarshaling tests that CreateProject properly handles nested issues in response
func TestCreateProjectUnmarshaling(t *testing.T) {
	graphqlResponse := `{
		"data": {
			"projectCreate": {
				"success": true,
				"project": {
					"id": "new-proj",
					"name": "New Project",
					"description": "Created project",
					"state": "planned",
					"createdAt": "2024-01-01T00:00:00Z",
					"updatedAt": "2024-01-01T00:00:00Z",
					"issues": {
						"nodes": []
					}
				}
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request uses teamIds (plural array) not teamId (singular)
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		if !strings.Contains(bodyStr, `"teamIds"`) {
			t.Errorf("Expected request to contain teamIds (plural), got: %s", bodyStr)
		}
		if strings.Contains(bodyStr, `"teamId"`) && !strings.Contains(bodyStr, `"teamIds"`) {
			t.Errorf("Request should use teamIds not teamId: %s", bodyStr)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(graphqlResponse))
	}))
	defer server.Close()

	base := &BaseClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
		apiToken:   "test-token",
	}
	pc := NewProjectClient(base)

	project, err := pc.CreateProject("New Project", "Created project", "team-123")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if project.ID != "new-proj" {
		t.Errorf("Expected project ID 'new-proj', got %s", project.ID)
	}

	// Verify issues field is properly initialized
	if project.Issues == nil {
		t.Error("Expected project to have issues field initialized")
	} else {
		issues := project.GetIssues()
		if len(issues) != 0 {
			t.Errorf("Expected new project to have 0 issues, got %d", len(issues))
		}
	}
}