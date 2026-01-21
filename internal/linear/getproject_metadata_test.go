package linear

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGetProjectWithMetadata(t *testing.T) {
	tests := []struct {
		name             string
		projectID        string
		mockResponse     func() string
		expectedMetadata map[string]interface{}
		expectedError    bool
	}{
		{
			name:      "project with metadata in description",
			projectID: "proj-123",
			mockResponse: func() string {
				// Build response with metadata in description
				type mockResponse struct {
					Data struct {
						Project *struct {
							ID          string `json:"id"`
							Name        string `json:"name"`
							Description string `json:"description"`
							Issues      struct {
								Nodes []struct {
									ID    string `json:"id"`
									Title string `json:"title"`
									State struct {
										Name string `json:"name"`
									} `json:"state"`
								} `json:"nodes"`
							} `json:"issues"`
						} `json:"project"`
					} `json:"data"`
				}

				resp := mockResponse{}
				resp.Data.Project = &struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Description string `json:"description"`
					Issues      struct {
						Nodes []struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							State struct {
								Name string `json:"name"`
							} `json:"state"`
						} `json:"nodes"`
					} `json:"issues"`
				}{}
				
				resp.Data.Project.ID = "proj-123"
				resp.Data.Project.Name = "Test Project"
				resp.Data.Project.Description = "Project description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"team\": \"backend\",\n  \"quarter\": \"Q1\",\n  \"budget\": 50000\n}\n```\n</details>"
				
				// Add some issues
				resp.Data.Project.Issues.Nodes = []struct {
					ID    string `json:"id"`
					Title string `json:"title"`
					State struct {
						Name string `json:"name"`
					} `json:"state"`
				}{
					{
						ID:    "issue-1",
						Title: "Issue 1",
						State: struct {
							Name string `json:"name"`
						}{Name: "In Progress"},
					},
					{
						ID:    "issue-2",
						Title: "Issue 2",
						State: struct {
							Name string `json:"name"`
						}{Name: "Done"},
					},
				}
				
				jsonBytes, _ := json.Marshal(resp)
				return string(jsonBytes)
			},
			expectedMetadata: map[string]interface{}{
				"team":    "backend",
				"quarter": "Q1",
				"budget":  float64(50000),
			},
			expectedError: false,
		},
		{
			name:      "project without metadata",
			projectID: "proj-456",
			mockResponse: func() string {
				type mockResponse struct {
					Data struct {
						Project *struct {
							ID          string `json:"id"`
							Name        string `json:"name"`
							Description string `json:"description"`
							Issues      struct {
								Nodes []struct {
									ID    string `json:"id"`
									Title string `json:"title"`
									State struct {
										Name string `json:"name"`
									} `json:"state"`
								} `json:"nodes"`
							} `json:"issues"`
						} `json:"project"`
					} `json:"data"`
				}

				resp := mockResponse{}
				resp.Data.Project = &struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Description string `json:"description"`
					Issues      struct {
						Nodes []struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							State struct {
								Name string `json:"name"`
							} `json:"state"`
						} `json:"nodes"`
					} `json:"issues"`
				}{}
				
				resp.Data.Project.ID = "proj-456"
				resp.Data.Project.Name = "Another Project"
				resp.Data.Project.Description = "Simple project description without metadata"
				resp.Data.Project.Issues.Nodes = []struct {
					ID    string `json:"id"`
					Title string `json:"title"`
					State struct {
						Name string `json:"name"`
					} `json:"state"`
				}{}
				
				jsonBytes, _ := json.Marshal(resp)
				return string(jsonBytes)
			},
			expectedMetadata: map[string]interface{}{},
			expectedError:    false,
		},
		{
			name:      "project not found",
			projectID: "proj-999",
			mockResponse: func() string {
				type mockResponse struct {
					Data struct {
						Project interface{} `json:"project"`
					} `json:"data"`
				}
				
				resp := mockResponse{}
				resp.Data.Project = nil
				
				jsonBytes, _ := json.Marshal(resp)
				return string(jsonBytes)
			},
			expectedMetadata: nil,
			expectedError:    true,
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

				// Check query contains GetProject
				query, ok := requestBody["query"].(string)
				if !ok || !contains(query, "query GetProject") {
					t.Error("Expected GetProject query")
				}

				// Check variables
				variables, ok := requestBody["variables"].(map[string]interface{})
				if !ok {
					t.Error("Expected variables in request")
				}
				if variables["id"] != tt.projectID {
					t.Errorf("Expected project ID %s, got %v", tt.projectID, variables["id"])
				}

				// Send mock response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.mockResponse()))
			}))
			defer server.Close()

			// Create client with test server
			client := NewClient("test-token")
			client.base.httpClient = &http.Client{}
			client.base.baseURL = server.URL

			// Call GetProject
			project, err := client.GetProject(tt.projectID)

			// Check error
			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check metadata if no error
			if !tt.expectedError && project != nil {
				if project.Metadata == nil && len(tt.expectedMetadata) > 0 {
					t.Error("Metadata is nil but expected values")
				} else if project.Metadata == nil && len(tt.expectedMetadata) == 0 {
					// Both are empty, that's ok
				} else if !reflect.DeepEqual(project.Metadata, tt.expectedMetadata) {
					t.Errorf("Metadata mismatch:\nexpected: %+v\ngot: %+v", tt.expectedMetadata, project.Metadata)
				}
			}
		})
	}
}

func TestGetIssueWithProjectContextIncludesProjectMetadata(t *testing.T) {
	// Create mock response structure
	type mockResponse struct {
		Data struct {
			Issue struct {
				ID          string `json:"id"`
				Identifier  string `json:"identifier"`
				Title       string `json:"title"`
				Description string `json:"description"`
				State       struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"state"`
				Assignee struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"assignee"`
				Creator struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"creator"`
				Project struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Description string `json:"description"`
					Issues      struct {
						Nodes []struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							State struct {
								Name string `json:"name"`
							} `json:"state"`
						} `json:"nodes"`
					} `json:"issues"`
				} `json:"project"`
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

	// Build response with project metadata
	resp := mockResponse{}
	resp.Data.Issue.ID = "issue-123"
	resp.Data.Issue.Identifier = "LIN-123"
	resp.Data.Issue.Title = "Test Issue"
	resp.Data.Issue.Description = "Issue description"
	resp.Data.Issue.State.Name = "In Progress"
	resp.Data.Issue.Assignee.ID = "user-1"
	resp.Data.Issue.Assignee.Name = "John Doe"
	resp.Data.Issue.Creator.ID = "user-2"
	resp.Data.Issue.Creator.Name = "Jane Smith"
	
	// Project with metadata
	resp.Data.Issue.Project.ID = "proj-1"
	resp.Data.Issue.Project.Name = "Test Project"
	resp.Data.Issue.Project.Description = "Project description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"department\": \"engineering\",\n  \"priority\": \"high\"\n}\n```\n</details>"
	resp.Data.Issue.Project.Issues.Nodes = []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		State struct {
			Name string `json:"name"`
		} `json:"state"`
	}{}
	
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

	issue, err := client.GetIssueWithProjectContext("issue-123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedProjectMetadata := map[string]interface{}{
		"department": "engineering",
		"priority":   "high",
	}

	if !reflect.DeepEqual(issue.Project.Metadata, expectedProjectMetadata) {
		t.Errorf("Project metadata mismatch:\nexpected: %+v\ngot: %+v", expectedProjectMetadata, issue.Project.Metadata)
	}
}