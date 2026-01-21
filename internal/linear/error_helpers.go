package linear

import (
	"fmt"
	"strings"
)

// ErrorWithGuidance creates an error with actionable guidance for agents
type ErrorWithGuidance struct {
	Operation   string   // What operation failed
	Reason      string   // Why it failed
	Guidance    []string // Steps to resolve
	Tools       []string // MCP tools that might help
	Example     string   // Example of correct usage
	OriginalErr error    // Original error for debugging
}

func (e *ErrorWithGuidance) Error() string {
	var sb strings.Builder

	// Primary error message
	sb.WriteString(fmt.Sprintf("%s failed: %s", e.Operation, e.Reason))

	// Add guidance steps
	if len(e.Guidance) > 0 {
		sb.WriteString("\n\nTo resolve this:")
		for i, step := range e.Guidance {
			sb.WriteString(fmt.Sprintf("\n%d. %s", i+1, step))
		}
	}

	// Add helpful tools
	if len(e.Tools) > 0 {
		sb.WriteString("\n\nHelpful tools:")
		for _, tool := range e.Tools {
			sb.WriteString(fmt.Sprintf("\n- %s", tool))
		}
	}

	// Add example if provided
	if e.Example != "" {
		sb.WriteString(fmt.Sprintf("\n\nExample:\n%s", e.Example))
	}

	// Add original error for debugging
	if e.OriginalErr != nil {
		sb.WriteString(fmt.Sprintf("\n\nDebug info: %v", e.OriginalErr))
	}

	return sb.String()
}

// Unwrap returns the original error for errors.As/Is/Unwrap support
func (e *ErrorWithGuidance) Unwrap() error {
	return e.OriginalErr
}

// Common error guidance helpers

// InvalidStateIDError creates an error for invalid state IDs with guidance
func InvalidStateIDError(stateID string, err error) error {
	return &ErrorWithGuidance{
		Operation: "Update issue state",
		Reason:    fmt.Sprintf("the state ID '%s' does not exist or you don't have access to it", stateID),
		Guidance: []string{
			"Use linear_get_workflow_states to discover valid state IDs for your team",
			"States are grouped by type (backlog, unstarted, started, completed, canceled)",
			"Use the exact state ID from the workflow states response",
			"State IDs are unique per workspace and team",
		},
		Tools: []string{
			"linear_get_workflow_states(teamId) - Get all valid states for a team",
			"linear_get_issue(issueId) - Check the current state and team of the issue",
		},
		Example: `states = linear_get_workflow_states(teamId="backend-team-id")
in_progress_id = states.byType.started[0].id
linear_update_issue_state(issueId, in_progress_id)`,
		OriginalErr: err,
	}
}

// AuthenticationRequiredError creates an error when authentication is needed
func AuthenticationRequiredError(operation string) error {
	return &ErrorWithGuidance{
		Operation: operation,
		Reason:    "authentication is required",
		Guidance: []string{
			"Run linear_login to authenticate with Linear",
			"Check if your token has expired",
			"Verify Linear OAuth credentials are configured",
		},
		Tools: []string{
			"linear_login - Authenticate with Linear",
			"linear_get_viewer - Check current authentication status",
		},
		Example: `linear_login()
// Browser will open for OAuth authentication
// After successful login, retry your operation`,
	}
}

// ResourceNotFoundError creates an error when a resource is not found
func ResourceNotFoundError(resourceType, resourceID string, err error) error {
	toolMap := map[string][]string{
		"issue": {
			"linear_search_issues - Find issues by various filters",
			"linear_get_issue(issueId) - Verify issue exists",
		},
		"user": {
			"linear_list_users(teamId) - List all users in a team",
			"linear_get_user(userId) - Get specific user details",
		},
		"team": {
			"linear_get_teams - List all available teams",
		},
		"project": {
			"linear_list_projects(filter='all') - List all projects",
			"linear_get_project(projectId) - Verify project exists",
		},
		"comment": {
			"linear_get_issue(issueId) - Check issue and its comments",
		},
	}
	
	return &ErrorWithGuidance{
		Operation: fmt.Sprintf("Find %s", resourceType),
		Reason:    fmt.Sprintf("%s with ID '%s' not found", resourceType, resourceID),
		Guidance: []string{
			fmt.Sprintf("Verify the %s ID is correct", resourceType),
			fmt.Sprintf("Check if you have access to this %s", resourceType),
			fmt.Sprintf("Use discovery tools to find valid %s IDs", resourceType),
		},
		Tools:       toolMap[resourceType],
		OriginalErr: err,
	}
}

