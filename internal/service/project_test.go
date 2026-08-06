package service

import (
	"errors"
	"testing"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/pkg/linear"
	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/projects"
)

type projectServiceMock struct {
	resolvedTeam string
	viewer       *core.User
	statusIDs    []string
	statusNames  []string
	projects     []core.Project
	teamCalls    int
	userCalls    int
	allCalls     int
	resolveErr   error
}

func (m *projectServiceMock) CreateProject(string, string, string) (*core.Project, error) {
	return nil, nil
}
func (m *projectServiceMock) GetProject(string) (*core.Project, error) { return nil, nil }
func (m *projectServiceMock) ListAllProjectsWithStatus(_ int, ids []string) ([]core.Project, error) {
	m.allCalls++
	m.statusIDs = append([]string(nil), ids...)
	return m.projects, nil
}
func (m *projectServiceMock) ListByTeamWithStatus(_ string, _ int, ids []string) ([]core.Project, error) {
	m.teamCalls++
	m.statusIDs = append([]string(nil), ids...)
	return m.projects, nil
}
func (m *projectServiceMock) ListUserProjectsWithStatus(_ string, _ int, ids []string) ([]core.Project, error) {
	m.userCalls++
	m.statusIDs = append([]string(nil), ids...)
	return m.projects, nil
}
func (m *projectServiceMock) ResolveProjectStatusNames(names []string) ([]string, error) {
	normalized, err := projects.NormalizeStatusNames(names)
	if err != nil {
		return nil, err
	}
	m.statusNames = append([]string(nil), normalized...)
	if m.resolveErr != nil {
		return nil, m.resolveErr
	}
	return []string{"status-1", "status-2"}, nil
}
func (m *projectServiceMock) GetViewer() (*core.User, error) { return m.viewer, nil }
func (m *projectServiceMock) UpdateProject(string, projects.UpdateProjectInput) (*core.Project, error) {
	return nil, nil
}
func (m *projectServiceMock) ResolveTeamIdentifier(string) (string, error) {
	return m.resolvedTeam, nil
}
func (m *projectServiceMock) ResolveUserIdentifier(string) (*linear.ResolvedUser, error) {
	return &linear.ResolvedUser{ID: "user-1"}, nil
}

func TestProjectServiceNormalizesStatusAndDispatchesTeamList(t *testing.T) {
	mock := &projectServiceMock{resolvedTeam: "team-1", projects: []core.Project{{ID: "p1", Name: "Project"}}}
	svc := NewProjectService(mock, format.New())

	_, err := svc.ListByTeamWithStatusOutput("ENG", 2, " In Progress,On Hold,in progress ", format.VerbosityCompact, format.OutputText)
	if err != nil {
		t.Fatalf("ListByTeamWithStatusOutput() error = %v", err)
	}
	if mock.teamCalls != 1 || len(mock.statusIDs) != 2 || mock.statusIDs[0] != "status-1" {
		t.Fatalf("team dispatch = calls %d, IDs %#v", mock.teamCalls, mock.statusIDs)
	}
	if len(mock.statusNames) != 2 || mock.statusNames[0] != "In Progress" || mock.statusNames[1] != "On Hold" {
		t.Fatalf("resolved names = %#v", mock.statusNames)
	}
}

func TestProjectServiceMineUsesViewerAndPropagatesStatusErrors(t *testing.T) {
	mock := &projectServiceMock{viewer: &core.User{ID: "viewer-1"}, resolveErr: errors.New("ambiguous")}
	svc := NewProjectService(mock, format.New())

	_, err := svc.ListUserProjectsWithStatusOutput(1, "In Progress", format.VerbosityCompact, format.OutputJSON)
	if err == nil || err.Error() != "ambiguous" {
		t.Fatalf("error = %v, want ambiguous", err)
	}
	if mock.userCalls != 0 {
		t.Fatal("project list called after status resolution failed")
	}
}

func TestProjectServiceUnfilteredListPreservesNilStatusIDs(t *testing.T) {
	mock := &projectServiceMock{resolvedTeam: "team-1", projects: []core.Project{{ID: "p1", Name: "Project"}}}
	svc := NewProjectService(mock, format.New())

	if _, err := svc.ListAllWithStatusOutput(1, "", format.VerbosityCompact, format.OutputJSON); err != nil {
		t.Fatalf("ListAllWithStatusOutput() error = %v", err)
	}
	if mock.allCalls != 1 || mock.statusIDs != nil {
		t.Fatalf("unfiltered dispatch = calls %d, IDs %#v", mock.allCalls, mock.statusIDs)
	}
}
