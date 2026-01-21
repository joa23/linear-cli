package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestGetWorkflowStatesWithCache(t *testing.T) {
	// Test that GetWorkflowStates caches results per team
	mockResponseBody1 := `{
		"data": {
			"workflowStates": {
				"nodes": [
					{
						"id": "state-1",
						"name": "Todo",
						"type": "unstarted",
						"color": "#e2e2e2",
						"position": 1.0,
						"description": "Work that needs to be done",
						"team": {
							"id": "team-123",
							"name": "Engineering"
						}
					},
					{
						"id": "state-2",
						"name": "In Progress",
						"type": "started",
						"color": "#f2c94c",
						"position": 2.0,
						"description": "Work in progress",
						"team": {
							"id": "team-123",
							"name": "Engineering"
						}
					}
				]
			}
		}
	}`

	mockResponseBody2 := `{
		"data": {
			"workflowStates": {
				"nodes": [
					{
						"id": "state-3",
						"name": "Backlog",
						"type": "backlog",
						"color": "#95a2b3",
						"position": 0.0,
						"description": "Future work",
						"team": {
							"id": "team-456",
							"name": "Product"
						}
					}
				]
			}
		}
	}`

	// Track number of API calls
	apiCalls := 0
	
	base := NewBaseClient("test-token")
	base.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			apiCalls++
			
			// Return different responses based on team ID in the request
			body, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewBuffer(body))
			
			if bytes.Contains(body, []byte("team-123")) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody1)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody2)),
			}, nil
		}),
	}
	
	workflowClient := NewWorkflowClient(base)
	
	// First call for team-123 should hit the API
	states1, err := workflowClient.GetWorkflowStates("team-123")
	if err != nil {
		t.Fatalf("First GetWorkflowStates failed: %v", err)
	}
	if len(states1) != 2 {
		t.Errorf("Expected 2 states for team-123, got %d", len(states1))
	}
	if apiCalls != 1 {
		t.Errorf("Expected 1 API call after first request, got %d", apiCalls)
	}
	
	// Second call for team-123 should use cache
	states2, err := workflowClient.GetWorkflowStates("team-123")
	if err != nil {
		t.Fatalf("Second GetWorkflowStates failed: %v", err)
	}
	if len(states2) != 2 {
		t.Errorf("Expected 2 cached states for team-123, got %d", len(states2))
	}
	if apiCalls != 1 {
		t.Errorf("Expected still 1 API call after cached request, got %d", apiCalls)
	}
	
	// Verify cached data is the same
	if states1[0].ID != states2[0].ID || states1[0].Name != states2[0].Name {
		t.Error("Cached data doesn't match original")
	}
	
	// Call for different team should hit the API
	states3, err := workflowClient.GetWorkflowStates("team-456")
	if err != nil {
		t.Fatalf("GetWorkflowStates for team-456 failed: %v", err)
	}
	if len(states3) != 1 {
		t.Errorf("Expected 1 state for team-456, got %d", len(states3))
	}
	if apiCalls != 2 {
		t.Errorf("Expected 2 API calls after new team request, got %d", apiCalls)
	}
	
	// Call without team ID should also hit API (different cache key)
	_, err = workflowClient.GetWorkflowStates("")
	if err != nil {
		t.Fatalf("GetWorkflowStates without team failed: %v", err)
	}
	if apiCalls != 3 {
		t.Errorf("Expected 3 API calls after no-team request, got %d", apiCalls)
	}
}

