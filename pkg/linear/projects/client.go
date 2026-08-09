package projects

import (
	"fmt"
	"strings"

	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/guidance"
)

// ProjectClient handles all project-related operations for the Linear API.
// It uses the shared BaseClient for HTTP communication and focuses on
// project management functionality.
type Client struct {
	base *core.BaseClient
}

// NewProjectClient creates a new project client with the provided base client
func NewClient(base *core.BaseClient) *Client {
	return &Client{base: base}
}

// CreateProject creates a new project in Linear
// Why: Projects are containers for organizing related issues. This method
// enables project creation with proper team assignment.
// The summary is capped at 255 characters by Linear; the description is not.
func (pc *Client) CreateProject(name, summary, description, teamID string) (*core.Project, error) {
	// Validate required inputs
	// Why: Name and teamID are mandatory for project creation. Early
	// validation provides clearer error messages than API errors.
	if name == "" {
		return nil, &core.ValidationError{Field: "name", Message: "name cannot be empty"}
	}
	if teamID == "" {
		return nil, &core.ValidationError{Field: "teamID", Message: "teamID cannot be empty"}
	}
	
	const mutation = `
		mutation CreateProject($input: ProjectCreateInput!) {
			projectCreate(input: $input) {
				success
				project {
					id
					name
					description
					content
					state
					createdAt
					updatedAt
					issues {
						nodes {
							id
							identifier
							title
						}
					}
				}
			}
		}
	`
	
	// Build the input object
	// Why: Linear's API expects specific fields. We conditionally include the
	// text fields only if provided to avoid sending empty strings.
	// Note: Linear API requires teamIds (plural, array) not teamId (singular).
	input := map[string]interface{}{
		"name":    name,
		"teamIds": []string{teamID},
	}
	if summary != "" {
		input["description"] = summary
	}
	if description != "" {
		input["content"] = description
	}
	
	variables := map[string]interface{}{
		"input": input,
	}
	
	var response struct {
		ProjectCreate struct {
			Success bool    `json:"success"`
			Project core.Project `json:"project"`
		} `json:"projectCreate"`
	}
	
	err := pc.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}
	
	if !response.ProjectCreate.Success {
		return nil, fmt.Errorf("project creation was not successful")
	}

	return &response.ProjectCreate.Project, nil
}

