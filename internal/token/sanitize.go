package token

import (
	"fmt"
	"strings"
	"unicode"
)

// SanitizeToken removes invalid HTTP header characters from a token.
// Removes all whitespace characters: spaces, tabs, newlines, carriage returns.
// This prevents "invalid header field value" errors when setting Authorization headers.
func SanitizeToken(token string) string {
	// Remove leading/trailing whitespace first
	token = strings.TrimSpace(token)

	// Remove all whitespace characters that could break HTTP headers
	token = strings.ReplaceAll(token, "\n", "")
	token = strings.ReplaceAll(token, "\r", "")
	token = strings.ReplaceAll(token, "\t", "")
	token = strings.ReplaceAll(token, " ", "")

	// Remove any remaining control characters
	token = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1 // Drop control characters
		}
		return r
	}, token)

	return token
}

// ValidateToken checks if a token contains invalid characters.
// Returns an error if the token contains characters that would cause HTTP header issues.
func ValidateToken(token string) error {
	if token == "" {
		return fmt.Errorf("token is empty")
	}

	// Check for invalid characters after sanitization
	sanitized := SanitizeToken(token)
	if len(sanitized) != len(token) {
		return fmt.Errorf("token contains invalid characters (whitespace or control characters)")
	}

	return nil
}

// FormatAuthHeader formats a token for use in an Authorization header.
// Ensures the token is sanitized and has the "Bearer " prefix.
// Handles tokens that already have the Bearer prefix to avoid duplication.
func FormatAuthHeader(token string) string {
	sanitized := SanitizeToken(token)

	// After sanitization, all spaces are removed, so "Bearer token123" becomes "Bearertoken123"
	// Check if already has Bearer prefix (without space since sanitization removes it)
	if strings.HasPrefix(sanitized, "Bearer") {
		// Extract the token part after "Bearer"
		tokenPart := sanitized[6:] // Skip "Bearer"
		return "Bearer " + tokenPart
	}

	return "Bearer " + sanitized
}
