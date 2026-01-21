package linear

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestErrorWrappingPatterns demonstrates proper error wrapping
func TestErrorWrappingPatterns(t *testing.T) {
	// Base errors for testing
	baseErr := errors.New("connection refused")
	
	tests := []struct {
		name           string
		wrapFunc       func(error) error
		expectedInMsg  []string
		shouldUnwrap   bool
	}{
		{
			name: "operation context wrapping",
			wrapFunc: func(err error) error {
				// Good: Provides operation context and wraps the error
				return fmt.Errorf("failed to get issue: %w", err)
			},
			expectedInMsg: []string{"failed to get issue", "connection refused"},
			shouldUnwrap:  true,
		},
		{
			name: "multi-level context wrapping",
			wrapFunc: func(err error) error {
				// Good: Multiple levels of context
				err = fmt.Errorf("GraphQL query failed: %w", err)
				err = fmt.Errorf("GetIssue operation: %w", err)
				return err
			},
			expectedInMsg: []string{"GetIssue operation", "GraphQL query failed", "connection refused"},
			shouldUnwrap:  true,
		},
		{
			name: "wrapping with additional details",
			wrapFunc: func(err error) error {
				// Good: Adds specific details
				issueID := "ISS-123"
				return fmt.Errorf("failed to get issue %s: %w", issueID, err)
			},
			expectedInMsg: []string{"failed to get issue ISS-123", "connection refused"},
			shouldUnwrap:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedErr := tt.wrapFunc(baseErr)
			
			// Check error message contains expected strings
			errMsg := wrappedErr.Error()
			for _, expected := range tt.expectedInMsg {
				if !strings.Contains(errMsg, expected) {
					t.Errorf("Error message %q should contain %q", errMsg, expected)
				}
			}
			
			// Check error unwrapping works
			if tt.shouldUnwrap {
				if !errors.Is(wrappedErr, baseErr) {
					t.Errorf("Wrapped error should unwrap to base error")
				}
			}
		})
	}
}

// TestErrorWrappingInMethods tests that our methods properly wrap errors
func TestErrorWrappingInMethods(t *testing.T) {
	// This test verifies that methods follow proper error wrapping patterns
	
	t.Run("makeRequestWithRetry error wrapping", func(t *testing.T) {
		// The makeRequestWithRetry method should wrap errors with context
		// Example of what we want to see:
		// Instead of: return nil, err
		// We want: return nil, fmt.Errorf("failed to execute request: %w", err)
		
		// This is a documentation test showing the pattern
		t.Skip("This is a pattern demonstration test")
	})
	
	t.Run("makeGraphQLRequest error wrapping", func(t *testing.T) {
		// GraphQL errors should include the operation context
		// Example of what we want to see:
		// Instead of: return nil, errors.New(graphQLResp.Errors[0].Message)
		// We want: return nil, fmt.Errorf("GraphQL query failed: %s", graphQLResp.Errors[0].Message)
		
		t.Skip("This is a pattern demonstration test")
	})
}

// TestCustomErrorWrapping tests wrapping with our custom error types
func TestCustomErrorWrapping(t *testing.T) {
	t.Run("wrap authentication error", func(t *testing.T) {
		authErr := &AuthenticationError{
			Message: "Invalid token",
			Code:    "INVALID_TOKEN",
		}
		
		// Wrap with additional context
		wrappedErr := fmt.Errorf("failed to authenticate for GetIssue: %w", authErr)
		
		// Should be able to extract the original error
		var extractedAuthErr *AuthenticationError
		if !errors.As(wrappedErr, &extractedAuthErr) {
			t.Error("Should be able to extract AuthenticationError from wrapped error")
		}
		
		if extractedAuthErr.Code != "INVALID_TOKEN" {
			t.Errorf("Expected code INVALID_TOKEN, got %s", extractedAuthErr.Code)
		}
	})
	
	t.Run("wrap rate limit error", func(t *testing.T) {
		rateLimitErr := &RateLimitError{
			RetryAfter: 5,
		}
		
		// Wrap with operation context
		wrappedErr := fmt.Errorf("rate limited during issue creation: %w", rateLimitErr)
		
		// Should be able to check if it's a rate limit error
		if !IsRateLimitError(wrappedErr) {
			t.Error("IsRateLimitError should return true for wrapped rate limit error")
		}
		
		// Should be able to extract retry duration
		if retryAfter := GetRetryAfter(wrappedErr); retryAfter != 5 {
			t.Errorf("Expected retry after 5, got %v", retryAfter)
		}
	})
	
	t.Run("wrap validation error", func(t *testing.T) {
		valErr := &ValidationError{
			Field:  "issueID",
			Value:  "",
			Reason: "cannot be empty",
		}
		
		// Wrap with method context
		wrappedErr := fmt.Errorf("GetIssue failed: %w", valErr)
		
		// Should preserve validation error info
		var extractedValErr *ValidationError
		if !errors.As(wrappedErr, &extractedValErr) {
			t.Error("Should be able to extract ValidationError from wrapped error")
		}
		
		if extractedValErr.Field != "issueID" {
			t.Errorf("Expected field issueID, got %s", extractedValErr.Field)
		}
	})
}

// TestErrorContextPatterns shows examples of good error context
func TestErrorContextPatterns(t *testing.T) {
	examples := []struct {
		name    string
		badCode string
		goodCode string
	}{
		{
			name:    "HTTP request error",
			badCode: `return nil, err`,
			goodCode: `return nil, fmt.Errorf("failed to execute HTTP request: %w", err)`,
		},
		{
			name:    "JSON unmarshaling",
			badCode: `return nil, err`,
			goodCode: `return nil, fmt.Errorf("failed to unmarshal response: %w", err)`,
		},
		{
			name:    "GraphQL errors",
			badCode: `return errors.New(graphQLResp.Errors[0].Message)`,
			goodCode: `return fmt.Errorf("GraphQL query failed: %s", graphQLResp.Errors[0].Message)`,
		},
		{
			name:    "Missing data errors",
			badCode: `return errors.New("no data returned from API")`,
			goodCode: `return fmt.Errorf("no data returned from GetIssue API call")`,
		},
		{
			name:    "Rate limit errors",
			badCode: `return fmt.Errorf("rate limit exceeded after %d retries", maxRetries)`,
			goodCode: `return &RateLimitError{RetryAfter: retryAfter}`,
		},
	}
	
	// This test documents the patterns we want to follow
	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			t.Logf("Bad:  %s", ex.badCode)
			t.Logf("Good: %s", ex.goodCode)
		})
	}
}