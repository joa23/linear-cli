package linear

import (
	"github.com/joa23/linear-cli/internal/token"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGetCycle verifies that GetCycle correctly fetches and unmarshals a cycle
func TestGetCycle(t *testing.T) {
	tests := []struct {
		name            string
		cycleID         string
		graphqlResponse string
		wantErr         bool
		wantCycle       *Cycle
	}{
		{
			name:    "successful cycle retrieval",
			cycleID: "cycle-123",
			graphqlResponse: `{
				"data": {
					"cycle": {
						"id": "cycle-123",
						"name": "Sprint 42",
						"number": 42,
						"description": "The meaning of life sprint",
						"startsAt": "2025-01-06T00:00:00Z",
						"endsAt": "2025-01-20T00:00:00Z",
						"completedAt": null,
						"progress": 0.5,
						"team": {
							"id": "team-123",
							"name": "Engineering",
							"key": "ENG"
						},
						"isActive": true,
						"isFuture": false,
						"isPast": false,
						"isNext": false,
						"isPrevious": false,
						"scopeHistory": [10, 12, 15],
						"completedScopeHistory": [0, 3, 7],
						"completedIssueCountHistory": [0, 2, 5],
						"inProgressScopeHistory": [2, 3, 4],
						"issueCountHistory": [5, 6, 8],
						"createdAt": "2025-01-01T00:00:00Z",
						"updatedAt": "2025-01-10T00:00:00Z",
						"archivedAt": null,
						"autoArchivedAt": null
					}
				}
			}`,
			wantErr: false,
			wantCycle: &Cycle{
				ID:          "cycle-123",
				Name:        "Sprint 42",
				Number:      42,
				Description: "The meaning of life sprint",
				StartsAt:    "2025-01-06T00:00:00Z",
				EndsAt:      "2025-01-20T00:00:00Z",
				Progress:    0.5,
				IsActive:    true,
				IsFuture:    false,
				IsPast:      false,
			},
		},
		{
			name:    "cycle not found",
			cycleID: "nonexistent",
			graphqlResponse: `{
				"data": {
					"cycle": {
						"id": ""
					}
				}
			}`,
			wantErr:   true,
			wantCycle: nil,
		},
		{
			name:            "empty cycle ID",
			cycleID:         "",
			graphqlResponse: `{}`,
			wantErr:         true,
			wantCycle:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.graphqlResponse))
			}))
			defer server.Close()

			base := &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			}
			cc := NewCycleClient(base)

			cycle, err := cc.GetCycle(tt.cycleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCycle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantCycle != nil && cycle != nil {
				if cycle.ID != tt.wantCycle.ID {
					t.Errorf("Expected cycle ID %s, got %s", tt.wantCycle.ID, cycle.ID)
				}
				if cycle.Name != tt.wantCycle.Name {
					t.Errorf("Expected cycle Name %s, got %s", tt.wantCycle.Name, cycle.Name)
				}
				if cycle.Number != tt.wantCycle.Number {
					t.Errorf("Expected cycle Number %d, got %d", tt.wantCycle.Number, cycle.Number)
				}
				if cycle.IsActive != tt.wantCycle.IsActive {
					t.Errorf("Expected cycle IsActive %v, got %v", tt.wantCycle.IsActive, cycle.IsActive)
				}
			}
		})
	}
}

