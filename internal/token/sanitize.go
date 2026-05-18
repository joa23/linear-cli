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
// Linear API keys (lin_api_*) are sent directly without a prefix.
// OAuth tokens use the "Bearer " prefix.
func FormatAuthHeader(token string) string {
	sanitized := SanitizeToken(token)
	if sanitized == "" {
		return ""
	}

	// Linear API keys must NOT use the Bearer prefix.
	// The API returns a 400 error: "It looks like you're trying to use an API key as a Bearer token"
	if strings.HasPrefix(sanitized, "lin_api_") {
		return sanitized
	}

	// After sanitization, all spaces are removed, so "Bearer token123" becomes "Bearertoken123".
	// Check if already has Bearer prefix (without space since sanitization removes it).
	if strings.HasPrefix(sanitized, "Bearer") {
		tokenPart := sanitized[6:] // Skip "Bearer"
		return "Bearer " + tokenPart
	}

	// OAuth access tokens require "Bearer ".
	return "Bearer " + sanitized
}
