package token

import (
	"testing"
)

func TestSanitizeToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean token unchanged",
			input:    "lin_api_validtoken123",
			expected: "lin_api_validtoken123",
		},
		{
			name:     "token with newline",
			input:    "lin_api_token123\n",
			expected: "lin_api_token123",
		},
		{
			name:     "token with carriage return",
			input:    "lin_api_token123\r",
			expected: "lin_api_token123",
		},
		{
			name:     "token with tab",
			input:    "lin_api_token\t123",
			expected: "lin_api_token123",
		},
		{
			name:     "token with multiple whitespace types",
			input:    "\r\nlin_api_token123\n\t",
			expected: "lin_api_token123",
		},
		{
			name:     "token with spaces",
			input:    " lin_api_token123 ",
			expected: "lin_api_token123",
		},
		{
			name:     "token with embedded newline",
			input:    "lin_api\ntoken123",
			expected: "lin_apitoken123",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "\n\r\t ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeToken(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeToken(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{
			name:      "valid token",
			token:     "lin_api_validtoken123",
			wantError: false,
		},
		{
			name:      "empty token",
			token:     "",
			wantError: true,
		},
		{
			name:      "token with newline",
			token:     "lin_api_token\n123",
			wantError: true,
		},
		{
			name:      "token with tab",
			token:     "lin_api_token\t123",
			wantError: true,
		},
		{
			name:      "token with space",
			token:     "lin_api token123",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToken(tt.token)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateToken(%q) error = %v, wantError %v", tt.token, err, tt.wantError)
			}
		})
	}
}

func TestFormatAuthHeader(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "clean token gets Bearer prefix",
			token:    "lin_api_token123",
			expected: "Bearer lin_api_token123",
		},
		{
			name:     "token with existing Bearer prefix unchanged",
			token:    "Bearer lin_api_token123",
			expected: "Bearer lin_api_token123",
		},
		{
			name:     "token with newline gets sanitized and Bearer prefix",
			token:    "lin_api_token123\n",
			expected: "Bearer lin_api_token123",
		},
		{
			name:     "token with existing Bearer and newline gets sanitized",
			token:    "Bearer lin_api_token123\n",
			expected: "Bearer lin_api_token123",
		},
		{
			name:     "token with whitespace gets sanitized",
			token:    " lin_api_token123 ",
			expected: "Bearer lin_api_token123",
		},
		{
			name:     "token with Bearer and extra whitespace",
			token:    " Bearer  lin_api_token123 ",
			expected: "Bearer lin_api_token123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAuthHeader(tt.token)
			if result != tt.expected {
				t.Errorf("FormatAuthHeader(%q) = %q, want %q", tt.token, result, tt.expected)
			}
		})
	}
}
