package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestWorkflowClient_GetWorkflowStates(t *testing.T) {
	tests := []struct {
		name           string
		teamID         string
		mockResponse   string
		expectedStates int
		expectedError  bool
	}{
		{
			name:   "Get workflow states for team",
			teamID: "team-123",
			mockResponse: `{
				"data": {
					"workflowStates": {
						"nodes": [
							{
								"id": "state-1",
								"name": "Todo",
								"type": "unstarted",
								"color": "#e2e2e2",
								"position": 1.0,
								"description": "Work to be done",
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
			}`,
			expectedStates: 2,
			expectedError:  false,
		},
		{
			name:   "Get all workflow states (no team filter)",
			teamID: "",
			mockResponse: `{
				"data": {
					"workflowStates": {
						"nodes": [
							{
								"id": "state-global-1",
								"name": "Backlog",
								"type": "backlog",
								"color": "#95a2b3",
								"position": 0.0,
								"description": "Future work"
							},
							{
								"id": "state-global-2",
								"name": "Todo",
								"type": "unstarted",
								"color": "#e2e2e2",
								"position": 1.0,
								"description": "Ready to start"
							},
							{
								"id": "state-global-3",
								"name": "Done",
								"type": "completed",
								"color": "#5e6ad2",
								"position": 3.0,
								"description": "Completed work"
							}
						]
					}
				}
			}`,
			expectedStates: 3,
			expectedError:  false,
		},
		{
			name:   "API error",
			teamID: "team-error",
			mockResponse: `{
				"errors": [
					{
						"message": "Team not found",
						"extensions": {
							"code": "NOT_FOUND"
						}
					}
				]
			}`,
			expectedStates: 0,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := NewBaseClient("test-token")
			base.httpClient = &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(tt.mockResponse)),
					}, nil
				}),
			}

			client := NewWorkflowClient(base)
			states, err := client.GetWorkflowStates(tt.teamID)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(states) != tt.expectedStates {
					t.Errorf("Expected %d states, got %d", tt.expectedStates, len(states))
				}
			}
		})
	}
}

func TestWorkflowClient_GetWorkflowStateByName(t *testing.T) {
	mockResponse := `{
		"data": {
			"workflowStates": {
				"nodes": [
					{
						"id": "state-todo",
						"name": "Todo",
						"type": "unstarted",
						"color": "#e2e2e2",
						"position": 1.0,
						"description": "Work to be done"
					},
					{
						"id": "state-in-progress",
						"name": "In Progress",
						"type": "started",
						"color": "#f2c94c",
						"position": 2.0,
						"description": "Work being done"
					},
					{
						"id": "state-done",
						"name": "Done",
						"type": "completed",
						"color": "#5e6ad2",
						"position": 3.0,
						"description": "Completed work"
					}
				]
			}
		}
	}`

	tests := []struct {
		name          string
		teamID        string
		stateName     string
		expectedID    string
		expectedNil   bool
	}{
		{
			name:        "Find existing state",
			teamID:      "team-123",
			stateName:   "In Progress",
			expectedID:  "state-in-progress",
			expectedNil: false,
		},
		{
			name:        "Find state case-insensitive",
			teamID:      "team-123",
			stateName:   "in progress",
			expectedID:  "state-in-progress",
			expectedNil: false,
		},
		{
			name:        "Find state with mixed case",
			teamID:      "team-123",
			stateName:   "TODO",
			expectedID:  "state-todo",
			expectedNil: false,
		},
		{
			name:        "State not found",
			teamID:      "team-123",
			stateName:   "Non-existent",
			expectedID:  "",
			expectedNil: true,
		},
	}

	base := NewBaseClient("test-token")
	base.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			}, nil
		}),
	}

	client := NewWorkflowClient(base)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := client.GetWorkflowStateByName(tt.teamID, tt.stateName)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectedNil {
				if state != nil {
					t.Errorf("Expected nil state, got %+v", state)
				}
			} else {
				if state == nil {
					t.Fatal("Expected state, got nil")
				}
				if state.ID != tt.expectedID {
					t.Errorf("Expected state ID %s, got %s", tt.expectedID, state.ID)
				}
			}
		})
	}
}

func TestWorkflowClient_Caching(t *testing.T) {
	mockResponse := `{
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
				Body:       io.NopCloser(bytes.NewBufferString(mockResponse)),
			}, nil
		}),
	}

	client := NewWorkflowClient(base)
	client.cacheTTL = 100 * time.Millisecond // Short TTL for testing

	// First call should hit API
	_, err := client.GetWorkflowStates("team-123")
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}
	if apiCalls != 1 {
		t.Errorf("Expected 1 API call, got %d", apiCalls)
	}

	// Second call should use cache
	_, err = client.GetWorkflowStates("team-123")
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}
	if apiCalls != 1 {
		t.Errorf("Expected still 1 API call (cached), got %d", apiCalls)
	}

	// Different team should hit API
	_, err = client.GetWorkflowStates("team-456")
	if err != nil {
		t.Fatalf("Different team call failed: %v", err)
	}
	if apiCalls != 2 {
		t.Errorf("Expected 2 API calls after different team, got %d", apiCalls)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Call after expiration should hit API
	_, err = client.GetWorkflowStates("team-123")
	if err != nil {
		t.Fatalf("Call after expiration failed: %v", err)
	}
	if apiCalls != 3 {
		t.Errorf("Expected 3 API calls after expiration, got %d", apiCalls)
	}
}