// ValidationError creates an error for invalid input with examples
func ValidationErrorWithExample(field, requirement, example string) error {
	// Create the underlying ValidationError
	valErr := &ValidationError{
		Field:   field,
		Message: requirement,
	}

	// Wrap it with guidance for better UX
	return &ErrorWithGuidance{
		Operation: "Validate input",
		Reason:    fmt.Sprintf("%s %s", field, requirement),
		Guidance: []string{
			fmt.Sprintf("Ensure %s is provided", field),
			"Check the format matches Linear's requirements",
		},
		Example:     example,
		OriginalErr: valErr,
	}
}

// NetworkRetryError creates an error with retry guidance
func NetworkRetryError(operation string, err error) error {
	return &ErrorWithGuidance{
		Operation: operation,
		Reason:    "network error occurred",
		Guidance: []string{
			"Wait a few seconds and retry the operation",
			"Check your internet connection",
			"If the error persists, Linear's API might be experiencing issues",
			"The operation will be automatically retried with exponential backoff",
		},
		OriginalErr: err,
	}
}

// RateLimitErrorWithGuidance creates an error with rate limit guidance
func RateLimitErrorWithGuidance(operation string, retryAfter int) error {
	return &ErrorWithGuidance{
		Operation: operation,
		Reason:    "rate limit exceeded",
		Guidance: []string{
			fmt.Sprintf("Wait %d seconds before retrying", retryAfter),
			"Reduce the frequency of API calls",
			"Consider batching operations where possible",
		},
		Tools: []string{
			"linear_batch_update_issues - Update multiple issues in one call (if available)",
		},
	}
}

// OperationFailedError creates an error when an operation fails without clear reason
func OperationFailedError(operation, resourceType string, guidance []string) error {
	defaultGuidance := []string{
		"Verify all required fields are provided",
		"Check if you have permission to perform this operation",
		"Try fetching the resource first to verify it exists",
	}
	
	if len(guidance) > 0 {
		defaultGuidance = append(defaultGuidance, guidance...)
	}
	
	return &ErrorWithGuidance{
		Operation: operation,
		Reason:    "operation was not successful",
		Guidance:  defaultGuidance,
	}
}

// BatchOperationError creates an error for batch operations
func BatchOperationError(operation string, err error) error {
	return &ErrorWithGuidance{
		Operation: operation,
		Reason:    "batch operation failed",
		Guidance: []string{
			"Check if all provided IDs are valid",
			"Verify you have permission to update all items",
			"Try updating items individually to identify the problematic one",
			"Ensure update fields are valid for all items",
		},
		Tools: []string{
			"linear_get_issue(issueId) - Verify each issue exists",
			"linear_update_issue_state(issueId, stateId) - Update individually",
		},
		OriginalErr: err,
	}
}

// ConfigurationError creates an error for configuration issues
func ConfigurationError(issue string, err error) error {
	return &ErrorWithGuidance{
		Operation: "Configuration",
		Reason:    issue,
		Guidance: []string{
			"Check file permissions in ~/.config/linear/",
			"Ensure the configuration file exists",
			"Verify the configuration format is valid JSON",
		},
		Example: `{
  "client_id": "YOUR_LINEAR_CLIENT_ID",
  "client_secret": "YOUR_LINEAR_CLIENT_SECRET"
}`,
		OriginalErr: err,
	}
}

// EnhanceGenericError takes a generic error and adds context
func EnhanceGenericError(operation string, err error) error {
	if err == nil {
		return nil
	}
	
	// Check for specific error patterns and enhance them
	errStr := err.Error()
	
	// State ID errors
	if strings.Contains(errStr, "Entity not found in validateAccess: stateId") {
		return InvalidStateIDError("unknown", err)
	}
	
	// Authentication errors
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") {
		return AuthenticationRequiredError(operation)
	}
	
	// Rate limit errors
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		return RateLimitErrorWithGuidance(operation, 60)
	}
	
	// Network errors
	if strings.Contains(errStr, "network") || strings.Contains(errStr, "connection") {
		return NetworkRetryError(operation, err)
	}
	
	// Resource not found
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "404") {
		return ResourceNotFoundError("resource", "unknown", err)
	}
	
	// Default enhancement
	return &ErrorWithGuidance{
		Operation:   operation,
		Reason:      err.Error(),
		Guidance:    []string{"Check the error details and try again"},
		OriginalErr: err,
	}
}