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

// relationCall records a CreateRelation call for assertion
type relationCall struct {
	issueID        string
	relatedIssueID string
	relationType   core.IssueRelationType
}

// mockIssueClientForRelation implements IssueClientOperations for relation tests
type mockIssueClientForRelation struct {
	// Capture calls
	relationCalls    []relationCall
	updateIssueCalls int
	lastUpdateInput  core.UpdateIssueInput

	// Error to return from CreateRelation
	createRelationErr error
}

func (m *mockIssueClientForRelation) CreateIssue(title, desc, team string) (*core.Issue, error) {
	return &core.Issue{ID: "new-issue-uuid", Identifier: "TEST-99"}, nil
}

func (m *mockIssueClientForRelation) GetIssue(identifier string) (*core.Issue, error) {
	return &core.Issue{
		ID:         "issue-uuid-123",
		Identifier: "TEST-1",
	}, nil
}

func (m *mockIssueClientForRelation) UpdateIssue(issueID string, input core.UpdateIssueInput) (*core.Issue, error) {
	m.updateIssueCalls++
	m.lastUpdateInput = input
	return &core.Issue{
		ID:         issueID,
		Identifier: "TEST-1",
	}, nil
}

func (m *mockIssueClientForRelation) CreateRelation(issueID, relatedIssueID string, relationType core.IssueRelationType) error {
	m.relationCalls = append(m.relationCalls, relationCall{
		issueID:        issueID,
		relatedIssueID: relatedIssueID,
		relationType:   relationType,
	})
	return m.createRelationErr
}

func (m *mockIssueClientForRelation) UpdateIssueState(id, state string) error { return nil }
func (m *mockIssueClientForRelation) AssignIssue(id, assignee string) error   { return nil }
func (m *mockIssueClientForRelation) ListAssignedIssues(limit int) ([]core.Issue, error) {
	return nil, nil
}
func (m *mockIssueClientForRelation) SearchIssues(filters *core.IssueSearchFilters) (*core.IssueSearchResult, error) {
	return nil, nil
}
func (m *mockIssueClientForRelation) ResolveTeamIdentifier(key string) (string, error) {
	return "team-uuid", nil
}
func (m *mockIssueClientForRelation) ResolveUserIdentifier(name string) (*linear.ResolvedUser, error) {
	return &linear.ResolvedUser{ID: "user-uuid"}, nil
}
func (m *mockIssueClientForRelation) ResolveCycleIdentifier(num, team string) (string, error) {
	return "cycle-uuid", nil
}
func (m *mockIssueClientForRelation) ResolveLabelIdentifier(label, team string) (string, error) {
	return "label-uuid", nil
}
func (m *mockIssueClientForRelation) UpdateIssueMetadataKey(id, key string, val interface{}) error {
	return nil
}
func (m *mockIssueClientForRelation) CommentClient() *comments.Client   { return nil }
func (m *mockIssueClientForRelation) WorkflowClient() *workflows.Client { return nil }
func (m *mockIssueClientForRelation) IssueClient() *issues.Client       { return nil }
func (m *mockIssueClientForRelation) TeamClient() *teams.Client         { return nil }

