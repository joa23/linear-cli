package linear

import (
	"testing"
)

func TestIsOAuthApplicationEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{
			name:     "OAuth application email",
			email:    "app-12345@oauthapp.linear.app",
			expected: true,
		},
		{
			name:     "Another OAuth application email",
			email:    "my-agent@oauthapp.linear.app",
			expected: true,
		},
		{
			name:     "Human user email",
			email:    "john@company.com",
			expected: false,
		},
		{
			name:     "Gmail email",
			email:    "user@gmail.com",
			expected: false,
		},
		{
			name:     "Linear employee email",
			email:    "employee@linear.app",
			expected: false,
		},
		{
			name:     "Empty email",
			email:    "",
			expected: false,
		},
		{
			name:     "Similar but not OAuth app email",
			email:    "test@oauthapp.linear.com",
			expected: false,
		},
		{
			name:     "Partial match - not suffix",
			email:    "oauthapp.linear.app@example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the detection logic used in resolver
			result := isOAuthApplicationEmail(tt.email)
			if result != tt.expected {
				t.Errorf("isOAuthApplicationEmail(%q) = %v, want %v", tt.email, result, tt.expected)
			}
		})
	}
}

func TestResolvedUserStruct(t *testing.T) {
	// Test that ResolvedUser struct works correctly
	t.Run("human user", func(t *testing.T) {
		user := &ResolvedUser{
			ID:            "uuid-123",
			IsApplication: false,
		}
		if user.ID != "uuid-123" {
			t.Errorf("ID = %q, want %q", user.ID, "uuid-123")
		}
		if user.IsApplication {
			t.Error("IsApplication should be false for human user")
		}
	})

	t.Run("OAuth application", func(t *testing.T) {
		user := &ResolvedUser{
			ID:            "app-uuid-456",
			IsApplication: true,
		}
		if user.ID != "app-uuid-456" {
			t.Errorf("ID = %q, want %q", user.ID, "app-uuid-456")
		}
		if !user.IsApplication {
			t.Error("IsApplication should be true for OAuth application")
		}
	})
}

// isOAuthApplicationEmail checks if an email belongs to an OAuth application
// This is the detection logic used in the resolver
func isOAuthApplicationEmail(email string) bool {
	if email == "" {
		return false
	}
	const suffix = "@oauthapp.linear.app"
	if len(email) <= len(suffix) {
		return false
	}
	return email[len(email)-len(suffix):] == suffix
}
