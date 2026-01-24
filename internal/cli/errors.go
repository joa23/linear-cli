package cli

import "fmt"

// Standard error messages
const (
	// ErrTeamRequired is returned when a team is required but not provided
	ErrTeamRequired = "--team is required (or run 'linear init' to set a default)"
)

// ResolutionError creates a standard error message when auto-resolution fails
func ResolutionError(resourceType, value, specificCommand string) error {
	return fmt.Errorf(
		"could not resolve %s '%s'\n\n"+
			"To see valid %ss:\n"+
			"  • Run 'linear onboard' for a complete summary\n"+
			"  • Run '%s' for detailed %s information",
		resourceType, value,
		resourceType,
		specificCommand, resourceType,
	)
}