// TestListCycles verifies that ListCycles correctly fetches and filters cycles
func TestListCycles(t *testing.T) {
	tests := []struct {
		name            string
		filter          *CycleFilter
		graphqlResponse string
		wantCycleCount  int
		wantHasNextPage bool
	}{
		{
			name:   "list all cycles",
			filter: &CycleFilter{Limit: 10},
			graphqlResponse: `{
				"data": {
					"cycles": {
						"nodes": [
							{
								"id": "cycle-1",
								"name": "Sprint 1",
								"number": 1,
								"isActive": false,
								"isFuture": false,
								"isPast": true
							},
							{
								"id": "cycle-2",
								"name": "Sprint 2",
								"number": 2,
								"isActive": true,
								"isFuture": false,
								"isPast": false
							}
						],
						"pageInfo": {
							"hasNextPage": false,
							"endCursor": "cursor123"
						}
					}
				}
			}`,
			wantCycleCount:  2,
			wantHasNextPage: false,
		},
		{
			name: "filter active cycles only",
			filter: func() *CycleFilter {
				active := true
				return &CycleFilter{IsActive: &active, Limit: 10}
			}(),
			graphqlResponse: `{
				"data": {
					"cycles": {
						"nodes": [
							{
								"id": "cycle-2",
								"name": "Sprint 2",
								"number": 2,
								"isActive": true,
								"isFuture": false,
								"isPast": false
							}
						],
						"pageInfo": {
							"hasNextPage": false,
							"endCursor": ""
						}
					}
				}
			}`,
			wantCycleCount:  1,
			wantHasNextPage: false,
		},
		{
			name: "paginated results",
			filter: &CycleFilter{Limit: 1},
			graphqlResponse: `{
				"data": {
					"cycles": {
						"nodes": [
							{
								"id": "cycle-1",
								"name": "Sprint 1",
								"number": 1,
								"isActive": false,
								"isFuture": false,
								"isPast": true
							}
						],
						"pageInfo": {
							"hasNextPage": true,
							"endCursor": "cursor456"
						}
					}
				}
			}`,
			wantCycleCount:  1,
			wantHasNextPage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.graphqlResponse))
			}))
			defer server.Close()

			base := &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			}
			cc := NewCycleClient(base)

			result, err := cc.ListCycles(tt.filter)
			if err != nil {
				t.Fatalf("ListCycles() error = %v", err)
			}

			if len(result.Cycles) != tt.wantCycleCount {
				t.Errorf("Expected %d cycles, got %d", tt.wantCycleCount, len(result.Cycles))
			}

			if result.HasNextPage != tt.wantHasNextPage {
				t.Errorf("Expected HasNextPage %v, got %v", tt.wantHasNextPage, result.HasNextPage)
			}
		})
	}
}

// TestGetActiveCycle verifies that GetActiveCycle returns the active cycle for a team
func TestGetActiveCycle(t *testing.T) {
	tests := []struct {
		name            string
		teamID          string
		graphqlResponse string
		wantErr         bool
		wantCycleName   string
	}{
		{
			name:   "active cycle found",
			teamID: "team-123",
			graphqlResponse: `{
				"data": {
					"cycles": {
						"nodes": [
							{
								"id": "cycle-active",
								"name": "Current Sprint",
								"number": 10,
								"isActive": true,
								"isFuture": false,
								"isPast": false
							}
						],
						"pageInfo": {
							"hasNextPage": false,
							"endCursor": ""
						}
					}
				}
			}`,
			wantErr:       false,
			wantCycleName: "Current Sprint",
		},
		{
			name:   "no active cycle",
			teamID: "team-456",
			graphqlResponse: `{
				"data": {
					"cycles": {
						"nodes": [],
						"pageInfo": {
							"hasNextPage": false,
							"endCursor": ""
						}
					}
				}
			}`,
			wantErr:       true,
			wantCycleName: "",
		},
		{
			name:            "empty team ID",
			teamID:          "",
			graphqlResponse: `{}`,
			wantErr:         true,
			wantCycleName:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.graphqlResponse))
			}))
			defer server.Close()

			base := &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			}
			cc := NewCycleClient(base)

			cycle, err := cc.GetActiveCycle(tt.teamID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetActiveCycle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cycle.Name != tt.wantCycleName {
				t.Errorf("Expected cycle name %s, got %s", tt.wantCycleName, cycle.Name)
			}
		})
	}
}

