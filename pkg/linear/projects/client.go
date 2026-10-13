package projects

import (
	"fmt"
	"strings"

	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/guidance"
	"github.com/joa23/linear-cli/pkg/linear/metadata"
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

// NormalizeStatusNames trims, validates, and de-duplicates project status names.
func NormalizeStatusNames(names []string) ([]string, error) {
	normalized := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("project status filter contains an empty value")
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, name)
	}
	return normalized, nil
}

// ListProjectStatuses returns active workspace project statuses.
func (pc *Client) ListProjectStatuses() ([]core.ProjectStatus, error) {
	const query = `
		query ListProjectStatuses {
			organization {
				projectStatuses {
					id
					name
					type
					archivedAt
				}
			}
		}
	`
	var response struct {
		Organization struct {
			ProjectStatuses []core.ProjectStatus `json:"projectStatuses"`
		} `json:"organization"`
	}
	if err := pc.base.ExecuteRequest(query, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to list project statuses: %w", err)
	}
	statuses := make([]core.ProjectStatus, 0, len(response.Organization.ProjectStatuses))
	for _, status := range response.Organization.ProjectStatuses {
		if status.ArchivedAt == nil {
			statuses = append(statuses, status)
		}
	}
	return statuses, nil
}

// ResolveProjectStatusNames resolves normalized names to active status IDs.
func (pc *Client) ResolveProjectStatusNames(names []string) ([]string, error) {
	normalized, err := NormalizeStatusNames(names)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	statuses, err := pc.ListProjectStatuses()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(normalized))
	for _, name := range normalized {
		key := strings.ToLower(name)
		var match *core.ProjectStatus
		for i := range statuses {
			if strings.ToLower(strings.TrimSpace(statuses[i].Name)) != key {
				continue
			}
			if match != nil {
				return nil, fmt.Errorf("project status '%s' is ambiguous", name)
			}
			match = &statuses[i]
		}
		if match == nil {
			return nil, fmt.Errorf("project status '%s' not found", name)
		}
		ids = append(ids, match.ID)
	}
	return ids, nil
}

func statusFilterMap(statusIDs []string) map[string]interface{} {
	if len(statusIDs) == 0 {
		return nil
	}
	return map[string]interface{}{
		"status": map[string]interface{}{
			"id": map[string]interface{}{"in": statusIDs},
		},
	}
}

// CreateProject creates a new project in Linear
// Why: Projects are containers for organizing related issues. This method
// enables project creation with proper team assignment.
func (pc *Client) CreateProject(name, description, teamID string) (*core.Project, error) {
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
					state
					status {
						id
						name
						type
					}
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
			Success bool         `json:"success"`
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

	// Extract metadata from description if present
	// Why: Projects can have metadata stored in descriptions. We extract
	// it immediately after creation for consistent access.
	if response.ProjectCreate.Project.Description != "" {
		metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.ProjectCreate.Project.Description)
		response.ProjectCreate.Project.Metadata = metadata
		response.ProjectCreate.Project.Description = cleanDesc
	}

	return &response.ProjectCreate.Project, nil
}

// GetProject retrieves a single project by ID
// Why: This is the primary method for fetching detailed project information
// including associated issues and metadata.
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
				status {
					id
					name
					type
				}
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

	// Extract metadata from content (or fallback to description for backwards compatibility)
	// Why: Metadata is embedded in project content as hidden markdown.
	// Content is preferred over description as it has no character limit.
	// Extracting it here ensures consistent access across all retrieval methods.
	if response.Project.Content != "" {
		metadata, cleanContent := metadata.ExtractMetadataFromDescription(response.Project.Content)
		response.Project.Metadata = metadata
		response.Project.Content = cleanContent
	} else if response.Project.Description != "" {
		// Fallback: check description for backwards compatibility with old metadata storage
		metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.Project.Description)
		response.Project.Metadata = metadata
		response.Project.Description = cleanDesc
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
					status {
						id
						name
						type
					}
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

	// Extract metadata from content (or fallback to description)
	// Why: Each project might have metadata. We extract it here to ensure
	// users can access metadata without additional calls.
	for i := range response.Projects.Nodes {
		if response.Projects.Nodes[i].Content != "" {
			metadata, cleanContent := metadata.ExtractMetadataFromDescription(response.Projects.Nodes[i].Content)
			response.Projects.Nodes[i].Metadata = metadata
			response.Projects.Nodes[i].Content = cleanContent
		} else if response.Projects.Nodes[i].Description != "" {
			// Fallback to description for backwards compatibility
			metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.Projects.Nodes[i].Description)
			response.Projects.Nodes[i].Metadata = metadata
			response.Projects.Nodes[i].Description = cleanDesc
		}
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
					status {
						id
						name
						type
					}
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

	// Extract metadata from content (or fallback to description)
	for i := range response.Team.Projects.Nodes {
		if response.Team.Projects.Nodes[i].Content != "" {
			metadata, cleanContent := metadata.ExtractMetadataFromDescription(response.Team.Projects.Nodes[i].Content)
			response.Team.Projects.Nodes[i].Metadata = metadata
			response.Team.Projects.Nodes[i].Content = cleanContent
		} else if response.Team.Projects.Nodes[i].Description != "" {
			metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.Team.Projects.Nodes[i].Description)
			response.Team.Projects.Nodes[i].Metadata = metadata
			response.Team.Projects.Nodes[i].Description = cleanDesc
		}
	}

	return response.Team.Projects.Nodes, nil
}

