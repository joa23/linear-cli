package linear

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func buildMockIssueResponse(id, identifier, title, description, stateName, projectName, projectDesc, creatorID, creatorName string) string {
	type mockIssue struct {
		Data struct {
			Issue struct {
				ID          string `json:"id"`
				Identifier  string `json:"identifier"`
				Title       string `json:"title"`
				Description string `json:"description"`
				State       struct {
					Name string `json:"name"`
				} `json:"state"`
				Project struct {
					Name        string `json:"name"`
					Description string `json:"description"`
				} `json:"project"`
				Creator struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"creator"`
			} `json:"issue"`
		} `json:"data"`
	}

	response := mockIssue{}
	response.Data.Issue.ID = id
	response.Data.Issue.Identifier = identifier
	response.Data.Issue.Title = title
	response.Data.Issue.Description = description
	response.Data.Issue.State.Name = stateName
	response.Data.Issue.Project.Name = projectName
	response.Data.Issue.Project.Description = projectDesc
	response.Data.Issue.Creator.ID = creatorID
	response.Data.Issue.Creator.Name = creatorName

	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes)
}

func TestGetIssueWithMetadata(t *testing.T) {
	tests := []struct {
		name             string
		issueID          string
		mockResponse     string
		expectedMetadata map[string]interface{}
		expectedError    bool
	}{
		{
			name:    "issue with metadata in description",
			issueID: "issue-123",
			mockResponse: buildMockIssueResponse(
				"issue-123",
				"LIN-123",
				"Test Issue",
				"This is a test issue.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"priority\": \"high\",\n  \"tags\": [\"bug\", \"urgent\"]\n}\n```\n</details>",
				"In Progress",
				"Test Project",
				"Project description",
				"user-1",
				"John Doe",
			),
			expectedMetadata: map[string]interface{}{
				"priority": "high",
				"tags":     []interface{}{"bug", "urgent"},
			},
			expectedError: false,
		},
		{
			name:    "issue without metadata in description",
			issueID: "issue-456",
			mockResponse: buildMockIssueResponse(
				"issue-456",
				"LIN-456",
				"Another Issue",
				"Simple description without metadata",
				"Todo",
				"Test Project",
				"Project description",
				"user-2",
				"Jane Smith",
			),
			expectedMetadata: map[string]interface{}{},
			expectedError:    false,
		},
		{
			name:    "issue with empty metadata",
			issueID: "issue-789",
			mockResponse: buildMockIssueResponse(
				"issue-789",
				"LIN-789",
				"Empty Metadata Issue",
				"Description with empty metadata.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{}\n```\n</details>",
				"Done",
				"Test Project",
				"Project description",
				"user-3",
				"Bob Johnson",
			),
			expectedMetadata: map[string]interface{}{},
			expectedError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Decode request body
				var requestBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				// Check query contains GetIssue
				query, ok := requestBody["query"].(string)
				if !ok || !contains(query, "query GetIssue") {
					t.Error("Expected GetIssue query")
				}

				// Check variables
				variables, ok := requestBody["variables"].(map[string]interface{})
				if !ok {
					t.Error("Expected variables in request")
				}
				if variables["id"] != tt.issueID {
					t.Errorf("Expected issue ID %s, got %v", tt.issueID, variables["id"])
				}

				// Send mock response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Create client with test server
			client := NewClient("test-token")
			client.base.httpClient = &http.Client{}
			client.base.baseURL = server.URL

			// Call GetIssue
			issue, err := client.GetIssue(tt.issueID)

			// Check error
			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check metadata
			if issue != nil {
				if issue.Metadata == nil && len(tt.expectedMetadata) > 0 {
					t.Error("Metadata is nil but expected values")
				} else if issue.Metadata == nil && len(tt.expectedMetadata) == 0 {
					// Both are empty, that's ok
				} else if !reflect.DeepEqual(issue.Metadata, tt.expectedMetadata) {
					t.Errorf("Metadata mismatch:\nexpected: %+v\ngot: %+v", tt.expectedMetadata, issue.Metadata)
				}
			}
		})
	}
}

