package cli

import (
	"testing"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/service"
)

type projectListServiceMock struct {
	mineCalls int
	teamCalls int
}

func (m *projectListServiceMock) Get(string) (string, error) { return "", nil }
func (m *projectListServiceMock) GetWithOutput(string, format.Verbosity, format.OutputType) (string, error) {
	return "[]", nil
}
func (m *projectListServiceMock) ListAll(int) (string, error) { return "", nil }
func (m *projectListServiceMock) ListAllWithOutput(int, format.Verbosity, format.OutputType) (string, error) {
	return "[]", nil
}
func (m *projectListServiceMock) ListAllWithStatusOutput(int, string, format.Verbosity, format.OutputType) (string, error) {
	return "[]", nil
}
func (m *projectListServiceMock) ListByTeam(string, int) (string, error) { return "", nil }
func (m *projectListServiceMock) ListByTeamWithOutput(string, int, format.Verbosity, format.OutputType) (string, error) {
	return "[]", nil
}
func (m *projectListServiceMock) ListByTeamWithStatusOutput(string, int, string, format.Verbosity, format.OutputType) (string, error) {
	m.teamCalls++
	return "[]", nil
}
func (m *projectListServiceMock) ListUserProjects(int) (string, error) { return "", nil }
func (m *projectListServiceMock) ListUserProjectsWithOutput(int, format.Verbosity, format.OutputType) (string, error) {
	m.mineCalls++
	return "[]", nil
}
func (m *projectListServiceMock) ListUserProjectsWithStatusOutput(int, string, format.Verbosity, format.OutputType) (string, error) {
	m.mineCalls++
	return "[]", nil
}
func (m *projectListServiceMock) Create(*service.CreateProjectInput) (string, error) { return "", nil }
func (m *projectListServiceMock) Update(string, *service.UpdateProjectInput) (string, error) {
	return "", nil
}

func TestProjectsListExplicitEmptyStatusFailsBeforeDispatch(t *testing.T) {
	mock := &projectListServiceMock{}
	deps := &Dependencies{Projects: mock}
	cmd := NewCmdWithDeps(deps, newProjectsListCmd)
	cmd.SetArgs([]string{"--mine", "--status", ""})

	err := cmd.Execute()
	if err == nil || err.Error() != "project status filter contains an empty value" {
		t.Fatalf("error = %v, want empty-value validation", err)
	}
	if mock.mineCalls != 0 {
		t.Fatal("project list service was called for an explicit empty status")
	}
}

func TestProjectsListMalformedStatusEntriesFailBeforeDispatch(t *testing.T) {
	for _, status := range []string{"", ",In Progress", "In Progress,", "In Progress,,On Hold", "   "} {
		t.Run(status, func(t *testing.T) {
			mock := &projectListServiceMock{}
			cmd := NewCmdWithDeps(&Dependencies{Projects: mock}, newProjectsListCmd)
			cmd.SetArgs([]string{"--mine", "--status", status})

			err := cmd.Execute()
			if err == nil || err.Error() != "project status filter contains an empty value" {
				t.Fatalf("error = %v, want empty-value validation", err)
			}
			if mock.mineCalls != 0 {
				t.Fatal("project list service was called for malformed status")
			}
		})
	}
}

func TestProjectsListOmittedStatusDispatchesMinePath(t *testing.T) {
	mock := &projectListServiceMock{}
	deps := &Dependencies{Projects: mock}
	cmd := NewCmdWithDeps(deps, newProjectsListCmd)
	cmd.SetArgs([]string{"--mine"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if mock.mineCalls != 1 {
		t.Fatalf("mine calls = %d, want 1", mock.mineCalls)
	}
}

func TestProjectsListTeamStatusDispatchesTeamPath(t *testing.T) {
	mock := &projectListServiceMock{}
	cmd := NewCmdWithDeps(&Dependencies{Projects: mock}, newProjectsListCmd)
	cmd.SetArgs([]string{"--team", "ENG", "--status", "In Progress"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if mock.teamCalls != 1 || mock.mineCalls != 0 {
		t.Fatalf("dispatch = team %d, mine %d; want team path", mock.teamCalls, mock.mineCalls)
	}
}

func TestProjectsListStatusFlagExists(t *testing.T) {
	if flag := newProjectsListCmd().Flags().Lookup("status"); flag == nil {
		t.Fatal("projects list is missing --status")
	}
}
