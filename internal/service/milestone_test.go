package service

import (
	"errors"
	"testing"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/milestones"
	"github.com/joa23/linear-cli/pkg/linear/projects"
)

// mockMilestoneClient implements MilestoneClientOperations with configurable
// return values and call tracking for MilestoneService tests.
type mockMilestoneClient struct {
	// Configured return values
	listResult   []core.ProjectMilestone
	listErr      error
	getResult    *core.ProjectMilestone
	getErr       error
	createResult *core.ProjectMilestone
	createErr    error
	updateResult *core.ProjectMilestone
	updateErr    error
	deleteErr    error

	resolveTeamResult      string
	resolveTeamErr         error
	resolveProjectResult   string
	resolveProjectErr      error
	resolveMilestoneResult string
	resolveMilestoneErr    error

	// Captured inputs
	lastListProjectID       string
	lastCreateInput         *core.CreateProjectMilestoneInput
	lastUpdateID            string
	lastUpdateInput         *core.UpdateProjectMilestoneInput
	lastDeleteID            string
	lastResolveMilestoneArg string
	lastResolveProjectArg   string
}

func (m *mockMilestoneClient) ListProjectMilestones(projectID string, limit int) ([]core.ProjectMilestone, error) {
	m.lastListProjectID = projectID
	return m.listResult, m.listErr
}

func (m *mockMilestoneClient) GetProjectMilestone(id string) (*core.ProjectMilestone, error) {
	return m.getResult, m.getErr
}

func (m *mockMilestoneClient) CreateProjectMilestone(input *core.CreateProjectMilestoneInput) (*core.ProjectMilestone, error) {
	m.lastCreateInput = input
	return m.createResult, m.createErr
}

func (m *mockMilestoneClient) UpdateProjectMilestone(id string, input *core.UpdateProjectMilestoneInput) (*core.ProjectMilestone, error) {
	m.lastUpdateID = id
	m.lastUpdateInput = input
	return m.updateResult, m.updateErr
}

func (m *mockMilestoneClient) DeleteProjectMilestone(id string) error {
	m.lastDeleteID = id
	return m.deleteErr
}

func (m *mockMilestoneClient) ResolveTeamIdentifier(keyOrName string) (string, error) {
	return m.resolveTeamResult, m.resolveTeamErr
}

func (m *mockMilestoneClient) ResolveProjectIdentifier(nameOrID, teamID string) (string, error) {
	m.lastResolveProjectArg = nameOrID
	return m.resolveProjectResult, m.resolveProjectErr
}

func (m *mockMilestoneClient) ResolveProjectMilestoneIdentifier(nameOrID, projectID string) (string, error) {
	m.lastResolveMilestoneArg = nameOrID
	return m.resolveMilestoneResult, m.resolveMilestoneErr
}

func (m *mockMilestoneClient) MilestoneClient() *milestones.Client { return nil }
func (m *mockMilestoneClient) ProjectClient() *projects.Client     { return nil }

func newMilestoneService(client *mockMilestoneClient) *MilestoneService {
	return NewMilestoneService(client, format.New())
}

func TestMilestoneService_List(t *testing.T) {
	t.Run("requires a project", func(t *testing.T) {
		client := &mockMilestoneClient{}
		s := newMilestoneService(client)

		_, err := s.List(&MilestoneListInput{}, format.VerbosityCompact, format.OutputText)
		if err == nil {
			t.Fatal("expected error when no project is provided")
		}
	})

	t.Run("resolves project and lists milestones", func(t *testing.T) {
		client := &mockMilestoneClient{
			resolveProjectResult: "project-uuid",
			listResult: []core.ProjectMilestone{
				{Name: "Alpha", Status: "done"},
			},
		}
		s := newMilestoneService(client)

		result, err := s.List(&MilestoneListInput{ProjectID: "Q3 Launch"}, format.VerbosityCompact, format.OutputText)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.lastListProjectID != "project-uuid" {
			t.Errorf("expected resolved project ID to be passed through, got %q", client.lastListProjectID)
		}
		if result == "" {
			t.Error("expected non-empty rendered result")
		}
	})

	t.Run("propagates client errors", func(t *testing.T) {
		client := &mockMilestoneClient{
			resolveProjectResult: "project-uuid",
			listErr:              errors.New("boom"),
		}
		s := newMilestoneService(client)

		_, err := s.List(&MilestoneListInput{ProjectID: "Q3 Launch"}, format.VerbosityCompact, format.OutputText)
		if err == nil {
			t.Fatal("expected error to propagate from client")
		}
	})
}

