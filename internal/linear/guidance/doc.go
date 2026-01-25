// Package guidance provides AI agent-friendly error messages with actionable guidance.
//
// This package wraps errors with contextual help, examples, and tool suggestions
// to help AI agents (and humans) resolve issues more effectively.
//
// # Error Enhancement
//
// The package provides functions to enhance generic errors with helpful context:
//
//	err := guidance.EnhanceGenericError("create issue", err)
//	// Returns error with operation context and recovery suggestions
//
//	err := guidance.ValidationErrorWithExample("teamID", "cannot be empty",
//	    `linear_create_issue("My Issue", teamId="CEN")`)
//	// Returns validation error with example showing correct usage
//
// # Structured Guidance
//
// For complex errors, use ErrorWithGuidance to provide:
//   - Operation context ("what were you trying to do")
//   - Reason ("why it failed")
//   - Guidance list ("how to fix it")
//   - Tool suggestions ("what commands to try")
//   - Code examples ("correct usage")
//
// Example:
//
//	return &guidance.ErrorWithGuidance{
//	    Operation: "Create issue",
//	    Reason:    "team 'INVALID' not found",
//	    Guidance: []string{
//	        "Use linear_list_teams() to see valid team keys",
//	        "Check for typos in team name",
//	    },
//	    Example: `team = linear_get_team("ENG")
//	linear_create_issue("Title", teamId=team.id)`,
//	    OriginalErr: err,
//	}
//
// # Design Philosophy
//
// This package is designed for AI agents that:
//   - Need to self-correct when operations fail
//   - Benefit from examples showing correct API usage
//   - Can execute suggested tools to gather more information
//
// All error messages are optimized for token efficiency while maintaining
// clarity and actionability.
package guidance