// GetProject retrieves a single project by ID
// Why: This is the primary method for fetching detailed project information
// including associated issues.
func (pc *Client) GetProject(projectID string) (*core.Project, error) {
	// Validate input
	// Why: Empty project ID would cause the query to fail with unclear
	// GraphQL errors. Early validation improves error clarity.
	if projectID == "" {
		return nil, &core.ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}
	
	const query = `
		query GetProject($id: String!) {
			project(id: $id) {
				id
				name
				description
				content
				state
				createdAt
				updatedAt
				issues {
					nodes {
						id
						identifier
						title
						state {
							id
							name
						}
						assignee {
							id
							name
							email
						}
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"id": projectID,
	}
	
	var response struct {
		Project core.Project `json:"project"`
	}
	
	err := pc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		// Check if this is a "not found" error and provide helpful guidance
		if strings.Contains(err.Error(), "Entity not found") || strings.Contains(err.Error(), "Project") {
			return nil, &guidance.ErrorWithGuidance{
				Operation: "Get project",
				Reason:    fmt.Sprintf("project with ID '%s' not found", projectID),
				Guidance: []string{
					"Verify the project ID is correct (Linear uses UUID format)",
					"Check if you have access to this project",
					"The project may have been deleted or archived",
					"Use project discovery tools to find valid project IDs",
				},
				Tools: []string{
					"linear_list_projects(filter='all') - List all projects to find the correct one",
					"linear_list_projects(filter='user') - List projects with your assigned issues",
					"linear_search_issues() - Find issues and get project IDs from them",
				},
				Example: fmt.Sprintf(`// Find projects first:
projects = linear_list_projects(filter="all")
// Look for your project by name, then use its ID:
correctProject = projects.find(p => p.name.includes("Your Project Name"))
linear_get_project(correctProject.id)`),
				OriginalErr: err,
			}
		}
		return nil, guidance.EnhanceGenericError("get project", err)
	}

	// Check if project was found
	if response.Project.ID == "" {
		return nil, &core.NotFoundError{
			ResourceType: "project",
			ResourceID:   projectID,
		}
	}

	return &response.Project, nil
}

// ListAllProjects retrieves all projects in the workspace
// Why: Users need to discover available projects. This method provides
// a complete list with optional limiting for performance.
func (pc *Client) ListAllProjects(limit int) ([]core.Project, error) {
	// Default limit if not specified
	// Why: Without a limit, the query could return too many results.
	// 50 is a reasonable default balancing completeness with performance.
	if limit <= 0 {
		limit = 50
	}
	
	const query = `
		query ListProjects($first: Int) {
			projects(first: $first) {
				nodes {
					id
					name
					description
					content
					state
					createdAt
					updatedAt
				}
			}
		}
	`

	variables := map[string]interface{}{
		"first": limit,
	}

	var response struct {
		Projects struct {
			Nodes []core.Project `json:"nodes"`
		} `json:"projects"`
	}

	err := pc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return response.Projects.Nodes, nil
}

// ListByTeam retrieves projects for a specific team
// Why: Teams often have many projects. Filtering by team makes it easier
// to find relevant projects without seeing all org-wide projects.
func (pc *Client) ListByTeam(teamID string, limit int) ([]core.Project, error) {
	// Validate input
	if teamID == "" {
		return nil, &core.ValidationError{Field: "teamID", Message: "teamID cannot be empty"}
	}

	if limit <= 0 {
		limit = 50
	}

	const query = `
		query ListProjectsByTeam($teamId: String!, $first: Int) {
			team(id: $teamId) {
				projects(first: $first) {
					nodes {
						id
						name
						description
						content
						state
						createdAt
						updatedAt
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"teamId": teamID,
		"first":  limit,
	}

	var response struct {
		Team struct {
			Projects struct {
				Nodes []core.Project `json:"nodes"`
			} `json:"projects"`
		} `json:"team"`
	}

	err := pc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects by team: %w", err)
	}

	return response.Team.Projects.Nodes, nil
}

// ListUserProjects retrieves projects that have issues assigned to a specific user
// Why: Users often want to see only projects they're actively working on.
// This method filters projects based on issue assignments.
func (pc *Client) ListUserProjects(userID string, limit int) ([]core.Project, error) {
	// Validate input
	// Why: User ID is required for filtering. Without it, we can't
	// determine which projects to return.
	if userID == "" {
		return nil, &core.ValidationError{Field: "userID", Message: "userID cannot be empty"}
	}
	
	if limit <= 0 {
		limit = 50
	}
	
	const query = `
		query ListUserProjects($filter: ProjectFilter, $first: Int) {
			projects(filter: $filter, first: $first) {
				nodes {
					id
					name
					description
					content
					state
					createdAt
					updatedAt
					issues {
						nodes {
							id
							assignee {
								id
							}
						}
					}
				}
			}
		}
	`

	// Filter for projects with issues assigned to the user
	// Why: Linear doesn't have direct user-project relationships.
	// We filter through issues to find projects the user is working on.
	filter := map[string]interface{}{
		"issues": map[string]interface{}{
			"assignee": map[string]interface{}{
				"id": map[string]interface{}{
					"eq": userID,
				},
			},
		},
	}

	variables := map[string]interface{}{
		"filter": filter,
		"first":  limit,
	}

	var response struct {
		Projects struct {
			Nodes []core.Project `json:"nodes"`
		} `json:"projects"`
	}

	err := pc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list user projects: %w", err)
	}

	// Filter projects to only include those with issues assigned to the user
	// Why: The API filter may not work as expected in all cases, and we need
	// to ensure we only return projects where the user actually has assigned issues.
	var filteredProjects []core.Project
	for _, project := range response.Projects.Nodes {
		if len(project.Issues.Nodes) > 0 {
			filteredProjects = append(filteredProjects, project)
		}
	}

	return filteredProjects, nil
}

// UpdateProjectInput represents the input for updating a project
type UpdateProjectInput struct {
	Name        *string `json:"name,omitempty"`
	Summary     *string `json:"description,omitempty"`
	Description *string `json:"content,omitempty"`
	State       *string `json:"state,omitempty"`
	LeadID      *string `json:"leadId,omitempty"`
	StartDate   *string `json:"startDate,omitempty"`
	TargetDate  *string `json:"targetDate,omitempty"`
}

// UpdateProject updates a project with the provided input
// Supports updating name, summary, description, state, lead, start date, and target date
func (pc *Client) UpdateProject(projectID string, input UpdateProjectInput) (*core.Project, error) {
	if projectID == "" {
		return nil, &core.ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}

	const mutation = `
		mutation UpdateProject($projectId: String!, $input: ProjectUpdateInput!) {
			projectUpdate(
				id: $projectId,
				input: $input
			) {
				success
				project {
					id
					name
					description
					content
					state
					createdAt
					updatedAt
				}
			}
		}
	`

	// Build input map with only non-nil fields
	inputMap := make(map[string]interface{})
	if input.Name != nil {
		inputMap["name"] = *input.Name
	}
	if input.Summary != nil {
		inputMap["description"] = *input.Summary
	}
	if input.Description != nil {
		inputMap["content"] = *input.Description
	}
	if input.State != nil {
		inputMap["state"] = *input.State
	}
	if input.LeadID != nil {
		inputMap["leadId"] = *input.LeadID
	}
	if input.StartDate != nil {
		inputMap["startDate"] = *input.StartDate
	}
	if input.TargetDate != nil {
		inputMap["targetDate"] = *input.TargetDate
	}

	if len(inputMap) == 0 {
		return nil, &core.ValidationError{Field: "input", Message: "at least one field must be provided"}
	}

	variables := map[string]interface{}{
		"projectId": projectID,
		"input":     inputMap,
	}

	var response struct {
		ProjectUpdate struct {
			Success bool    `json:"success"`
			Project core.Project `json:"project"`
		} `json:"projectUpdate"`
	}

	err := pc.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	if !response.ProjectUpdate.Success {
		return nil, fmt.Errorf("project update was not successful")
	}

	return &response.ProjectUpdate.Project, nil
}

// UpdateProjectState updates the state of a project
// Why: Projects have states (planned, started, completed, etc.) that need
// to be updated as work progresses. This method provides that capability.
func (pc *Client) UpdateProjectState(projectID, state string) error {
	// Validate inputs
	// Why: Both project ID and state are required. Empty values would
	// cause the mutation to fail with unclear errors.
	if projectID == "" {
		return &core.ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}
	if state == "" {
		return &core.ValidationError{Field: "state", Message: "state cannot be empty"}
	}
	
	const mutation = `
		mutation UpdateProjectState($projectId: String!, $state: String!) {
			projectUpdate(
				id: $projectId,
				input: { state: $state }
			) {
				success
				project {
					id
					state
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"projectId": projectID,
		"state":     state,
	}
	
	var response struct {
		ProjectUpdate struct {
			Success bool `json:"success"`
			Project struct {
				ID    string `json:"id"`
				State string `json:"state"`
			} `json:"project"`
		} `json:"projectUpdate"`
	}
	
	err := pc.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to update project state: %w", err)
	}
	
	if !response.ProjectUpdate.Success {
		return fmt.Errorf("project state update was not successful")
	}
	
	return nil
}

// UpdateProjectDescription replaces a project's content.
// Note: Linear has two fields - 'description' (255 char limit) and 'content' (no limit).
// We use 'content' for longer text to avoid the character limit.
func (pc *Client) UpdateProjectDescription(projectID, newContent string) error {
	if projectID == "" {
		return &core.ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}

	const mutation = `
		mutation UpdateProjectContent($projectId: String!, $content: String!) {
			projectUpdate(
				id: $projectId,
				input: { content: $content }
			) {
				success
			}
		}
	`

	variables := map[string]interface{}{
		"projectId": projectID,
		"content":   newContent,
	}

	var response struct {
		ProjectUpdate struct {
			Success bool `json:"success"`
		} `json:"projectUpdate"`
	}

	err := pc.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to update project content: %w", err)
	}

	if !response.ProjectUpdate.Success {
		return fmt.Errorf("project content update was not successful")
	}

	return nil
}
