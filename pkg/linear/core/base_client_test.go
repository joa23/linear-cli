package core

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestBaseClientExecuteRequestGraphQLErrorPreservesExtensionsWithoutQuery(t *testing.T) {
	const query = `mutation SecretOperation($token: String!) {
	  issueCreate(input: {description: $token}) { success }
	}`
	const message = "You do not have permission to perform this action"

	client := newBaseClientForResponse(`{
		"errors": [{
			"message": "` + message + `",
			"extensions": {
				"code": "FORBIDDEN",
				"classification": "authorization",
				"details": {"reason": "insufficient_scope"}
			}
		}]
	}`)

	err := client.ExecuteRequest(query, nil, nil)
	if err == nil {
		t.Fatal("expected GraphQL error")
	}

	var graphQLError *GraphQLError
	if !errors.As(err, &graphQLError) {
		t.Fatalf("expected GraphQLError, got %T: %v", err, err)
	}
	if !IsGraphQLError(err) {
		t.Fatal("expected IsGraphQLError to classify the error")
	}
	if graphQLError.Message != message {
		t.Errorf("message = %q, want %q", graphQLError.Message, message)
	}
	if got := graphQLError.Extensions["code"]; got != "FORBIDDEN" {
		t.Errorf("extensions[code] = %#v, want FORBIDDEN", got)
	}
	if got := graphQLError.Extensions["classification"]; got != "authorization" {
		t.Errorf("extensions[classification] = %#v, want authorization", got)
	}
	details, ok := graphQLError.Extensions["details"].(map[string]interface{})
	if !ok || details["reason"] != "insufficient_scope" {
		t.Errorf("extensions[details] = %#v, want reason insufficient_scope", graphQLError.Extensions["details"])
	}

	errString := err.Error()
	if !strings.Contains(errString, "code: FORBIDDEN") {
		t.Errorf("error = %q, want GraphQL code classification", errString)
	}
	for _, queryFragment := range []string{"SecretOperation", "issueCreate", "$token", "query:"} {
		if strings.Contains(errString, queryFragment) {
			t.Errorf("error = %q contains query fragment %q", errString, queryFragment)
		}
	}
}

func TestBaseClientExecuteRequestGraphQLErrorWithoutExtensionsRemainsTyped(t *testing.T) {
	client := newBaseClientForResponse(`{"errors":[{"message":"validation failed"}]}`)

	err := client.ExecuteRequest("query PrivateQuery { viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected GraphQL error")
	}

	var graphQLError *GraphQLError
	if !errors.As(err, &graphQLError) {
		t.Fatalf("expected GraphQLError, got %T: %v", err, err)
	}
	if graphQLError.Extensions != nil {
		t.Errorf("extensions = %#v, want nil", graphQLError.Extensions)
	}
	if got, want := err.Error(), "GraphQL error: validation failed"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
	if strings.Contains(err.Error(), "PrivateQuery") {
		t.Errorf("error = %q contains query text", err)
	}
}

func TestBaseClientExecuteRequestHTTPErrorRemainsClassified(t *testing.T) {
	client := newBaseClientForResponseWithStatus(http.StatusBadRequest, `{"error":"bad request"}`)

	err := client.ExecuteRequest("query PrivateQuery { viewer { id } }", nil, nil)
	if err == nil {
		t.Fatal("expected HTTP error")
	}

	var httpError *HTTPError
	if !errors.As(err, &httpError) {
		t.Fatalf("expected HTTPError, got %T: %v", err, err)
	}
	if httpError.StatusCode != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", httpError.StatusCode, http.StatusBadRequest)
	}
	if httpError.Body != `{"error":"bad request"}` {
		t.Errorf("body = %q, want response body preserved", httpError.Body)
	}
}

func newBaseClientForResponse(body string) *BaseClient {
	return newBaseClientForResponseWithStatus(http.StatusOK, body)
}

func newBaseClientForResponseWithStatus(statusCode int, body string) *BaseClient {
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	return NewTestBaseClient("test-token", "https://example.test/graphql", httpClient)
}