// ListUserProjects retrieves projects that have issues assigned to a specific user.
// The visible limit is applied after the assignee predicate has been verified.
func (pc *Client) ListUserProjects(userID string, limit int) ([]core.Project, error) {
	return pc.listUserProjects(userID, limit, nil)
}

// ListAllProjectsWithStatus retrieves projects matching any of the supplied status IDs.
// The status predicate is sent to Linear so it is applied before the limit.
func (pc *Client) ListAllProjectsWithStatus(limit int, statusIDs []string) ([]core.Project, error) {
	if len(statusIDs) == 0 {
		return pc.ListAllProjects(limit)
	}
	if limit <= 0 {
		limit = 50
	}
	const query = `
		query ListProjects($filter: ProjectFilter, $first: Int) {
			projects(filter: $filter, first: $first) {
				nodes {
					id
					name
					description
					content
					state
					status { id name type }
					createdAt
					updatedAt
				}
			}
		}
	`
	var response struct {
		Projects struct {
			Nodes []core.Project `json:"nodes"`
		} `json:"projects"`
	}
	variables := map[string]interface{}{"filter": statusFilterMap(statusIDs), "first": limit}
	if err := pc.base.ExecuteRequest(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	pc.extractMetadata(response.Projects.Nodes)
	return response.Projects.Nodes, nil
}

// ListByTeamWithStatus retrieves team projects matching any supplied status IDs.
func (pc *Client) ListByTeamWithStatus(teamID string, limit int, statusIDs []string) ([]core.Project, error) {
	if len(statusIDs) == 0 {
		return pc.ListByTeam(teamID, limit)
	}
	if teamID == "" {
		return nil, &core.ValidationError{Field: "teamID", Message: "teamID cannot be empty"}
	}
	if limit <= 0 {
		limit = 50
	}
	const query = `
		query ListProjectsByTeam($teamId: String!, $filter: ProjectFilter, $first: Int) {
			team(id: $teamId) {
				projects(filter: $filter, first: $first) {
					nodes {
						id
						name
						description
						content
						state
						status { id name type }
						createdAt
						updatedAt
					}
				}
			}
		}
	`
	var response struct {
		Team struct {
			Projects struct {
				Nodes []core.Project `json:"nodes"`
			} `json:"projects"`
		} `json:"team"`
	}
	variables := map[string]interface{}{"teamId": teamID, "filter": statusFilterMap(statusIDs), "first": limit}
	if err := pc.base.ExecuteRequest(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to list projects by team: %w", err)
	}
	pc.extractMetadata(response.Team.Projects.Nodes)
	return response.Team.Projects.Nodes, nil
}

// ListUserProjectsWithStatus retrieves user projects with both predicates applied server-side.
func (pc *Client) ListUserProjectsWithStatus(userID string, limit int, statusIDs []string) ([]core.Project, error) {
	return pc.listUserProjects(userID, limit, statusIDs)
}

func (pc *Client) listUserProjects(userID string, limit int, statusIDs []string) ([]core.Project, error) {
	if userID == "" {
		return nil, &core.ValidationError{Field: "userID", Message: "userID cannot be empty"}
	}
	if limit <= 0 {
		limit = 50
	}

	const query = `
		query ListUserProjects($filter: ProjectFilter, $issueFilter: IssueFilter, $first: Int, $after: String) {
			projects(filter: $filter, first: $first, after: $after) {
				nodes {
					id
					name
					description
					content
					state
					status { id name type }
					createdAt
					updatedAt
					issues(filter: $issueFilter, first: 1) { nodes { id assignee { id } } }
				}
				pageInfo { hasNextPage endCursor }
			}
		}
	`

	issueFilter := map[string]interface{}{"assignee": map[string]interface{}{"id": map[string]interface{}{"eq": userID}}}
	filter := map[string]interface{}{
		"issues": map[string]interface{}{"some": issueFilter},
	}
	if len(statusIDs) > 0 {
		filter["status"] = map[string]interface{}{"id": map[string]interface{}{"in": statusIDs}}
	}

	const pageSize = 250
	filtered := make([]core.Project, 0, limit)
	seenCursors := map[string]struct{}{}
	var after string
	for {
		variables := map[string]interface{}{"filter": filter, "issueFilter": issueFilter, "first": pageSize}
		if after != "" {
			variables["after"] = after
		}
		var response struct {
			Projects struct {
				Nodes    []core.Project `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"projects"`
		}
		if err := pc.base.ExecuteRequest(query, variables, &response); err != nil {
			return nil, fmt.Errorf("failed to list user projects: %w", err)
		}
		for _, project := range response.Projects.Nodes {
			if project.Issues != nil && len(project.Issues.Nodes) > 0 {
				filtered = append(filtered, project)
				if len(filtered) == limit {
					pc.extractMetadata(filtered)
					return filtered, nil
				}
			}
		}
		if !response.Projects.PageInfo.HasNextPage {
			break
		}
		next := response.Projects.PageInfo.EndCursor
		if next == "" {
			return nil, fmt.Errorf("failed to list user projects: pagination returned an empty cursor")
		}
		if _, ok := seenCursors[next]; ok {
			return nil, fmt.Errorf("failed to list user projects: pagination returned a repeated cursor")
		}
		seenCursors[next] = struct{}{}
		after = next
	}
	pc.extractMetadata(filtered)
	return filtered, nil
}

func (pc *Client) extractMetadata(projects []core.Project) {
	for i := range projects {
		if projects[i].Content != "" {
			projectMetadata, cleanContent := metadata.ExtractMetadataFromDescription(projects[i].Content)
			projects[i].Metadata = projectMetadata
			projects[i].Content = cleanContent
		} else if projects[i].Description != "" {
			projectMetadata, cleanDescription := metadata.ExtractMetadataFromDescription(projects[i].Description)
			projects[i].Metadata = projectMetadata
			projects[i].Description = cleanDescription
		}
	}
}

// UpdateProjectInput contains the legacy project write fields used by this client.
// The checked-in Linear schema exposes statusId, not state, on ProjectUpdateInput.
// State is intentionally retained for compatibility with the existing --state
// surface; this ticket does not change that write semantics or infer a named
// status because multiple named statuses may share one status type.
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
					status {
						id
						name
						type
					}
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
		return nil, &core.ValidationError{Field: "input", Message: "at least one field must be provided"}
	}

	variables := map[string]interface{}{
		"projectId": projectID,
		"input":     inputMap,
	}

	var response struct {
		ProjectUpdate struct {
			Success bool         `json:"success"`
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

// UpdateProjectState updates the state of a project.
// UpdateProjectStateWithResult retains the mutation's project response for
// callers that need the named status; the deprecated state write is preserved
// for compatibility and is not migrated to statusId in this ticket.
func (pc *Client) UpdateProjectState(projectID, state string) error {
	_, err := pc.UpdateProjectStateWithResult(projectID, state)
	return err
}

// UpdateProjectStateWithResult updates a project using the legacy state input
// and returns the response including its named status. The current checked-in
// schema declares statusId rather than state for ProjectUpdateInput; the legacy
// state write is deliberately preserved and may be rejected by newer APIs.
// A type such as started cannot be converted safely to statusId because several
// named statuses can share the same type.
func (pc *Client) UpdateProjectStateWithResult(projectID, state string) (*core.Project, error) {
	if projectID == "" {
		return nil, &core.ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}
	if state == "" {
		return nil, &core.ValidationError{Field: "state", Message: "state cannot be empty"}
	}

	const mutation = `
		mutation UpdateProjectState($projectId: String!, $state: String!) {
			projectUpdate(id: $projectId, input: { state: $state }) {
				success
				project {
					id
					state
					status { id name type }
				}
			}
		}
	`
	var response struct {
		ProjectUpdate struct {
			Success bool         `json:"success"`
			Project core.Project `json:"project"`
		} `json:"projectUpdate"`
	}
	variables := map[string]interface{}{"projectId": projectID, "state": state}
	if err := pc.base.ExecuteRequest(mutation, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to update project state: %w", err)
	}
	if !response.ProjectUpdate.Success {
		return nil, fmt.Errorf("project state update was not successful")
	}
	return &response.ProjectUpdate.Project, nil
}

// UpdateProjectDescription updates a project's content while preserving metadata
// Why: Project content may contain both user content and metadata. This
// method ensures metadata is preserved during content updates.
// Note: Linear has two fields - 'description' (255 char limit) and 'content' (no limit).
// We use 'content' for longer text to avoid the character limit.
func (pc *Client) UpdateProjectDescription(projectID, newContent string) error {
	if projectID == "" {
		return &core.ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
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
		contentWithMetadata = metadata.InjectMetadataIntoDescription(newContent, project.Metadata)
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

	err = pc.base.ExecuteRequest(mutation, variables, &response)
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
func (pc *Client) UpdateProjectMetadataKey(projectID, key string, value interface{}) error {
	if projectID == "" {
		return &core.ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}
	if key == "" {
		return &core.ValidationError{Field: "key", Message: "key cannot be empty"}
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
	contentWithMetadata := metadata.InjectMetadataIntoDescription(project.Content, project.Metadata)

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

	err = pc.base.ExecuteRequest(mutation, variables, &response)
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
func (pc *Client) RemoveProjectMetadataKey(projectID, key string) error {
	if projectID == "" {
		return &core.ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}
	if key == "" {
		return &core.ValidationError{Field: "key", Message: "key cannot be empty"}
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
			contentWithMetadata = metadata.InjectMetadataIntoDescription(project.Content, project.Metadata)
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

		err = pc.base.ExecuteRequest(mutation, variables, &response)
		if err != nil {
			return fmt.Errorf("failed to update project content: %w", err)
		}

		if !response.ProjectUpdate.Success {
			return fmt.Errorf("project metadata removal was not successful")
		}
	}

	return nil
}
