package linear

import (
	"regexp"
	"strings"
)

// Issue identifier pattern: uppercase letters, hyphen, digits
// Examples: CEN-123, ABC-1, ENGINEERING-9999
var issueIdentifierRegex = regexp.MustCompile(`^[A-Z]+-\d+$`)

// Email pattern: basic email validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// isIssueIdentifier checks if a string matches Linear's issue identifier format
// Valid format: UPPERCASE-NUMBER (e.g., CEN-123, ABC-1)
//
// Why: Issue identifiers are the human-readable way to reference issues.
// We need to detect them to know when to resolve them to UUIDs internally.
func isIssueIdentifier(s string) bool {
	return issueIdentifierRegex.MatchString(s)
}

// parseIssueIdentifier extracts the team key and issue number from an identifier
// Returns the team key (e.g., "CEN"), the number (e.g., "123"), and any error
//
// Why: We need to extract components for searching or validation purposes.
func parseIssueIdentifier(s string) (teamKey string, number string, error error) {
	if !isIssueIdentifier(s) {
		return "", "", &ValidationError{
			Field:  "identifier",
			Value:  s,
			Reason: "must be in format TEAM-NUMBER (e.g., CEN-123)",
		}
	}

	// Split on hyphen
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return "", "", &ValidationError{
			Field:  "identifier",
			Value:  s,
			Reason: "invalid format",
		}
	}

	return parts[0], parts[1], nil
}

// isEmail checks if a string is a valid email address
//
// Why: Users can be identified by email, so we need to detect email format
// to route to the correct resolution method.
func isEmail(s string) bool {
	return emailRegex.MatchString(s)
}

// isUUID checks if a string is a UUID format
// UUIDs: 8-4-4-4-12 format (e.g., "550e8400-e29b-41d4-a716-446655440000")
//
// Why: We need to detect UUIDs to avoid unnecessary resolution lookups.
// UUIDs can be used directly without resolution.
func isUUID(s string) bool {
	return len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}
