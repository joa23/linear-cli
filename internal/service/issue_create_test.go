package service

import (
	"fmt"
	"testing"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
	"github.com/joa23/linear-cli/internal/linear/comments"
	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/joa23/linear-cli/internal/linear/issues"
	"github.com/joa23/linear-cli/internal/linear/teams"
	"github.com/joa23/linear-cli/internal/linear/workflows"
)

// mockIssueClientForCreate records CreateIssue and UpdateIssue calls to verify
// that issue creation is atomic (UpdateIssue must never be called after CreateIssue).
type mockIssueClientForCreate struct {
	// Captured inputs
	lastCreateInput *core.IssueCreateInput

	// Call tracking
	createCalled bool
	updateCalled bool

	// Configured return values
	createResult *core.Issue
	createErr    error
}

func (m *mockIssueClientForCreate) CreateIssue(input *core.IssueCreateInput) (*core.Issue, error) {
	m.createCalled = true
	m.lastCreateInput = input
	return m.createResult, m.createErr
}

func (m *mockIssueClientForCreate) UpdateIssue(id string, input core.UpdateIssueInput) (*core.Issue, error) {
	m.updateCalled = true
	return nil, nil
}

// Resolver stubs — return predictable UUIDs.
func (m *mockIssueClientForCreate) ResolveTeamIdentifier(key string) (string, error) {
	return "team-uuid", nil
}
func (m *mockIssueClientForCreate) ResolveUserIdentifier(nameOrEmail string) (*linear.ResolvedUser, error) {
	return &linear.ResolvedUser{ID: "user-uuid", IsApplication: false}, nil
}
func (m *mockIssueClientForCreate) ResolveCycleIdentifier(num, team string) (string, error) {
	return "cycle-uuid", nil
}
func (m *mockIssueClientForCreate) ResolveLabelIdentifier(label, team string) (string, error) {
	return "label-uuid-" + label, nil
}

// Unused interface methods.
func (m *mockIssueClientForCreate) GetIssue(id string) (*core.Issue, error) { return nil, nil }
func (m *mockIssueClientForCreate) UpdateIssueState(id, state string) error  { return nil }
func (m *mockIssueClientForCreate) AssignIssue(id, assignee string) error    { return nil }
func (m *mockIssueClientForCreate) ListAssignedIssues(limit int) ([]core.Issue, error) {
	return nil, nil
}
func (m *mockIssueClientForCreate) SearchIssues(filters *core.IssueSearchFilters) (*core.IssueSearchResult, error) {
	return nil, nil
}
func (m *mockIssueClientForCreate) UpdateIssueMetadataKey(id, key string, val interface{}) error {
	return nil
}
func (m *mockIssueClientForCreate) CreateRelation(issueID, relatedIssueID string, relationType core.IssueRelationType) error {
	return nil
}
func (m *mockIssueClientForCreate) ResolveProjectIdentifier(nameOrID, teamID string) (string, error) {
	return "project-uuid", nil
}
func (m *mockIssueClientForCreate) CommentClient() *comments.Client   { return nil }
func (m *mockIssueClientForCreate) WorkflowClient() *workflows.Client { return nil }
func (m *mockIssueClientForCreate) IssueClient() *issues.Client       { return nil }
func (m *mockIssueClientForCreate) TeamClient() *teams.Client         { return nil }

// makeIssueServiceForCreate creates an IssueService backed by the given mock.
func makeIssueServiceForCreate(mock *mockIssueClientForCreate) *IssueService {
	return NewIssueService(mock, format.New())
}

func TestIssueService_Create_AtomicFields(t *testing.T) {
	priority := 1
	estimate := 3.0

	t.Run("all optional fields go through CreateIssue, UpdateIssue never called", func(t *testing.T) {
		fakeIssue := &core.Issue{ID: "issue-123", Identifier: "TL-1", Title: "My issue"}
		mock := &mockIssueClientForCreate{
			createResult: fakeIssue,
		}
		svc := makeIssueServiceForCreate(mock)

		_, err := svc.Create(&CreateIssueInput{
			Title:     "My issue",
			TeamID:    "TL",
			AssigneeID: "john@company.com",
			LabelIDs:  []string{"Bugfix", "Feature"},
			Priority:  &priority,
			Estimate:  &estimate,
			DueDate:   "2026-03-01",
			ParentID:  "parent-uuid",
			ProjectID: "project-uuid",
			CycleID:   "65",
		})

		if err != nil {
			t.Fatalf("Create() returned unexpected error: %v", err)
		}
		if !mock.createCalled {
			t.Fatal("CreateIssue was not called")
		}
		if mock.updateCalled {
			t.Fatal("UpdateIssue was called — issue creation is not atomic")
		}

		in := mock.lastCreateInput
		if in == nil {
			t.Fatal("lastCreateInput is nil")
		}
		if in.Title != "My issue" {
			t.Errorf("Title = %q, want %q", in.Title, "My issue")
		}
		if in.TeamID != "team-uuid" {
			t.Errorf("TeamID = %q, want %q", in.TeamID, "team-uuid")
		}
		if in.AssigneeID != "user-uuid" {
			t.Errorf("AssigneeID = %q, want %q", in.AssigneeID, "user-uuid")
		}
		if len(in.LabelIDs) != 2 {
			t.Errorf("len(LabelIDs) = %d, want 2", len(in.LabelIDs))
		}
		if in.Priority == nil || *in.Priority != priority {
			t.Errorf("Priority = %v, want %d", in.Priority, priority)
		}
		if in.Estimate == nil || *in.Estimate != estimate {
			t.Errorf("Estimate = %v, want %f", in.Estimate, estimate)
		}
		if in.DueDate != "2026-03-01" {
			t.Errorf("DueDate = %q, want %q", in.DueDate, "2026-03-01")
		}
		if in.ParentID != "parent-uuid" {
			t.Errorf("ParentID = %q, want %q", in.ParentID, "parent-uuid")
		}
		if in.ProjectID != "project-uuid" {
			t.Errorf("ProjectID = %q, want %q", in.ProjectID, "project-uuid")
		}
		if in.CycleID != "cycle-uuid" {
			t.Errorf("CycleID = %q, want %q", in.CycleID, "cycle-uuid")
		}
	})

	t.Run("minimal creation (title + team only) never calls UpdateIssue", func(t *testing.T) {
		fakeIssue := &core.Issue{ID: "issue-456", Identifier: "TL-2", Title: "Minimal"}
		mock := &mockIssueClientForCreate{
			createResult: fakeIssue,
		}
		svc := makeIssueServiceForCreate(mock)

		_, err := svc.Create(&CreateIssueInput{
			Title:  "Minimal",
			TeamID: "TL",
		})

		if err != nil {
			t.Fatalf("Create() returned unexpected error: %v", err)
		}
		if !mock.createCalled {
			t.Fatal("CreateIssue was not called")
		}
		if mock.updateCalled {
			t.Fatal("UpdateIssue was called for minimal creation")
		}
	})

	t.Run("CreateIssue failure returns error without calling UpdateIssue", func(t *testing.T) {
		mock := &mockIssueClientForCreate{
			createErr: fmt.Errorf("simulated API error"),
		}
		svc := makeIssueServiceForCreate(mock)

		_, err := svc.Create(&CreateIssueInput{
			Title:    "Will fail",
			TeamID:   "TL",
			LabelIDs: []string{"Bugfix"},
			Priority: &priority,
		})

		if err == nil {
			t.Fatal("Create() should have returned an error")
		}
		if mock.updateCalled {
			t.Fatal("UpdateIssue was called after CreateIssue failure — orphaned issue risk")
		}
	})
}