func TestMilestoneService_Get(t *testing.T) {
	client := &mockMilestoneClient{
		resolveMilestoneResult: "milestone-uuid",
		getResult:              &core.ProjectMilestone{Name: "Beta"},
	}
	s := newMilestoneService(client)

	result, err := s.Get("Beta", "Q3 Launch", "", format.VerbosityFull, format.OutputText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.lastResolveMilestoneArg != "Beta" {
		t.Errorf("expected resolver to be called with 'Beta', got %q", client.lastResolveMilestoneArg)
	}
	if result == "" {
		t.Error("expected non-empty rendered result")
	}
}

func TestMilestoneService_Create(t *testing.T) {
	t.Run("requires input", func(t *testing.T) {
		s := newMilestoneService(&mockMilestoneClient{})
		if _, err := s.Create(nil, format.VerbosityFull, format.OutputText); err == nil {
			t.Fatal("expected error for nil input")
		}
	})

	t.Run("requires a name", func(t *testing.T) {
		s := newMilestoneService(&mockMilestoneClient{})
		_, err := s.Create(&CreateMilestoneInput{ProjectID: "Q3 Launch"}, format.VerbosityFull, format.OutputText)
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("resolves project and creates milestone", func(t *testing.T) {
		client := &mockMilestoneClient{
			resolveProjectResult: "project-uuid",
			createResult:         &core.ProjectMilestone{Name: "Beta"},
		}
		s := newMilestoneService(client)

		_, err := s.Create(&CreateMilestoneInput{
			Name:      "Beta",
			ProjectID: "Q3 Launch",
		}, format.VerbosityFull, format.OutputText)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.lastCreateInput == nil || client.lastCreateInput.ProjectID != "project-uuid" {
			t.Error("expected create input to use the resolved project ID")
		}
	})
}

func TestMilestoneService_Update(t *testing.T) {
	t.Run("requires input", func(t *testing.T) {
		s := newMilestoneService(&mockMilestoneClient{})
		if _, err := s.Update("Beta", nil, format.VerbosityFull, format.OutputText); err == nil {
			t.Fatal("expected error for nil input")
		}
	})

	t.Run("does not move milestone between projects", func(t *testing.T) {
		// UpdateMilestoneInput has no ProjectID field: --project only scopes
		// the lookup, it never becomes the mutation's projectId.
		client := &mockMilestoneClient{
			resolveMilestoneResult: "milestone-uuid",
			updateResult:           &core.ProjectMilestone{Name: "Private beta"},
		}
		s := newMilestoneService(client)

		name := "Private beta"
		_, err := s.Update("Beta", &UpdateMilestoneInput{
			Name:            &name,
			LookupProjectID: "Q3 Launch",
		}, format.VerbosityFull, format.OutputText)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.lastUpdateID != "milestone-uuid" {
			t.Errorf("expected resolved milestone ID, got %q", client.lastUpdateID)
		}
		if client.lastUpdateInput.Name == nil || *client.lastUpdateInput.Name != "Private beta" {
			t.Error("expected name to be passed through to the update input")
		}
	})
}

func TestMilestoneService_Delete(t *testing.T) {
	client := &mockMilestoneClient{
		resolveMilestoneResult: "milestone-uuid",
	}
	s := newMilestoneService(client)

	_, err := s.Delete("Beta", "Q3 Launch", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.lastDeleteID != "milestone-uuid" {
		t.Errorf("expected resolved milestone ID to be deleted, got %q", client.lastDeleteID)
	}
}
