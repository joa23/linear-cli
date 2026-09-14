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

// CapturingTransport implements http.RoundTripper for testing HTTP clients.
// It records the outgoing request body (so tests can assert on the GraphQL
// query text that was actually sent) while returning a canned response.
type CapturingTransport struct {
	Response *http.Response

	// CapturedBody holds the raw bytes of the last request body seen by
	// RoundTrip. Populated after the request under test has been made.
	CapturedBody []byte
}

// RoundTrip implements the http.RoundTripper interface.
func (c *CapturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close()
		c.CapturedBody = body
	}
	return c.Response, nil
}

// NewCapturingTransport creates a capturing transport with a JSON response body.
func NewCapturingTransport(statusCode int, body interface{}) *CapturingTransport {
	var bodyBytes []byte
	switch v := body.(type) {
	case string:
		bodyBytes = []byte(v)
	case []byte:
		bodyBytes = v
	default:
		bodyBytes, _ = json.Marshal(body)
	}

	return &CapturingTransport{
		Response: &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewBuffer(bodyBytes)),
			Header:     make(http.Header),
		},
	}
}

// CapturedQuery extracts the "query" field from the captured GraphQL request
// body, so tests can assert on the selection set that was actually sent.
func (c *CapturingTransport) CapturedQuery() (string, error) {
	var payload struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(c.CapturedBody, &payload); err != nil {
		return "", err
	}
	return payload.Query, nil
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
