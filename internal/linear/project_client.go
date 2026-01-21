package linear

import (
	"fmt"
	"strings"
)

// ProjectClient handles all project-related operations for the Linear API.
// It uses the shared BaseClient for HTTP communication and focuses on
// project management functionality.
type ProjectClient struct {
	base *BaseClient
}

// NewProjectClient creates a new project client with the provided base client
func NewProjectClient(base *BaseClient) *ProjectClient {
	return &ProjectClient{base: base}
}

// CreateProject creates a new project in Linear
// Why: Projects are containers for organizing related issues. This method
// enables project creation with proper team assignment.
func (pc *ProjectClient) CreateProject(name, description, teamID string) (*Project, error) {
	// Validate required inputs
	// Why: Name and teamID are mandatory for project creation. Early
	// validation provides clearer error messages than API errors.
	if name == "" {
		return nil, &ValidationError{Field: "name", Message: "name cannot be empty"}
	}
	if teamID == "" {
		return nil, &ValidationError{Field: "teamID", Message: "teamID cannot be empty"}
	}
	
	const mutation = `
		mutation CreateProject($input: ProjectCreateInput!) {
			projectCreate(input: $input) {
				success
				project {
					id
					name
					description
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
	// Why: Linear's API expects specific fields. We conditionally include
	// description only if provided to avoid sending empty strings.
	// Note: Linear API requires teamIds (plural, array) not teamId (singular).
	input := map[string]interface{}{
		"name":    name,
		"teamIds": []string{teamID},
	}
	if description != "" {
		input["description"] = description
	}
	
	variables := map[string]interface{}{
		"input": input,
	}
	
	var response struct {
		ProjectCreate struct {
			Success bool    `json:"success"`
			Project Project `json:"project"`
		} `json:"projectCreate"`
	}
	
	err := pc.base.executeRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}
	
	if !response.ProjectCreate.Success {
		return nil, fmt.Errorf("project creation was not successful")
	}
	
	// Extract metadata from description if present
	// Why: Projects can have metadata stored in descriptions. We extract
	// it immediately after creation for consistent access.
	if response.ProjectCreate.Project.Description != "" {
		metadata, cleanDesc := extractMetadataFromDescription(response.ProjectCreate.Project.Description)
		response.ProjectCreate.Project.Metadata = metadata
		response.ProjectCreate.Project.Description = cleanDesc
	}
	
	return &response.ProjectCreate.Project, nil
}

// GetProject retrieves a single project by ID
// Why: This is the primary method for fetching detailed project information
// including associated issues and metadata.
func (pc *ProjectClient) GetProject(projectID string) (*Project, error) {
	// Validate input
	// Why: Empty project ID would cause the query to fail with unclear
	// GraphQL errors. Early validation improves error clarity.
	if projectID == "" {
		return nil, &ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
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
		Project Project `json:"project"`
	}
	
	err := pc.base.executeRequest(query, variables, &response)
	if err != nil {
		// Check if this is a "not found" error and provide helpful guidance
		if strings.Contains(err.Error(), "Entity not found") || strings.Contains(err.Error(), "Project") {
			return nil, &ErrorWithGuidance{
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
		return nil, EnhanceGenericError("get project", err)
	}

	// Check if project was found
	if response.Project.ID == "" {
		return nil, &NotFoundError{
			ResourceType: "project",
			ResourceID:   projectID,
		}
	}

	// Extract metadata from content (or fallback to description for backwards compatibility)
	// Why: Metadata is embedded in project content as hidden markdown.
	// Content is preferred over description as it has no character limit.
	// Extracting it here ensures consistent access across all retrieval methods.
	if response.Project.Content != "" {
		metadata, cleanContent := extractMetadataFromDescription(response.Project.Content)
		response.Project.Metadata = metadata
		response.Project.Content = cleanContent
	} else if response.Project.Description != "" {
		// Fallback: check description for backwards compatibility with old metadata storage
		metadata, cleanDesc := extractMetadataFromDescription(response.Project.Description)
		response.Project.Metadata = metadata
		response.Project.Description = cleanDesc
	}

	return &response.Project, nil
}

// ListAllProjects retrieves all projects in the workspace
// Why: Users need to discover available projects. This method provides
// a complete list with optional limiting for performance.
func (pc *ProjectClient) ListAllProjects(limit int) ([]Project, error) {
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
			Nodes []Project `json:"nodes"`
		} `json:"projects"`
	}

	err := pc.base.executeRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Extract metadata from content (or fallback to description)
	// Why: Each project might have metadata. We extract it here to ensure
	// users can access metadata without additional calls.
	for i := range response.Projects.Nodes {
		if response.Projects.Nodes[i].Content != "" {
			metadata, cleanContent := extractMetadataFromDescription(response.Projects.Nodes[i].Content)
			response.Projects.Nodes[i].Metadata = metadata
			response.Projects.Nodes[i].Content = cleanContent
		} else if response.Projects.Nodes[i].Description != "" {
			// Fallback to description for backwards compatibility
			metadata, cleanDesc := extractMetadataFromDescription(response.Projects.Nodes[i].Description)
			response.Projects.Nodes[i].Metadata = metadata
			response.Projects.Nodes[i].Description = cleanDesc
		}
	}

	return response.Projects.Nodes, nil
}

// ListUserProjects retrieves projects that have issues assigned to a specific user
// Why: Users often want to see only projects they're actively working on.
// This method filters projects based on issue assignments.
func (pc *ProjectClient) ListUserProjects(userID string, limit int) ([]Project, error) {
	// Validate input
	// Why: User ID is required for filtering. Without it, we can't
	// determine which projects to return.
	if userID == "" {
		return nil, &ValidationError{Field: "userID", Message: "userID cannot be empty"}
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
			Nodes []Project `json:"nodes"`
		} `json:"projects"`
	}

	err := pc.base.executeRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list user projects: %w", err)
	}

	// Filter projects to only include those with issues assigned to the user
	// Why: The API filter may not work as expected in all cases, and we need
	// to ensure we only return projects where the user actually has assigned issues.
	var filteredProjects []Project
	for _, project := range response.Projects.Nodes {
		if len(project.Issues.Nodes) > 0 {
			filteredProjects = append(filteredProjects, project)
		}
	}

	// Extract metadata from content (or fallback to description)
	for i := range filteredProjects {
		if filteredProjects[i].Content != "" {
			metadata, cleanContent := extractMetadataFromDescription(filteredProjects[i].Content)
			filteredProjects[i].Metadata = metadata
			filteredProjects[i].Content = cleanContent
		} else if filteredProjects[i].Description != "" {
			// Fallback to description for backwards compatibility
			metadata, cleanDesc := extractMetadataFromDescription(filteredProjects[i].Description)
			filteredProjects[i].Metadata = metadata
			filteredProjects[i].Description = cleanDesc
		}
	}

	return filteredProjects, nil
}

// UpdateProjectInput represents the input for updating a project
type UpdateProjectInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Content     *string `json:"content,omitempty"`
	State       *string `json:"state,omitempty"`
	LeadID      *string `json:"leadId,omitempty"`
	StartDate   *string `json:"startDate,omitempty"`
	TargetDate  *string `json:"targetDate,omitempty"`
}

// UpdateProject updates a project with the provided input
// Supports updating name, description, state, lead, start date, and target date
func (pc *ProjectClient) UpdateProject(projectID string, input UpdateProjectInput) (*Project, error) {
	if projectID == "" {
		return nil, &ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
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
	if input.Description != nil {
		inputMap["description"] = *input.Description
	}
	if input.Content != nil {
		inputMap["content"] = *input.Content
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
		return nil, &ValidationError{Field: "input", Message: "at least one field must be provided"}
	}

	variables := map[string]interface{}{
		"projectId": projectID,
		"input":     inputMap,
	}

	var response struct {
		ProjectUpdate struct {
			Success bool    `json:"success"`
			Project Project `json:"project"`
		} `json:"projectUpdate"`
	}

	err := pc.base.executeRequest(mutation, variables, &response)
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
func (pc *ProjectClient) UpdateProjectState(projectID, state string) error {
	// Validate inputs
	// Why: Both project ID and state are required. Empty values would
	// cause the mutation to fail with unclear errors.
	if projectID == "" {
		return &ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}
	if state == "" {
		return &ValidationError{Field: "state", Message: "state cannot be empty"}
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
	
	err := pc.base.executeRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to update project state: %w", err)
	}
	
	if !response.ProjectUpdate.Success {
		return fmt.Errorf("project state update was not successful")
	}
	
	return nil
}

// UpdateProjectDescription updates a project's content while preserving metadata
// Why: Project content may contain both user content and metadata. This
// method ensures metadata is preserved during content updates.
// Note: Linear has two fields - 'description' (255 char limit) and 'content' (no limit).
// We use 'content' for longer text to avoid the character limit.
func (pc *ProjectClient) UpdateProjectDescription(projectID, newContent string) error {
	if projectID == "" {
		return &ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}

	// First, get the current project to preserve metadata
	// Why: We need to extract existing metadata before updating to ensure
	// it's not lost during the content update.
	project, err := pc.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get current project: %w", err)
	}

	// Preserve existing metadata
	// Why: The project.Metadata field contains extracted metadata that
	// needs to be injected back into the new content.
	contentWithMetadata := newContent
	if project.Metadata != nil && len(project.Metadata) > 0 {
		contentWithMetadata = injectMetadataIntoDescription(newContent, project.Metadata)
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
		"content":   contentWithMetadata,
	}

	var response struct {
		ProjectUpdate struct {
			Success bool `json:"success"`
		} `json:"projectUpdate"`
	}

	err = pc.base.executeRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to update project content: %w", err)
	}

	if !response.ProjectUpdate.Success {
		return fmt.Errorf("project content update was not successful")
	}

	return nil
}

// UpdateProjectMetadataKey updates a specific metadata key for a project
// Why: Granular metadata updates allow changing individual values without
// affecting other metadata. This is more efficient than full replacements.
// Note: Uses 'content' field instead of 'description' to avoid 255 char limit.
func (pc *ProjectClient) UpdateProjectMetadataKey(projectID, key string, value interface{}) error {
	if projectID == "" {
		return &ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}
	if key == "" {
		return &ValidationError{Field: "key", Message: "key cannot be empty"}
	}

	// Get current project to access existing metadata
	// Why: We need to merge the new key-value with existing metadata
	// to preserve other metadata entries.
	project, err := pc.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get current project: %w", err)
	}

	// Special handling for projects with null/empty content
	// Linear's API may reject updates to projects with null content
	// For now, we'll work around this by ensuring we always have content
	if project.Content == "" {
		// Set a minimal placeholder that won't be visible in Linear UI
		// but ensures the API accepts our update
		project.Content = " " // Single space
	}

	// Initialize metadata if needed and update the key
	// Why: The project might not have metadata yet. We initialize it
	// before adding the new key-value pair.
	if project.Metadata == nil {
		project.Metadata = make(map[string]interface{})
	}
	project.Metadata[key] = value

	// Update the content with new metadata
	contentWithMetadata := injectMetadataIntoDescription(project.Content, project.Metadata)

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
		"content":   contentWithMetadata,
	}

	var response struct {
		ProjectUpdate struct {
			Success bool `json:"success"`
		} `json:"projectUpdate"`
	}

	err = pc.base.executeRequest(mutation, variables, &response)
	if err != nil {
		// Add more context for debugging
		return fmt.Errorf("failed to update project metadata (projectID: %s, key: %s, content length: %d): %w",
			projectID, key, len(contentWithMetadata), err)
	}

	if !response.ProjectUpdate.Success {
		return fmt.Errorf("project metadata update was not successful")
	}

	return nil
}

// RemoveProjectMetadataKey removes a specific metadata key from a project
// Why: Metadata keys may become obsolete. This method allows selective
// removal without affecting other metadata.
// Note: Uses 'content' field instead of 'description' to avoid 255 char limit.
func (pc *ProjectClient) RemoveProjectMetadataKey(projectID, key string) error {
	if projectID == "" {
		return &ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}
	if key == "" {
		return &ValidationError{Field: "key", Message: "key cannot be empty"}
	}

	// Get current project
	project, err := pc.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get current project: %w", err)
	}

	// Remove the key if metadata exists
	// Why: We only update if there's metadata and the key exists.
	// No API call needed if there's nothing to remove.
	if project.Metadata != nil {
		delete(project.Metadata, key)

		// Update content with modified metadata
		// Why: After removing the key, we either update with remaining
		// metadata or remove the metadata section entirely if empty.
		var contentWithMetadata string
		if len(project.Metadata) > 0 {
			contentWithMetadata = injectMetadataIntoDescription(project.Content, project.Metadata)
		} else {
			// No metadata left, just use the clean content
			contentWithMetadata = project.Content
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
			"content":   contentWithMetadata,
		}

		var response struct {
			ProjectUpdate struct {
				Success bool `json:"success"`
			} `json:"projectUpdate"`
		}

		err = pc.base.executeRequest(mutation, variables, &response)
		if err != nil {
			return fmt.Errorf("failed to update project content: %w", err)
		}

		if !response.ProjectUpdate.Success {
			return fmt.Errorf("project metadata removal was not successful")
		}
	}

	return nil
}