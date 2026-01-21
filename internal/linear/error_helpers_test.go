package linear

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorWithGuidance(t *testing.T) {
	tests := []struct {
		name            string
		error           error
		expectedContent []string
	}{
		{
			name: "InvalidStateIDError",
			error: InvalidStateIDError("test-state-id", errors.New("original error")),
			expectedContent: []string{
				"Update issue state failed",
				"test-state-id",
				"does not exist",
				"linear_get_workflow_states",
				"grouped by type",
				"Example:",
				"states.byType.started",
			},
		},
		{
			name: "AuthenticationRequiredError",
			error: AuthenticationRequiredError("create issue"),
			expectedContent: []string{
				"create issue failed",
				"authentication is required",
				"linear_login",
				"Check if your token has expired",
				"Example:",
			},
		},
		{
			name: "ResourceNotFoundError",
			error: ResourceNotFoundError("issue", "LIN-123", errors.New("404")),
			expectedContent: []string{
				"Find issue failed",
				"issue with ID 'LIN-123' not found",
				"Verify the issue ID is correct",
				"linear_search_issues",
				"linear_get_issue",
			},
		},
		{
			name: "ValidationErrorWithExample",
			error: ValidationErrorWithExample("teamId", "cannot be empty", "linear_get_teams()"),
			expectedContent: []string{
				"Validate input failed",
				"teamId cannot be empty",
				"Example:",
				"linear_get_teams()",
			},
		},
		{
			name: "NetworkRetryError",
			error: NetworkRetryError("fetch issues", errors.New("connection reset")),
			expectedContent: []string{
				"fetch issues failed",
				"network error occurred",
				"Wait a few seconds and retry",
				"exponential backoff",
			},
		},
		{
			name: "RateLimitError",
			error: RateLimitErrorWithGuidance("update issues", 30),
			expectedContent: []string{
				"update issues failed",
				"rate limit exceeded",
				"Wait 30 seconds",
				"batching operations",
			},
		},
		{
			name: "OperationFailedError",
			error: OperationFailedError("Create comment", "comment", []string{"Custom guidance"}),
			expectedContent: []string{
				"Create comment failed",
				"operation was not successful",
				"Verify all required fields",
				"Custom guidance",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.error.Error()
			
			// Check that all expected content is present
			for _, expected := range tt.expectedContent {
				if !strings.Contains(errStr, expected) {
					t.Errorf("Expected error to contain '%s', but it doesn't.\nFull error: %s", expected, errStr)
				}
			}
			
			// Verify the error is formatted with sections
			if !strings.Contains(errStr, "To resolve this:") && !strings.Contains(errStr, "Helpful tools:") {
				t.Logf("Warning: Error might be missing structured sections: %s", errStr)
			}
		})
	}
}

func TestEnhanceGenericError(t *testing.T) {
	tests := []struct {
		name            string
		operation       string
		originalError   error
		expectedType    string
		expectedContent []string
	}{
		{
			name:          "State ID error",
			operation:     "update state",
			originalError: errors.New("Entity not found in validateAccess: stateId"),
			expectedType:  "*ErrorWithGuidance",
			expectedContent: []string{
				"does not exist",
				"linear_get_workflow_states",
			},
		},
		{
			name:          "Authentication error",
			operation:     "list issues",
			originalError: errors.New("401 unauthorized"),
			expectedType:  "*ErrorWithGuidance",
			expectedContent: []string{
				"authentication is required",
				"linear_login",
			},
		},
		{
			name:          "Rate limit error",
			operation:     "bulk update",
			originalError: errors.New("429 rate limit exceeded"),
			expectedType:  "*ErrorWithGuidance",
			expectedContent: []string{
				"rate limit exceeded",
				"Wait",
			},
		},
		{
			name:          "Network error",
			operation:     "fetch data",
			originalError: errors.New("network connection timeout"),
			expectedType:  "*ErrorWithGuidance",
			expectedContent: []string{
				"network error occurred",
				"retry",
			},
		},
		{
			name:          "Not found error",
			operation:     "get resource",
			originalError: errors.New("404 not found"),
			expectedType:  "*ErrorWithGuidance",
			expectedContent: []string{
				"not found",
				"Verify",
			},
		},
		{
			name:          "Generic error",
			operation:     "unknown operation",
			originalError: errors.New("something went wrong"),
			expectedType:  "*ErrorWithGuidance",
			expectedContent: []string{
				"unknown operation failed",
				"something went wrong",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enhanced := EnhanceGenericError(tt.operation, tt.originalError)
			
			// Check error type
			if _, ok := enhanced.(*ErrorWithGuidance); !ok {
				t.Errorf("Expected error type %s, got %T", tt.expectedType, enhanced)
			}
			
			errStr := enhanced.Error()
			
			// Check expected content
			for _, expected := range tt.expectedContent {
				if !strings.Contains(errStr, expected) {
					t.Errorf("Expected error to contain '%s', but it doesn't.\nFull error: %s", expected, errStr)
				}
			}
		})
	}
}

func TestErrorWithGuidance_Structure(t *testing.T) {
	err := &ErrorWithGuidance{
		Operation: "Test operation",
		Reason:    "test failed",
		Guidance: []string{
			"Step 1",
			"Step 2",
		},
		Tools: []string{
			"tool1",
			"tool2",
		},
		Example:     "example code",
		OriginalErr: errors.New("original"),
	}
	
	errStr := err.Error()
	
	// Check structure
	expectedSections := []string{
		"Test operation failed: test failed",
		"To resolve this:",
		"1. Step 1",
		"2. Step 2",
		"Helpful tools:",
		"- tool1",
		"- tool2",
		"Example:",
		"example code",
		"Debug info: original",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(errStr, section) {
			t.Errorf("Expected error to contain section '%s', but it doesn't.\nFull error: %s", section, errStr)
		}
	}
}