// TestGetCycleIssues verifies that GetCycleIssues returns issues in a cycle
func TestGetCycleIssues(t *testing.T) {
	graphqlResponse := `{
		"data": {
			"cycle": {
				"issues": {
					"nodes": [
						{
							"id": "issue-1",
							"identifier": "ENG-123",
							"title": "First Issue",
							"state": {"id": "state-1", "name": "In Progress"},
							"assignee": {"id": "user-1", "name": "Test User", "email": "test@example.com"},
							"priority": 2,
							"estimate": 3.0
						},
						{
							"id": "issue-2",
							"identifier": "ENG-124",
							"title": "Second Issue",
							"state": {"id": "state-2", "name": "Done"},
							"assignee": null,
							"priority": 1,
							"estimate": 1.0
						}
					]
				}
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(graphqlResponse))
	}))
	defer server.Close()

	base := &BaseClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
		tokenProvider: token.NewStaticProvider("test-token"),
	}
	cc := NewCycleClient(base)

	issues, err := cc.GetCycleIssues("cycle-123", 50)
	if err != nil {
		t.Fatalf("GetCycleIssues() error = %v", err)
	}

	if len(issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(issues))
	}

	if issues[0].Identifier != "ENG-123" {
		t.Errorf("Expected first issue identifier ENG-123, got %s", issues[0].Identifier)
	}

	if issues[1].Title != "Second Issue" {
		t.Errorf("Expected second issue title 'Second Issue', got %s", issues[1].Title)
	}
}

// TestCreateCycle verifies that CreateCycle creates a new cycle
func TestCreateCycle(t *testing.T) {
	tests := []struct {
		name            string
		input           *CreateCycleInput
		graphqlResponse string
		wantErr         bool
		wantCycleName   string
	}{
		{
			name: "successful cycle creation",
			input: &CreateCycleInput{
				TeamID:      "team-123",
				Name:        "Sprint 43",
				Description: "New sprint",
				StartsAt:    "2025-01-20T00:00:00Z",
				EndsAt:      "2025-02-03T00:00:00Z",
			},
			graphqlResponse: `{
				"data": {
					"cycleCreate": {
						"success": true,
						"cycle": {
							"id": "cycle-new",
							"name": "Sprint 43",
							"number": 43,
							"description": "New sprint",
							"startsAt": "2025-01-20T00:00:00Z",
							"endsAt": "2025-02-03T00:00:00Z",
							"isActive": false,
							"isFuture": true,
							"isPast": false
						}
					}
				}
			}`,
			wantErr:       false,
			wantCycleName: "Sprint 43",
		},
		{
			name:            "nil input",
			input:           nil,
			graphqlResponse: `{}`,
			wantErr:         true,
			wantCycleName:   "",
		},
		{
			name: "missing required field",
			input: &CreateCycleInput{
				TeamID: "team-123",
				// Missing StartsAt and EndsAt
			},
			graphqlResponse: `{}`,
			wantErr:         true,
			wantCycleName:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.graphqlResponse))
			}))
			defer server.Close()

			base := &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			}
			cc := NewCycleClient(base)

			cycle, err := cc.CreateCycle(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCycle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cycle.Name != tt.wantCycleName {
				t.Errorf("Expected cycle name %s, got %s", tt.wantCycleName, cycle.Name)
			}
		})
	}
}

// TestUpdateCycle verifies that UpdateCycle updates an existing cycle
func TestUpdateCycle(t *testing.T) {
	tests := []struct {
		name            string
		cycleID         string
		input           *UpdateCycleInput
		graphqlResponse string
		wantErr         bool
		wantCycleName   string
	}{
		{
			name:    "successful cycle update",
			cycleID: "cycle-123",
			input: func() *UpdateCycleInput {
				name := "Updated Sprint Name"
				return &UpdateCycleInput{Name: &name}
			}(),
			graphqlResponse: `{
				"data": {
					"cycleUpdate": {
						"success": true,
						"cycle": {
							"id": "cycle-123",
							"name": "Updated Sprint Name",
							"number": 42,
							"isActive": true
						}
					}
				}
			}`,
			wantErr:       false,
			wantCycleName: "Updated Sprint Name",
		},
		{
			name:            "empty cycle ID",
			cycleID:         "",
			input:           &UpdateCycleInput{},
			graphqlResponse: `{}`,
			wantErr:         true,
			wantCycleName:   "",
		},
		{
			name:            "nil input",
			cycleID:         "cycle-123",
			input:           nil,
			graphqlResponse: `{}`,
			wantErr:         true,
			wantCycleName:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.graphqlResponse))
			}))
			defer server.Close()

			base := &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			}
			cc := NewCycleClient(base)

			cycle, err := cc.UpdateCycle(tt.cycleID, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateCycle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cycle.Name != tt.wantCycleName {
				t.Errorf("Expected cycle name %s, got %s", tt.wantCycleName, cycle.Name)
			}
		})
	}
}

// TestArchiveCycle verifies that ArchiveCycle archives a cycle
func TestArchiveCycle(t *testing.T) {
	tests := []struct {
		name            string
		cycleID         string
		graphqlResponse string
		wantErr         bool
	}{
		{
			name:    "successful archive",
			cycleID: "cycle-123",
			graphqlResponse: `{
				"data": {
					"cycleArchive": {
						"success": true
					}
				}
			}`,
			wantErr: false,
		},
		{
			name:            "empty cycle ID",
			cycleID:         "",
			graphqlResponse: `{}`,
			wantErr:         true,
		},
		{
			name:    "archive failure",
			cycleID: "cycle-123",
			graphqlResponse: `{
				"data": {
					"cycleArchive": {
						"success": false
					}
				}
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.graphqlResponse))
			}))
			defer server.Close()

			base := &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			}
			cc := NewCycleClient(base)

			err := cc.ArchiveCycle(tt.cycleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ArchiveCycle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCycleFormatConversions verifies that Cycle format conversions work correctly
func TestCycleFormatConversions(t *testing.T) {
	cycle := Cycle{
		ID:          "cycle-123",
		Name:        "Sprint 42",
		Number:      42,
		Description: "Test description",
		StartsAt:    "2025-01-06T00:00:00Z",
		EndsAt:      "2025-01-20T00:00:00Z",
		Progress:    0.75,
		Team: &Team{
			ID:   "team-123",
			Name: "Engineering",
			Key:  "ENG",
		},
		IsActive:  true,
		IsFuture:  false,
		IsPast:    false,
		IsNext:    false,
		CreatedAt: "2025-01-01T00:00:00Z",
		UpdatedAt: "2025-01-10T00:00:00Z",
	}

	t.Run("ToMinimal", func(t *testing.T) {
		minimal := cycle.ToMinimal()
		if minimal.ID != cycle.ID {
			t.Errorf("Expected ID %s, got %s", cycle.ID, minimal.ID)
		}
		if minimal.Name != cycle.Name {
			t.Errorf("Expected Name %s, got %s", cycle.Name, minimal.Name)
		}
		if minimal.Number != cycle.Number {
			t.Errorf("Expected Number %d, got %d", cycle.Number, minimal.Number)
		}
		if minimal.IsActive != cycle.IsActive {
			t.Errorf("Expected IsActive %v, got %v", cycle.IsActive, minimal.IsActive)
		}
	})

	t.Run("ToCompact", func(t *testing.T) {
		compact := cycle.ToCompact()
		if compact.ID != cycle.ID {
			t.Errorf("Expected ID %s, got %s", cycle.ID, compact.ID)
		}
		if compact.Name != cycle.Name {
			t.Errorf("Expected Name %s, got %s", cycle.Name, compact.Name)
		}
		if compact.StartsAt != cycle.StartsAt {
			t.Errorf("Expected StartsAt %s, got %s", cycle.StartsAt, compact.StartsAt)
		}
		if compact.EndsAt != cycle.EndsAt {
			t.Errorf("Expected EndsAt %s, got %s", cycle.EndsAt, compact.EndsAt)
		}
		if compact.Progress != cycle.Progress {
			t.Errorf("Expected Progress %f, got %f", cycle.Progress, compact.Progress)
		}
	})
}

// TestListCyclesRequestBody verifies that the correct GraphQL query is sent
func TestListCyclesRequestBody(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": {
				"cycles": {
					"nodes": [],
					"pageInfo": {"hasNextPage": false, "endCursor": ""}
				}
			}
		}`))
	}))
	defer server.Close()

	base := &BaseClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
		tokenProvider: token.NewStaticProvider("test-token"),
	}
	cc := NewCycleClient(base)

	active := true
	filter := &CycleFilter{
		TeamID:   "team-123",
		IsActive: &active,
		Limit:    5,
	}

	_, err := cc.ListCycles(filter)
	if err != nil {
		t.Fatalf("ListCycles() error = %v", err)
	}

	// Verify the query was sent
	if receivedBody["query"] == nil {
		t.Error("Expected query to be sent")
	}

	// Verify variables were sent
	variables, ok := receivedBody["variables"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected variables to be sent")
	}

	// Check first (limit) variable
	if first, ok := variables["first"].(float64); !ok || int(first) != 5 {
		t.Errorf("Expected first=5, got %v", variables["first"])
	}

	// Check filter variable contains the team ID filter
	filterVar, ok := variables["filter"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected filter to be sent")
	}

	teamFilter, ok := filterVar["team"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected team filter to be sent")
	}

	idFilter, ok := teamFilter["id"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected id filter to be sent")
	}

	if idFilter["eq"] != "team-123" {
		t.Errorf("Expected team ID team-123, got %v", idFilter["eq"])
	}
}

// TestGetCycleIssuesEmptyCycleID verifies validation
func TestGetCycleIssuesEmptyCycleID(t *testing.T) {
	base := &BaseClient{
		httpClient: http.DefaultClient,
		baseURL:    "http://example.com",
		tokenProvider: token.NewStaticProvider("test-token"),
	}
	cc := NewCycleClient(base)

	_, err := cc.GetCycleIssues("", 50)
	if err == nil {
		t.Error("Expected error for empty cycle ID")
	}

	if !strings.Contains(err.Error(), "cycleID cannot be empty") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}
