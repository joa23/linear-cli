package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestIssueClientSearchIssuesEnhanced(t *testing.T) {
	t.Run("search with multiple filters", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"issues": {
					"nodes": [
						{
							"id": "issue-1",
							"identifier": "TEAM-123",
							"title": "Fix authentication bug",
							"description": "Users cannot login",
							"state": {
								"id": "state-1",
								"name": "In Progress"
							},
							"assignee": {
								"id": "user-1",
								"name": "John Doe",
								"email": "john@example.com"
							},
							"labels": {
								"nodes": [
									{
										"id": "label-1",
										"name": "bug",
										"color": "#FF0000"
									}
								]
							},
							"priority": 1,
							"createdAt": "2024-01-01T00:00:00Z",
							"updatedAt": "2024-01-02T00:00:00Z"
						}
					],
					"pageInfo": {
						"hasNextPage": false,
						"endCursor": "cursor-123"
					}
				}
			}
		}`

		client := NewClient("test-token")
		transport := &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			},
		}
		client.base.httpClient = &http.Client{Transport: transport}
		client.base.baseURL = "https://api.linear.app"

		// Test with enhanced search filters
		filters := &IssueSearchFilters{
			TeamID:     "team-123",
			StateIDs:   []string{"state-1", "state-2"},
			LabelIDs:   []string{"label-1"},
			AssigneeID: "user-1",
			Priority:   IntPtr(1),
			SearchTerm: "authentication",
			Limit:      20,
		}

		result, err := client.Issues.SearchIssuesEnhanced(filters)
		if err != nil {
			t.Fatalf("SearchIssuesEnhanced failed: %v", err)
		}

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		if result.Issues[0].Title != "Fix authentication bug" {
			t.Errorf("Expected title 'Fix authentication bug', got '%s'", result.Issues[0].Title)
		}
	})

	t.Run("search with empty filters returns all accessible issues", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"issues": {
					"nodes": [
						{
							"id": "issue-1",
							"identifier": "TEAM-123",
							"title": "Issue 1"
						},
						{
							"id": "issue-2",
							"identifier": "TEAM-124",
							"title": "Issue 2"
						}
					],
					"pageInfo": {
						"hasNextPage": true,
						"endCursor": "cursor-456"
					}
				}
			}
		}`

		client := NewClient("test-token")
		transport := &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			},
		}
		client.base.httpClient = &http.Client{Transport: transport}
		client.base.baseURL = "https://api.linear.app"

		// Test with no filters
		filters := &IssueSearchFilters{
			Limit: 50,
		}

		result, err := client.Issues.SearchIssuesEnhanced(filters)
		if err != nil {
			t.Fatalf("SearchIssuesEnhanced failed: %v", err)
		}

		if len(result.Issues) != 2 {
			t.Errorf("Expected 2 issues, got %d", len(result.Issues))
		}

		if !result.HasNextPage {
			t.Error("Expected HasNextPage to be true")
		}

		if result.EndCursor != "cursor-456" {
			t.Errorf("Expected EndCursor 'cursor-456', got '%s'", result.EndCursor)
		}
	})

	t.Run("search with date range filter", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"issues": {
					"nodes": [
						{
							"id": "issue-1",
							"identifier": "TEAM-125",
							"title": "Recent issue",
							"createdAt": "2024-01-15T00:00:00Z"
						}
					],
					"pageInfo": {
						"hasNextPage": false,
						"endCursor": null
					}
				}
			}
		}`

		client := NewClient("test-token")
		transport := &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			},
		}
		client.base.httpClient = &http.Client{Transport: transport}
		client.base.baseURL = "https://api.linear.app"

		// Test with date range
		filters := &IssueSearchFilters{
			CreatedAfter:  "2024-01-10T00:00:00Z",
			CreatedBefore: "2024-01-20T00:00:00Z",
			Limit:         10,
		}

		result, err := client.Issues.SearchIssuesEnhanced(filters)
		if err != nil {
			t.Fatalf("SearchIssuesEnhanced failed: %v", err)
		}

		if len(result.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(result.Issues))
		}

		if result.Issues[0].Title != "Recent issue" {
			t.Errorf("Expected title 'Recent issue', got '%s'", result.Issues[0].Title)
		}
	})
}

// Helper function to create int pointer
func IntPtr(i int) *int {
	return &i
}