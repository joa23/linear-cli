package linear

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestStateChangeIntegration tests the state change functionality
func TestStateChangeIntegration(t *testing.T) {
	t.Run("change issue state successfully", func(t *testing.T) {
		// Create mock response for successful state change
		mockResponse := `{
			"data": {
				"issueUpdate": {
					"success": true,
					"issue": {
						"id": "integration-test-issue",
						"state": {
							"name": "Done"
						}
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
		
		// Execute state change
		err := client.UpdateIssueState("integration-test-issue", "done-state-id")
		
		// Verify no error occurred
		if err != nil {
			t.Fatalf("Expected successful state change, got error: %v", err)
		}
	})
	
	t.Run("handle state change error", func(t *testing.T) {
		// Create mock response for failed state change
		mockResponse := `{
			"errors": [{
				"message": "Cannot transition from Done to In Progress",
				"extensions": {
					"code": "INVALID_STATE_TRANSITION"
				}
			}]
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
		
		// Execute invalid state change
		err := client.UpdateIssueState("integration-test-issue", "in-progress-state-id")
		
		// Verify error occurred
		if err == nil {
			t.Fatal("Expected error for invalid state transition, got nil")
		}
		
		// Verify error message contains the key error
		expectedError := "Cannot transition from Done to In Progress"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
		}
	})
}

// TestAssignmentIntegration tests the assignment functionality  
func TestAssignmentIntegration(t *testing.T) {
	t.Run("assign issue to user successfully", func(t *testing.T) {
		// Create mock response for successful assignment
		mockResponse := `{
			"data": {
				"issueUpdate": {
					"success": true,
					"issue": {
						"id": "integration-test-issue",
						"assignee": {
							"id": "550e8400-e29b-41d4-a716-446655440006",
							"name": "Test User"
						}
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
		
		// Execute assignment
		err := client.AssignIssue("integration-test-issue", "550e8400-e29b-41d4-a716-446655440006")
		
		// Verify no error occurred
		if err != nil {
			t.Fatalf("Expected successful assignment, got error: %v", err)
		}
	})
	
	t.Run("unassign issue", func(t *testing.T) {
		// Create mock response for unassigning (assignee is null)
		mockResponse := `{
			"data": {
				"issueUpdate": {
					"success": true,
					"issue": {
						"id": "integration-test-issue",
						"assignee": null
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
		
		// Execute unassignment by passing empty string
		err := client.AssignIssue("integration-test-issue", "")
		
		// Verify no error occurred
		if err != nil {
			t.Fatalf("Expected successful unassignment, got error: %v", err)
		}
	})
	
	t.Run("handle assignment error", func(t *testing.T) {
		// Create mock response for failed assignment
		mockResponse := `{
			"errors": [{
				"message": "User with ID 'invalid-user' not found",
				"extensions": {
					"code": "USER_NOT_FOUND"
				}
			}]
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
		
		// Execute assignment with invalid user
		err := client.AssignIssue("integration-test-issue", "invalid-user")
		
		// Verify error occurred
		if err == nil {
			t.Fatal("Expected error for invalid user, got nil")
		}
		
		// Verify error message contains the key error
		expectedError := "User with ID 'invalid-user' not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
		}
	})
}

// TestCombinedWorkflow tests state change and assignment together
func TestCombinedWorkflow(t *testing.T) {
	t.Run("assign and change state in workflow", func(t *testing.T) {
		// Track calls to ensure both operations are executed
		callCount := 0
		
		// Create client with mocked transport that returns different responses
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockRoundTripper{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					callCount++
					
					// Read the request body to determine which operation
					bodyBytes, _ := io.ReadAll(req.Body)
					req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					
					// First call: assign issue
					if callCount == 1 {
						mockResponse := `{
							"data": {
								"issueUpdate": {
									"success": true,
									"issue": {
										"id": "workflow-test-issue",
										"assignee": {
											"id": "550e8400-e29b-41d4-a716-446655440005",
											"name": "Workflow User"
										}
									}
								}
							}
						}`
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
						}, nil
					}
					
					// Second call: change state
					mockResponse := `{
						"data": {
							"issueUpdate": {
								"success": true,
								"issue": {
									"id": "workflow-test-issue",
									"state": {
										"name": "In Review"
									}
								}
							}
						}
					}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
					}, nil
				},
			},
		}
		
		// Step 1: Assign issue to user
		err := client.AssignIssue("workflow-test-issue", "550e8400-e29b-41d4-a716-446655440005")
		if err != nil {
			t.Fatalf("Failed to assign issue: %v", err)
		}
		
		// Step 2: Change issue state
		err = client.UpdateIssueState("workflow-test-issue", "in-review-state-id")
		if err != nil {
			t.Fatalf("Failed to change state: %v", err)
		}
		
		// Verify both operations were called
		if callCount != 2 {
			t.Errorf("Expected 2 API calls, got %d", callCount)
		}
	})
}

