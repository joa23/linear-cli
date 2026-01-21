package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// TestGetIssueReturnsIdentifier tests that GetIssue returns the issue identifier
func TestGetIssueReturnsIdentifier(t *testing.T) {
	t.Run("GetIssue should return issue identifier", func(t *testing.T) {
		// Create mock response with identifier field
		mockResponse := `{
			"data": {
				"issue": {
					"id": "test-issue-123",
					"identifier": "CEN-42",
					"title": "Test Issue",
					"description": "Test Description",
					"state": {
						"name": "Todo"
					},
					"project": {
						"name": "Test Project",
						"description": "Test Project Description"
					},
					"creator": {
						"id": "creator-123",
						"name": "Test Creator"
					}
				}
			}
		}`
		
		// Create client with mocked transport
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
				},
			},
		}
		
		// Execute GetIssue
		issue, err := client.GetIssue("test-issue-123")
		
		// Verify no error occurred
		if err != nil {
			t.Fatalf("Expected successful GetIssue, got error: %v", err)
		}
		
		// Verify issue is not nil
		if issue == nil {
			t.Fatal("Expected issue to be returned, got nil")
		}
		
		// Verify identifier is returned
		if issue.Identifier == "" {
			t.Error("Expected issue identifier to be returned, but it was empty")
		}
		
		// Verify identifier has correct value
		expectedIdentifier := "CEN-42"
		if issue.Identifier != expectedIdentifier {
			t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, issue.Identifier)
		}
	})
	
	t.Run("GetIssueWithProjectContext should return issue identifier", func(t *testing.T) {
		// Create mock response with identifier field  
		mockResponse := `{
			"data": {
				"issue": {
					"id": "test-issue-456",
					"identifier": "CEN-99",
					"title": "Test Issue with Project",
					"description": "Test Description",
					"state": {
						"name": "In Progress"
					},
					"assignee": {
						"id": "user-123",
						"name": "Test User"
					},
					"creator": {
						"id": "creator-456",
						"name": "Test Creator"
					},
					"project": {
						"id": "project-789",
						"name": "Test Project",
						"description": "Test Project Description",
						"issues": {
							"nodes": []
						}
					},
					"comments": {
						"nodes": []
					}
				}
			}
		}`
		
		// Create client with mocked transport
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
				},
			},
		}
		
		// Execute GetIssueWithProjectContext
		issue, err := client.GetIssueWithProjectContext("test-issue-456")
		
		// Verify no error occurred
		if err != nil {
			t.Fatalf("Expected successful GetIssueWithProjectContext, got error: %v", err)
		}
		
		// Verify identifier is returned
		expectedIdentifier := "CEN-99"
		if issue.Identifier != expectedIdentifier {
			t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, issue.Identifier)
		}
	})
}