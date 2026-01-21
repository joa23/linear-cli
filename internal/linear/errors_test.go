package linear

import (
	"errors"
	"testing"
	"time"
)

func TestRateLimitError(t *testing.T) {
	tests := []struct {
		name          string
		retryAfter    time.Duration
		expectedMsg   string
		expectedRetry time.Duration
	}{
		{
			name:          "rate limit with 5 second retry",
			retryAfter:    5 * time.Second,
			expectedMsg:   "rate limit exceeded, retry after 5s",
			expectedRetry: 5 * time.Second,
		},
		{
			name:          "rate limit with 1 minute retry",
			retryAfter:    60 * time.Second,
			expectedMsg:   "rate limit exceeded, retry after 1m0s",
			expectedRetry: 60 * time.Second,
		},
		{
			name:          "rate limit with no retry time",
			retryAfter:    0,
			expectedMsg:   "rate limit exceeded",
			expectedRetry: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &RateLimitError{RetryAfter: tt.retryAfter}
			
			// Test Error() method
			if err.Error() != tt.expectedMsg {
				t.Errorf("Error() = %v, want %v", err.Error(), tt.expectedMsg)
			}
			
			// Test RetryAfter field
			if err.RetryAfter != tt.expectedRetry {
				t.Errorf("RetryAfter = %v, want %v", err.RetryAfter, tt.expectedRetry)
			}
			
			// Test that it implements error interface
			var _ error = err
		})
	}
}

func TestAuthenticationError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		code        string
		expectedMsg string
	}{
		{
			name:        "invalid token error",
			message:     "Invalid API token",
			code:        "INVALID_TOKEN",
			expectedMsg: "authentication failed: Invalid API token (code: INVALID_TOKEN)",
		},
		{
			name:        "expired token error",
			message:     "Token has expired",
			code:        "TOKEN_EXPIRED",
			expectedMsg: "authentication failed: Token has expired (code: TOKEN_EXPIRED)",
		},
		{
			name:        "no code provided",
			message:     "Authentication required",
			code:        "",
			expectedMsg: "authentication failed: Authentication required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &AuthenticationError{
				Message: tt.message,
				Code:    tt.code,
			}
			
			// Test Error() method
			if err.Error() != tt.expectedMsg {
				t.Errorf("Error() = %v, want %v", err.Error(), tt.expectedMsg)
			}
			
			// Test that it implements error interface
			var _ error = err
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		value       interface{}
		reason      string
		expectedMsg string
	}{
		{
			name:        "empty string validation",
			field:       "issueID",
			value:       "",
			reason:      "cannot be empty",
			expectedMsg: "validation error: field 'issueID' with value '' cannot be empty",
		},
		{
			name:        "invalid format validation",
			field:       "teamID",
			value:       "invalid-uuid",
			reason:      "must be a valid UUID",
			expectedMsg: "validation error: field 'teamID' with value 'invalid-uuid' must be a valid UUID",
		},
		{
			name:        "numeric range validation",
			field:       "limit",
			value:       -1,
			reason:      "must be positive",
			expectedMsg: "validation error: field 'limit' with value '-1' must be positive",
		},
		{
			name:        "nil value",
			field:       "data",
			value:       nil,
			reason:      "cannot be nil",
			expectedMsg: "validation error: field 'data' with value '<nil>' cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ValidationError{
				Field:  tt.field,
				Value:  tt.value,
				Reason: tt.reason,
			}
			
			// Test Error() method
			if err.Error() != tt.expectedMsg {
				t.Errorf("Error() = %v, want %v", err.Error(), tt.expectedMsg)
			}
			
			// Test that it implements error interface
			var _ error = err
		})
	}
}

func TestNotFoundError(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceID   string
		expectedMsg  string
	}{
		{
			name:         "issue not found",
			resourceType: "issue",
			resourceID:   "ISS-123",
			expectedMsg:  "issue not found: ISS-123",
		},
		{
			name:         "project not found",
			resourceType: "project",
			resourceID:   "PROJ-456",
			expectedMsg:  "project not found: PROJ-456",
		},
		{
			name:         "user not found",
			resourceType: "user",
			resourceID:   "user-789",
			expectedMsg:  "user not found: user-789",
		},
		{
			name:         "empty resource ID",
			resourceType: "comment",
			resourceID:   "",
			expectedMsg:  "comment not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &NotFoundError{
				ResourceType: tt.resourceType,
				ResourceID:   tt.resourceID,
			}
			
			// Test Error() method
			if err.Error() != tt.expectedMsg {
				t.Errorf("Error() = %v, want %v", err.Error(), tt.expectedMsg)
			}
			
			// Test that it implements error interface
			var _ error = err
		})
	}
}

