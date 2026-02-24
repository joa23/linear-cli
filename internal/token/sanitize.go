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
// Linear personal API keys (lin_api_*) must be sent without a Bearer prefix.
// OAuth access tokens are sent with "Bearer ".
func FormatAuthHeader(token string) string {
	sanitized := SanitizeToken(token)
	if sanitized == "" {
		return ""
	}

	// After sanitization, all spaces are removed, so "Bearer token123" becomes "Bearertoken123".
	// If caller already passed a bearer token, normalize it first.
	if strings.HasPrefix(sanitized, "Bearer") {
		sanitized = SanitizeToken(sanitized[6:]) // Skip "Bearer"
	}

	// Linear API keys must not include "Bearer ".
	if strings.HasPrefix(sanitized, "lin_api_") {
		return sanitized
	}

	// OAuth access tokens require "Bearer ".
	return "Bearer " + sanitized
}
