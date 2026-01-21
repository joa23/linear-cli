package linear

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateIssueMetadataKey(t *testing.T) {
	tests := []struct {
		name                  string
		issueID               string
		key                   string
		value                 interface{}
		mockGetResponse       func() string
		mockUpdateResponse    func() string
		expectedGetCalled     bool
		expectedUpdateCalled  bool
		validateDescription   func(string) error
		expectedError         bool
		errorMessage          string
	}{
		{
			name:    "update existing key in metadata",
			issueID: "issue-123",
			key:     "priority",
			value:   "critical",
			mockGetResponse: func() string {
				// Mock response with existing metadata
				resp := buildMockIssueResponse(
					"issue-123",
					"LIN-123",
					"Test Issue",
					"Issue description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"priority\": \"high\",\n  \"category\": \"bug\"\n}\n```\n</details>",
					"In Progress",
					"Test Project",
					"Project desc",
					"user-1",
					"John Doe",
				)
				return resp
			},
			mockUpdateResponse: func() string {
				return `{
					"data": {
						"issueUpdate": {
							"success": true,
							"issue": {
								"id": "issue-123"
							}
						}
					}
				}`
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: true,
			validateDescription: func(desc string) error {
				// Extract metadata from the description
				metadata, _ := extractMetadataFromDescription(desc)
				
				// Check that priority was updated
				if priority, ok := metadata["priority"].(string); !ok || priority != "critical" {
					return fmt.Errorf("priority not updated correctly, got: %v", metadata["priority"])
				}
				
				// Check that category is still there
				if category, ok := metadata["category"].(string); !ok || category != "bug" {
					return fmt.Errorf("category was lost, got: %v", metadata["category"])
				}
				
				return nil
			},
			expectedError: false,
		},
		{
			name:    "add new key to existing metadata",
			issueID: "issue-456",
			key:     "assignedTeam",
			value:   "backend",
			mockGetResponse: func() string {
				resp := buildMockIssueResponse(
					"issue-456",
					"LIN-456",
					"Another Issue",
					"Description with metadata.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"status\": \"active\"\n}\n```\n</details>",
					"Todo",
					"Test Project",
					"Project desc",
					"user-2",
					"Jane Smith",
				)
				return resp
			},
			mockUpdateResponse: func() string {
				return `{
					"data": {
						"issueUpdate": {
							"success": true,
							"issue": {
								"id": "issue-456"
							}
						}
					}
				}`
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: true,
			validateDescription: func(desc string) error {
				// Extract metadata from the description
				metadata, _ := extractMetadataFromDescription(desc)
				
				// Check that new key was added
				if team, ok := metadata["assignedTeam"].(string); !ok || team != "backend" {
					return fmt.Errorf("assignedTeam not added correctly, got: %v", metadata["assignedTeam"])
				}
				
				// Check that existing key is still there
				if status, ok := metadata["status"].(string); !ok || status != "active" {
					return fmt.Errorf("status was lost, got: %v", metadata["status"])
				}
				
				return nil
			},
			expectedError: false,
		},
		{
			name:    "add key to issue without metadata",
			issueID: "issue-789",
			key:     "type",
			value:   "feature",
			mockGetResponse: func() string {
				resp := buildMockIssueResponse(
					"issue-789",
					"LIN-789",
					"Issue without metadata",
					"Simple description without any metadata.",
					"Done",
					"Test Project",
					"Project desc",
					"user-3",
					"Bob Johnson",
				)
				return resp
			},
			mockUpdateResponse: func() string {
				return `{
					"data": {
						"issueUpdate": {
							"success": true,
							"issue": {
								"id": "issue-789"
							}
						}
					}
				}`
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: true,
			validateDescription: func(desc string) error {
				// Extract metadata from the description
				metadata, cleanDesc := extractMetadataFromDescription(desc)
				
				// Check that metadata was added
				if metadata == nil || len(metadata) == 0 {
					return fmt.Errorf("metadata was not added")
				}
				
				// Check that the key was set
				if issueType, ok := metadata["type"].(string); !ok || issueType != "feature" {
					return fmt.Errorf("type not set correctly, got: %v", metadata["type"])
				}
				
				// Check that original description is preserved
				if cleanDesc != "Simple description without any metadata." {
					return fmt.Errorf("original description not preserved, got: %s", cleanDesc)
				}
				
				return nil
			},
			expectedError: false,
		},
		{
			name:    "update with complex value",
			issueID: "issue-complex",
			key:     "details",
			value: map[string]interface{}{
				"component": "api",
				"version":   "2.0",
				"features":  []string{"auth", "logging"},
			},
			mockGetResponse: func() string {
				resp := buildMockIssueResponse(
					"issue-complex",
					"LIN-COMPLEX",
					"Complex Issue",
					"Issue with complex metadata update.",
					"In Progress",
					"Test Project",
					"Project desc",
					"user-4",
					"Alice Cooper",
				)
				return resp
			},
			mockUpdateResponse: func() string {
				return `{
					"data": {
						"issueUpdate": {
							"success": true,
							"issue": {
								"id": "issue-complex"
							}
						}
					}
				}`
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: true,
			validateDescription: func(desc string) error {
				// Extract metadata from the description
				metadata, _ := extractMetadataFromDescription(desc)
				
				// Check that complex value was set
				details, ok := metadata["details"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("details not set as map, got: %T", metadata["details"])
				}
				
				if details["component"] != "api" || details["version"] != "2.0" {
					return fmt.Errorf("complex value not set correctly, got: %v", details)
				}
				
				return nil
			},
			expectedError: false,
		},
		{
			name:    "issue not found",
			issueID: "issue-notfound",
			key:     "status",
			value:   "active",
			mockGetResponse: func() string {
				// Return a proper GraphQL error response
				return `{
					"data": {
						"issue": {
							"id": "",
							"identifier": "",
							"title": "",
							"description": "",
							"state": {
								"name": ""
							},
							"project": {
								"name": "",
								"description": ""
							},
							"creator": {
								"id": "",
								"name": ""
							}
						}
					},
					"errors": [{
						"message": "Issue not found"
					}]
				}`
			},
			mockUpdateResponse: func() string {
				return "" // Should not be called
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: false,
			expectedError:        true,
			errorMessage:         "Issue not found",
		},
		{
			name:    "update fails",
			issueID: "issue-fail",
			key:     "priority",
			value:   "high",
			mockGetResponse: func() string {
				resp := buildMockIssueResponse(
					"issue-fail",
					"LIN-FAIL",
					"Failing Issue",
					"Description",
					"Todo",
					"Test Project",
					"Project desc",
					"user-5",
					"Charlie Brown",
				)
				return resp
			},
			mockUpdateResponse: func() string {
				return `{
					"data": {
						"issueUpdate": {
							"success": false
						}
					}
				}`
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: true,
			expectedError:        true,
			errorMessage:         "was not successful",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getCalled := false
			updateCalled := false
			var capturedDescription string

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Decode request to determine which query it is
				var requestBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				query, ok := requestBody["query"].(string)
				if !ok {
					t.Fatal("No query in request")
				}

				// Check if it's a GetIssue query or an update mutation
				if contains(query, "query GetIssue") {
					getCalled = true
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(tt.mockGetResponse()))
				} else if contains(query, "mutation UpdateIssueDescription") {
					updateCalled = true
					
					// Capture the description from the request
					if variables, ok := requestBody["variables"].(map[string]interface{}); ok {
						if desc, ok := variables["description"].(string); ok {
							capturedDescription = desc
						}
					}
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(tt.mockUpdateResponse()))
				} else {
					t.Fatalf("Unexpected query: %s", query)
				}
			}))
			defer server.Close()

			// Create client with test server
			client := NewClient("test-token")
			client.base.httpClient = &http.Client{}
			client.base.baseURL = server.URL

			// Call UpdateIssueMetadataKey
			err := client.UpdateIssueMetadataKey(tt.issueID, tt.key, tt.value)

			// Check if calls were made as expected
			if getCalled != tt.expectedGetCalled {
				t.Errorf("GetIssue called = %v, want %v", getCalled, tt.expectedGetCalled)
			}
			if updateCalled != tt.expectedUpdateCalled {
				t.Errorf("UpdateIssue called = %v, want %v", updateCalled, tt.expectedUpdateCalled)
			}

			// Check error
			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMessage != "" && !contains(err.Error(), tt.errorMessage) {
					t.Errorf("Error message = %v, want to contain %v", err.Error(), tt.errorMessage)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Validate captured description
			if updateCalled && tt.validateDescription != nil {
				if err := tt.validateDescription(capturedDescription); err != nil {
					t.Errorf("Description validation failed: %v", err)
				}
			}
		})
	}
}

func TestRemoveIssueMetadataKey(t *testing.T) {
	tests := []struct {
		name                  string
		issueID               string
		key                   string
		mockGetResponse       func() string
		mockUpdateResponse    func() string
		expectedGetCalled     bool
		expectedUpdateCalled  bool
		validateDescription   func(string) error
		expectedError         bool
		errorMessage          string
	}{
		{
			name:    "remove existing key from metadata",
			issueID: "issue-123",
			key:     "priority",
			mockGetResponse: func() string {
				// Mock response with existing metadata
				resp := buildMockIssueResponse(
					"issue-123",
					"LIN-123",
					"Test Issue",
					"Issue description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"priority\": \"high\",\n  \"category\": \"bug\",\n  \"status\": \"active\"\n}\n```\n</details>",
					"In Progress",
					"Test Project",
					"Project desc",
					"user-1",
					"John Doe",
				)
				return resp
			},
			mockUpdateResponse: func() string {
				return `{
					"data": {
						"issueUpdate": {
							"success": true,
							"issue": {
								"id": "issue-123"
							}
						}
					}
				}`
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: true,
			validateDescription: func(desc string) error {
				// Extract metadata from the description
				metadata, _ := extractMetadataFromDescription(desc)
				
				// Check that priority was removed
				if _, exists := metadata["priority"]; exists {
					return fmt.Errorf("priority key was not removed")
				}
				
				// Check that other keys are still there
				if category, ok := metadata["category"].(string); !ok || category != "bug" {
					return fmt.Errorf("category was lost, got: %v", metadata["category"])
				}
				if status, ok := metadata["status"].(string); !ok || status != "active" {
					return fmt.Errorf("status was lost, got: %v", metadata["status"])
				}
				
				return nil
			},
			expectedError: false,
		},
		{
			name:    "remove non-existent key",
			issueID: "issue-456",
			key:     "nonExistentKey",
			mockGetResponse: func() string {
				resp := buildMockIssueResponse(
					"issue-456",
					"LIN-456",
					"Another Issue",
					"Description with metadata.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"status\": \"active\",\n  \"type\": \"feature\"\n}\n```\n</details>",
					"Todo",
					"Test Project",
					"Project desc",
					"user-2",
					"Jane Smith",
				)
				return resp
			},
			mockUpdateResponse: func() string {
				return `{
					"data": {
						"issueUpdate": {
							"success": true,
							"issue": {
								"id": "issue-456"
							}
						}
					}
				}`
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: false, // No update needed if key doesn't exist
			validateDescription: func(desc string) error {
				// Extract metadata from the description
				metadata, _ := extractMetadataFromDescription(desc)
				
				// Check that existing keys are still there
				if status, ok := metadata["status"].(string); !ok || status != "active" {
					return fmt.Errorf("status was lost, got: %v", metadata["status"])
				}
				if issueType, ok := metadata["type"].(string); !ok || issueType != "feature" {
					return fmt.Errorf("type was lost, got: %v", metadata["type"])
				}
				
				// Ensure we still have 2 keys
				if len(metadata) != 2 {
					return fmt.Errorf("expected 2 keys, got %d", len(metadata))
				}
				
				return nil
			},
			expectedError: false,
		},
		{
			name:    "remove last key - metadata section should be removed",
			issueID: "issue-789",
			key:     "onlyKey",
			mockGetResponse: func() string {
				resp := buildMockIssueResponse(
					"issue-789",
					"LIN-789",
					"Issue with single metadata key",
					"Simple description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"onlyKey\": \"value\"\n}\n```\n</details>",
					"Done",
					"Test Project",
					"Project desc",
					"user-3",
					"Bob Johnson",
				)
				return resp
			},
			mockUpdateResponse: func() string {
				return `{
					"data": {
						"issueUpdate": {
							"success": true,
							"issue": {
								"id": "issue-789"
							}
						}
					}
				}`
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: true,
			validateDescription: func(desc string) error {
				// Extract metadata from the description
				metadata, cleanDesc := extractMetadataFromDescription(desc)
				
				// Check that metadata is empty
				if len(metadata) != 0 {
					return fmt.Errorf("metadata should be empty after removing last key, got: %v", metadata)
				}
				
				// Check that description is clean (no metadata section)
				if strings.Contains(desc, "<details>") || strings.Contains(desc, "🤖 Metadata") {
					return fmt.Errorf("metadata section should be removed when empty")
				}
				
				// Check that original description is preserved
				if cleanDesc != "Simple description." {
					return fmt.Errorf("original description not preserved, got: %s", cleanDesc)
				}
				
				return nil
			},
			expectedError: false,
		},
		{
			name:    "remove key from issue without metadata",
			issueID: "issue-no-meta",
			key:     "someKey",
			mockGetResponse: func() string {
				resp := buildMockIssueResponse(
					"issue-no-meta",
					"LIN-NOMETA",
					"Issue without metadata",
					"Plain description without any metadata.",
					"Todo",
					"Test Project",
					"Project desc",
					"user-4",
					"Alice Cooper",
				)
				return resp
			},
			mockUpdateResponse: func() string {
				return "" // Should not be called
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: false,
			expectedError:        false, // No error, just no-op
		},
		{
			name:    "issue not found",
			issueID: "issue-notfound",
			key:     "anyKey",
			mockGetResponse: func() string {
				return `{
					"data": {
						"issue": {
							"id": "",
							"identifier": "",
							"title": "",
							"description": "",
							"state": {
								"name": ""
							},
							"project": {
								"name": "",
								"description": ""
							},
							"creator": {
								"id": "",
								"name": ""
							}
						}
					},
					"errors": [{
						"message": "Issue not found"
					}]
				}`
			},
			mockUpdateResponse: func() string {
				return "" // Should not be called
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: false,
			expectedError:        true,
			errorMessage:         "Issue not found",
		},
		{
			name:    "update fails",
			issueID: "issue-fail",
			key:     "priority",
			mockGetResponse: func() string {
				resp := buildMockIssueResponse(
					"issue-fail",
					"LIN-FAIL",
					"Failing Issue",
					"Description.\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\n  \"priority\": \"high\"\n}\n```\n</details>",
					"Todo",
					"Test Project",
					"Project desc",
					"user-5",
					"Charlie Brown",
				)
				return resp
			},
			mockUpdateResponse: func() string {
				return `{
					"data": {
						"issueUpdate": {
							"success": false
						}
					}
				}`
			},
			expectedGetCalled:    true,
			expectedUpdateCalled: true,
			expectedError:        true,
			errorMessage:         "was not successful",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getCalled := false
			updateCalled := false
			var capturedDescription string

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Decode request to determine which query it is
				var requestBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				query, ok := requestBody["query"].(string)
				if !ok {
					t.Fatal("No query in request")
				}

				// Check if it's a GetIssue query or an update mutation
				if contains(query, "query GetIssue") {
					getCalled = true
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(tt.mockGetResponse()))
				} else if contains(query, "mutation UpdateIssueDescription") {
					updateCalled = true
					
					// Capture the description from the request
					if variables, ok := requestBody["variables"].(map[string]interface{}); ok {
						if desc, ok := variables["description"].(string); ok {
							capturedDescription = desc
						}
					}
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(tt.mockUpdateResponse()))
				} else {
					t.Fatalf("Unexpected query: %s", query)
				}
			}))
			defer server.Close()

			// Create client with test server
			client := NewClient("test-token")
			client.base.httpClient = &http.Client{}
			client.base.baseURL = server.URL

			// Call RemoveIssueMetadataKey
			err := client.RemoveIssueMetadataKey(tt.issueID, tt.key)

			// Check if calls were made as expected
			if getCalled != tt.expectedGetCalled {
				t.Errorf("GetIssue called = %v, want %v", getCalled, tt.expectedGetCalled)
			}
			if updateCalled != tt.expectedUpdateCalled {
				t.Errorf("UpdateIssue called = %v, want %v", updateCalled, tt.expectedUpdateCalled)
			}

			// Check error
			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMessage != "" && !contains(err.Error(), tt.errorMessage) {
					t.Errorf("Error message = %v, want to contain %v", err.Error(), tt.errorMessage)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Validate captured description
			if updateCalled && tt.validateDescription != nil {
				if err := tt.validateDescription(capturedDescription); err != nil {
					t.Errorf("Description validation failed: %v", err)
				}
			}
		})
	}
}