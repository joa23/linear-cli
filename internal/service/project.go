package service

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
)

// ProjectService handles project-related operations
type ProjectService struct {
	client    *linear.Client
	formatter *format.Formatter
}

// NewProjectService creates a new ProjectService
func NewProjectService(client *linear.Client, formatter *format.Formatter) *ProjectService {
	return &ProjectService{
		client:    client,
		formatter: formatter,
	}
}

// Get retrieves a single project by ID
func (s *ProjectService) Get(projectID string) (string, error) {
	project, err := s.client.Projects.GetProject(projectID)
	if err != nil {
		return "", fmt.Errorf("failed to get project %s: %w", projectID, err)
	}

	return s.formatter.Project(project), nil
}

// ListAll lists all projects in the workspace
func (s *ProjectService) ListAll(limit int) (string, error) {
	if limit <= 0 {
		limit = 50
	}

	projects, err := s.client.Projects.ListAllProjects(limit)
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	return s.formatter.ProjectList(projects, nil), nil
}

// ListUserProjects lists projects that have issues assigned to the user
func (s *ProjectService) ListUserProjects(limit int) (string, error) {
	if limit <= 0 {
		limit = 50
	}

	// Get current user
	viewer, err := s.client.GetViewer()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	projects, err := s.client.Projects.ListUserProjects(viewer.ID, limit)
	if err != nil {
		return "", fmt.Errorf("failed to list user projects: %w", err)
	}

	return s.formatter.ProjectList(projects, nil), nil
}

// CreateProjectInput represents input for creating a project
type CreateProjectInput struct {
	Name        string
	Description string
	TeamID      string
	State       string // planned, started, paused, completed, canceled
	LeadID      string // Project lead user ID
	StartDate   string // Start date YYYY-MM-DD
	EndDate     string // Target end date YYYY-MM-DD
}

// Create creates a new project
func (s *ProjectService) Create(input *CreateProjectInput) (string, error) {
	if input.Name == "" {
		return "", fmt.Errorf("name is required")
	}
	if input.TeamID == "" {
		return "", fmt.Errorf("teamId is required")
	}

	// Resolve team identifier
	teamID, err := s.client.ResolveTeamIdentifier(input.TeamID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", input.TeamID, err)
	}

	project, err := s.client.CreateProject(input.Name, input.Description, teamID)
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	// Update with additional fields if provided
	needsUpdate := false
	updateInput := &UpdateProjectInput{}

	if input.State != "" {
		updateInput.State = &input.State
		needsUpdate = true
	}
	if input.LeadID != "" {
		updateInput.LeadID = &input.LeadID
		needsUpdate = true
	}
	if input.StartDate != "" {
		updateInput.StartDate = &input.StartDate
		needsUpdate = true
	}
	if input.EndDate != "" {
		updateInput.EndDate = &input.EndDate
		needsUpdate = true
	}

	if needsUpdate {
		_, err = s.Update(project.ID, updateInput)
		if err != nil {
			return "", fmt.Errorf("failed to update project after creation: %w", err)
		}
		// Re-fetch to get updated project
		project, err = s.client.Projects.GetProject(project.ID)
		if err != nil {
			return "", fmt.Errorf("failed to get updated project: %w", err)
		}
	}

	return s.formatter.Project(project), nil
}

// UpdateProjectInput represents input for updating a project
type UpdateProjectInput struct {
	Name        *string
	Description *string
	State       *string // planned, started, paused, completed, canceled
	LeadID      *string // Project lead user ID
	StartDate   *string // Start date YYYY-MM-DD
	EndDate     *string // Target end date YYYY-MM-DD
}

// Update updates an existing project
func (s *ProjectService) Update(projectID string, input *UpdateProjectInput) (string, error) {
	// Build Linear client input
	linearInput := linear.UpdateProjectInput{}

	if input.Name != nil {
		linearInput.Name = input.Name
	}
	if input.Description != nil {
		linearInput.Description = input.Description
	}
	if input.State != nil {
		linearInput.State = input.State
	}
	if input.LeadID != nil {
		// Resolve lead user identifier
		if *input.LeadID != "" {
			userID, err := s.client.ResolveUserIdentifier(*input.LeadID)
			if err != nil {
				return "", fmt.Errorf("failed to resolve lead '%s': %w", *input.LeadID, err)
			}
			linearInput.LeadID = &userID
		} else {
			linearInput.LeadID = input.LeadID
		}
	}
	if input.StartDate != nil {
		linearInput.StartDate = input.StartDate
	}
	if input.EndDate != nil {
		linearInput.TargetDate = input.EndDate
	}

	// Update project
	project, err := s.client.Projects.UpdateProject(projectID, linearInput)
	if err != nil {
		return "", fmt.Errorf("failed to update project: %w", err)
	}

	return s.formatter.Project(project), nil
}
