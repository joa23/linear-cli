package linear

import (
	"errors"
	"strings"
	"testing"
)

// TestInputValidation tests input validation for public methods
func TestInputValidation(t *testing.T) {
	client := NewClient("test-token")

	t.Run("ListAssignedIssues validation", func(t *testing.T) {
		// Test empty user ID
		_, err := client.ListAssignedIssues("")
		if err == nil {
			t.Error("Expected error for empty user ID, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}
		
		// Verify error details
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			if valErr.Field != "userID" {
				t.Errorf("Expected field 'userID', got '%s'", valErr.Field)
			}
			if valErr.Value != "" {
				t.Errorf("Expected empty value, got '%v'", valErr.Value)
			}
		}
	})

	t.Run("GetProject validation", func(t *testing.T) {
		// Test empty project ID
		_, err := client.GetProject("")
		if err == nil {
			t.Error("Expected error for empty project ID, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("GetIssueWithProjectContext validation", func(t *testing.T) {
		// Test empty issue ID
		_, err := client.GetIssueWithProjectContext("")
		if err == nil {
			t.Error("Expected error for empty issue ID, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("AssignIssue validation", func(t *testing.T) {
		// Test empty issue ID
		err := client.AssignIssue("", "550e8400-e29b-41d4-a716-446655440006")
		if err == nil {
			t.Error("Expected error for empty issue ID, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Note: empty assigneeID is valid (unassigns the issue)
	})

	t.Run("CreateComment validation", func(t *testing.T) {
		// Test empty issue ID
		_, err := client.CreateComment("", "comment body")
		if err == nil {
			t.Error("Expected error for empty issue ID, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Test empty body (should be allowed)
		// Linear allows empty comments, so this should not error
	})

	t.Run("AddReaction validation", func(t *testing.T) {
		// Test empty subject ID
		err := client.AddReaction("", "👍")
		if err == nil {
			t.Error("Expected error for empty subject ID, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Test empty emoji
		err = client.AddReaction("550e8400-e29b-41d4-a716-446655440004", "")
		if err == nil {
			t.Error("Expected error for empty emoji, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Test invalid emoji (not an emoji character)
		err = client.AddReaction("550e8400-e29b-41d4-a716-446655440004", "not-an-emoji")
		if err == nil {
			t.Error("Expected error for invalid emoji, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("UpdateIssueMetadataKey validation", func(t *testing.T) {
		// Test empty issue ID
		err := client.UpdateIssueMetadataKey("", "key", "value")
		if err == nil {
			t.Error("Expected error for empty issue ID, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Test empty key
		err = client.UpdateIssueMetadataKey("550e8400-e29b-41d4-a716-446655440004", "", "value")
		if err == nil {
			t.Error("Expected error for empty key, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Test invalid key format (contains spaces)
		err = client.UpdateIssueMetadataKey("550e8400-e29b-41d4-a716-446655440004", "invalid key", "value")
		if err == nil {
			t.Error("Expected error for invalid key format, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("GetNotifications validation", func(t *testing.T) {
		// Test negative limit
		_, err := client.GetNotifications(false, -1)
		if err == nil {
			t.Error("Expected error for negative limit, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Test zero limit
		_, err = client.GetNotifications(false, 0)
		if err == nil {
			t.Error("Expected error for zero limit, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Test excessive limit
		_, err = client.GetNotifications(false, 1001)
		if err == nil {
			t.Error("Expected error for excessive limit, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("CreateIssue validation", func(t *testing.T) {
		// Test empty title
		_, err := client.Issues.CreateIssue("", "Description", "550e8400-e29b-41d4-a716-446655440003")
		if err == nil {
			t.Error("Expected error for empty title, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Test empty team ID
		_, err = client.Issues.CreateIssue("Test Issue", "Description", "")
		if err == nil {
			t.Error("Expected error for empty team ID, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Test title too long (Linear limit is around 255 characters)
		longTitle := strings.Repeat("a", 256)
		_, err = client.Issues.CreateIssue(longTitle, "Description", "550e8400-e29b-41d4-a716-446655440003")
		if err == nil {
			t.Error("Expected error for title too long, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}

		// Test description too long (Linear limit is around 100k characters)
		longDesc := strings.Repeat("a", 100001)
		_, err = client.Issues.CreateIssue("Test Issue", longDesc, "550e8400-e29b-41d4-a716-446655440003")
		if err == nil {
			t.Error("Expected error for description too long, got nil")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T: %v", err, err)
		}
	})
}

// TestValidationHelpers tests validation helper functions
func TestValidationHelpers(t *testing.T) {
	t.Run("isValidMetadataKey", func(t *testing.T) {
		tests := []struct {
			key   string
			valid bool
		}{
			{"validKey", true},
			{"valid_key", true},
			{"valid-key", true},
			{"validKey123", true},
			{"", false},
			{"invalid key", false}, // spaces not allowed
			{"invalid.key", false}, // dots not allowed
			{"invalid/key", false}, // slashes not allowed
			{"invalid\\key", false}, // backslashes not allowed
			{"123key", false}, // can't start with number
			{"-key", false}, // can't start with dash
			{"_key", true}, // can start with underscore
		}

		for _, tt := range tests {
			t.Run(tt.key, func(t *testing.T) {
				result := isValidMetadataKey(tt.key)
				if result != tt.valid {
					t.Errorf("isValidMetadataKey(%q) = %v, want %v", tt.key, result, tt.valid)
				}
			})
		}
	})

	t.Run("isValidEmoji", func(t *testing.T) {
		tests := []struct {
			emoji string
			valid bool
		}{
			{"👍", true},
			{"✅", true},
			{"🎉", true},
			{"👀", true},
			{"", false},
			{"not-emoji", false},
			{"a", false},
			{"123", false},
			{"👍👍", false}, // multiple emojis not allowed
		}

		for _, tt := range tests {
			t.Run(tt.emoji, func(t *testing.T) {
				result := isValidEmoji(tt.emoji)
				if result != tt.valid {
					t.Errorf("isValidEmoji(%q) = %v, want %v", tt.emoji, result, tt.valid)
				}
			})
		}
	})
}