package linear

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestTeamClientGetUser tests the GetUser functionality
func TestTeamClientGetUser(t *testing.T) {
	t.Run("get user by ID with basic info", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"user": {
					"id": "user-123",
					"name": "John Doe",
					"displayName": "John",
					"email": "john@example.com",
					"avatarUrl": "https://example.com/avatar.jpg",
					"active": true,
					"admin": false,
					"createdAt": "2023-01-01T00:00:00Z",
					"isMe": false
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

		user, err := client.Teams.GetUser("user-123")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if user.ID != "user-123" {
			t.Errorf("Expected user ID to be 'user-123', got '%s'", user.ID)
		}
		if user.Name != "John Doe" {
			t.Errorf("Expected user name to be 'John Doe', got '%s'", user.Name)
		}
		if user.Email != "john@example.com" {
			t.Errorf("Expected user email to be 'john@example.com', got '%s'", user.Email)
		}
		if !user.Active {
			t.Error("Expected user to be active")
		}
	})

	t.Run("get user by ID with team memberships", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"user": {
					"id": "user-123",
					"name": "John Doe",
					"displayName": "John",
					"email": "john@example.com",
					"avatarUrl": "https://example.com/avatar.jpg",
					"active": true,
					"admin": false,
					"createdAt": "2023-01-01T00:00:00Z",
					"isMe": false,
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

		user, err := client.Teams.GetUser("user-123")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(user.Teams) != 2 {
			t.Errorf("Expected 2 teams, got %d", len(user.Teams))
		}
		if user.Teams[0].Key != "ENG" {
			t.Errorf("Expected first team key to be 'ENG', got '%s'", user.Teams[0].Key)
		}
		if user.Teams[1].Name != "Product" {
			t.Errorf("Expected second team name to be 'Product', got '%s'", user.Teams[1].Name)
		}
	})


	t.Run("user not found", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"user": null
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

		user, err := client.Teams.GetUser("non-existent-user")
		if err == nil {
			t.Fatal("Expected error for non-existent user, got nil")
		}
		if user != nil {
			t.Error("Expected nil user for non-existent user")
		}
		if !strings.Contains(err.Error(), "user not found") {
			t.Errorf("Expected error to contain 'user not found', got '%s'", err.Error())
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

		_, err := client.Teams.GetUser("user-123")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !strings.Contains(err.Error(), "Not authorized") {
			t.Errorf("Expected error to contain 'Not authorized', got '%s'", err.Error())
		}
	})
}

// TestTeamClientGetUserByEmail tests the GetUserByEmail functionality
func TestTeamClientGetUserByEmail(t *testing.T) {
	t.Run("get user by email", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"users": {
					"nodes": [
						{
							"id": "user-123",
							"name": "John Doe",
							"displayName": "John",
							"email": "john@example.com",
							"avatarUrl": "https://example.com/avatar.jpg",
							"active": true,
							"admin": false,
							"createdAt": "2023-01-01T00:00:00Z",
							"isMe": false
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

		user, err := client.Teams.GetUserByEmail("john@example.com")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if user.ID != "user-123" {
			t.Errorf("Expected user ID to be 'user-123', got '%s'", user.ID)
		}
		if user.Email != "john@example.com" {
			t.Errorf("Expected user email to be 'john@example.com', got '%s'", user.Email)
		}
	})

	t.Run("get user by email with team memberships", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"users": {
					"nodes": [
						{
							"id": "user-123",
							"name": "John Doe",
							"displayName": "John",
							"email": "john@example.com",
							"avatarUrl": "https://example.com/avatar.jpg",
							"active": true,
							"admin": false,
							"createdAt": "2023-01-01T00:00:00Z",
							"isMe": false,
							"teams": {
								"nodes": [
									{
										"id": "team-1",
										"name": "Engineering",
										"key": "ENG",
										"description": "Engineering team"
									}
								]
							}
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

		user, err := client.Teams.GetUserByEmail("john@example.com")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(user.Teams) != 1 {
			t.Errorf("Expected 1 team, got %d", len(user.Teams))
		}
		if user.Teams[0].Key != "ENG" {
			t.Errorf("Expected team key to be 'ENG', got '%s'", user.Teams[0].Key)
		}
	})

	t.Run("user not found by email", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"users": {
					"nodes": []
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

		user, err := client.Teams.GetUserByEmail("nonexistent@example.com")
		if err == nil {
			t.Fatal("Expected error for non-existent email, got nil")
		}
		if user != nil {
			t.Error("Expected nil user for non-existent email")
		}
		if !strings.Contains(err.Error(), "user not found") {
			t.Errorf("Expected error to contain 'user not found', got '%s'", err.Error())
		}
	})

	t.Run("multiple users with same email", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"users": {
					"nodes": [
						{
							"id": "user-123",
							"name": "John Doe",
							"email": "john@example.com"
						},
						{
							"id": "user-456",
							"name": "John Doe Jr",
							"email": "john@example.com"
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

		_, err := client.Teams.GetUserByEmail("john@example.com")
		if err == nil {
			t.Fatal("Expected error for multiple users, got nil")
		}
		if !strings.Contains(err.Error(), "multiple users") {
			t.Errorf("Expected error to contain 'multiple users', got '%s'", err.Error())
		}
	})
}