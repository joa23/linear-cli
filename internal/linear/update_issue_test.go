package linear

import (
	"github.com/joa23/linear-cli/internal/token"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

// createTestBaseClient creates a BaseClient for testing
func createTestBaseClient(serverURL, tokenStr string) *BaseClient {
	return &BaseClient{
		tokenProvider: token.NewStaticProvider(tokenStr),
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		baseURL:       serverURL,
	}
}

func TestUpdateIssue(t *testing.T) {
	tests := []struct {
		name           string
		issueID        string
		input          UpdateIssueInput
		mockResponse   string
		expectedError  string
		validateRequest func(t *testing.T, req map[string]interface{})
	}{
		{
			name:    "Update title only",
			issueID: "issue-123",
			input: UpdateIssueInput{
				Title: stringPtr("Updated Title"),
			},
			mockResponse: `{
				"data": {
					"issueUpdate": {
						"success": true,
						"issue": {
							"id": "issue-123",
							"title": "Updated Title",
							"identifier": "ENG-123",
							"description": "Original description",
							"state": {
								"id": "state-1",
								"name": "In Progress"
							},
							"updatedAt": "2024-01-01T00:00:00Z"
						}
					}
				}
			}`,
			validateRequest: func(t *testing.T, req map[string]interface{}) {
				input := req["input"].(map[string]interface{})
				if input["title"] != "Updated Title" {
					t.Errorf("Expected title 'Updated Title', got %v", input["title"])
				}
				// Ensure other fields are not included
				if _, exists := input["description"]; exists {
					t.Error("Description should not be included when not specified")
				}
			},
		},
		{
			name:    "Update multiple fields",
			issueID: "issue-456",
			input: UpdateIssueInput{
				Title:       stringPtr("New Title"),
				Description: stringPtr("New description"),
				Priority:    intPtr(2), // High priority
				Estimate:    float64Ptr(5.0),
				DueDate:     stringPtr("2024-12-31T00:00:00Z"),
			},
			mockResponse: `{
				"data": {
					"issueUpdate": {
						"success": true,
						"issue": {
							"id": "issue-456",
							"title": "New Title",
							"identifier": "ENG-456",
							"description": "New description",
							"priority": 2,
							"estimate": 5.0,
							"dueDate": "2024-12-31T00:00:00Z",
							"state": {
								"id": "state-1",
								"name": "In Progress"
							},
							"updatedAt": "2024-01-01T00:00:00Z"
						}
					}
				}
			}`,
			validateRequest: func(t *testing.T, req map[string]interface{}) {
				input := req["input"].(map[string]interface{})
				if input["title"] != "New Title" {
					t.Errorf("Expected title 'New Title', got %v", input["title"])
				}
				if input["description"] != "New description" {
					t.Errorf("Expected description 'New description', got %v", input["description"])
				}
				if input["priority"] != float64(2) {
					t.Errorf("Expected priority 2, got %v", input["priority"])
				}
				if input["estimate"] != float64(5.0) {
					t.Errorf("Expected estimate 5.0, got %v", input["estimate"])
				}
				if input["dueDate"] != "2024-12-31T00:00:00Z" {
					t.Errorf("Expected dueDate '2024-12-31T00:00:00Z', got %v", input["dueDate"])
				}
			},
		},
		// Skipping metadata preservation test - handled separately in TestUpdateIssueMetadataPreservation
		{
			name:    "Update state ID",
			issueID: "issue-111",
			input: UpdateIssueInput{
				StateID: stringPtr("state-done"),
			},
			mockResponse: `{
				"data": {
					"issueUpdate": {
						"success": true,
						"issue": {
							"id": "issue-111",
							"state": {
								"id": "state-done",
								"name": "Done"
							}
						}
					}
				}
			}`,
			validateRequest: func(t *testing.T, req map[string]interface{}) {
				input := req["input"].(map[string]interface{})
				if input["stateId"] != "state-done" {
					t.Errorf("Expected stateId 'state-done', got %v", input["stateId"])
				}
			},
		},
		{
			name:    "Update assignee with null value (unassign)",
			issueID: "issue-222",
			input: UpdateIssueInput{
				AssigneeID: stringPtr(""), // Empty string means unassign
			},
			mockResponse: `{
				"data": {
					"issueUpdate": {
						"success": true,
						"issue": {
							"id": "issue-222",
							"assignee": null
						}
					}
				}
			}`,
			validateRequest: func(t *testing.T, req map[string]interface{}) {
				input := req["input"].(map[string]interface{})
				assigneeInput := input["assigneeId"]
				if assigneeInput != nil {
					t.Errorf("Expected assigneeId to be nil for unassignment, got %v", assigneeInput)
				}
			},
		},
		{
			name:    "Update with label IDs",
			issueID: "issue-333",
			input: UpdateIssueInput{
				LabelIDs: []string{"label-1", "label-2"},
			},
			mockResponse: `{
				"data": {
					"issueUpdate": {
						"success": true,
						"issue": {
							"id": "issue-333"
						}
					}
				}
			}`,
			validateRequest: func(t *testing.T, req map[string]interface{}) {
				input := req["input"].(map[string]interface{})
				labelIDs := input["labelIds"].([]interface{})
				if len(labelIDs) != 2 {
					t.Errorf("Expected 2 label IDs, got %d", len(labelIDs))
				}
				if labelIDs[0] != "label-1" || labelIDs[1] != "label-2" {
					t.Errorf("Unexpected label IDs: %v", labelIDs)
				}
			},
		},
		{
			name:    "Update with cycle ID",
			issueID: "issue-888",
			input: UpdateIssueInput{
				CycleID: stringPtr("cycle-65"),
			},
			mockResponse: `{
				"data": {
					"issueUpdate": {
						"success": true,
						"issue": {
							"id": "issue-888",
							"cycle": {
								"id": "cycle-65",
								"name": "Cycle 65"
							}
						}
					}
				}
			}`,
			validateRequest: func(t *testing.T, req map[string]interface{}) {
				input := req["input"].(map[string]interface{})
				if input["cycleId"] != "cycle-65" {
					t.Errorf("Expected cycleId 'cycle-65', got %v", input["cycleId"])
				}
			},
		},
		{
			name:    "Empty issue ID",
			issueID: "",
			input: UpdateIssueInput{
				Title: stringPtr("Title"),
			},
			expectedError: "issueID cannot be empty",
		},
		{
			name:    "No fields to update",
			issueID: "issue-444",
			input:   UpdateIssueInput{},
			expectedError: "no fields to update",
		},
		{
			name:    "Invalid priority value",
			issueID: "issue-555",
			input: UpdateIssueInput{
				Priority: intPtr(5), // Invalid priority (should be 0-4)
			},
			expectedError: "invalid priority value: 5 (must be between 0-4)",
		},
		{
			name:    "Update fails",
			issueID: "issue-666",
			input: UpdateIssueInput{
				Title: stringPtr("New Title"),
			},
			mockResponse: `{
				"data": {
					"issueUpdate": {
						"success": false
					}
				}
			}`,
			expectedError: "issue update was not successful",
		},
		{
			name:    "GraphQL error",
			issueID: "issue-777",
			input: UpdateIssueInput{
				Title: stringPtr("New Title"),
			},
			mockResponse: `{
				"errors": [
					{
						"message": "Issue not found",
						"extensions": {
							"code": "NOT_FOUND"
						}
					}
				]
			}`,
			expectedError: "Issue not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequest map[string]interface{}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Capture the request for validation
				var req struct {
					Variables map[string]interface{} `json:"variables"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("Failed to decode request: %v", err)
				}

				// Check if this is a GetIssue query (has "id" but no "input")
				// This happens when UpdateIssue needs to preserve metadata
				if _, hasInput := req.Variables["input"]; !hasInput {
					// This is a GetIssue call - return a minimal issue response
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{
						"data": {
							"issue": {
								"id": "` + tt.issueID + `",
								"identifier": "ENG-` + tt.issueID[6:] + `",
								"title": "Current Title",
								"description": "Current description",
								"state": {
									"id": "state-1",
									"name": "In Progress"
								},
								"createdAt": "2024-01-01T00:00:00Z",
								"updatedAt": "2024-01-01T00:00:00Z",
								"url": "https://linear.app/test/issue/ENG-` + tt.issueID[6:] + `",
								"children": { "nodes": [] },
								"attachments": { "nodes": [] }
							}
						}
					}`))
					return
				}

				// This is an issueUpdate call - capture for validation
				capturedRequest = req.Variables

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			base := createTestBaseClient(server.URL, "test-token")
			issueClient := NewIssueClient(base)

			_, err := issueClient.UpdateIssue(tt.issueID, tt.input)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error '%s', got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError && !containsError(err.Error(), tt.expectedError) {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.validateRequest != nil && capturedRequest != nil {
					tt.validateRequest(t, capturedRequest)
				}
			}
		})
	}
}

