package cli

import "fmt"

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
