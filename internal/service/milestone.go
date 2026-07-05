package service

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/pkg/linear/core"
)

// MilestoneService handles project milestone operations.
type MilestoneService struct {
	client    MilestoneClientOperations
	formatter *format.Formatter
}

// NewMilestoneService creates a new MilestoneService.
func NewMilestoneService(client MilestoneClientOperations, formatter *format.Formatter) *MilestoneService {
	return &MilestoneService{client: client, formatter: formatter}
}

// MilestoneListInput contains filters for listing project milestones.
type MilestoneListInput struct {
	ProjectID string
	TeamID    string
	Limit     int
}

// CreateMilestoneInput contains CLI-level input for creating a project milestone.
type CreateMilestoneInput struct {
	Name        string
	Description string
	ProjectID   string
	TeamID      string
	TargetDate  string
}

// UpdateMilestoneInput contains CLI-level input for updating a project milestone.
type UpdateMilestoneInput struct {
	Name            *string
	Description     *string
	LookupProjectID string
	TeamID          string
	TargetDate      *string
}

// List lists milestones for a project.
func (s *MilestoneService) List(input *MilestoneListInput, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	projectID, err := s.resolveProject(input.ProjectID, input.TeamID)
	if err != nil {
		return "", err
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}

	milestones, err := s.client.ListProjectMilestones(projectID, limit)
	if err != nil {
		return "", fmt.Errorf("failed to list milestones: %w", err)
	}

	return s.formatter.RenderMilestoneList(milestones, verbosity, outputType), nil
}

// Get retrieves a milestone by ID or by name within a project.
func (s *MilestoneService) Get(identifier string, projectID string, teamID string, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	id, err := s.resolveMilestone(identifier, projectID, teamID)
	if err != nil {
		return "", err
	}

	milestone, err := s.client.GetProjectMilestone(id)
	if err != nil {
		return "", fmt.Errorf("failed to get milestone: %w", err)
	}

	return s.formatter.RenderMilestone(milestone, verbosity, outputType), nil
}

// Create creates a project milestone.
func (s *MilestoneService) Create(input *CreateMilestoneInput, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	if input == nil {
		return "", fmt.Errorf("input is required")
	}
	if input.Name == "" {
		return "", fmt.Errorf("name is required")
	}

	projectID, err := s.resolveProject(input.ProjectID, input.TeamID)
	if err != nil {
		return "", err
	}

	milestone, err := s.client.CreateProjectMilestone(&core.CreateProjectMilestoneInput{
		Name:        input.Name,
		Description: input.Description,
		ProjectID:   projectID,
		TargetDate:  input.TargetDate,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create milestone: %w", err)
	}

	return s.formatter.RenderMilestone(milestone, verbosity, outputType), nil
}

// Update updates a project milestone.
func (s *MilestoneService) Update(identifier string, input *UpdateMilestoneInput, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	if input == nil {
		return "", fmt.Errorf("input is required")
	}

	id, err := s.resolveMilestone(identifier, input.LookupProjectID, input.TeamID)
	if err != nil {
		return "", err
	}

	updateInput := &core.UpdateProjectMilestoneInput{
		Name:        input.Name,
		Description: input.Description,
		TargetDate:  input.TargetDate,
	}

	milestone, err := s.client.UpdateProjectMilestone(id, updateInput)
	if err != nil {
		return "", fmt.Errorf("failed to update milestone: %w", err)
	}

	return s.formatter.RenderMilestone(milestone, verbosity, outputType), nil
}

// Delete deletes a project milestone.
func (s *MilestoneService) Delete(identifier string, projectID string, teamID string) (string, error) {
	id, err := s.resolveMilestone(identifier, projectID, teamID)
	if err != nil {
		return "", err
	}

	if err := s.client.DeleteProjectMilestone(id); err != nil {
		return "", fmt.Errorf("failed to delete milestone: %w", err)
	}

	return fmt.Sprintf("Deleted milestone: %s", identifier), nil
}

func (s *MilestoneService) resolveProject(projectIdentifier string, teamIdentifier string) (string, error) {
	if projectIdentifier == "" {
		return "", fmt.Errorf("project is required")
	}

	var teamID string
	if teamIdentifier != "" {
		resolvedTeamID, err := s.client.ResolveTeamIdentifier(teamIdentifier)
		if err != nil {
			return "", fmt.Errorf("failed to resolve team '%s': %w", teamIdentifier, err)
		}
		teamID = resolvedTeamID
	}

	projectID, err := s.client.ResolveProjectIdentifier(projectIdentifier, teamID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project '%s': %w", projectIdentifier, err)
	}
	return projectID, nil
}

func (s *MilestoneService) resolveMilestone(identifier string, projectIdentifier string, teamIdentifier string) (string, error) {
	if identifier == "" {
		return "", fmt.Errorf("milestone is required")
	}

	projectID := projectIdentifier
	if projectID != "" {
		resolvedProjectID, err := s.resolveProject(projectID, teamIdentifier)
		if err != nil {
			return "", err
		}
		projectID = resolvedProjectID
	}

	milestoneID, err := s.client.ResolveProjectMilestoneIdentifier(identifier, projectID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve milestone '%s': %w", identifier, err)
	}
	return milestoneID, nil
}
