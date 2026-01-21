package linear

import (
	"errors"
	"fmt"
	"time"
)

// RateLimitError represents a rate limit error from the Linear API
// It includes information about when to retry the request
type RateLimitError struct {
	RetryAfter time.Duration // How long to wait before retrying
}

// Error implements the error interface for RateLimitError
func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limit exceeded, retry after %v", e.RetryAfter)
	}
	return "rate limit exceeded"
}

// AuthenticationError represents an authentication failure
// It provides details about why authentication failed
type AuthenticationError struct {
	Message string // Human-readable error message
	Code    string // Error code from the API (e.g., "INVALID_TOKEN", "TOKEN_EXPIRED")
}

// Error implements the error interface for AuthenticationError
func (e *AuthenticationError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("authentication failed: %s (code: %s)", e.Message, e.Code)
	}
	return fmt.Sprintf("authentication failed: %s", e.Message)
}

// ValidationError represents an input validation failure
// It provides details about which field failed validation and why
type ValidationError struct {
	Field   string      // The field that failed validation
	Value   interface{} // The invalid value that was provided
	Message string      // Simple error message
	Reason  string      // Why the validation failed
}

// Error implements the error interface for ValidationError
func (e *ValidationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("validation error: %s %s", e.Field, e.Message)
	}
	if e.Reason != "" {
		return fmt.Sprintf("validation error: field '%s' with value '%v' %s", e.Field, e.Value, e.Reason)
	}
	return fmt.Sprintf("validation error: invalid %s", e.Field)
}

// NotFoundError represents a resource not found error
// It indicates that a requested resource doesn't exist
type NotFoundError struct {
	ResourceType string // Type of resource (e.g., "issue", "project", "user")
	ResourceID   string // ID of the resource that wasn't found
}

// Error implements the error interface for NotFoundError
func (e *NotFoundError) Error() string {
	if e.ResourceID != "" {
		return fmt.Sprintf("%s not found: %s", e.ResourceType, e.ResourceID)
	}
	return fmt.Sprintf("%s not found", e.ResourceType)
}

// GraphQLError represents an error returned by the Linear GraphQL API
// It includes the error message and any extensions provided by the API
type GraphQLError struct {
	Message    string                 // The error message from GraphQL
	Extensions map[string]interface{} // Additional error context from the API
}

// Error implements the error interface for GraphQLError
func (e *GraphQLError) Error() string {
	// Check if there's a code in extensions
	if code, ok := e.Extensions["code"].(string); ok && code != "" {
		return fmt.Sprintf("GraphQL error: %s (code: %s)", e.Message, code)
	}
	return fmt.Sprintf("GraphQL error: %s", e.Message)
}

// HTTPError represents an HTTP-level error
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// Helper functions to check error types

// IsRateLimitError checks if an error is a RateLimitError
// It uses errors.As to handle wrapped errors
func IsRateLimitError(err error) bool {
	var rateLimitErr *RateLimitError
	return errors.As(err, &rateLimitErr)
}

// IsAuthenticationError checks if an error is an AuthenticationError
// It uses errors.As to handle wrapped errors
func IsAuthenticationError(err error) bool {
	var authErr *AuthenticationError
	return errors.As(err, &authErr)
}

// IsValidationError checks if an error is a ValidationError
// It uses errors.As to handle wrapped errors
func IsValidationError(err error) bool {
	var valErr *ValidationError
	return errors.As(err, &valErr)
}

// IsNotFoundError checks if an error is a NotFoundError
// It uses errors.As to handle wrapped errors
func IsNotFoundError(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

// IsGraphQLError checks if an error is a GraphQLError
// It uses errors.As to handle wrapped errors
func IsGraphQLError(err error) bool {
	var gqlErr *GraphQLError
	return errors.As(err, &gqlErr)
}

// GetRetryAfter extracts the retry duration from a RateLimitError
// Returns 0 if the error is not a RateLimitError or has no retry duration
func GetRetryAfter(err error) time.Duration {
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.RetryAfter
	}
	return 0
}

// Helper functions for validation

// validateNonEmptyString validates that a string is not empty
func validateNonEmptyString(value, fieldName string) error {
	if value == "" {
		return &ValidationError{
			Field:   fieldName,
			Message: "cannot be empty",
		}
	}
	return nil
}

// validatePositiveInt validates that an integer is positive
func validatePositiveInt(value int, fieldName string) error {
	if value <= 0 {
		return &ValidationError{
			Field:   fieldName,
			Value:   value,
			Message: "must be positive",
		}
	}
	return nil
}