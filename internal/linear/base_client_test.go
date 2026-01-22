package linear

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAuthorizationHeaderFormat verifies that the Authorization header
// is properly formatted with Bearer prefix and sanitized token
func TestAuthorizationHeaderFormat(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		expectedAuth  string
		expectSuccess bool
	}{
		{
			name:          "clean token gets Bearer prefix",
			token:         "lin_api_token123",
			expectedAuth:  "Bearer lin_api_token123",
			expectSuccess: true,
		},
		{
			name:          "token with newline gets sanitized",
			token:         "lin_api_token123\n",
			expectedAuth:  "Bearer lin_api_token123",
			expectSuccess: true,
		},
		{
			name:          "token with existing Bearer prefix",
			token:         "Bearer lin_api_token123",
			expectedAuth:  "Bearer lin_api_token123",
			expectSuccess: true,
		},
		{
			name:          "token with Bearer and newline",
			token:         "Bearer lin_api_token123\n",
			expectedAuth:  "Bearer lin_api_token123",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that captures the Authorization header
			var capturedAuthHeader string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedAuthHeader = r.Header.Get("Authorization")

				// Return a minimal valid GraphQL response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":{}}`))
			}))
			defer server.Close()

			// Create a base client with test server URL
			client := NewTestBaseClient(tt.token, server.URL, server.Client())

			// Make a request (doesn't matter what query, we just want to check the header)
			err := client.executeRequest("query { viewer { id } }", nil, nil)

			if tt.expectSuccess && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify the Authorization header
			if capturedAuthHeader != tt.expectedAuth {
				t.Errorf("Authorization header = %q, want %q", capturedAuthHeader, tt.expectedAuth)
			}

			// Verify no invalid characters remain
			if strings.ContainsAny(capturedAuthHeader, "\n\r\t") {
				t.Errorf("Authorization header contains invalid characters: %q", capturedAuthHeader)
			}
		})
	}
}
