package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/pkg/linear"
	"github.com/joa23/linear-cli/pkg/linear/comments"
	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/issues"
	"github.com/joa23/linear-cli/pkg/linear/teams"
	"github.com/joa23/linear-cli/pkg/linear/workflows"
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
func (m *mockIssueClientForCreate) UpdateIssueState(id, state string) error { return nil }
func (m *mockIssueClientForCreate) AssignIssue(id, assignee string) error   { return nil }
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
			Title:      "My issue",
			TeamID:     "TL",
			AssigneeID: "john@company.com",
			LabelIDs:   []string{"Bugfix", "Feature"},
			Priority:   &priority,
			Estimate:   &estimate,
			DueDate:    "2026-03-01",
			ParentID:   "parent-uuid",
			ProjectID:  "project-uuid",
			CycleID:    "65",
		}, format.OutputText)

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
		}, format.OutputText)

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
		}, format.OutputText)

		if err == nil {
			t.Fatal("Create() should have returned an error")
		}
		if mock.updateCalled {
			t.Fatal("UpdateIssue was called after CreateIssue failure — orphaned issue risk")
		}
	})
}

// Create must report the issue it created, never the description it was created
// with. Rendering the new issue at Full echoed the whole body back and buried the
// identifier on line 1, so a caller reading the tail of that output saw only its
// own description and concluded the create had failed — then retried a write that
// had already landed and filed a duplicate issue.
func TestIssueService_Create_ReportsIdentifierNotDescription(t *testing.T) {
	// A body long enough to bury the identifier, carrying the kind of dedupe
	// marker an automated filer embeds so it can recognize its own issues later.
	const body = "Filed automatically.\n\n<!-- dedupe:0123456789 -->"
	const identifier = "ABC-123"
	const issueURL = "https://linear.app/acme/issue/ABC-123"

	newSvc := func() (*IssueService, *mockIssueClientForCreate) {
		mock := &mockIssueClientForCreate{
			createResult: &core.Issue{
				ID:          "issue-uuid",
				Identifier:  identifier,
				Title:       "Fix the thing",
				Description: body,
				URL:         issueURL,
			},
		}
		return makeIssueServiceForCreate(mock), mock
	}

	t.Run("text output leads with the identifier and omits the description", func(t *testing.T) {
		svc, _ := newSvc()

		out, err := svc.Create(&CreateIssueInput{Title: "t", TeamID: "ABC", Description: body}, format.OutputText)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		firstLine, _, _ := strings.Cut(out, "\n")
		if !strings.HasPrefix(firstLine, identifier+":") {
			t.Errorf("first line = %q, want it to start with the identifier", firstLine)
		}
		if strings.Contains(out, "<!-- dedupe:0123456789 -->") || strings.Contains(out, "Filed automatically") {
			t.Errorf("create echoed the description back:\n%s", out)
		}
		if !strings.Contains(out, issueURL) {
			t.Errorf("create did not report the issue URL:\n%s", out)
		}
	})

	t.Run("json output carries the identifier for scripted callers", func(t *testing.T) {
		svc, _ := newSvc()

		out, err := svc.Create(&CreateIssueInput{Title: "t", TeamID: "ABC", Description: body}, format.OutputJSON)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		var got struct {
			Identifier string `json:"identifier"`
			URL        string `json:"url"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("create --output json is not valid JSON: %v\n%s", err, out)
		}
		if got.Identifier != identifier {
			t.Errorf("identifier = %q, want %q", got.Identifier, identifier)
		}
		if got.URL != issueURL {
			t.Errorf("url = %q, want the issue URL", got.URL)
		}
	})
}
