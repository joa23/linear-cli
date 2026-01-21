package linear

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const linearAPIURL = "https://api.linear.app/graphql"

// BaseClient contains the shared HTTP client and common request functionality
// that all sub-clients will use. This ensures we have a single HTTP client
// instance and consistent request handling across all Linear API operations.
type BaseClient struct {
	apiToken   string
	httpClient *http.Client
	baseURL    string
}

// NewBaseClient creates a new base client with a shared HTTP client
func NewBaseClient(apiToken string) *BaseClient {
	// Create a single HTTP client instance that will be reused
	// for all requests. This improves performance by reusing
	// TCP connections through HTTP keep-alive.
	httpClient := NewOptimizedHTTPClient()

	return &BaseClient{
		apiToken:   apiToken,
		httpClient: httpClient,
		baseURL:    linearAPIURL,
	}
}

// SetHTTPClient sets a custom HTTP client for testing purposes
func (bc *BaseClient) SetHTTPClient(client *http.Client) {
	bc.httpClient = client
}

// makeRequestWithRetry makes an HTTP request with exponential backoff for rate limiting and network errors.
// This method implements a comprehensive retry strategy to handle:
// - Rate limiting (429 responses) with respect for Retry-After headers
// - Temporary network failures (connection resets, timeouts)
// - Server errors (5xx responses) that might be transient
//
// The retry logic uses exponential backoff with a base delay of 100ms, doubling
// on each retry up to a maximum of 5 retries. For rate limits, it respects the
// server's Retry-After header if provided.
//
// Why this approach: Linear's API has strict rate limits and network issues are
// common in distributed systems. This retry mechanism ensures resilience without
// overwhelming the server or failing prematurely on temporary issues.
func (bc *BaseClient) makeRequestWithRetry(req *http.Request) (*http.Response, error) {
	const maxRetries = 5
	const baseDelay = 100 * time.Millisecond
	
	// Store the original body so we can recreate the request for retries
	// Why: HTTP request bodies can only be read once. Since we might retry
	// the request multiple times, we need to preserve the original body data
	// and create a new reader for each attempt.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()
	}
	
	// Add auth header once, it will persist across retries
	req.Header.Set("Authorization", bc.apiToken)
	req.Header.Set("Content-Type", "application/json")
	
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Reset body for each attempt if we have one
		// Why: Each HTTP request attempt needs a fresh reader because
		// the previous attempt consumed the body stream.
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
		
		resp, err := bc.httpClient.Do(req)
		
		// Handle network errors with retry logic
		// Why: Network errors are often transient (e.g., connection reset,
		// DNS failures, timeouts). Retrying with backoff gives the network
		// time to recover without immediately failing the operation.
		if err != nil {
			// Check if it's a retryable error (network errors, EOF, connection reset, etc.)
			shouldRetry := false
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				shouldRetry = true
			} else if err == io.EOF || strings.Contains(err.Error(), "EOF") ||
				strings.Contains(err.Error(), "connection reset") ||
				strings.Contains(err.Error(), "broken pipe") {
				shouldRetry = true
			}

			if shouldRetry {
				lastErr = err
				if attempt < maxRetries {
					delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
					time.Sleep(delay)
					continue
				}
			}
			return nil, fmt.Errorf("network error: %w", err)
		}
		
		// Success - return the response
		// Why: 2xx status codes indicate successful API calls that don't
		// need retry logic. We return immediately to avoid unnecessary delays.
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}
		
		// Handle rate limiting with Retry-After header
		// Why: Linear enforces rate limits to protect their infrastructure.
		// When we hit these limits, they tell us exactly when we can retry
		// via the Retry-After header. Respecting this prevents ban escalation.
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			if retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil {
					time.Sleep(time.Duration(seconds) * time.Second)
					resp.Body.Close()
					continue
				}
			}
			// If no Retry-After header, use exponential backoff
			// Why: Even without explicit guidance, we should still back off
			// to avoid hammering the server with rapid retry attempts.
			if attempt < maxRetries {
				delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay * 10
				time.Sleep(delay)
				resp.Body.Close()
				continue
			}
		}
		
		// Handle server errors (5xx) with retry
		// Why: Server errors are often temporary (e.g., deployments, database
		// issues, load problems). Retrying gives the server time to recover
		// from transient issues that aren't the client's fault.
		if resp.StatusCode >= 500 && attempt < maxRetries {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
			
			delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
			time.Sleep(delay)
			continue
		}
		
		// For client errors or final attempt, return the response
		// Why: 4xx errors (except 429) indicate client issues that won't
		// be fixed by retrying. We return these immediately to let the
		// caller handle the error appropriately.
		return resp, nil
	}
	
	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
	}
	return nil, fmt.Errorf("request failed after %d retries", maxRetries)
}

// executeRequest is a helper method to execute GraphQL requests
// Why: All Linear API calls follow the same pattern - send GraphQL query,
// handle errors, decode response. This method centralizes that logic to
// avoid duplication and ensure consistent error handling.
func (bc *BaseClient) executeRequest(query string, variables map[string]interface{}, result interface{}) error {
	// Prepare the GraphQL request payload
	// Why: Linear uses GraphQL which requires a specific JSON structure
	// with "query" and optional "variables" fields.
	payload := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		payload["variables"] = variables
	}
	
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create the HTTP request
	// Why: We need to construct a proper HTTP POST request with the
	// GraphQL payload to send to Linear's API endpoint.
	req, err := http.NewRequest("POST", bc.baseURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Execute the request with retry logic
	// Why: We delegate to our retry-aware method to handle transient
	// failures gracefully without failing the entire operation.
	resp, err := bc.makeRequestWithRetry(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read the response body
	// Why: We need to consume the entire response to check for errors
	// and decode the result, even if the status code indicates failure.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// Check for non-2xx status codes
	// Why: Even though makeRequestWithRetry handles retries, it still
	// returns error responses that we need to handle appropriately.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}
	
	// Decode the GraphQL response
	// Why: GraphQL responses have a standard structure with "data" and
	// "errors" fields. We need to decode this to extract the actual result.
	var graphQLResp struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	
	if err := json.Unmarshal(respBody, &graphQLResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	// Check for GraphQL errors
	// Why: GraphQL can return a 200 OK status but still contain errors
	// in the response. We need to check for these and surface them.
	if len(graphQLResp.Errors) > 0 {
		// Enhanced error with query context for better debugging
		errMsg := graphQLResp.Errors[0].Message
		
		// Add query context for debugging
		queryPreview := query
		if len(queryPreview) > 100 {
			queryPreview = queryPreview[:100] + "..."
		}
		
		return &GraphQLError{
			Message: fmt.Sprintf("%s (query: %s)", errMsg, queryPreview),
		}
	}
	
	// Decode the data portion into the result
	// Why: The actual query result is nested under the "data" field.
	// We decode this into the caller's provided result structure.
	if result != nil && graphQLResp.Data != nil {
		if err := json.Unmarshal(graphQLResp.Data, result); err != nil {
			return fmt.Errorf("failed to decode response data: %w", err)
		}
	}
	
	return nil
}

// NewTestBaseClient creates a new base client for testing with custom URL
func NewTestBaseClient(apiToken string, baseURL string, httpClient *http.Client) *BaseClient {
	return &BaseClient{
		apiToken:   apiToken,
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}