func TestUpdateIssueWithMetadataPreservation(t *testing.T) {
	// Mock server that simulates getting issue and then updating it
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		callCount++
		if callCount == 1 {
			// First call: GetIssue to retrieve current metadata
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"issue": map[string]interface{}{
						"id":         "issue-meta-123",
						"identifier": "ENG-123",
						"title":      "Original Title",
						"description": "Original description\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\"preserved\":\"data\",\"count\":42}\n```\n</details>",
						"state": map[string]interface{}{
							"id":   "state-1",
							"name": "In Progress",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			// Second call: UpdateIssue with preserved metadata
			var req struct {
				Variables map[string]interface{} `json:"variables"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			
			input := req.Variables["input"].(map[string]interface{})
			desc := input["description"].(string)
			
			// Verify metadata is preserved in the update
			if !strings.Contains(desc, "New description") {
				t.Errorf("Description should start with 'New description', got: %s", desc)
			}
			if !strings.Contains(desc, "🤖 Metadata") {
				t.Errorf("Description should contain metadata section, got: %s", desc) 
			}
			// Extract and verify the metadata JSON
			metadataMatch := regexp.MustCompile(`(?s)` + "```json\n(.+?)\n```").FindStringSubmatch(desc)
			if len(metadataMatch) < 2 {
				t.Errorf("Could not extract metadata JSON from description: %s", desc)
			} else {
				var extractedMeta map[string]interface{}
				if err := json.Unmarshal([]byte(metadataMatch[1]), &extractedMeta); err != nil {
					t.Errorf("Failed to parse metadata JSON: %v", err)
				} else {
					if extractedMeta["preserved"] != "data" || extractedMeta["count"] != float64(42) {
						t.Errorf("Metadata not preserved correctly: %v", extractedMeta)
					}
				}
			}
			
			w.Write([]byte(`{
				"data": {
					"issueUpdate": {
						"success": true,
						"issue": {
							"id": "issue-meta-123"
						}
					}
				}
			}`))
		}
	}))
	defer server.Close()

	base := createTestBaseClient(server.URL, "test-token")
	issueClient := NewIssueClient(base)

	input := UpdateIssueInput{
		Description: stringPtr("New description"),
	}

	_, err := issueClient.UpdateIssue("issue-meta-123", input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 API calls (GetIssue + UpdateIssue), got %d", callCount)
	}
}

// Helper functions
func containsError(actual, expected string) bool {
	// Check exact match first
	if actual == expected {
		return true
	}

	// Check common error wrapping patterns
	patterns := []string{
		"validation error: issueID " + expected,
		"validation error: input " + expected,
		"validation error: priority " + expected,
		"failed to update issue: " + expected,
		"failed to update issue: GraphQL error: " + expected,
	}

	for _, pattern := range patterns {
		if actual == pattern {
			return true
		}
	}

	// Check for GraphQL error with query context (includes query preview)
	// Pattern: "failed to update issue: GraphQL error: {expected} (query: ...)"
	if strings.Contains(actual, "failed to update issue: GraphQL error: "+expected+" (query:") {
		return true
	}

	return false
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}