//go:build integration

package linear

import (
	"os"
	"testing"
)

// TestListUsersIntegration tests user listing functionality with real API
func TestListUsersIntegration(t *testing.T) {
	// Skip if no token is available
	if os.Getenv("LINEAR_API_TOKEN") == "" && os.Getenv("LINEAR_TOKEN") == "" {
		t.Skip("Skipping integration test: No LINEAR_API_TOKEN or LINEAR_TOKEN environment variable")
	}

	// Skip if integration tests are disabled
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test: SKIP_INTEGRATION_TESTS is set")
	}

	client := NewDefaultClient()
	if client.apiToken == "" {
		t.Skip("Skipping integration test: No valid token found")
	}

	t.Run("list all users", func(t *testing.T) {
		users, err := client.Teams.ListUsers(nil)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		if len(users) == 0 {
			t.Log("No users found in workspace")
		} else {
			t.Logf("Found %d users in workspace", len(users))
			
			// Log first user details
			firstUser := users[0]
			t.Logf("First user: ID=%s, Name=%s, Active=%v, Admin=%v", 
				firstUser.ID, firstUser.Name, firstUser.Active, firstUser.Admin)
		}
	})

	t.Run("list users with pagination", func(t *testing.T) {
		filter := &UserFilter{
			First: 2, // Get only 2 users at a time
		}

		result, err := client.Teams.ListUsersWithPagination(filter)
		if err != nil {
			t.Fatalf("Failed to list users with pagination: %v", err)
		}

		t.Logf("Retrieved %d users", len(result.Users))
		t.Logf("Has next page: %v", result.HasNextPage)
		if result.HasNextPage {
			t.Logf("End cursor: %s", result.EndCursor)
		}
	})

	t.Run("list active users only", func(t *testing.T) {
		activeOnly := true
		filter := &UserFilter{
			ActiveOnly: &activeOnly,
		}

		users, err := client.Teams.ListUsers(filter)
		if err != nil {
			t.Fatalf("Failed to list active users: %v", err)
		}

		// Verify all returned users are active
		for _, user := range users {
			if !user.Active {
				t.Errorf("Expected all users to be active, but user %s is not active", user.ID)
			}
		}

		t.Logf("Found %d active users", len(users))
	})

	t.Run("list team members", func(t *testing.T) {
		// First get teams to test team member listing
		teams, err := client.Teams.GetTeams()
		if err != nil {
			t.Fatalf("Failed to get teams: %v", err)
		}

		if len(teams) == 0 {
			t.Skip("No teams found to test member listing")
		}

		// Test listing members of the first team
		firstTeam := teams[0]
		filter := &UserFilter{
			TeamID: firstTeam.ID,
		}

		users, err := client.Teams.ListUsers(filter)
		if err != nil {
			t.Fatalf("Failed to list team members: %v", err)
		}

		t.Logf("Team '%s' has %d members", firstTeam.Name, len(users))
	})
}

// TestListUsersRealAPIWithStoredToken tests with stored token from OAuth flow
func TestListUsersRealAPIWithStoredToken(t *testing.T) {
	// Skip if integration tests are disabled
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test: SKIP_INTEGRATION_TESTS is set")
	}

	// Use the default client which tries stored token first
	client := NewDefaultClient()
	if client.apiToken == "" {
		t.Skip("No stored token found, skipping test")
	}

	// Test basic functionality
	users, err := client.Teams.ListUsers(nil)
	if err != nil {
		t.Fatalf("Failed to list users with stored token: %v", err)
	}

	t.Logf("Successfully listed %d users with stored token", len(users))
}