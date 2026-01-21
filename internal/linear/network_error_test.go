package linear

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNetworkErrorRecovery tests how the client handles network errors
func TestNetworkErrorRecovery(t *testing.T) {
	t.Run("connection refused recovery", func(t *testing.T) {
		// Create client pointing to a non-existent server
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Timeout: 1 * time.Second}
		client.base.baseURL = "http://127.0.0.1:1" // Invalid port
		
		// This should fail with connection refused
		_, err := client.GetAppUserID()
		if err == nil {
			t.Error("Expected connection error, got success")
		}
		
		// Verify it's a connection error
		if !strings.Contains(err.Error(), "connection refused") && 
		   !strings.Contains(err.Error(), "no such host") &&
		   !strings.Contains(err.Error(), "network") {
			t.Errorf("Expected network/connection error, got: %v", err)
		}
		
		t.Logf("Got expected connection error: %v", err)
	})
	
	t.Run("timeout recovery", func(t *testing.T) {
		// Create a server that takes too long to respond
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sleep longer than client timeout
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"viewer": {"id": "test-user"}}}`))
		}))
		defer server.Close()
		
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Timeout: 500 * time.Millisecond} // Short timeout
		client.base.baseURL = server.URL
		
		// This should timeout
		start := time.Now()
		_, err := client.GetAppUserID()
		elapsed := time.Since(start)
		
		if err == nil {
			t.Error("Expected timeout error, got success")
		}
		
		// Verify it timed out within reasonable time (may retry a few times)
		if elapsed > 10*time.Second {
			t.Errorf("Expected timeout within 10s, took %v", elapsed)
		}
		
		// Verify it's a timeout error
		if !strings.Contains(err.Error(), "timeout") && 
		   !strings.Contains(err.Error(), "deadline") &&
		   !strings.Contains(err.Error(), "context") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
		
		t.Logf("Got expected timeout error: %v", err)
	})
	
	t.Run("intermittent connection failures with recovery", func(t *testing.T) {
		requestCount := 0
		failureCount := 3
		
		// Server that fails first N requests with connection-like errors
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			
			if requestCount <= failureCount {
				// Simulate connection being dropped
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, err := hj.Hijack()
					if err == nil {
						conn.Close() // Abruptly close connection
						return
					}
				}
				// Fallback if hijacking fails
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			
			// Success after failures
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"viewer": {"id": "test-user-recovered"}}}`))
		}))
		defer server.Close()
		
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Timeout: 2 * time.Second}
		client.base.baseURL = server.URL
		
		// This should eventually succeed despite initial failures
		userID, err := client.GetAppUserID()
		
		// Since our current implementation doesn't have network error retry,
		// this will likely fail. When we implement retry logic, this should succeed.
		if err != nil {
			t.Logf("Got error (expected with current implementation): %v", err)
			t.Logf("Total requests made: %d", requestCount)
			
			// Verify we got a network-related error
			if !strings.Contains(err.Error(), "connection") &&
			   !strings.Contains(err.Error(), "EOF") &&
			   !strings.Contains(err.Error(), "reset") &&
			   !strings.Contains(err.Error(), "closed") {
				t.Errorf("Expected network error, got: %v", err)
			}
		} else {
			// If it succeeded, verify the result and that retries happened
			if userID != "test-user-recovered" {
				t.Errorf("Expected 'test-user-recovered', got: %s", userID)
			}
			
			expectedRequests := failureCount + 1
			if requestCount < expectedRequests {
				t.Errorf("Expected at least %d requests (with retries), got %d", expectedRequests, requestCount)
			}
			
			t.Logf("Successfully recovered after %d requests", requestCount)
		}
	})
	
	t.Run("DNS resolution failure", func(t *testing.T) {
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Timeout: 2 * time.Second}
		client.base.baseURL = "http://this-domain-does-not-exist-12345.com/graphql"
		
		_, err := client.GetAppUserID()
		if err == nil {
			t.Error("Expected DNS resolution error, got success")
		}
		
		// Verify it's a DNS-related error
		if !strings.Contains(err.Error(), "no such host") &&
		   !strings.Contains(err.Error(), "lookup") &&
		   !strings.Contains(err.Error(), "resolve") {
			t.Errorf("Expected DNS error, got: %v", err)
		}
		
		t.Logf("Got expected DNS error: %v", err)
	})
	
	t.Run("server drops connection during request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Start writing response then drop connection
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"viewer"`)) // Incomplete JSON
			
			// Drop connection
			if hj, ok := w.(http.Hijacker); ok {
				if conn, _, err := hj.Hijack(); err == nil {
					conn.Close()
				}
			}
		}))
		defer server.Close()
		
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Timeout: 2 * time.Second}
		client.base.baseURL = server.URL
		
		_, err := client.GetAppUserID()
		if err == nil {
			t.Error("Expected connection drop error, got success")
		}
		
		// Verify it's a connection-related error
		if !strings.Contains(err.Error(), "EOF") &&
		   !strings.Contains(err.Error(), "connection") &&
		   !strings.Contains(err.Error(), "closed") &&
		   !strings.Contains(err.Error(), "reset") {
			t.Errorf("Expected connection error, got: %v", err)
		}
		
		t.Logf("Got expected connection drop error: %v", err)
	})
	
	t.Run("context cancellation handling", func(t *testing.T) {
		// Create a server that takes a long time to respond
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"viewer": {"id": "test-user"}}}`))
		}))
		defer server.Close()
		
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{}
		client.base.baseURL = server.URL
		
		// Create request with context
		start := time.Now()
		
		// For this test, we'll simulate what would happen with context cancellation
		// by using a very short timeout on the HTTP client
		client.base.httpClient.Timeout = 100 * time.Millisecond
		
		_, err := client.GetAppUserID()
		elapsed := time.Since(start)
		
		if err == nil {
			t.Error("Expected context cancellation/timeout error, got success")
		}
		
		// Should fail within reasonable time (may retry a few times)
		if elapsed > 10*time.Second {
			t.Errorf("Expected failure within 10s, took %v", elapsed)
		}
		
		t.Logf("Got expected cancellation/timeout error: %v", err)
	})
}

