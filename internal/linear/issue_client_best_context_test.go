package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestIssueClientGetIssueWithBestContext(t *testing.T) {
	t.Run("issue with parent context", func(t *testing.T) {
		// First call - basic issue info shows it has a parent
		basicResponse := `{
			"data": {
				"issue": {
					"id": "issue-1",
					"identifier": "TEAM-123",
					"title": "Sub-task Issue",
					"description": "A sub-task",
					"state": {"id": "state-1", "name": "Todo"},
					"parent": {
						"id": "parent-1",
						"identifier": "TEAM-100",
						"title": "Parent Issue"
					}
				}
			}
		}`

		// Second call - full parent context
		parentContextResponse := `{
			"data": {
				"issue": {
					"id": "issue-1",
					"identifier": "TEAM-123",
					"title": "Sub-task Issue",
					"description": "A sub-task",
					"state": {"id": "state-1", "name": "Todo"},
					"parent": {
						"id": "parent-1",
						"identifier": "TEAM-100",
						"title": "Parent Issue",
						"description": "Parent description",
						"state": {"id": "state-2", "name": "In Progress"},
						"children": {
							"nodes": [
								{
									"id": "issue-1",
									"identifier": "TEAM-123",
									"title": "Sub-task Issue",
									"state": {"id": "state-1", "name": "Todo"}
								},
								{
									"id": "issue-2",
									"identifier": "TEAM-124",
									"title": "Another Sub-task",
									"state": {"id": "state-1", "name": "Todo"}
								}
							]
						}
					}
				}
			}
		}`

		callCount := 0
		client := NewClient("test-token")
		transport := &mockTransportWithFunc{}
		
		// Override RoundTrip to return different responses
		transport.roundTripFunc = func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(basicResponse)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(parentContextResponse)),
			}, nil
		}
		
		client.base.httpClient = &http.Client{Transport: transport}

		issue, err := client.Issues.GetIssueWithBestContext("issue-1")
		if err != nil {
			t.Fatalf("GetIssueWithBestContext failed: %v", err)
		}

		// Should have made 2 calls
		if callCount != 2 {
			t.Errorf("Expected 2 API calls, got %d", callCount)
		}

		// Should have full parent context
		if issue.Parent.Description != "Parent description" {
			t.Error("Expected full parent context")
		}
	})

	t.Run("issue with project context", func(t *testing.T) {
		// First call - basic issue info shows it has a project but no parent
		basicResponse := `{
			"data": {
				"issue": {
					"id": "issue-1",
					"identifier": "TEAM-200",
					"title": "Project Issue",
					"description": "An issue in a project",
					"state": {"id": "state-1", "name": "Todo"},
					"project": {
						"id": "project-1",
						"name": "My Project"
					}
				}
			}
		}`

		// Second call - full project context
		projectContextResponse := `{
			"data": {
				"issue": {
					"id": "issue-1",
					"identifier": "TEAM-200",
					"title": "Project Issue",
					"description": "An issue in a project",
					"state": {"id": "state-1", "name": "Todo"},
					"project": {
						"id": "project-1",
						"name": "My Project",
						"description": "Project description with metadata",
						"state": "started",
						"createdAt": "2024-01-01T00:00:00Z",
						"updatedAt": "2024-01-02T00:00:00Z"
					}
				}
			}
		}`

		callCount := 0
		client := NewClient("test-token")
		transport := &mockTransportWithFunc{}
		
		transport.roundTripFunc = func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(basicResponse)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(projectContextResponse)),
			}, nil
		}
		
		client.base.httpClient = &http.Client{Transport: transport}

		issue, err := client.Issues.GetIssueWithBestContext("issue-1")
		if err != nil {
			t.Fatalf("GetIssueWithBestContext failed: %v", err)
		}

		// Should have made 2 calls
		if callCount != 2 {
			t.Errorf("Expected 2 API calls, got %d", callCount)
		}

		// Should have full project context
		if issue.Project.Description != "Project description with metadata" {
			t.Error("Expected full project context")
		}
	})

	t.Run("standalone issue", func(t *testing.T) {
		// Issue with no parent or project
		response := `{
			"data": {
				"issue": {
					"id": "issue-1",
					"identifier": "TEAM-300",
					"title": "Standalone Issue",
					"description": "An issue without parent or project",
					"state": {"id": "state-1", "name": "Todo"}
				}
			}
		}`

		callCount := 0
		client := NewClient("test-token")
		transport := &mockTransportWithFunc{}
		
		transport.roundTripFunc = func(req *http.Request) (*http.Response, error) {
			callCount++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(response)),
			}, nil
		}
		
		client.base.httpClient = &http.Client{Transport: transport}

		issue, err := client.Issues.GetIssueWithBestContext("issue-1")
		if err != nil {
			t.Fatalf("GetIssueWithBestContext failed: %v", err)
		}

		// Should have made only 1 call
		if callCount != 1 {
			t.Errorf("Expected 1 API call for standalone issue, got %d", callCount)
		}

		if issue.Title != "Standalone Issue" {
			t.Errorf("Expected title 'Standalone Issue', got '%s'", issue.Title)
		}
	})

	t.Run("handles errors gracefully", func(t *testing.T) {
		client := NewClient("test-token")
		transport := &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"errors":[{"message":"Not found"}]}`)),
			},
		}
		client.base.httpClient = &http.Client{Transport: transport}

		_, err := client.Issues.GetIssueWithBestContext("nonexistent")
		if err == nil {
			t.Fatal("Expected error for nonexistent issue")
		}

		if !contains(err.Error(), "Not found") {
			t.Errorf("Expected 'Not found' error, got: %v", err)
		}
	})
}

// mockTransportWithFunc extends the basic mockTransport to allow custom RoundTrip functions
type mockTransportWithFunc struct {
	mockTransport
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransportWithFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.roundTripFunc != nil {
		return m.roundTripFunc(req)
	}
	return m.mockTransport.RoundTrip(req)
}