// mockRoundTripper allows us to mock multiple sequential requests
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

// TestParentChildContextIntegration verifies that GetIssue returns complete parent and child context
// including ID, identifier (key), and title for both parent and child issues
func TestParentChildContextIntegration(t *testing.T) {
	t.Run("parent issue context includes all required fields", func(t *testing.T) {
		response := `{
			"data": {
				"issue": {
					"id": "550e8400-e29b-41d4-a716-446655440016",
					"identifier": "PROJ-456",
					"title": "Child Issue",
					"description": "This is a child issue",
					"state": {"id": "state-1", "name": "Todo"},
					"assignee": null,
					"createdAt": "2024-01-01T00:00:00Z",
					"updatedAt": "2024-01-01T00:00:00Z",
					"url": "https://linear.app/team/issue/PROJ-456",
					"project": null,
					"parent": {
						"id": "550e8400-e29b-41d4-a716-446655440017",
						"identifier": "PROJ-123",
						"title": "Parent Issue Title"
					},
					"children": {
						"nodes": []
					},
					"attachments": {
						"nodes": []
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(response)),
				},
			},
		}

		issue, err := client.Issues.GetIssue("550e8400-e29b-41d4-a716-446655440016")
		if err != nil {
			t.Fatalf("GetIssue failed: %v", err)
		}

		// Verify issue has parent
		if issue.Parent == nil {
			t.Fatal("Expected issue to have parent, but parent was nil")
		}

		// Verify parent context includes all required fields
		if issue.Parent.ID != "550e8400-e29b-41d4-a716-446655440017" {
			t.Errorf("Expected parent ID '550e8400-e29b-41d4-a716-446655440017', got '%s'", issue.Parent.ID)
		}

		if issue.Parent.Identifier != "PROJ-123" {
			t.Errorf("Expected parent identifier 'PROJ-123', got '%s'", issue.Parent.Identifier)
		}

		if issue.Parent.Title != "Parent Issue Title" {
			t.Errorf("Expected parent title 'Parent Issue Title', got '%s'", issue.Parent.Title)
		}
	})

	t.Run("child issues context includes all required fields", func(t *testing.T) {
		response := `{
			"data": {
				"issue": {
					"id": "550e8400-e29b-41d4-a716-446655440017",
					"identifier": "PROJ-123",
					"title": "Parent Issue",
					"description": "This is a parent issue",
					"state": {"id": "state-1", "name": "In Progress"},
					"assignee": null,
					"createdAt": "2024-01-01T00:00:00Z",
					"updatedAt": "2024-01-01T00:00:00Z",
					"url": "https://linear.app/team/issue/PROJ-123",
					"project": null,
					"parent": null,
					"children": {
						"nodes": [
							{
								"id": "550e8400-e29b-41d4-a716-446655440016",
								"identifier": "PROJ-456",
								"title": "First Child Issue",
								"state": {"id": "state-2", "name": "Todo"}
							},
							{
								"id": "550e8400-e29b-41d4-a716-446655440018",
								"identifier": "PROJ-457",
								"title": "Second Child Issue",
								"state": {"id": "state-3", "name": "Done"}
							}
						]
					},
					"attachments": {
						"nodes": []
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(response)),
				},
			},
		}

		issue, err := client.Issues.GetIssue("550e8400-e29b-41d4-a716-446655440017")
		if err != nil {
			t.Fatalf("GetIssue failed: %v", err)
		}

		// Verify issue has children
		if len(issue.Children.Nodes) != 2 {
			t.Fatalf("Expected 2 children, got %d", len(issue.Children.Nodes))
		}

		// Verify first child context includes all required fields
		child1 := issue.Children.Nodes[0]
		if child1.ID != "550e8400-e29b-41d4-a716-446655440016" {
			t.Errorf("Expected child 1 ID '550e8400-e29b-41d4-a716-446655440016', got '%s'", child1.ID)
		}

		if child1.Identifier != "PROJ-456" {
			t.Errorf("Expected child 1 identifier 'PROJ-456', got '%s'", child1.Identifier)
		}

		if child1.Title != "First Child Issue" {
			t.Errorf("Expected child 1 title 'First Child Issue', got '%s'", child1.Title)
		}

		// Verify second child context includes all required fields
		child2 := issue.Children.Nodes[1]
		if child2.ID != "550e8400-e29b-41d4-a716-446655440018" {
			t.Errorf("Expected child 2 ID '550e8400-e29b-41d4-a716-446655440018', got '%s'", child2.ID)
		}

		if child2.Identifier != "PROJ-457" {
			t.Errorf("Expected child 2 identifier 'PROJ-457', got '%s'", child2.Identifier)
		}

		if child2.Title != "Second Child Issue" {
			t.Errorf("Expected child 2 title 'Second Child Issue', got '%s'", child2.Title)
		}
	})

	t.Run("complete parent-child-grandchild hierarchy", func(t *testing.T) {
		// Test a grandchild issue that has both parent context and siblings
		response := `{
			"data": {
				"issue": {
					"id": "550e8400-e29b-41d4-a716-446655440019",
					"identifier": "PROJ-789",
					"title": "Grandchild Issue",
					"description": "This is a grandchild issue",
					"state": {"id": "state-1", "name": "Todo"},
					"assignee": null,
					"createdAt": "2024-01-01T00:00:00Z",
					"updatedAt": "2024-01-01T00:00:00Z",
					"url": "https://linear.app/team/issue/PROJ-789",
					"project": null,
					"parent": {
						"id": "550e8400-e29b-41d4-a716-446655440016",
						"identifier": "PROJ-456",
						"title": "Child Issue (Parent of Grandchild)"
					},
					"children": {
						"nodes": []
					},
					"attachments": {
						"nodes": []
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{
			Transport: &mockTransport{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(response)),
				},
			},
		}

		issue, err := client.Issues.GetIssue("550e8400-e29b-41d4-a716-446655440019")
		if err != nil {
			t.Fatalf("GetIssue failed: %v", err)
		}

		// Verify grandchild has proper parent context
		if issue.Parent == nil {
			t.Fatal("Expected grandchild to have parent, but parent was nil")
		}

		if issue.Parent.ID != "550e8400-e29b-41d4-a716-446655440016" {
			t.Errorf("Expected parent ID '550e8400-e29b-41d4-a716-446655440016', got '%s'", issue.Parent.ID)
		}

		if issue.Parent.Identifier != "PROJ-456" {
			t.Errorf("Expected parent identifier 'PROJ-456', got '%s'", issue.Parent.Identifier)
		}

		if issue.Parent.Title != "Child Issue (Parent of Grandchild)" {
			t.Errorf("Expected parent title 'Child Issue (Parent of Grandchild)', got '%s'", issue.Parent.Title)
		}

		// Verify grandchild itself has correct context
		if issue.ID != "550e8400-e29b-41d4-a716-446655440019" {
			t.Errorf("Expected issue ID '550e8400-e29b-41d4-a716-446655440019', got '%s'", issue.ID)
		}

		if issue.Identifier != "PROJ-789" {
			t.Errorf("Expected issue identifier 'PROJ-789', got '%s'", issue.Identifier)
		}

		if issue.Title != "Grandchild Issue" {
			t.Errorf("Expected issue title 'Grandchild Issue', got '%s'", issue.Title)
		}
	})
}