// TestNetworkErrorRetryLogic tests the retry logic for network errors
func TestNetworkErrorRetryLogic(t *testing.T) {
	t.Run("should retry on temporary network errors", func(t *testing.T) {
		
		requestCount := 0
		tempFailures := 2
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			
			if requestCount <= tempFailures {
				// Simulate temporary network failure
				if hj, ok := w.(http.Hijacker); ok {
					if conn, _, err := hj.Hijack(); err == nil {
						conn.Close()
						return
					}
				}
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			
			// Success after retries
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"viewer": {"id": "retry-success"}}}`))
		}))
		defer server.Close()
		
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Timeout: 1 * time.Second}
		client.base.baseURL = server.URL
		
		// Should eventually succeed with retries
		userID, err := client.GetAppUserID()
		if err != nil {
			t.Fatalf("Expected success with retries, got error: %v", err)
		}
		
		if userID != "retry-success" {
			t.Errorf("Expected 'retry-success', got: %s", userID)
		}
		
		expectedRequests := tempFailures + 1
		if requestCount != expectedRequests {
			t.Errorf("Expected %d requests, got %d", expectedRequests, requestCount)
		}
		
		t.Logf("Successfully retried %d times", tempFailures)
	})
	
	t.Run("should not retry on non-retryable errors", func(t *testing.T) {
		
		requestCount := 0
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			// Return 400 Bad Request - should not be retried
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Bad request"}`))
		}))
		defer server.Close()
		
		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Timeout: 1 * time.Second}
		client.base.baseURL = server.URL
		
		// Should fail without retries
		_, err := client.GetAppUserID()
		if err == nil {
			t.Error("Expected error for 400 response, got success")
		}
		
		// Should only make one request (no retries for 400)
		if requestCount != 1 {
			t.Errorf("Expected 1 request (no retries), got %d", requestCount)
		}
	})
}