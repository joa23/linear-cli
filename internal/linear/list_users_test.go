package linear

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestListUsersBasic tests basic user listing without filters
func TestListUsersBasic(t *testing.T) {
	mockResponseBody := `{
		"data": {
			"users": {
				"nodes": [
					{
						"id": "user-1",
						"name": "John Doe",
						"displayName": "John",
						"email": "john@example.com",
						"avatarUrl": "https://example.com/avatar1.jpg",
						"active": true,
						"admin": false,
						"createdAt": "2023-01-01T00:00:00Z"
					},
					{
						"id": "user-2",
						"name": "Jane Smith",
						"displayName": "Jane",
						"email": "jane@example.com",
						"avatarUrl": "https://example.com/avatar2.jpg",
						"active": true,
						"admin": true,
						"createdAt": "2023-01-02T00:00:00Z"
					},
					{
						"id": "user-3",
						"name": "Bob Wilson",
						"displayName": "Bob",
						"email": "bob@example.com",
						"avatarUrl": "",
						"active": false,
						"admin": false,
						"createdAt": "2023-01-03T00:00:00Z"
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

	users, err := client.Teams.ListUsers(nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}

	// Verify first user details
	if users[0].ID != "user-1" {
		t.Errorf("Expected first user ID to be 'user-1', got '%s'", users[0].ID)
	}
	if users[0].Name != "John Doe" {
		t.Errorf("Expected first user name to be 'John Doe', got '%s'", users[0].Name)
	}
	if !users[0].Active {
		t.Error("Expected first user to be active")
	}
	if users[0].Admin {
		t.Error("Expected first user not to be admin")
	}

	// Verify inactive user
	if users[2].Active {
		t.Error("Expected third user to be inactive")
	}

	// Verify admin user
	if !users[1].Admin {
		t.Error("Expected second user to be admin")
	}
}

// TestListUsersWithTeamFilter tests listing users filtered by team
func TestListUsersWithTeamFilter(t *testing.T) {
	mockResponseBody := `{
		"data": {
			"team": {
				"members": {
					"nodes": [
						{
							"id": "user-1",
							"name": "John Doe",
							"displayName": "John",
							"email": "john@example.com",
							"avatarUrl": "https://example.com/avatar1.jpg",
							"active": true,
							"admin": false,
							"createdAt": "2023-01-01T00:00:00Z"
						},
						{
							"id": "user-2",
							"name": "Jane Smith",
							"displayName": "Jane",
							"email": "jane@example.com",
							"avatarUrl": "https://example.com/avatar2.jpg",
							"active": true,
							"admin": true,
							"createdAt": "2023-01-02T00:00:00Z"
						}
					]
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

	filter := &UserFilter{
		TeamID: "team-123",
	}

	users, err := client.Teams.ListUsers(filter)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users in team, got %d", len(users))
	}

	// Verify team members
	if users[0].ID != "user-1" {
		t.Errorf("Expected first user ID to be 'user-1', got '%s'", users[0].ID)
	}
	if users[1].ID != "user-2" {
		t.Errorf("Expected second user ID to be 'user-2', got '%s'", users[1].ID)
	}
}

// TestListUsersActiveOnly tests listing only active users
func TestListUsersActiveOnly(t *testing.T) {
	mockResponseBody := `{
		"data": {
			"users": {
				"nodes": [
					{
						"id": "user-1",
						"name": "John Doe",
						"displayName": "John",
						"email": "john@example.com",
						"avatarUrl": "https://example.com/avatar1.jpg",
						"active": true,
						"admin": false,
						"createdAt": "2023-01-01T00:00:00Z"
					},
					{
						"id": "user-2",
						"name": "Jane Smith",
						"displayName": "Jane",
						"email": "jane@example.com",
						"avatarUrl": "https://example.com/avatar2.jpg",
						"active": true,
						"admin": true,
						"createdAt": "2023-01-02T00:00:00Z"
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

	activeOnly := true
	filter := &UserFilter{
		ActiveOnly: &activeOnly,
	}

	users, err := client.Teams.ListUsers(filter)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 active users, got %d", len(users))
	}

	// Verify all users are active
	for _, user := range users {
		if !user.Active {
			t.Errorf("Expected all users to be active, but user %s is not", user.ID)
		}
	}
}

// TestListUsersWithPagination tests pagination support
func TestListUsersWithPagination(t *testing.T) {
	mockResponseBody := `{
		"data": {
			"users": {
				"nodes": [
					{
						"id": "user-1",
						"name": "John Doe",
						"displayName": "John",
						"email": "john@example.com",
						"avatarUrl": "https://example.com/avatar1.jpg",
						"active": true,
						"admin": false,
						"createdAt": "2023-01-01T00:00:00Z"
					}
				],
				"pageInfo": {
					"hasNextPage": true,
					"endCursor": "cursor-123"
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

	filter := &UserFilter{
		First: 1,
	}

	result, err := client.Teams.ListUsersWithPagination(filter)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.Users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(result.Users))
	}

	if !result.HasNextPage {
		t.Error("Expected hasNextPage to be true")
	}

	if result.EndCursor != "cursor-123" {
		t.Errorf("Expected endCursor to be 'cursor-123', got '%s'", result.EndCursor)
	}
}

// TestListUsersErrorHandling tests error handling scenarios
func TestListUsersErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		statusCode     int
		expectedErrMsg string
	}{
		{
			name:           "GraphQL error",
			statusCode:     http.StatusOK,
			responseBody:   `{"errors": [{"message": "Not authorized"}]}`,
			expectedErrMsg: "Not authorized",
		},
		{
			name:           "Network error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "",
			expectedErrMsg: "500",
		},
		{
			name:           "Invalid JSON response",
			statusCode:     http.StatusOK,
			responseBody:   `{invalid json}`,
			expectedErrMsg: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("test-token")
			client.base.httpClient = &http.Client{
				Transport: &mockTransport{
					response: &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
					},
				},
			}

			_, err := client.Teams.ListUsers(nil)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if !containsString(err.Error(), tt.expectedErrMsg) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedErrMsg, err.Error())
			}
		})
	}
}

// containsString checks if the string contains the substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}