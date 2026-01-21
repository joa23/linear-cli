package linear

import (
	"net/http"
	"testing"
)

func TestResolveUser_ByEmail(t *testing.T) {
	// Setup mock server
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Mock response for user search by email
		mockResponse := `{
			"data": {
				"users": {
					"nodes": [
						{
							"id": "user-123",
							"name": "John Doe",
							"email": "john@company.com",
							"displayName": "John"
						}
					]
				}
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	})
	defer server.Close()

	resolver := &Resolver{
		client: client,
		cache:  newResolverCache(defaultCacheTTL),
	}

	userID, err := resolver.ResolveUser("john@company.com")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if userID != "user-123" {
		t.Errorf("Expected user-123, got %s", userID)
	}

	// Second call should use cache (no additional API call)
	userID2, err := resolver.ResolveUser("john@company.com")
	if err != nil {
		t.Fatalf("Expected no error on cached lookup, got: %v", err)
	}

	if userID2 != "user-123" {
		t.Errorf("Expected cached user-123, got %s", userID2)
	}
}

func TestResolveUser_ByDisplayName(t *testing.T) {
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Mock response for user search by name
		mockResponse := `{
			"data": {
				"users": {
					"nodes": [
						{
							"id": "user-123",
							"name": "John Doe",
							"email": "john@company.com",
							"displayName": "John Doe"
						}
					]
				}
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	})
	defer server.Close()

	resolver := &Resolver{
		client: client,
		cache:  newResolverCache(defaultCacheTTL),
	}

	userID, err := resolver.ResolveUser("John Doe")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if userID != "user-123" {
		t.Errorf("Expected user-123, got %s", userID)
	}
}

func TestResolveUser_AmbiguousName(t *testing.T) {
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Mock response with multiple matching users
		mockResponse := `{
			"data": {
				"users": {
					"nodes": [
						{
							"id": "user-123",
							"name": "John Doe",
							"email": "john.doe@company.com",
							"displayName": "John"
						},
						{
							"id": "user-456",
							"name": "John Smith",
							"email": "john.smith@company.com",
							"displayName": "John"
						}
					]
				}
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	})
	defer server.Close()

	resolver := &Resolver{
		client: client,
		cache:  newResolverCache(defaultCacheTTL),
	}

	_, err := resolver.ResolveUser("John")
	if err == nil {
		t.Fatal("Expected error for ambiguous name, got nil")
	}

	// Check that error contains suggestions
	errMsg := err.Error()
	if !stringContainsSubstr(errMsg, "john.doe@company.com") || !stringContainsSubstr(errMsg, "john.smith@company.com") {
		t.Errorf("Expected error to include email suggestions, got: %s", errMsg)
	}
}

func TestResolveUser_NotFound(t *testing.T) {
	server, client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Mock response with no users
		mockResponse := `{
			"data": {
				"users": {
					"nodes": []
				}
			}
		}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	})
	defer server.Close()

	resolver := &Resolver{
		client: client,
		cache:  newResolverCache(defaultCacheTTL),
	}

	_, err := resolver.ResolveUser("nonexistent@company.com")
	if err == nil {
		t.Fatal("Expected error for non-existent user, got nil")
	}

	// The error should indicate user not found
	// Note: GetUserByEmail returns an error when no users match, which gets wrapped
	errMsg := err.Error()
	if !stringContainsSubstr(errMsg, "nonexistent") || !stringContainsSubstr(errMsg, "user") {
		t.Errorf("Expected error about non-existent user, got: %s", errMsg)
	}
}

func TestResolveUser_EmptyInput(t *testing.T) {
	resolver := &Resolver{
		cache: newResolverCache(defaultCacheTTL),
	}

	_, err := resolver.ResolveUser("")
	if err == nil {
		t.Fatal("Expected error for empty input, got nil")
	}

	// Check that it's a ValidationError
	if !IsValidationError(err) {
		t.Errorf("Expected ValidationError, got: %T", err)
	}
}

// Helper function for string contains check
func stringContainsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