// Test for other GetIssue variants
func TestGetIssueWithProjectContextIncludesMetadata(t *testing.T) {
	// Create a more complex response structure for project context
	mockResponse := `{
		"data": {
			"issue": {
				"id": "issue-123",
				"identifier": "LIN-123",
				"title": "Test Issue",
				"description": "Issue with metadata.\n\n<details><summary>🤖 Metadata</summary>\n\n` + "```" + `json\n{\n  \"custom\": \"value\"\n}\n` + "```" + `\n</details>",
				"state": {
					"name": "In Progress"
				},
				"assignee": {
					"id": "user-1",
					"name": "John Doe"
				},
				"creator": {
					"id": "user-2",
					"name": "Jane Smith"
				},
				"project": {
					"id": "proj-1",
					"name": "Test Project",
					"description": "Project desc",
					"issues": {
						"nodes": []
					}
				}
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base.httpClient = &http.Client{}
	client.base.baseURL = server.URL

	issue, err := client.GetIssueWithProjectContext("issue-123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedMetadata := map[string]interface{}{
		"custom": "value",
	}

	if !reflect.DeepEqual(issue.Metadata, expectedMetadata) {
		t.Errorf("Metadata mismatch:\nexpected: %+v\ngot: %+v", expectedMetadata, issue.Metadata)
	}
}

func TestGetIssueWithParentContextIncludesMetadata(t *testing.T) {
	// Create mock response structure
	type mockResponse struct {
		Data struct {
			Issue struct {
				ID          string `json:"id"`
				Identifier  string `json:"identifier"`
				Title       string `json:"title"`
				Description string `json:"description"`
				State       struct {
					Name string `json:"name"`
				} `json:"state"`
				Creator struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"creator"`
				Assignee struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"assignee"`
				Parent *struct {
					ID          string `json:"id"`
					Title       string `json:"title"`
					Description string `json:"description"`
					State       struct {
						Name string `json:"name"`
					} `json:"state"`
					Children struct {
						Nodes []struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							State struct {
								Name string `json:"name"`
							} `json:"state"`
						} `json:"nodes"`
					} `json:"children"`
				} `json:"parent"`
				Comments struct {
					Nodes []struct {
						ID        string `json:"id"`
						Body      string `json:"body"`
						CreatedAt string `json:"createdAt"`
						User      struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"user"`
					} `json:"nodes"`
				} `json:"comments"`
			} `json:"issue"`
		} `json:"data"`
	}

	// Build response with metadata
	resp := mockResponse{}
	resp.Data.Issue.ID = "issue-123"
	resp.Data.Issue.Identifier = "LIN-123"
	resp.Data.Issue.Title = "Test Issue with Parent"
	resp.Data.Issue.Description = "Issue description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"priority\": \"medium\",\n  \"sprint\": 12\n}\n```\n</details>"
	resp.Data.Issue.State.Name = "In Progress"
	resp.Data.Issue.Creator.ID = "user-1"
	resp.Data.Issue.Creator.Name = "John Doe"
	resp.Data.Issue.Assignee.ID = "user-2"
	resp.Data.Issue.Assignee.Name = "Jane Smith"
	
	// Add parent
	resp.Data.Issue.Parent = &struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       struct {
			Name string `json:"name"`
		} `json:"state"`
		Children struct {
			Nodes []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				State struct {
					Name string `json:"name"`
				} `json:"state"`
			} `json:"nodes"`
		} `json:"children"`
	}{}
	resp.Data.Issue.Parent.ID = "parent-issue-1"
	resp.Data.Issue.Parent.Title = "Parent Issue"
	resp.Data.Issue.Parent.Description = "Parent description"
	resp.Data.Issue.Parent.State.Name = "In Progress"
	resp.Data.Issue.Parent.Children.Nodes = []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		State struct {
			Name string `json:"name"`
		} `json:"state"`
	}{
		{
			ID:    "issue-123",
			Title: "Test Issue with Parent",
			State: struct {
				Name string `json:"name"`
			}{Name: "In Progress"},
		},
	}
	
	// Empty comments
	resp.Data.Issue.Comments.Nodes = []struct {
		ID        string `json:"id"`
		Body      string `json:"body"`
		CreatedAt string `json:"createdAt"`
		User      struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"user"`
	}{}

	jsonBytes, _ := json.Marshal(resp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonBytes)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.base.httpClient = &http.Client{}
	client.base.baseURL = server.URL

	issue, err := client.GetIssueWithParentContext("issue-123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedMetadata := map[string]interface{}{
		"priority": "medium",
		"sprint":   float64(12),
	}

	if !reflect.DeepEqual(issue.Metadata, expectedMetadata) {
		t.Errorf("Metadata mismatch:\nexpected: %+v\ngot: %+v", expectedMetadata, issue.Metadata)
	}
}