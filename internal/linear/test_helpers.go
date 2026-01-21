//go:build !production

package linear

import (
	"bytes"
	"io"
	"net/http"
)

// mockTransport is a mock HTTP transport for testing.
// This is shared across all test files in the linear package.
type mockTransport struct {
	response *http.Response
}

// RoundTrip implements the http.RoundTripper interface.
func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.response, nil
}

// newMockResponse creates a mock HTTP response with the given body string.
func newMockResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}
