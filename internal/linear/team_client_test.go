package linear

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestTeamClientListUsers tests the ListUsers functionality
func TestTeamClientListUsers(t *testing.T) {
	t.Run("list all users", func(t *testing.T) {
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

		users, err := client.Teams.ListUsers(nil)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(users) != 2 {
			t.Errorf("Expected 2 users, got %d", len(users))
		}

		// Verify user details
		if users[0].ID != "user-1" {
			t.Errorf("Expected first user ID to be 'user-1', got '%s'", users[0].ID)
		}
		if !users[1].Admin {
			t.Error("Expected second user to be admin")
		}
	})

	t.Run("list team members", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"team": {
					"members": {
						"nodes": [
							{
								"id": "user-1",
								"name": "Team Member",
								"displayName": "Member",
								"email": "member@example.com",
								"avatarUrl": "",
								"active": true,
								"admin": false,
								"createdAt": "2023-01-01T00:00:00Z"
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

		if len(users) != 1 {
			t.Errorf("Expected 1 user, got %d", len(users))
		}

		if users[0].Name != "Team Member" {
			t.Errorf("Expected user name to be 'Team Member', got '%s'", users[0].Name)
		}
	})

	t.Run("list active users only", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"users": {
					"nodes": [
						{
							"id": "user-1",
							"name": "Active User",
							"displayName": "Active",
							"email": "active@example.com",
							"avatarUrl": "",
							"active": true,
							"admin": false,
							"createdAt": "2023-01-01T00:00:00Z"
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

		// Verify all users are active
		for _, user := range users {
			if !user.Active {
				t.Errorf("Expected all users to be active, but user %s is not", user.ID)
			}
		}
	})

	t.Run("pagination", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"users": {
					"nodes": [
						{
							"id": "user-1",
							"name": "User 1",
							"displayName": "U1",
							"email": "user1@example.com",
							"avatarUrl": "",
							"active": true,
							"admin": false,
							"createdAt": "2023-01-01T00:00:00Z"
						}
					],
					"pageInfo": {
						"hasNextPage": true,
						"endCursor": "cursor-next"
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

		if result.EndCursor != "cursor-next" {
			t.Errorf("Expected endCursor to be 'cursor-next', got '%s'", result.EndCursor)
		}
	})

	t.Run("error handling", func(t *testing.T) {
		mockResponseBody := `{"errors": [{"message": "Not authorized"}]}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
				},
			},
		}

		_, err := client.Teams.ListUsers(nil)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !strings.Contains(err.Error(), "Not authorized") {
			t.Errorf("Expected error to contain 'Not authorized', got '%s'", err.Error())
		}
	})
}

// TestTeamClientGetTeams tests the GetTeams functionality
func TestTeamClientGetTeams(t *testing.T) {
	mockResponseBody := `{
		"data": {
			"teams": {
				"nodes": [
					{
						"id": "team-1",
						"name": "Engineering",
						"key": "ENG",
						"description": "Engineering team"
					},
					{
						"id": "team-2",
						"name": "Product",
						"key": "PROD",
						"description": "Product team"
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

	teams, err := client.Teams.GetTeams()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(teams) != 2 {
		t.Errorf("Expected 2 teams, got %d", len(teams))
	}

	// Verify team details
	if teams[0].ID != "team-1" {
		t.Errorf("Expected first team ID to be 'team-1', got '%s'", teams[0].ID)
	}
	if teams[0].Key != "ENG" {
		t.Errorf("Expected first team key to be 'ENG', got '%s'", teams[0].Key)
	}
	if teams[1].Name != "Product" {
		t.Errorf("Expected second team name to be 'Product', got '%s'", teams[1].Name)
	}
}

// TestTeamClientGetViewer tests the GetViewer functionality
func TestTeamClientGetViewer(t *testing.T) {
	mockResponseBody := `{
		"data": {
			"viewer": {
				"id": "viewer-id",
				"name": "Current User",
				"email": "current@example.com",
				"displayName": "Current",
				"avatarUrl": "https://example.com/avatar.jpg",
				"createdAt": "2023-01-01T00:00:00Z",
				"isMe": true
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

	viewer, err := client.Teams.GetViewer()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if viewer.ID != "viewer-id" {
		t.Errorf("Expected viewer ID to be 'viewer-id', got '%s'", viewer.ID)
	}
	if viewer.Email != "current@example.com" {
		t.Errorf("Expected viewer email to be 'current@example.com', got '%s'", viewer.Email)
	}
	if !viewer.IsMe {
		t.Error("Expected viewer.IsMe to be true")
	}
}