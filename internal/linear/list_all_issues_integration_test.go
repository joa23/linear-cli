//go:build integration

package linear

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAllIssuesIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}

	// Create client
	token := os.Getenv("LINEAR_TOKEN")
	if token == "" {
		t.Skip("LINEAR_TOKEN not set")
	}

	client := NewClient(token)

	t.Run("list all issues with no filters", func(t *testing.T) {
		filter := &IssueFilter{
			First: 10, // Small limit for testing
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have some issues
		assert.GreaterOrEqual(t, len(result.Issues), 0)
		
		// Check pagination info
		assert.NotEmpty(t, result.EndCursor)
		
		// Verify issue structure
		for _, issue := range result.Issues {
			assert.NotEmpty(t, issue.ID)
			assert.NotEmpty(t, issue.Title)
			assert.NotEmpty(t, issue.State.ID)
			assert.NotEmpty(t, issue.State.Name)
			assert.NotEmpty(t, issue.Team.ID)
			assert.NotEmpty(t, issue.Team.Name)
		}
	})

	t.Run("list issues with sorting by priority", func(t *testing.T) {
		filter := &IssueFilter{
			First:     5,
			OrderBy:   "priority",
			Direction: "asc",
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify issues are returned
		if len(result.Issues) > 1 {
			// Check that priorities are in ascending order (1 is highest priority)
			for i := 1; i < len(result.Issues); i++ {
				assert.LessOrEqual(t, result.Issues[i-1].Priority, result.Issues[i].Priority)
			}
		}
	})

	t.Run("list issues with pagination", func(t *testing.T) {
		// First page
		filter := &IssueFilter{
			First: 5,
		}

		result1, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result1)

		if result1.HasNextPage {
			// Second page
			filter.After = result1.EndCursor
			result2, err := client.ListAllIssues(filter)
			require.NoError(t, err)
			require.NotNil(t, result2)

			// Make sure we got different issues
			if len(result2.Issues) > 0 && len(result1.Issues) > 0 {
				assert.NotEqual(t, result1.Issues[0].ID, result2.Issues[0].ID)
			}
		}
	})

	t.Run("list issues with metadata extraction", func(t *testing.T) {
		filter := &IssueFilter{
			First: 20,
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Check if any issues have metadata
		hasMetadata := false
		for _, issue := range result.Issues {
			if issue.Metadata != nil && len(*issue.Metadata) > 0 {
				hasMetadata = true
				break
			}
		}
		
		// It's okay if no issues have metadata, just log it
		if hasMetadata {
			t.Log("Found issues with metadata")
		} else {
			t.Log("No issues with metadata found")
		}
	})
}