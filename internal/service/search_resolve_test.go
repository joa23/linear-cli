package service

import (
	"fmt"
	"testing"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
	"github.com/joa23/linear-cli/internal/linear/comments"
	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/joa23/linear-cli/internal/linear/issues"
	"github.com/joa23/linear-cli/internal/linear/projects"
	"github.com/joa23/linear-cli/internal/linear/teams"
	"github.com/joa23/linear-cli/internal/linear/workflows"
)

// mockIssueClient implements IssueClientOperations for search resolution tests
type mockIssueClient struct {
	resolveTeamResult    string
	resolveTeamErr       error
	resolveLabelResult   string
	resolveLabelErr      error
	searchResult         *core.IssueSearchResult
	searchErr            error
	workflowClient       *workflows.Client
}

func (m *mockIssueClient) CreateIssue(input *core.IssueCreateInput) (*core.Issue, error) {
	return nil, nil
}
func (m *mockIssueClient) GetIssue(id string) (*core.Issue, error) { return nil, nil }
func (m *mockIssueClient) UpdateIssue(id string, input core.UpdateIssueInput) (*core.Issue, error) {
	return nil, nil
}
func (m *mockIssueClient) UpdateIssueState(id, state string) error { return nil }
func (m *mockIssueClient) AssignIssue(id, assignee string) error   { return nil }
func (m *mockIssueClient) ListAssignedIssues(limit int) ([]core.Issue, error) {
	return nil, nil
}
func (m *mockIssueClient) SearchIssues(filters *core.IssueSearchFilters) (*core.IssueSearchResult, error) {
	return m.searchResult, m.searchErr
}
func (m *mockIssueClient) ResolveTeamIdentifier(key string) (string, error) {
	return m.resolveTeamResult, m.resolveTeamErr
}
func (m *mockIssueClient) ResolveUserIdentifier(name string) (*linear.ResolvedUser, error) {
	return &linear.ResolvedUser{ID: "user-uuid"}, nil
}
func (m *mockIssueClient) ResolveCycleIdentifier(num, team string) (string, error) {
	return "cycle-uuid", nil
}
func (m *mockIssueClient) ResolveLabelIdentifier(label, team string) (string, error) {
	return m.resolveLabelResult, m.resolveLabelErr
}
func (m *mockIssueClient) UpdateIssueMetadataKey(id, key string, val interface{}) error {
	return nil
}
func (m *mockIssueClient) CommentClient() *comments.Client   { return nil }
func (m *mockIssueClient) WorkflowClient() *workflows.Client { return m.workflowClient }
func (m *mockIssueClient) IssueClient() *issues.Client       { return nil }
func (m *mockIssueClient) TeamClient() *teams.Client         { return nil }

// mockSearchClient implements SearchClientOperations for search resolution tests
type mockSearchClient struct {
	resolveTeamResult  string
	resolveTeamErr     error
	resolveLabelResult string
	resolveLabelErr    error
	searchResult       *core.IssueSearchResult
	searchErr          error
	workflowClient     *workflows.Client
}

func (m *mockSearchClient) SearchIssues(filters *core.IssueSearchFilters) (*core.IssueSearchResult, error) {
	return m.searchResult, m.searchErr
}
func (m *mockSearchClient) ListCycles(filter *core.CycleFilter) (*core.CycleSearchResult, error) {
	return nil, nil
}
func (m *mockSearchClient) ResolveTeamIdentifier(key string) (string, error) {
	return m.resolveTeamResult, m.resolveTeamErr
}
func (m *mockSearchClient) ResolveUserIdentifier(name string) (*linear.ResolvedUser, error) {
	return &linear.ResolvedUser{ID: "user-uuid"}, nil
}
func (m *mockSearchClient) ResolveCycleIdentifier(num, team string) (string, error) {
	return "cycle-uuid", nil
}
func (m *mockSearchClient) ResolveLabelIdentifier(label, team string) (string, error) {
	return m.resolveLabelResult, m.resolveLabelErr
}
func (m *mockSearchClient) IssueClient() *issues.Client       { return nil }
func (m *mockSearchClient) ProjectClient() *projects.Client    { return nil }
func (m *mockSearchClient) TeamClient() *teams.Client          { return nil }
func (m *mockSearchClient) WorkflowClient() *workflows.Client  { return m.workflowClient }

// --- IssueService.Search tests ---

func TestIssueService_Search_StateRequiresTeam(t *testing.T) {
	svc := NewIssueService(&mockIssueClient{}, format.New())

	_, err := svc.Search(&SearchFilters{
		StateIDs: []string{"In Progress"},
	})

	if err == nil {
		t.Fatal("expected error when filtering by state without team")
	}
	expected := "--team is required when filtering by state"
	if err.Error() != expected {
		t.Errorf("got error %q, want %q", err.Error(), expected)
	}
}

