package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestSubIssueWithIdentifier(t *testing.T) {
	// Test that SubIssue struct includes identifier field
	t.Run("GetSubIssues returns identifier field", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"issue": {
					"id": "parent-123",
					"title": "Parent Issue",
					"children": {
						"nodes": [
							{
								"id": "sub-1",
								"identifier": "PROJ-124",
								"title": "First sub-task",
								"state": {
									"name": "Todo"
								}
							},
							{
								"id": "sub-2",
								"identifier": "PROJ-125",
								"title": "Second sub-task",
								"state": {
									"name": "In Progress"
								}
							}
						]
					}
				}
			}
		}`
		
		// Create a client with mocked HTTP transport
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
				},
			},
		}
		
		// Get sub-issues
		subIssues, err := client.GetSubIssues("parent-123")
		if err != nil {
			t.Fatalf("GetSubIssues failed: %v", err)
		}
		
		// Verify we got 2 sub-issues
		if len(subIssues) != 2 {
			t.Errorf("Expected 2 sub-issues, got %d", len(subIssues))
		}
		
		// Verify first sub-issue has identifier
		if subIssues[0].Identifier != "PROJ-124" {
			t.Errorf("Expected identifier 'PROJ-124', got '%s'", subIssues[0].Identifier)
		}
		if subIssues[0].ID != "sub-1" {
			t.Errorf("Expected ID 'sub-1', got '%s'", subIssues[0].ID)
		}
		if subIssues[0].Title != "First sub-task" {
			t.Errorf("Expected title 'First sub-task', got '%s'", subIssues[0].Title)
		}
		
		// Verify second sub-issue
		if subIssues[1].Identifier != "PROJ-125" {
			t.Errorf("Expected identifier 'PROJ-125', got '%s'", subIssues[1].Identifier)
		}
	})
	
	t.Run("handle sub-issues without identifier gracefully", func(t *testing.T) {
		// Some older issues might not have identifiers
		mockResponseBody := `{
			"data": {
				"issue": {
					"id": "parent-456",
					"title": "Old Parent Issue",
					"children": {
						"nodes": [
							{
								"id": "old-sub-1",
								"title": "Old sub-task",
								"state": {
									"name": "Done"
								}
							}
						]
					}
				}
			}
		}`
		
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
				},
			},
		}
		
		subIssues, err := client.GetSubIssues("parent-456")
		if err != nil {
			t.Fatalf("GetSubIssues failed: %v", err)
		}
		
		// Should handle missing identifier gracefully
		if len(subIssues) != 1 {
			t.Errorf("Expected 1 sub-issue, got %d", len(subIssues))
		}
		
		// Identifier should be empty string when not provided
		if subIssues[0].Identifier != "" {
			t.Errorf("Expected empty identifier, got '%s'", subIssues[0].Identifier)
		}
	})
}

func TestIssueWithChildren(t *testing.T) {
	// Test that Issue struct includes children field when fetched
	t.Run("GetIssue returns children field", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"issue": {
					"id": "parent-789",
					"identifier": "PROJ-789",
					"title": "Parent task with subtasks",
					"description": "This has children",
					"state": {
						"name": "In Progress"
					},
					"project": {
						"name": "Test Project",
						"description": "Test project description"
					},
					"creator": {
						"id": "user-1",
						"name": "Test User"
					},
					"children": {
						"nodes": [
							{
								"id": "child-1",
								"identifier": "PROJ-790",
								"title": "Child task 1",
								"state": {
									"name": "Todo"
								}
							},
							{
								"id": "child-2",
								"identifier": "PROJ-791",
								"title": "Child task 2",
								"state": {
									"name": "Done"
								}
							}
						]
					}
				}
			}
		}`
		
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
				},
			},
		}
		
		issue, err := client.GetIssue("parent-789")
		if err != nil {
			t.Fatalf("GetIssue failed: %v", err)
		}
		
		// Verify issue has children
		if len(issue.Children.Nodes) != 2 {
			t.Errorf("Expected 2 children, got %d", len(issue.Children.Nodes))
		}
		
		// Verify first child
		if issue.Children.Nodes[0].ID != "child-1" {
			t.Errorf("Expected child ID 'child-1', got '%s'", issue.Children.Nodes[0].ID)
		}
		if issue.Children.Nodes[0].Identifier != "PROJ-790" {
			t.Errorf("Expected child identifier 'PROJ-790', got '%s'", issue.Children.Nodes[0].Identifier)
		}
		if issue.Children.Nodes[0].Title != "Child task 1" {
			t.Errorf("Expected child title 'Child task 1', got '%s'", issue.Children.Nodes[0].Title)
		}
		if issue.Children.Nodes[0].State.Name != "Todo" {
			t.Errorf("Expected child state 'Todo', got '%s'", issue.Children.Nodes[0].State.Name)
		}
		
		// Verify second child
		if issue.Children.Nodes[1].Identifier != "PROJ-791" {
			t.Errorf("Expected child identifier 'PROJ-791', got '%s'", issue.Children.Nodes[1].Identifier)
		}
		if issue.Children.Nodes[1].State.Name != "Done" {
			t.Errorf("Expected child state 'Done', got '%s'", issue.Children.Nodes[1].State.Name)
		}
	})
	
	t.Run("GetIssue handles issues without children", func(t *testing.T) {
		mockResponseBody := `{
			"data": {
				"issue": {
					"id": "leaf-123",
					"identifier": "PROJ-999",
					"title": "Leaf task without children",
					"description": "No children here",
					"state": {
						"name": "Todo"
					},
					"project": {
						"name": "Test Project",
						"description": "Test project description"
					},
					"creator": {
						"id": "user-1",
						"name": "Test User"
					}
				}
			}
		}`
		
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
				},
			},
		}
		
		issue, err := client.GetIssue("leaf-123")
		if err != nil {
			t.Fatalf("GetIssue failed: %v", err)
		}
		
		// Verify issue has no children
		if len(issue.Children.Nodes) != 0 {
			t.Errorf("Expected 0 children, got %d", len(issue.Children.Nodes))
		}
		
		// Verify other fields are still populated
		if issue.Identifier != "PROJ-999" {
			t.Errorf("Expected identifier 'PROJ-999', got '%s'", issue.Identifier)
		}
	})
}