func TestGraphQLError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		extensions  map[string]interface{}
		expectedMsg string
	}{
		{
			name:    "simple GraphQL error",
			message: "Field 'invalidField' doesn't exist on type 'Issue'",
			expectedMsg: "GraphQL error: Field 'invalidField' doesn't exist on type 'Issue'",
		},
		{
			name:    "GraphQL error with code",
			message: "Unauthorized access",
			extensions: map[string]interface{}{
				"code": "UNAUTHORIZED",
			},
			expectedMsg: "GraphQL error: Unauthorized access (code: UNAUTHORIZED)",
		},
		{
			name:    "GraphQL error with multiple extensions",
			message: "Rate limit exceeded",
			extensions: map[string]interface{}{
				"code":       "RATE_LIMITED",
				"retryAfter": 60,
			},
			expectedMsg: "GraphQL error: Rate limit exceeded (code: RATE_LIMITED)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &GraphQLError{
				Message:    tt.message,
				Extensions: tt.extensions,
			}
			
			// Test Error() method
			if err.Error() != tt.expectedMsg {
				t.Errorf("Error() = %v, want %v", err.Error(), tt.expectedMsg)
			}
			
			// Test that it implements error interface
			var _ error = err
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that our custom errors work with errors.Is and errors.As
	
	t.Run("errors.Is with RateLimitError", func(t *testing.T) {
		originalErr := &RateLimitError{RetryAfter: 5 * time.Second}
		wrappedErr := errors.Join(originalErr, errors.New("additional context"))
		
		if !errors.Is(wrappedErr, originalErr) {
			t.Error("errors.Is should recognize wrapped RateLimitError")
		}
	})
	
	t.Run("errors.As with AuthenticationError", func(t *testing.T) {
		originalErr := &AuthenticationError{Message: "Invalid token", Code: "INVALID"}
		wrappedErr := errors.Join(originalErr, errors.New("during API call"))
		
		var authErr *AuthenticationError
		if !errors.As(wrappedErr, &authErr) {
			t.Error("errors.As should extract AuthenticationError from wrapped error")
		}
		
		if authErr.Code != "INVALID" {
			t.Errorf("Expected code INVALID, got %s", authErr.Code)
		}
	})
	
	t.Run("errors.As with ValidationError", func(t *testing.T) {
		originalErr := &ValidationError{Field: "issueID", Value: "", Reason: "cannot be empty"}
		wrappedErr := errors.Join(originalErr, errors.New("in CreateIssue"))
		
		var valErr *ValidationError
		if !errors.As(wrappedErr, &valErr) {
			t.Error("errors.As should extract ValidationError from wrapped error")
		}
		
		if valErr.Field != "issueID" {
			t.Errorf("Expected field issueID, got %s", valErr.Field)
		}
	})
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct RateLimitError",
			err:      &RateLimitError{RetryAfter: 5 * time.Second},
			expected: true,
		},
		{
			name:     "wrapped RateLimitError",
			err:      errors.Join(&RateLimitError{RetryAfter: 5 * time.Second}, errors.New("context")),
			expected: true,
		},
		{
			name:     "different error type",
			err:      &AuthenticationError{Message: "test"},
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRateLimitError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRateLimitError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct AuthenticationError",
			err:      &AuthenticationError{Message: "Invalid token"},
			expected: true,
		},
		{
			name:     "wrapped AuthenticationError",
			err:      errors.Join(&AuthenticationError{Message: "Invalid"}, errors.New("context")),
			expected: true,
		},
		{
			name:     "different error type",
			err:      &ValidationError{Field: "test"},
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthenticationError(tt.err)
			if result != tt.expected {
				t.Errorf("IsAuthenticationError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct NotFoundError",
			err:      &NotFoundError{ResourceType: "issue", ResourceID: "123"},
			expected: true,
		},
		{
			name:     "wrapped NotFoundError",
			err:      errors.Join(&NotFoundError{ResourceType: "issue"}, errors.New("context")),
			expected: true,
		},
		{
			name:     "different error type",
			err:      &RateLimitError{},
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", result, tt.expected)
			}
		})
	}
}