func TestIssueService_Search_LabelsRequiresTeam(t *testing.T) {
	svc := NewIssueService(&mockIssueClient{}, format.New())

	_, err := svc.Search(&SearchFilters{
		LabelIDs: []string{"bug"},
	})

	if err == nil {
		t.Fatal("expected error when filtering by labels without team")
	}
	expected := "--team is required when filtering by labels"
	if err.Error() != expected {
		t.Errorf("got error %q, want %q", err.Error(), expected)
	}
}

func TestIssueService_Search_LabelResolutionSuccess(t *testing.T) {
	mock := &mockIssueClient{
		resolveTeamResult:  "team-uuid-123",
		resolveLabelResult: "label-uuid-456",
		searchResult: &core.IssueSearchResult{
			Issues: []core.Issue{},
		},
	}
	svc := NewIssueService(mock, format.New())

	_, err := svc.Search(&SearchFilters{
		TeamID:   "CEN",
		LabelIDs: []string{"bug"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueService_Search_LabelResolutionError(t *testing.T) {
	mock := &mockIssueClient{
		resolveTeamResult: "team-uuid-123",
		resolveLabelErr:   fmt.Errorf("label 'nonexistent' not found"),
	}
	svc := NewIssueService(mock, format.New())

	_, err := svc.Search(&SearchFilters{
		TeamID:   "CEN",
		LabelIDs: []string{"nonexistent"},
	})

	if err == nil {
		t.Fatal("expected error from label resolution failure")
	}
}

// --- IssueService.SearchWithOutput tests ---

func TestIssueService_SearchWithOutput_StateRequiresTeam(t *testing.T) {
	svc := NewIssueService(&mockIssueClient{}, format.New())

	_, err := svc.SearchWithOutput(&SearchFilters{
		StateIDs: []string{"In Progress"},
	}, format.VerbosityCompact, format.OutputText)

	if err == nil {
		t.Fatal("expected error when filtering by state without team")
	}
	expected := "--team is required when filtering by state"
	if err.Error() != expected {
		t.Errorf("got error %q, want %q", err.Error(), expected)
	}
}

func TestIssueService_SearchWithOutput_LabelsRequiresTeam(t *testing.T) {
	svc := NewIssueService(&mockIssueClient{}, format.New())

	_, err := svc.SearchWithOutput(&SearchFilters{
		LabelIDs: []string{"bug"},
	}, format.VerbosityCompact, format.OutputText)

	if err == nil {
		t.Fatal("expected error when filtering by labels without team")
	}
	expected := "--team is required when filtering by labels"
	if err.Error() != expected {
		t.Errorf("got error %q, want %q", err.Error(), expected)
	}
}

func TestIssueService_SearchWithOutput_LabelResolutionSuccess(t *testing.T) {
	mock := &mockIssueClient{
		resolveTeamResult:  "team-uuid-123",
		resolveLabelResult: "label-uuid-456",
		searchResult: &core.IssueSearchResult{
			Issues: []core.Issue{},
		},
	}
	svc := NewIssueService(mock, format.New())

	_, err := svc.SearchWithOutput(&SearchFilters{
		TeamID:   "CEN",
		LabelIDs: []string{"bug"},
	}, format.VerbosityCompact, format.OutputText)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- SearchService.Search tests ---

func TestSearchService_Search_StateRequiresTeam(t *testing.T) {
	svc := NewSearchService(&mockSearchClient{}, format.New())

	_, err := svc.Search(&SearchOptions{
		StateIDs: []string{"In Progress"},
	})

	if err == nil {
		t.Fatal("expected error when filtering by state without team")
	}
	expected := "--team is required when filtering by state"
	if err.Error() != expected {
		t.Errorf("got error %q, want %q", err.Error(), expected)
	}
}

func TestSearchService_Search_LabelsRequiresTeam(t *testing.T) {
	svc := NewSearchService(&mockSearchClient{}, format.New())

	_, err := svc.Search(&SearchOptions{
		LabelIDs: []string{"bug"},
	})

	if err == nil {
		t.Fatal("expected error when filtering by labels without team")
	}
	expected := "--team is required when filtering by labels"
	if err.Error() != expected {
		t.Errorf("got error %q, want %q", err.Error(), expected)
	}
}

func TestSearchService_Search_LabelResolutionSuccess(t *testing.T) {
	mock := &mockSearchClient{
		resolveTeamResult:  "team-uuid-123",
		resolveLabelResult: "label-uuid-456",
		searchResult: &core.IssueSearchResult{
			Issues: []core.Issue{},
		},
	}
	svc := NewSearchService(mock, format.New())

	_, err := svc.Search(&SearchOptions{
		TeamID:   "CEN",
		LabelIDs: []string{"bug"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchService_Search_LabelResolutionError(t *testing.T) {
	mock := &mockSearchClient{
		resolveTeamResult: "team-uuid-123",
		resolveLabelErr:   fmt.Errorf("label 'nonexistent' not found"),
	}
	svc := NewSearchService(mock, format.New())

	_, err := svc.Search(&SearchOptions{
		TeamID:   "CEN",
		LabelIDs: []string{"nonexistent"},
	})

	if err == nil {
		t.Fatal("expected error from label resolution failure")
	}
}
