// Package validation provides input validation utilities for Linear API operations.
//
// This package contains validation functions and constants used throughout
// the Linear client to ensure data integrity before making API calls.
//
// # Validation Constants
//
// Linear API enforces specific limits on field lengths:
//
//	validation.MaxTitleLength        // 255 characters
//	validation.MaxDescriptionLength  // 100,000 characters
//	validation.MaxNotificationLimit  // 250 notifications per request
//
// # String Validation
//
// Validate string lengths against Linear's limits:
//
//	err := validation.ValidateStringLength(title, "title", validation.MaxTitleLength)
//	if err != nil {
//	    // Title exceeds 255 characters
//	}
//
// # Numeric Validation
//
// Validate numeric ranges with helpful error messages:
//
//	err := validation.ValidatePositiveIntWithRange(priority, "priority", 0, 4)
//	if err != nil {
//	    // Priority must be between 0 and 4
//	}
//
// # Special Format Validation
//
// Validate emoji and metadata key formats:
//
//	if !validation.IsValidEmoji("👍") {
//	    return errors.New("invalid emoji")
//	}
//
//	if !validation.IsValidMetadataKey("my-key_123") {
//	    return errors.New("key must be alphanumeric with hyphens/underscores")
//	}
//
// # Design Principles
//
// Validation in this package follows these principles:
//   - Fail fast: Validate before making API calls
//   - Clear errors: Messages explain what's wrong and the valid range
//   - Prevent API errors: Catch issues before they reach Linear's API
//
// This approach improves user experience by providing immediate feedback
// rather than waiting for round-trip API errors.
package validation
