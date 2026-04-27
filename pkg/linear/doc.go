// Package linear provides a Go client for the Linear API with comprehensive
// GraphQL support, automatic rate limiting, and robust error handling.
//
// The client supports all major Linear operations including issue management,
// comments, notifications, team operations, and custom metadata storage using
// a description-based approach.
//
// # Authentication
//
// The client requires a Linear API token for authentication. You can provide
// the token directly or load it from a file:
//
//	// Direct token
//	client := linear.NewClient("your-api-token")
//
//	// Token from file with fallback to environment variable
//	client := linear.NewClientWithTokenPath("/path/to/token")
//
// # Basic Usage
//
// Get an issue:
//
//	issue, err := client.GetIssue("issue-id")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Issue: %s - %s\n", issue.Identifier, issue.Title)
//
// Create an issue:
//
//	issue, err := client.CreateIssue("team-id", "Bug Report", "Description here")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Update issue state:
//
//	err := client.UpdateIssueState("issue-id", "state-id")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Metadata Management
//
// This client supports storing custom metadata in Linear issue and project
// descriptions using a collapsible markdown format. Metadata is automatically
// extracted when fetching issues/projects and preserved when updating descriptions.
//
// Update metadata for an issue:
//
//	err := client.UpdateIssueMetadataKey("issue-id", "priority", "high")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Remove metadata:
//
//	err := client.RemoveIssueMetadataKey("issue-id", "priority")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Error Handling
//
// The package defines custom error types for better error handling:
//
//	err := client.UpdateIssueState("issue-id", "invalid-state")
//	if linear.IsValidationError(err) {
//	    // Handle validation error
//	}
//	if linear.IsRateLimitError(err) {
//	    // Handle rate limit with retry
//	    retryAfter := linear.GetRetryAfter(err)
//	}
//
// # Rate Limiting
//
// The client automatically handles rate limiting with exponential backoff
// and respects Linear's rate limit headers. When rate limits are exceeded,
// a RateLimitError is returned with retry information.
//
// # Network Resilience
//
// The client includes automatic retry logic for transient network errors
// with exponential backoff. Connection failures, timeouts, and temporary
// DNS issues are automatically retried.
package linear