package linear

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestGetWorkflowStates(t *testing.T) {
	// Test that GetWorkflowStates returns all available workflow states
	mockResponseBody := `{
		"data": {
			"workflowStates": {
				"nodes": [
					{
						"id": "state-backlog-123",
						"name": "Backlog",
						"type": "backlog"
					},
					{
						"id": "state-todo-456",
						"name": "Todo",
						"type": "unstarted"
					},
					{
						"id": "state-inprogress-789",
						"name": "In Progress",
						"type": "started"
					},
					{
						"id": "state-inreview-999",
						"name": "In Review",
						"type": "started"
					},
					{
						"id": "state-done-111",
						"name": "Done",
						"type": "completed"
					},
					{
						"id": "state-canceled-222",
						"name": "Canceled",
						"type": "canceled"
					}
				]
			}
		}
	}`
	
	client := NewClient("test-token")
	client.base.httpClient = &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			},
		},
	}
	
	states, err := client.GetWorkflowStates("")
	if err != nil {
		t.Fatalf("GetWorkflowStates failed: %v", err)
	}
	
	// Verify we got all states
	if len(states) != 6 {
		t.Fatalf("Expected 6 workflow states, got %d", len(states))
	}
	
	// Verify state details
	expectedStates := map[string]string{
		"Backlog":     "backlog",
		"Todo":        "unstarted",
		"In Progress": "started",
		"In Review":   "started",
		"Done":        "completed",
		"Canceled":    "canceled",
	}
	
	for _, state := range states {
		expectedType, ok := expectedStates[state.Name]
		if !ok {
			t.Errorf("Unexpected state name: %s", state.Name)
		}
		if state.Type != expectedType {
			t.Errorf("State %s: expected type '%s', got '%s'", state.Name, expectedType, state.Type)
		}
		if state.ID == "" {
			t.Errorf("State %s has empty ID", state.Name)
		}
	}
}

func TestGetWorkflowStatesForTeam(t *testing.T) {
	// Test that GetWorkflowStates can filter by team
	mockResponseBody := `{
		"data": {
			"workflowStates": {
				"nodes": [
					{
						"id": "team-state-1",
						"name": "New",
						"type": "unstarted"
					},
					{
						"id": "team-state-2",
						"name": "Active",
						"type": "started"
					},
					{
						"id": "team-state-3",
						"name": "Resolved",
						"type": "completed"
					}
				]
			}
		}
	}`
	
	client := NewClient("test-token")
	client.base.httpClient = &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
			},
		},
	}
	
	states, err := client.GetWorkflowStates("team-123")
	if err != nil {
		t.Fatalf("GetWorkflowStates with team failed: %v", err)
	}
	
	// Verify we got team-specific states
	if len(states) != 3 {
		t.Fatalf("Expected 3 team workflow states, got %d", len(states))
	}
	
	// Verify we have different states than the default
	if states[0].Name != "New" {
		t.Errorf("Expected first state to be 'New', got '%s'", states[0].Name)
	}
}