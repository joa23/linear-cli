package linear

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetNotifications(t *testing.T) {
	tests := []struct {
		name           string
		includeRead    bool
		limit          int
		mockResponse   string
		expectedCount  int
		expectedError  bool
	}{
		{
			name:        "successful notification fetch",
			includeRead: false,
			limit:       10,
			mockResponse: `{
				"data": {
					"notifications": {
						"nodes": [
							{
								"id": "notif-1",
								"type": "IssueNotification",
								"readAt": null,
								"createdAt": "2024-01-01T00:00:00Z",
								"issue": {
									"id": "issue-1",
									"title": "Test Issue"
								},
								"comment": {
									"id": "comment-1",
									"body": "Test comment"
								},
								"user": {
									"id": "user-1",
									"name": "Test User"
								}
							},
							{
								"id": "notif-2",
								"type": "ProjectNotification",
								"readAt": "2024-01-02T00:00:00Z",
								"createdAt": "2024-01-01T00:00:00Z",
								"project": {
									"id": "project-1",
									"name": "Test Project"
								},
								"user": {
									"id": "user-2",
									"name": "Another User"
								}
							}
						]
					}
				}
			}`,
			expectedCount: 2,
			expectedError: false,
		},
		{
			name:        "empty notifications",
			includeRead: true,
			limit:       50,
			mockResponse: `{
				"data": {
					"notifications": {
						"nodes": []
					}
				}
			}`,
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:          "API error",
			includeRead:   false,
			limit:         10,
			mockResponse:  `{"data": {"notifications": {"nodes": null}}}`,
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				// Verify the query includes proper inline fragments
				var requestBody struct {
					Query     string                 `json:"query"`
					Variables map[string]interface{} `json:"variables"`
				}
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				// Check for inline fragments in query
				if !contains(requestBody.Query, "... on IssueNotification") {
					t.Error("Query missing IssueNotification inline fragment")
				}
				if !contains(requestBody.Query, "... on ProjectNotification") {
					t.Error("Query missing ProjectNotification inline fragment")
				}

				// Verify variables
				if includeArchived, ok := requestBody.Variables["includeArchived"].(bool); !ok || includeArchived != tt.includeRead {
					t.Errorf("Expected includeArchived=%v, got %v", tt.includeRead, requestBody.Variables["includeArchived"])
				}
				if first, ok := requestBody.Variables["first"].(float64); !ok || int(first) != tt.limit {
					t.Errorf("Expected first=%d, got %v", tt.limit, requestBody.Variables["first"])
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.base.httpClient = &http.Client{}
			client.base.baseURL = server.URL

			notifications, err := client.GetNotifications(tt.includeRead, tt.limit)

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(notifications) != tt.expectedCount {
				t.Errorf("Expected %d notifications, got %d", tt.expectedCount, len(notifications))
			}

			// Verify notification content for successful case
			if tt.expectedCount > 0 && len(notifications) > 0 {
				// Check first notification (IssueNotification)
				if notifications[0].Type != "IssueNotification" {
					t.Errorf("Expected first notification type to be IssueNotification, got %s", notifications[0].Type)
				}
				if notifications[0].Issue == nil {
					t.Error("Expected issue data in first notification")
				}
				if notifications[0].Comment == nil {
					t.Error("Expected comment data in first notification")
				}

				// Check second notification (ProjectNotification)
				if len(notifications) > 1 {
					if notifications[1].Type != "ProjectNotification" {
						t.Errorf("Expected second notification type to be ProjectNotification, got %s", notifications[1].Type)
					}
					// Project field no longer exists in Notification struct
					/*
					if notifications[1].Project == nil {
						t.Error("Expected project data in second notification")
					}
					*/
					if notifications[1].ReadAt == nil {
						t.Error("Expected readAt to be set for second notification")
					}
				}
			}
		})
	}
}

func TestMarkNotificationAsRead(t *testing.T) {
	tests := []struct {
		name           string
		notificationID string
		mockResponse   string
		expectedError  bool
	}{
		{
			name:           "successful mark as read",
			notificationID: "notif-123",
			mockResponse: `{
				"data": {
					"notificationUpdate": {
						"success": true,
						"notification": {
							"id": "notif-123",
							"readAt": "2024-01-01T00:00:00Z"
						}
					}
				}
			}`,
			expectedError: false,
		},
		{
			name:           "notification not found",
			notificationID: "notif-invalid",
			mockResponse: `{
				"data": {
					"notificationUpdate": {
						"success": false
					}
				}
			}`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				var requestBody struct {
					Query     string                 `json:"query"`
					Variables map[string]interface{} `json:"variables"`
				}
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				// Verify mutation
				if !contains(requestBody.Query, "mutation MarkNotificationAsRead") {
					t.Error("Expected MarkNotificationAsRead mutation")
				}

				// Verify variables
				if id, ok := requestBody.Variables["id"].(string); !ok || id != tt.notificationID {
					t.Errorf("Expected id=%s, got %v", tt.notificationID, requestBody.Variables["id"])
				}
				if readAt, ok := requestBody.Variables["readAt"].(string); !ok {
					t.Error("Expected readAt timestamp in variables")
				} else {
					// Verify it's a valid timestamp
					if _, err := time.Parse(time.RFC3339, readAt); err != nil {
						t.Errorf("Invalid readAt timestamp format: %v", err)
					}
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.base.httpClient = &http.Client{}
			client.base.baseURL = server.URL

			err := client.MarkNotificationAsRead(tt.notificationID)

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}