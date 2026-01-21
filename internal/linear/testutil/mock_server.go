// Package testutil provides testing utilities for Linear client tests.
//
// This package provides two approaches for mocking Linear API calls in tests:
//
// 1. GoQL Mock Server (graphql_test.Server): Higher-level mocking at the GraphQL
//    operation level. Good for testing complete request/response cycles.
//
// 2. Mock Transport (MockTransport): Lower-level mocking at the HTTP level.
//    Good for fine-grained control over raw responses.
//
// Example using mock transport:
//
//	mockResponse := `{"data": {"viewer": {"id": "123"}}}`
//	client := linear.NewClient("test-token")
//	client.GetBase().SetHTTPClient(&http.Client{Transport: testutil.NewSuccessTransport(mockResponse)})
package testutil

import (
	"testing"

	"github.com/getoutreach/goql/graphql_test"
)

// NewLinearMockServer creates a new GraphQL mock server for Linear tests.
// The server is automatically cleaned up when the test finishes.
func NewLinearMockServer(t *testing.T) *graphql_test.Server {
	t.Helper()
	ts := graphql_test.NewServer(t, false)
	t.Cleanup(ts.Close)
	return ts
}

// RegisterIssueQuery registers a mock response for an issue query.
func RegisterIssueQuery(ts *graphql_test.Server, issueID string, issue interface{}) {
	ts.RegisterQuery(graphql_test.Operation{
		Identifier: "issue",
		Variables:  map[string]interface{}{"id": issueID},
		Response:   issue,
	})
}

// RegisterIssueByIdentifierQuery registers a mock response for an issue query by identifier.
func RegisterIssueByIdentifierQuery(ts *graphql_test.Server, identifier string, issue interface{}) {
	ts.RegisterQuery(graphql_test.Operation{
		Identifier: "issue",
		Variables:  map[string]interface{}{"id": identifier},
		Response:   issue,
	})
}

// RegisterTeamsQuery registers a mock response for the teams query.
func RegisterTeamsQuery(ts *graphql_test.Server, teams interface{}) {
	ts.RegisterQuery(graphql_test.Operation{
		Identifier: "teams",
		Variables:  nil,
		Response:   teams,
	})
}

// RegisterViewerQuery registers a mock response for the viewer query.
func RegisterViewerQuery(ts *graphql_test.Server, viewer interface{}) {
	ts.RegisterQuery(graphql_test.Operation{
		Identifier: "viewer",
		Variables:  nil,
		Response:   viewer,
	})
}

// RegisterIssuesQuery registers a mock response for the issues query.
func RegisterIssuesQuery(ts *graphql_test.Server, filters map[string]interface{}, issues interface{}) {
	ts.RegisterQuery(graphql_test.Operation{
		Identifier: "issues",
		Variables:  filters,
		Response:   issues,
	})
}

// RegisterCreateIssueMutation registers a mock response for issue creation.
func RegisterCreateIssueMutation(ts *graphql_test.Server, input map[string]interface{}, response interface{}) {
	ts.RegisterMutation(graphql_test.Operation{
		Identifier: "issueCreate",
		Variables:  input,
		Response:   response,
	})
}

// RegisterUpdateIssueMutation registers a mock response for issue update.
func RegisterUpdateIssueMutation(ts *graphql_test.Server, issueID string, input map[string]interface{}, response interface{}) {
	vars := map[string]interface{}{
		"id":    issueID,
		"input": input,
	}
	ts.RegisterMutation(graphql_test.Operation{
		Identifier: "issueUpdate",
		Variables:  vars,
		Response:   response,
	})
}

// RegisterCommentCreateMutation registers a mock response for comment creation.
func RegisterCommentCreateMutation(ts *graphql_test.Server, issueID string, body string, response interface{}) {
	ts.RegisterMutation(graphql_test.Operation{
		Identifier: "commentCreate",
		Variables: map[string]interface{}{
			"issueId": issueID,
			"body":    body,
		},
		Response: response,
	})
}

// RegisterReactionCreateMutation registers a mock response for reaction creation.
func RegisterReactionCreateMutation(ts *graphql_test.Server, issueID, emoji string, response interface{}) {
	ts.RegisterMutation(graphql_test.Operation{
		Identifier: "reactionCreate",
		Variables: map[string]interface{}{
			"id":    issueID,
			"emoji": emoji,
		},
		Response: response,
	})
}

// RegisterErrorResponse registers an error response for an operation.
func RegisterErrorResponse(ts *graphql_test.Server, operationName string, status int, err error) {
	ts.RegisterError(graphql_test.OperationError{
		Identifier: operationName,
		Status:     status,
		Error:      err,
	})
}

// IssueResponse is a helper type for building issue mock responses.
type IssueResponse struct {
	ID          string       `json:"id"`
	Identifier  string       `json:"identifier"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	State       *StateInfo   `json:"state,omitempty"`
	Team        *TeamInfo    `json:"team,omitempty"`
	Creator     *UserInfo    `json:"creator,omitempty"`
	Assignee    *UserInfo    `json:"assignee,omitempty"`
	Project     *ProjectInfo `json:"project,omitempty"`
	URL         string       `json:"url,omitempty"`
}

// StateInfo represents workflow state information.
type StateInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// TeamInfo represents team information.
type TeamInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// UserInfo represents user information.
type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// ProjectInfo represents project information.
type ProjectInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// NewTestIssue creates a standard test issue response.
func NewTestIssue(id, identifier, title string) IssueResponse {
	return IssueResponse{
		ID:         id,
		Identifier: identifier,
		Title:      title,
		State: &StateInfo{
			ID:   "state-todo",
			Name: "Todo",
			Type: "unstarted",
		},
		Team: &TeamInfo{
			ID:   "team-test",
			Name: "Test Team",
			Key:  "TEST",
		},
		URL: "https://linear.app/test/issue/" + identifier,
	}
}

// NewTestTeam creates a standard test team response.
func NewTestTeam(id, name, key string) TeamInfo {
	return TeamInfo{
		ID:   id,
		Name: name,
		Key:  key,
	}
}

// NewTestUser creates a standard test user response.
func NewTestUser(id, name, email string) UserInfo {
	return UserInfo{
		ID:    id,
		Name:  name,
		Email: email,
	}
}
