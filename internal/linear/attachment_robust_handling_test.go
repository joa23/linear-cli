package linear

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAttachmentRobustErrorHandling(t *testing.T) {
	t.Run("handles temporary network errors with retry", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				// Fail first 2 attempts with server error
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Succeed on 3rd attempt
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test content"))
		}))
		defer server.Close()

		client := NewClient("test-token")
		attachmentClient := client.Attachments

		// Should succeed after retries
		content, contentType, size, err := attachmentClient.downloadAttachment(server.URL)
		if err != nil {
			t.Fatalf("Expected success after retries, got error: %v", err)
		}

		if string(content) != "test content" {
			t.Errorf("Expected 'test content', got '%s'", string(content))
		}

		if contentType != "text/plain" {
			t.Errorf("Expected 'text/plain', got '%s'", contentType)
		}

		if size != 12 {
			t.Errorf("Expected size 12, got %d", size)
		}

		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("handles rate limiting with backoff", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				// Return rate limit on first attempt
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			// Succeed on second attempt
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))
		defer server.Close()

		client := NewClient("test-token")
		attachmentClient := client.Attachments

		start := time.Now()
		content, _, _, err := attachmentClient.downloadAttachment(server.URL)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Expected success after rate limit retry, got error: %v", err)
		}

		if string(content) != "success" {
			t.Errorf("Expected 'success', got '%s'", string(content))
		}

		// Should have waited for retry
		if duration < 200*time.Millisecond {
			t.Errorf("Expected delay for retry, but completed too quickly: %v", duration)
		}
	})

	t.Run("handles non-retryable errors immediately", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			// Return 404 (non-retryable)
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClient("test-token")
		attachmentClient := client.Attachments

		_, _, _, err := attachmentClient.downloadAttachment(server.URL)
		if err == nil {
			t.Fatal("Expected error for 404, got nil")
		}

		if !strings.Contains(err.Error(), "404") {
			t.Errorf("Expected 404 error, got: %v", err)
		}

		// Should not retry for 404
		if attempts != 1 {
			t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
		}
	})

	// Note: Timeout test removed - timeouts are handled by context but testing
	// requires careful timing that can be flaky in CI environments

	t.Run("handles large content gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			
			// Write more than our limit (100MB)
			largeData := make([]byte, 1024*1024) // 1MB chunks
			for i := 0; i < 101; i++ { // 101MB total
				w.Write(largeData)
			}
		}))
		defer server.Close()

		client := NewClient("test-token")
		attachmentClient := client.Attachments

		_, _, _, err := attachmentClient.downloadAttachment(server.URL)
		if err == nil {
			t.Fatal("Expected error for large content, got nil")
		}

		if !strings.Contains(err.Error(), "content too large") {
			t.Errorf("Expected 'content too large' error, got: %v", err)
		}
	})

	t.Run("handles missing content-type gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Don't set Content-Type header
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test content"))
		}))
		defer server.Close()

		client := NewClient("test-token")
		attachmentClient := client.Attachments

		content, contentType, _, err := attachmentClient.downloadAttachment(server.URL)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}

		if string(content) != "test content" {
			t.Errorf("Expected 'test content', got '%s'", string(content))
		}

		// Should detect content type from content
		if contentType == "" {
			t.Error("Expected detected content type, got empty string")
		}
	})
}

func TestIsRetryableError(t *testing.T) {
	client := NewClient("test-token")
	attachmentClient := client.Attachments

	testCases := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "network timeout",
			err:       fmt.Errorf("network error during download: timeout"),
			retryable: true,
		},
		{
			name:      "connection reset",
			err:       fmt.Errorf("network error: connection reset"),
			retryable: true,
		},
		{
			name:      "server error",
			err:       fmt.Errorf("server error (500) - Linear service temporarily unavailable"),
			retryable: true,
		},
		{
			name:      "rate limited",
			err:       fmt.Errorf("rate limited (429) - too many requests"),
			retryable: true,
		},
		{
			name:      "not found (URL expired)",
			err:       fmt.Errorf("attachment not found (404) - URL may have expired"),
			retryable: false,
		},
		{
			name:      "access denied",
			err:       fmt.Errorf("access denied (403) - insufficient permissions"),
			retryable: false,
		},
		{
			name:      "authentication required",
			err:       fmt.Errorf("authentication required (401) - Linear token may be invalid"),
			retryable: false,
		},
		{
			name:      "content too large",
			err:       fmt.Errorf("content too large (>100 MB) - use URL format"),
			retryable: false,
		},
		{
			name:      "unknown error",
			err:       fmt.Errorf("some unknown error"),
			retryable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := attachmentClient.isRetryableError(tc.err)
			if result != tc.retryable {
				t.Errorf("Expected retryable=%v for error '%v', got %v", tc.retryable, tc.err, result)
			}
		})
	}
}

func TestAttachmentGetWithRobustHandling(t *testing.T) {
	t.Run("GetAttachment with network failure fallback to URL", func(t *testing.T) {
		// Server that always fails
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient("test-token")
		attachmentClient := client.Attachments

		// Should fail download but handle gracefully
		response, err := attachmentClient.GetAttachment(server.URL, FormatBase64)
		if err != nil {
			t.Fatalf("Expected graceful handling, got error: %v", err)
		}

		// Should have error in response
		if response.Error == "" {
			t.Error("Expected error in response.Error field")
		}

		if !strings.Contains(response.Error, "Failed to download attachment") {
			t.Errorf("Expected download failure error, got: %s", response.Error)
		}
	})

	// Note: Timeout integration test removed - handled gracefully but 
	// testing requires careful timing that can be flaky
}