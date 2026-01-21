package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestIssueClientBatchUpdateIssues(t *testing.T) {
	t.Run("batch update multiple issues", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"issueBatchUpdate": {
					"success": true,
					"updatedIssues": [
						{
							"id": "issue-1",
							"identifier": "TEAM-123",
							"title": "Issue 1",
							"state": {
								"id": "state-done",
								"name": "Done"
							}
						},
						{
							"id": "issue-2",
							"identifier": "TEAM-124",
							"title": "Issue 2",
							"state": {
								"id": "state-done",
								"name": "Done"
							}
						},
						{
							"id": "issue-3",
							"identifier": "TEAM-125",
							"title": "Issue 3",
							"state": {
								"id": "state-done",
								"name": "Done"
							}
						}
					]
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

		// Test batch update
		issueIDs := []string{"issue-1", "issue-2", "issue-3"}
		update := BatchIssueUpdate{
			StateID: "state-done",
		}

		result, err := client.Issues.BatchUpdateIssues(issueIDs, update)
		if err != nil {
			t.Fatalf("BatchUpdateIssues failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success to be true")
		}

		if len(result.UpdatedIssues) != 3 {
			t.Errorf("Expected 3 updated issues, got %d", len(result.UpdatedIssues))
		}

		for _, issue := range result.UpdatedIssues {
			if issue.State.ID != "state-done" {
				t.Errorf("Expected state ID 'state-done', got '%s'", issue.State.ID)
			}
		}
	})

	t.Run("batch update with labels and assignee", func(t *testing.T) {
		// Mock response for updating labels and assignee
		mockResponseBody := `{
			"data": {
				"issueBatchUpdate": {
					"success": true,
					"updatedIssues": [
						{
							"id": "issue-1",
							"identifier": "TEAM-123",
							"title": "Issue 1",
							"assignee": {
								"id": "user-123",
								"name": "John Doe"
							},
							"labels": {
								"nodes": [
									{
										"id": "label-1",
										"name": "urgent"
									},
									{
										"id": "label-2",
										"name": "bug"
									}
								]
							}
						}
					]
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
		
		// We can't capture request with the simple mockTransport
		// Just verify the response handling
		
		client.base.httpClient = &http.Client{Transport: transport}
		client.base.baseURL = "https://api.linear.app"

		issueIDs := []string{"issue-1"}
		update := BatchIssueUpdate{
			AssigneeID: "user-123",
			LabelIDs:   []string{"label-1", "label-2"},
		}

		result, err := client.Issues.BatchUpdateIssues(issueIDs, update)
		if err != nil {
			t.Fatalf("BatchUpdateIssues failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected success to be true")
		}

		if len(result.UpdatedIssues) != 1 {
			t.Errorf("Expected 1 updated issue, got %d", len(result.UpdatedIssues))
		}

		issue := result.UpdatedIssues[0]
		if issue.Assignee.ID != "user-123" {
			t.Errorf("Expected assignee ID 'user-123', got '%s'", issue.Assignee.ID)
		}

		if len(issue.Labels.Nodes) != 2 {
			t.Errorf("Expected 2 labels, got %d", len(issue.Labels.Nodes))
		}
	})

	t.Run("batch update fails with invalid issue IDs", func(t *testing.T) {
		mockResponseBody := `{
			"errors": [
				{
					"message": "Issues not found",
					"extensions": {
						"code": "NOT_FOUND"
					}
				}
			]
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

		issueIDs := []string{"invalid-id-1", "invalid-id-2"}
		update := BatchIssueUpdate{
			StateID: "state-done",
		}

		_, err := client.Issues.BatchUpdateIssues(issueIDs, update)
		if err == nil {
			t.Fatal("Expected error for invalid issue IDs")
		}

		if !contains(err.Error(), "Issues not found") {
			t.Errorf("Expected error to contain 'Issues not found', got: %v", err)
		}
	})
}

