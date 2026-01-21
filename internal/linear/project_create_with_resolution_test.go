package linear

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestCreateProject_WithTeamName(t *testing.T) {
	callCount := 0
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query string `json:"query"`
		}
		json.Unmarshal(body, &req)

		if callCount == 1 {
			// First call: list teams to resolve team name
			mockResponse := `{
				"data": {
					"teams": {
						"nodes": [
							{
								"id": "team-123",
								"name": "Engineering",
								"key": "ENG"
							}
						]
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		} else {
			// Second call: create project with resolved team ID
			mockResponse := `{
				"data": {
					"projectCreate": {
						"success": true,
						"project": {
							"id": "project-123",
							"name": "Test Project",
							"description": "Test description",
							"state": "backlog",
							"createdAt": "2024-01-01T00:00:00Z",
							"updatedAt": "2024-01-01T00:00:00Z",
							"issues": {
								"nodes": []
							}
						}
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		}
	})
	defer server.Close()

	resolver := NewResolver(client)
	client.resolver = resolver

	// Test: CreateProject with team name should work
	project, err := client.CreateProject("Test Project", "Test description", "Engineering")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if project.Name != "Test Project" {
		t.Errorf("Expected name 'Test Project', got %s", project.Name)
	}

	if project.ID != "project-123" {
		t.Errorf("Expected ID 'project-123', got %s", project.ID)
	}
}

func TestCreateProject_WithTeamKey(t *testing.T) {
	callCount := 0
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// First call: list teams to resolve team key
			mockResponse := `{
				"data": {
					"teams": {
						"nodes": [
							{
								"id": "team-123",
								"name": "Engineering",
								"key": "ENG"
							}
						]
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		} else {
			// Second call: create project
			mockResponse := `{
				"data": {
					"projectCreate": {
						"success": true,
						"project": {
							"id": "project-123",
							"name": "Test Project",
							"description": "Test description",
							"state": "backlog",
							"createdAt": "2024-01-01T00:00:00Z",
							"updatedAt": "2024-01-01T00:00:00Z",
							"issues": {
								"nodes": []
							}
						}
					}
				}
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		}
	})
	defer server.Close()

	resolver := NewResolver(client)
	client.resolver = resolver

	// Test: CreateProject with team key should work
	project, err := client.CreateProject("Test Project", "Test description", "ENG")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if project.Name != "Test Project" {
		t.Errorf("Expected name 'Test Project', got %s", project.Name)
	}
}

func TestCreateProject_WithUUID(t *testing.T) {
	// Test that UUIDs still work (backward compatibility)
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{
			"data": {
				"projectCreate": {
					"success": true,
					"project": {
						"id": "project-123",
						"name": "Test Project",
						"description": "Test description",
						"state": "backlog",
						"createdAt": "2024-01-01T00:00:00Z",
						"updatedAt": "2024-01-01T00:00:00Z",
						"issues": {
							"nodes": []
						}
					}
				}
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	})
	defer server.Close()

	resolver := NewResolver(client)
	client.resolver = resolver

	// Should work with UUID format (no resolution needed)
	// Use proper UUID format: 8-4-4-4-12
	project, err := client.CreateProject("Test Project", "Test description", "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if project.Name != "Test Project" {
		t.Errorf("Expected name 'Test Project', got %s", project.Name)
	}
}

func TestCreateProject_WithInvalidTeam(t *testing.T) {
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Return empty teams list
		mockResponse := `{
			"data": {
				"teams": {
					"nodes": []
				}
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	})
	defer server.Close()

	resolver := NewResolver(client)
	client.resolver = resolver

	_, err := client.CreateProject("Test Project", "Test description", "NonExistentTeam")
	if err == nil {
		t.Fatal("Expected error for non-existent team, got nil")
	}

	if !IsNotFoundError(err) {
		t.Errorf("Expected NotFoundError, got: %T", err)
	}
}