func TestGetWorkflowStateByName(t *testing.T) {
	// Test the helper function that finds a state by name
	mockResponseBody := `{
		"data": {
			"workflowStates": {
				"nodes": [
					{
						"id": "state-todo",
						"name": "Todo",
						"type": "unstarted",
						"color": "#e2e2e2",
						"position": 1.0,
						"description": "Work that needs to be done",
						"team": {
							"id": "team-123",
							"name": "Engineering"
						}
					},
					{
						"id": "state-in-progress",
						"name": "In Progress",
						"type": "started",
						"color": "#f2c94c",
						"position": 2.0,
						"description": "Work in progress",
						"team": {
							"id": "team-123",
							"name": "Engineering"
						}
					},
					{
						"id": "state-done",
						"name": "Done",
						"type": "completed",
						"color": "#5e6ad2",
						"position": 3.0,
						"description": "Completed work",
						"team": {
							"id": "team-123",
							"name": "Engineering"
						}
					}
				]
			}
		}
	}`

	apiCalls := 0
	
	base := NewBaseClient("test-token")
	base.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			apiCalls++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			}, nil
		}),
	}
	
	workflowClient := NewWorkflowClient(base)
	
	// Test finding existing state
	state, err := workflowClient.GetWorkflowStateByName("team-123", "In Progress")
	if err != nil {
		t.Fatalf("GetWorkflowStateByName failed: %v", err)
	}
	if state == nil {
		t.Fatal("Expected to find 'In Progress' state, got nil")
	}
	if state.ID != "state-in-progress" {
		t.Errorf("Expected state ID 'state-in-progress', got '%s'", state.ID)
	}
	if state.Name != "In Progress" {
		t.Errorf("Expected state name 'In Progress', got '%s'", state.Name)
	}
	if apiCalls != 1 {
		t.Errorf("Expected 1 API call, got %d", apiCalls)
	}
	
	// Second call should use cache
	state2, err := workflowClient.GetWorkflowStateByName("team-123", "Done")
	if err != nil {
		t.Fatalf("Second GetWorkflowStateByName failed: %v", err)
	}
	if state2 == nil {
		t.Fatal("Expected to find 'Done' state, got nil")
	}
	if state2.ID != "state-done" {
		t.Errorf("Expected state ID 'state-done', got '%s'", state2.ID)
	}
	if apiCalls != 1 {
		t.Errorf("Expected still 1 API call (cached), got %d", apiCalls)
	}
	
	// Test non-existent state
	state3, err := workflowClient.GetWorkflowStateByName("team-123", "Non-existent")
	if err != nil {
		t.Fatalf("GetWorkflowStateByName for non-existent state failed: %v", err)
	}
	if state3 != nil {
		t.Errorf("Expected nil for non-existent state, got %+v", state3)
	}
	
	// Test case-insensitive search
	state4, err := workflowClient.GetWorkflowStateByName("team-123", "in progress")
	if err != nil {
		t.Fatalf("GetWorkflowStateByName with lowercase failed: %v", err)
	}
	if state4 == nil {
		t.Fatal("Expected to find state with case-insensitive search")
	}
	if state4.ID != "state-in-progress" {
		t.Errorf("Expected state ID 'state-in-progress' for case-insensitive search, got '%s'", state4.ID)
	}
}

func TestWorkflowStateCacheExpiration(t *testing.T) {
	// Test that cache expires after TTL
	mockResponseBody := `{
		"data": {
			"workflowStates": {
				"nodes": [
					{
						"id": "state-1",
						"name": "Todo",
						"type": "unstarted"
					}
				]
			}
		}
	}`

	apiCalls := 0
	
	base := NewBaseClient("test-token")
	base.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			apiCalls++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			}, nil
		}),
	}
	
	// Create workflow client with short cache TTL for testing
	workflowClient := NewWorkflowClient(base)
	workflowClient.cacheTTL = 100 * time.Millisecond
	
	// First call should hit API
	_, err := workflowClient.GetWorkflowStates("team-123")
	if err != nil {
		t.Fatalf("First GetWorkflowStates failed: %v", err)
	}
	if apiCalls != 1 {
		t.Errorf("Expected 1 API call, got %d", apiCalls)
	}
	
	// Immediate second call should use cache
	_, err = workflowClient.GetWorkflowStates("team-123")
	if err != nil {
		t.Fatalf("Second GetWorkflowStates failed: %v", err)
	}
	if apiCalls != 1 {
		t.Errorf("Expected still 1 API call (cached), got %d", apiCalls)
	}
	
	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)
	
	// Call after expiration should hit API again
	_, err = workflowClient.GetWorkflowStates("team-123")
	if err != nil {
		t.Fatalf("GetWorkflowStates after cache expiration failed: %v", err)
	}
	if apiCalls != 2 {
		t.Errorf("Expected 2 API calls after cache expiration, got %d", apiCalls)
	}
}

// roundTripFunc is a helper to create a RoundTripper from a function
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}