func TestIssueService_Update_BlockedByOnly_SkipsUpdateIssue(t *testing.T) {
	mock := &mockIssueClientForRelation{}
	svc := NewIssueService(mock, format.New())

	_, err := svc.Update("TEST-1", &UpdateIssueInput{
		BlockedBy: []string{"TEST-170"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.updateIssueCalls != 0 {
		t.Errorf("UpdateIssue called %d times, want 0 (should skip when only relations)", mock.updateIssueCalls)
	}
	if len(mock.relationCalls) != 1 {
		t.Fatalf("CreateRelation called %d times, want 1", len(mock.relationCalls))
	}
	call := mock.relationCalls[0]
	if call.issueID != "TEST-170" {
		t.Errorf("issueID = %q, want %q", call.issueID, "TEST-170")
	}
	if call.relatedIssueID != "TEST-1" {
		t.Errorf("relatedIssueID = %q, want %q", call.relatedIssueID, "TEST-1")
	}
	if call.relationType != core.RelationBlocks {
		t.Errorf("relationType = %q, want %q", call.relationType, core.RelationBlocks)
	}
}

func TestIssueService_Update_DependsOnOnly_SkipsUpdateIssue(t *testing.T) {
	mock := &mockIssueClientForRelation{}
	svc := NewIssueService(mock, format.New())

	_, err := svc.Update("TEST-1", &UpdateIssueInput{
		DependsOn: []string{"TEST-100"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.updateIssueCalls != 0 {
		t.Errorf("UpdateIssue called %d times, want 0", mock.updateIssueCalls)
	}
	if len(mock.relationCalls) != 1 {
		t.Fatalf("CreateRelation called %d times, want 1", len(mock.relationCalls))
	}
	call := mock.relationCalls[0]
	if call.issueID != "TEST-100" {
		t.Errorf("issueID = %q, want %q (dependency should be the blocker)", call.issueID, "TEST-100")
	}
	if call.relatedIssueID != "TEST-1" {
		t.Errorf("relatedIssueID = %q, want %q", call.relatedIssueID, "TEST-1")
	}
}

func TestIssueService_Update_BlockedByWithState_CallsBothUpdateAndRelation(t *testing.T) {
	mock := &mockIssueClientForRelation{}
	// WorkflowClient returns nil which would panic when resolving state.
	// Instead, test with a field that doesn't require resolution.
	priority := 1
	svc := NewIssueService(mock, format.New())

	_, err := svc.Update("TEST-1", &UpdateIssueInput{
		Priority:  &priority,
		BlockedBy: []string{"TEST-170"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.updateIssueCalls != 1 {
		t.Errorf("UpdateIssue called %d times, want 1", mock.updateIssueCalls)
	}
	if len(mock.relationCalls) != 1 {
		t.Fatalf("CreateRelation called %d times, want 1", len(mock.relationCalls))
	}
}

func TestIssueService_Update_MultipleBlockedBy_CreatesMultipleRelations(t *testing.T) {
	mock := &mockIssueClientForRelation{}
	svc := NewIssueService(mock, format.New())

	_, err := svc.Update("TEST-1", &UpdateIssueInput{
		BlockedBy: []string{"TEST-170", "TEST-171"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.updateIssueCalls != 0 {
		t.Errorf("UpdateIssue called %d times, want 0", mock.updateIssueCalls)
	}
	if len(mock.relationCalls) != 2 {
		t.Fatalf("CreateRelation called %d times, want 2", len(mock.relationCalls))
	}
	if mock.relationCalls[0].issueID != "TEST-170" {
		t.Errorf("first relation issueID = %q, want %q", mock.relationCalls[0].issueID, "TEST-170")
	}
	if mock.relationCalls[1].issueID != "TEST-171" {
		t.Errorf("second relation issueID = %q, want %q", mock.relationCalls[1].issueID, "TEST-171")
	}
}

func TestIssueService_Update_RelationError_FailsFast(t *testing.T) {
	mock := &mockIssueClientForRelation{
		createRelationErr: fmt.Errorf("API error: relation already exists"),
	}
	svc := NewIssueService(mock, format.New())

	_, err := svc.Update("TEST-1", &UpdateIssueInput{
		BlockedBy: []string{"TEST-170", "TEST-171"},
	})

	if err == nil {
		t.Fatal("expected error from relation creation failure")
	}
	// Should fail on first relation and not attempt second
	if len(mock.relationCalls) != 1 {
		t.Errorf("CreateRelation called %d times, want 1 (should fail fast)", len(mock.relationCalls))
	}
}

func TestIssueService_Create_DependsOn_CreatesRelation(t *testing.T) {
	mock := &mockIssueClientForRelation{}
	svc := NewIssueService(mock, format.New())

	_, err := svc.Create(&CreateIssueInput{
		Title:     "New issue",
		TeamID:    "TEST",
		DependsOn: []string{"TEST-100"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.relationCalls) != 1 {
		t.Fatalf("CreateRelation called %d times, want 1", len(mock.relationCalls))
	}
	call := mock.relationCalls[0]
	if call.issueID != "TEST-100" {
		t.Errorf("issueID = %q, want %q (dependency should be the blocker)", call.issueID, "TEST-100")
	}
	if call.relatedIssueID != "TEST-99" {
		t.Errorf("relatedIssueID = %q, want %q (new issue should be the blocked one)", call.relatedIssueID, "TEST-99")
	}
	if call.relationType != core.RelationBlocks {
		t.Errorf("relationType = %q, want %q", call.relationType, core.RelationBlocks)
	}
}

func TestIssueService_Update_BothDependsOnAndBlockedBy(t *testing.T) {
	mock := &mockIssueClientForRelation{}
	svc := NewIssueService(mock, format.New())

	_, err := svc.Update("TEST-1", &UpdateIssueInput{
		DependsOn: []string{"TEST-100"},
		BlockedBy: []string{"TEST-200"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.updateIssueCalls != 0 {
		t.Errorf("UpdateIssue called %d times, want 0", mock.updateIssueCalls)
	}
	if len(mock.relationCalls) != 2 {
		t.Fatalf("CreateRelation called %d times, want 2", len(mock.relationCalls))
	}
	// First call: depends-on (TEST-100 blocks TEST-1)
	if mock.relationCalls[0].issueID != "TEST-100" {
		t.Errorf("first relation issueID = %q, want %q", mock.relationCalls[0].issueID, "TEST-100")
	}
	// Second call: blocked-by (TEST-200 blocks TEST-1)
	if mock.relationCalls[1].issueID != "TEST-200" {
		t.Errorf("second relation issueID = %q, want %q", mock.relationCalls[1].issueID, "TEST-200")
	}
}

func TestHasServiceFieldsToUpdate(t *testing.T) {
	tests := []struct {
		name   string
		input  core.UpdateIssueInput
		expect bool
	}{
		{
			name:   "empty input has no fields",
			input:  core.UpdateIssueInput{},
			expect: false,
		},
		{
			name:   "title set",
			input:  core.UpdateIssueInput{Title: strPtr("new title")},
			expect: true,
		},
		{
			name:   "stateID set",
			input:  core.UpdateIssueInput{StateID: strPtr("state-uuid")},
			expect: true,
		},
		{
			name:   "labelIDs set",
			input:  core.UpdateIssueInput{LabelIDs: []string{"label-1"}},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasServiceFieldsToUpdate(tt.input)
			if got != tt.expect {
				t.Errorf("hasServiceFieldsToUpdate() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
