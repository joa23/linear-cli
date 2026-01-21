package linear

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateCommentReply(t *testing.T) {
	tests := []struct {
		name          string
		issueID       string
		parentID      string
		body          string
		mockResponse  string
		expectedID    string
		expectedError bool
	}{
		{
			name:     "successful reply creation",
			issueID:  "issue-123",
			parentID: "comment-parent-456",
			body:     "This is a reply to the parent comment",
			mockResponse: `{
				"data": {
					"commentCreate": {
						"success": true,
						"comment": {
							"id": "comment-reply-789",
							"body": "This is a reply to the parent comment",
							"createdAt": "2024-01-01T00:00:00Z",
							"parent": {
								"id": "comment-parent-456"
							}
						}
					}
				}
			}`,
			expectedID:    "comment-reply-789",
			expectedError: false,
		},
		{
			name:     "reply creation failure",
			issueID:  "issue-123",
			parentID: "comment-invalid",
			body:     "This reply will fail",
			mockResponse: `{
				"data": {
					"commentCreate": {
						"success": false
					}
				}
			}`,
			expectedID:    "",
			expectedError: true,
		},
		{
			name:     "GraphQL error",
			issueID:  "issue-123",
			parentID: "comment-456",
			body:     "This will cause an error",
			mockResponse: `{
				"errors": [
					{
						"message": "Parent comment not found",
						"extensions": {
							"code": "NOT_FOUND"
						}
					}
				]
			}`,
			expectedID:    "",
			expectedError: true,
		},
		{
			name:          "empty body",
			issueID:       "issue-123",
			parentID:      "comment-456",
			body:          "",
			mockResponse:  "",
			expectedID:    "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if body is empty (we'll validate before making request)
			if tt.body == "" {
				client := &Client{}
				_, err := client.CreateCommentReply(tt.issueID, tt.parentID, tt.body)
				if err == nil {
					t.Error("Expected error for empty body, got nil")
				}
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				var requestBody struct {
					Query     string                 `json:"query"`
					Variables map[string]interface{} `json:"variables"`
				}
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				// Verify mutation includes parent field
				if !contains(requestBody.Query, "mutation CreateCommentReply") {
					t.Error("Expected CreateCommentReply mutation")
				}
				if !contains(requestBody.Query, "parentId:") {
					t.Error("Expected parentId in mutation input")
				}

				// Verify variables
				if issueId, ok := requestBody.Variables["issueId"].(string); !ok || issueId != tt.issueID {
					t.Errorf("Expected issueId=%s, got %v", tt.issueID, requestBody.Variables["issueId"])
				}
				if parentId, ok := requestBody.Variables["parentId"].(string); !ok || parentId != tt.parentID {
					t.Errorf("Expected parentId=%s, got %v", tt.parentID, requestBody.Variables["parentId"])
				}
				if body, ok := requestBody.Variables["body"].(string); !ok || body != tt.body {
					t.Errorf("Expected body=%s, got %v", tt.body, requestBody.Variables["body"])
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Create a client with custom base URL for testing
			client := NewClient("test-token")
			client.base.baseURL = server.URL

			commentID, err := client.CreateCommentReply(tt.issueID, tt.parentID, tt.body)

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if commentID != tt.expectedID {
				t.Errorf("Expected comment ID %s, got %s", tt.expectedID, commentID)
			}
		})
	}
}

func TestGetCommentWithReplies(t *testing.T) {
	tests := []struct {
		name           string
		issueID        string
		commentID      string
		mockResponse   string
		expectedReplies int
		expectedError  bool
	}{
		{
			name:      "comment with replies",
			issueID:   "issue-123",
			commentID: "comment-456",
			mockResponse: `{
				"data": {
					"comment": {
						"id": "comment-456",
						"body": "Parent comment",
						"createdAt": "2024-01-01T00:00:00Z",
						"user": {
							"id": "user-1",
							"name": "User One"
						},
						"children": {
							"nodes": [
								{
									"id": "reply-1",
									"body": "First reply",
									"createdAt": "2024-01-01T01:00:00Z",
									"user": {
										"id": "user-2",
										"name": "User Two"
									}
								},
								{
									"id": "reply-2",
									"body": "Second reply",
									"createdAt": "2024-01-01T02:00:00Z",
									"user": {
										"id": "user-3",
										"name": "User Three"
									}
								}
							]
						}
					}
				}
			}`,
			expectedReplies: 2,
			expectedError:   false,
		},
		{
			name:      "comment without replies",
			issueID:   "issue-123",
			commentID: "comment-789",
			mockResponse: `{
				"data": {
					"comment": {
						"id": "comment-789",
						"body": "Comment with no replies",
						"createdAt": "2024-01-01T00:00:00Z",
						"user": {
							"id": "user-1",
							"name": "User One"
						},
						"children": {
							"nodes": []
						}
					}
				}
			}`,
			expectedReplies: 0,
			expectedError:   false,
		},
		{
			name:      "comment not found",
			issueID:   "issue-123",
			commentID: "comment-invalid",
			mockResponse: `{
				"data": {
					"comment": null
				}
			}`,
			expectedReplies: 0,
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				var requestBody struct {
					Query     string                 `json:"query"`
					Variables map[string]interface{} `json:"variables"`
				}
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				// Verify query
				if !contains(requestBody.Query, "query GetCommentWithReplies") {
					t.Error("Expected GetCommentWithReplies query")
				}
				if !contains(requestBody.Query, "children") {
					t.Error("Expected children field in query")
				}

				// Verify variables
				if id, ok := requestBody.Variables["id"].(string); !ok || id != tt.commentID {
					t.Errorf("Expected id=%s, got %v", tt.commentID, requestBody.Variables["id"])
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Create a client with custom base URL for testing
			client := NewClient("test-token")
			client.base.baseURL = server.URL

			comment, err := client.GetCommentWithReplies(tt.commentID)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if comment == nil {
				t.Error("Expected comment but got nil")
				return
			}

			if len(comment.Replies) != tt.expectedReplies {
				t.Errorf("Expected %d replies, got %d", tt.expectedReplies, len(comment.Replies))
			}

			// Verify reply content for the test with replies
			if tt.expectedReplies > 0 {
				if comment.Replies[0].ID != "reply-1" {
					t.Errorf("Expected first reply ID to be reply-1, got %s", comment.Replies[0].ID)
				}
				if comment.Replies[0].Body != "First reply" {
					t.Errorf("Expected first reply body to be 'First reply', got %s", comment.Replies[0].Body)
				}
			}
		})
	}
}