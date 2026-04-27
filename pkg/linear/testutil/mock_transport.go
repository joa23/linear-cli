package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// MockTransport implements http.RoundTripper for testing HTTP clients.
type MockTransport struct {
	Response *http.Response
}

// RoundTrip implements the http.RoundTripper interface.
func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.Response, nil
}

// NewMockTransport creates a mock transport with a JSON response body.
func NewMockTransport(statusCode int, body interface{}) *MockTransport {
	var bodyBytes []byte
	switch v := body.(type) {
	case string:
		bodyBytes = []byte(v)
	case []byte:
		bodyBytes = v
	default:
		bodyBytes, _ = json.Marshal(body)
	}

	return &MockTransport{
		Response: &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewBuffer(bodyBytes)),
			Header:     make(http.Header),
		},
	}
}

// NewSuccessTransport creates a mock transport with a 200 OK response.
func NewSuccessTransport(body interface{}) *MockTransport {
	return NewMockTransport(http.StatusOK, body)
}

// GraphQLResponse is a helper for building GraphQL response bodies.
type GraphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message    string                 `json:"message"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// NewGraphQLDataResponse creates a successful GraphQL response with data.
func NewGraphQLDataResponse(data interface{}) GraphQLResponse {
	return GraphQLResponse{Data: data}
}

// NewGraphQLErrorResponse creates a GraphQL error response.
func NewGraphQLErrorResponse(message, code string) GraphQLResponse {
	return GraphQLResponse{
		Errors: []GraphQLError{
			{
				Message: message,
				Extensions: map[string]interface{}{
					"code": code,
				},
			},
		},
	}
}

// IssueData wraps issue data for GraphQL responses.
type IssueData struct {
	Issue interface{} `json:"issue"`
}

// IssuesData wraps issues list data for GraphQL responses.
type IssuesData struct {
	Issues IssuesNodes `json:"issues"`
}

// IssuesNodes contains the nodes array for paginated results.
type IssuesNodes struct {
	Nodes    []interface{} `json:"nodes"`
	PageInfo *PageInfo     `json:"pageInfo,omitempty"`
}

// PageInfo contains pagination information.
type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor,omitempty"`
}

// TeamsData wraps teams data for GraphQL responses.
type TeamsData struct {
	Teams TeamsNodes `json:"teams"`
}

// TeamsNodes contains the nodes array for teams.
type TeamsNodes struct {
	Nodes []interface{} `json:"nodes"`
}

// ViewerData wraps viewer data for GraphQL responses.
type ViewerData struct {
	Viewer interface{} `json:"viewer"`
}

// IssueCreateData wraps issue creation response.
type IssueCreateData struct {
	IssueCreate IssueCreateResult `json:"issueCreate"`
}

// IssueCreateResult contains issue creation result.
type IssueCreateResult struct {
	Success bool        `json:"success"`
	Issue   interface{} `json:"issue"`
}

// IssueUpdateData wraps issue update response.
type IssueUpdateData struct {
	IssueUpdate IssueUpdateResult `json:"issueUpdate"`
}

// IssueUpdateResult contains issue update result.
type IssueUpdateResult struct {
	Success bool        `json:"success"`
	Issue   interface{} `json:"issue"`
}

// CommentCreateData wraps comment creation response.
type CommentCreateData struct {
	CommentCreate CommentCreateResult `json:"commentCreate"`
}

// CommentCreateResult contains comment creation result.
type CommentCreateResult struct {
	Success bool        `json:"success"`
	Comment interface{} `json:"comment"`
}

// ReactionCreateData wraps reaction creation response.
type ReactionCreateData struct {
	ReactionCreate ReactionCreateResult `json:"reactionCreate"`
}

// ReactionCreateResult contains reaction creation result.
type ReactionCreateResult struct {
	Success  bool        `json:"success"`
	Reaction interface{} `json:"reaction"`
}
