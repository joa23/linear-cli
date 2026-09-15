package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
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

	// resolveLabelErr, when set, makes label resolution fail. Its zero value
	// preserves the default resolver behaviour so existing tests are unaffected.
	resolveLabelErr error

	// realCreateClient, when set, makes CreateIssue delegate to a real issues
	// client instead of returning createResult. That lets one test exercise the
	// genuine GraphQL mutation and deserializer against a fake server.
	realCreateClient *issues.Client
}

func (m *mockIssueClientForCreate) CreateIssue(input *core.IssueCreateInput) (*core.Issue, error) {
	m.createCalled = true
	m.lastCreateInput = input
	if m.realCreateClient != nil {
		return m.realCreateClient.CreateIssue(input)
	}
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
	if m.resolveLabelErr != nil {
		return "", m.resolveLabelErr
	}
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

// Create's JSON output must report the labels that were applied. It previously
// reported `"labels": null` whatever was passed, because the issueCreate mutation
// never asked for them back — so a scripted caller confirming label application
// from the create response got a false negative and had to pay an extra
// `issues get` round-trip.
//
// These tests exercise the DTO/formatter path only: the mock hands back a
// core.Issue the test built itself, so they can never prove the GraphQL selection
// set is correct. That guard lives in pkg/linear/issues/client_create_test.go.
// Both layers are required; neither covers this on its own.
func TestIssueService_Create_ReportsAppliedLabels(t *testing.T) {
	const identifier = "ABC-123"
	const issueURL = "https://linear.app/acme/issue/ABC-123"

	// createdIssueWithLabels mirrors what the fixed mutation now returns. The
	// mock's default createResult carries no labels at all, so a test copied from
	// the surrounding pattern would pass vacuously without this.
	createdIssueWithLabels := func(labels ...core.Label) *core.Issue {
		issue := &core.Issue{
			ID:         "issue-uuid",
			Identifier: identifier,
			Title:      "Fix the thing",
			URL:        issueURL,
		}
		if labels != nil {
			issue.Labels = &core.LabelConnection{Nodes: labels}
		}
		return issue
	}

	// labelsFrom unmarshals just the labels array out of create's JSON output.
	labelsFrom := func(t *testing.T, out string) []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} {
		t.Helper()
		var got struct {
			Labels []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"labels"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("create --output json is not valid JSON: %v\n%s", err, out)
		}
		return got.Labels
	}

	t.Run("single label is reported, not null", func(t *testing.T) {
		mock := &mockIssueClientForCreate{
			createResult: createdIssueWithLabels(core.Label{ID: "label-1", Name: "Bugfix"}),
		}
		svc := makeIssueServiceForCreate(mock)

		out, err := svc.Create(&CreateIssueInput{
			Title:    "Fix the thing",
			TeamID:   "ABC",
			LabelIDs: []string{"Bugfix"},
		}, format.OutputJSON)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		labels := labelsFrom(t, out)
		if len(labels) != 1 {
			t.Fatalf("expected 1 label, got %d\n%s", len(labels), out)
		}
		if labels[0].ID != "label-1" || labels[0].Name != "Bugfix" {
			t.Errorf("labels[0] = {%s %s}, want {label-1 Bugfix}", labels[0].ID, labels[0].Name)
		}
	})

	t.Run("multiple labels are all reported", func(t *testing.T) {
		mock := &mockIssueClientForCreate{
			createResult: createdIssueWithLabels(
				core.Label{ID: "label-1", Name: "Bugfix"},
				core.Label{ID: "label-2", Name: "Feature"},
			),
		}
		svc := makeIssueServiceForCreate(mock)

		out, err := svc.Create(&CreateIssueInput{
			Title:    "Fix the thing",
			TeamID:   "ABC",
			LabelIDs: []string{"Bugfix", "Feature"},
		}, format.OutputJSON)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		// Order is legitimate to assert here because this test supplies the
		// core.Issue itself and the DTO loop preserves slice order. Live responses
		// are not documented to preserve labelIds order, so the manual
		// create-vs-get parity check compares them as a set.
		labels := labelsFrom(t, out)
		if len(labels) != 2 {
			t.Fatalf("expected 2 labels, got %d\n%s", len(labels), out)
		}
		if labels[0].Name != "Bugfix" || labels[1].Name != "Feature" {
			t.Errorf("labels = [%s %s], want [Bugfix Feature]", labels[0].Name, labels[1].Name)
		}
	})

	t.Run("no labels renders an empty array, never null", func(t *testing.T) {
		mock := &mockIssueClientForCreate{createResult: createdIssueWithLabels()}
		svc := makeIssueServiceForCreate(mock)

		out, err := svc.Create(&CreateIssueInput{Title: "Fix the thing", TeamID: "ABC"}, format.OutputJSON)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		// The service path pretty-prints, so match the key tolerantly rather than
		// pinning the exact spacing.
		if !regexp.MustCompile(`"labels":\s*\[\s*\]`).MatchString(out) {
			t.Errorf("expected labels to render as [], got:\n%s", out)
		}
		if regexp.MustCompile(`"labels":\s*null`).MatchString(out) {
			t.Errorf("labels rendered as null — indistinguishable from 'not reported':\n%s", out)
		}
	})

	t.Run("text output is unchanged by the labels fix", func(t *testing.T) {
		// Text mode renders via formatter.IssueCreated, a path the DTO change does
		// not touch at all.
		mock := &mockIssueClientForCreate{
			createResult: createdIssueWithLabels(core.Label{ID: "label-1", Name: "Bugfix"}),
		}
		svc := makeIssueServiceForCreate(mock)

		out, err := svc.Create(&CreateIssueInput{
			Title:    "Fix the thing",
			TeamID:   "ABC",
			LabelIDs: []string{"Bugfix"},
		}, format.OutputText)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		firstLine, _, _ := strings.Cut(out, "\n")
		if firstLine != identifier+": Fix the thing" {
			t.Errorf("first line = %q, want %q", firstLine, identifier+": Fix the thing")
		}
		if !strings.Contains(out, issueURL) {
			t.Errorf("text output did not report the issue URL:\n%s", out)
		}
		if strings.Contains(out, "Bugfix") || strings.Contains(out, "label") {
			t.Errorf("text output should not mention labels:\n%s", out)
		}
	})

	t.Run("fields create already reported correctly still survive", func(t *testing.T) {
		issue := createdIssueWithLabels(core.Label{ID: "label-1", Name: "Bugfix"})
		issue.State.ID = "state-uuid"
		issue.State.Name = "Todo"
		issue.Assignee = &core.User{ID: "user-1", Name: "Ada", Email: "ada@example.com"}
		issue.Creator = &core.User{ID: "user-2", Name: "Grace", Email: "grace@example.com"}
		issue.Project = &core.Project{ID: "project-uuid", Name: "Platform"}
		issue.Parent = &core.ParentIssue{ID: "parent-uuid", Identifier: "ABC-100", Title: "Parent issue"}
		issue.CreatedAt = "2026-03-01T10:00:00.000Z"
		issue.UpdatedAt = "2026-03-01T11:00:00.000Z"

		mock := &mockIssueClientForCreate{createResult: issue}
		svc := makeIssueServiceForCreate(mock)

		out, err := svc.Create(&CreateIssueInput{
			Title:    "Fix the thing",
			TeamID:   "ABC",
			LabelIDs: []string{"Bugfix"},
		}, format.OutputJSON)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		var got struct {
			Identifier string                 `json:"identifier"`
			Title      string                 `json:"title"`
			URL        string                 `json:"url"`
			CreatedAt  string                 `json:"createdAt"`
			UpdatedAt  string                 `json:"updatedAt"`
			State      *struct{ Name string } `json:"state"`
			Assignee   *struct{ Name string } `json:"assignee"`
			Creator    *struct{ Name string } `json:"creator"`
			Project    *struct{ Name string } `json:"project"`
			Parent     *struct {
				Identifier string `json:"identifier"`
			} `json:"parent"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("create --output json is not valid JSON: %v\n%s", err, out)
		}

		if got.Identifier != identifier || got.Title != "Fix the thing" || got.URL != issueURL {
			t.Errorf("identifier/title/url = %q/%q/%q", got.Identifier, got.Title, got.URL)
		}
		if got.CreatedAt == "" || got.UpdatedAt == "" {
			t.Errorf("timestamps missing: createdAt=%q updatedAt=%q", got.CreatedAt, got.UpdatedAt)
		}
		if got.State == nil || got.State.Name != "Todo" {
			t.Errorf("state = %+v, want Todo", got.State)
		}
		if got.Assignee == nil || got.Assignee.Name != "Ada" {
			t.Errorf("assignee = %+v, want Ada", got.Assignee)
		}
		if got.Creator == nil || got.Creator.Name != "Grace" {
			t.Errorf("creator = %+v, want Grace", got.Creator)
		}
		if got.Project == nil || got.Project.Name != "Platform" {
			t.Errorf("project = %+v, want Platform", got.Project)
		}
		if got.Parent == nil || got.Parent.Identifier != "ABC-100" {
			t.Errorf("parent = %+v, want ABC-100", got.Parent)
		}
	})
}

// A label name that cannot be resolved must fail before the mutation runs, so no
// issue is created with the wrong labels and no JSON is emitted for it.
func TestIssueService_Create_UnresolvableLabelFailsBeforeMutation(t *testing.T) {
	mock := &mockIssueClientForCreate{
		resolveLabelErr: fmt.Errorf("label not found"),
		createResult:    &core.Issue{ID: "issue-uuid", Identifier: "ABC-123", Title: "Fix the thing"},
	}
	svc := makeIssueServiceForCreate(mock)

	out, err := svc.Create(&CreateIssueInput{
		Title:    "Fix the thing",
		TeamID:   "ABC",
		LabelIDs: []string{"definitely-not-a-label"},
	}, format.OutputJSON)

	if err == nil {
		t.Fatal("Create() should have failed on an unresolvable label")
	}
	if !strings.Contains(err.Error(), "definitely-not-a-label") {
		t.Errorf("error should name the offending label, got: %v", err)
	}
	if mock.createCalled {
		t.Error("CreateIssue was called despite label resolution failing — orphaned issue risk")
	}
	if out != "" {
		t.Errorf("no JSON should be emitted on failure, got: %s", out)
	}
}

// A failed mutation must surface as an error, never as a partially-populated
// JSON issue a caller could mistake for a successful create.
func TestIssueService_Create_MutationFailureEmitsNoJSON(t *testing.T) {
	mock := &mockIssueClientForCreate{
		createErr: fmt.Errorf("issue creation was not successful"),
	}
	svc := makeIssueServiceForCreate(mock)

	out, err := svc.Create(&CreateIssueInput{
		Title:    "Fix the thing",
		TeamID:   "ABC",
		LabelIDs: []string{"Bugfix"},
	}, format.OutputJSON)

	if err == nil {
		t.Fatal("Create() should have returned an error")
	}
	if out != "" {
		t.Errorf("no JSON should be emitted on failure, got: %s", out)
	}
}

// TestIssueService_Create_EndToEndReportsLabels drives Create through a real
// issues.Client pointed at a fake Linear server, so the assertion covers the
// whole chain: the GraphQL selection set, the response deserializer, the DTO
// layer, and the formatter.
//
// The mock-based tests above cannot do this — they hand back a core.Issue the
// test built itself, so they would stay green if the mutation stopped asking for
// labels. This one fails if either layer regresses, but only because its handler
// asserts on the captured mutation: the canned response below carries labels
// unconditionally, so without that assertion the selection set would go
// uncovered here. The per-field guard lives in
// TestCreateIssue_MutationRequestsCallerSettableFields
// (pkg/linear/issues/client_create_test.go).
//
// This test does not replace the mocks either: their injectable failures are
// what cover the error paths.
func TestIssueService_Create_EndToEndReportsLabels(t *testing.T) {
	const response = `{
		"data": {
			"issueCreate": {
				"success": true,
				"issue": {
					"id": "issue-uuid",
					"identifier": "ABC-123",
					"title": "Fix the thing",
					"url": "https://linear.app/acme/issue/ABC-123",
					"labels": {
						"nodes": [
							{"id": "label-1", "name": "Bugfix", "color": "#eb5757"}
						]
					}
				}
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The canned response carries labels whatever the mutation asked for, so
		// the selection set is only covered if the request is actually inspected.
		// Brace-anchored: a bare "labels" would also match labelIds in the input.
		var payload struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("outgoing GraphQL request is not valid JSON: %v", err)
		} else if !regexp.MustCompile(`(?m)^\s*labels\s*\{\s*$`).MatchString(payload.Query) {
			t.Errorf("the issueCreate mutation never asks for labels back:\n%s", payload.Query)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	realClient := issues.NewClient(core.NewTestBaseClient("test-token", server.URL, server.Client()))
	mock := &mockIssueClientForCreate{realCreateClient: realClient}
	svc := makeIssueServiceForCreate(mock)

	out, err := svc.Create(&CreateIssueInput{
		Title:    "Fix the thing",
		TeamID:   "ABC",
		LabelIDs: []string{"Bugfix"},
	}, format.OutputJSON)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var got struct {
		Identifier string `json:"identifier"`
		Labels     []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("create --output json is not valid JSON: %v\n%s", err, out)
	}

	if got.Identifier != "ABC-123" {
		t.Errorf("identifier = %q, want ABC-123", got.Identifier)
	}
	if len(got.Labels) != 1 {
		t.Fatalf("expected 1 label to survive the full chain, got %d\n%s", len(got.Labels), out)
	}
	if got.Labels[0].ID != "label-1" || got.Labels[0].Name != "Bugfix" {
		t.Errorf("labels[0] = {%s %s}, want {label-1 Bugfix}", got.Labels[0].ID, got.Labels[0].Name)
	}
}
