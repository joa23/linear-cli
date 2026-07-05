package milestones

import (
	"fmt"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// Client handles project milestone operations for the Linear API.
type Client struct {
	base *core.BaseClient
}

// NewClient creates a new milestones client with the provided base client.
func NewClient(base *core.BaseClient) *Client {
	return &Client{base: base}
}

const milestoneFields = `
	id
	name
	description
	targetDate
	status
	progress
	createdAt
	updatedAt
	archivedAt
	project {
		id
		name
	}
`

// milestoneDetailFields extends milestoneFields with the issues connection.
// Only Get uses it; list/create/update skip issues to keep responses token-efficient.
const milestoneDetailFields = milestoneFields + `
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
`

// List returns milestones for a project.
func (mc *Client) List(projectID string, limit int) ([]core.ProjectMilestone, error) {
	if projectID == "" {
		return nil, &core.ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 250 {
		limit = 250
	}

	query := fmt.Sprintf(`
		query ListProjectMilestones($first: Int, $filter: ProjectMilestoneFilter) {
			projectMilestones(first: $first, filter: $filter) {
				nodes {
					%s
				}
			}
		}
	`, milestoneFields)

	variables := map[string]interface{}{
		"first": limit,
		"filter": map[string]interface{}{
			"project": map[string]interface{}{
				"id": map[string]interface{}{
					"eq": projectID,
				},
			},
		},
	}

	var response struct {
		ProjectMilestones struct {
			Nodes []core.ProjectMilestone `json:"nodes"`
		} `json:"projectMilestones"`
	}

	if err := mc.base.ExecuteRequest(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to list project milestones: %w", err)
	}

	return response.ProjectMilestones.Nodes, nil
}

// Get returns a project milestone by ID.
func (mc *Client) Get(id string) (*core.ProjectMilestone, error) {
	if id == "" {
		return nil, &core.ValidationError{Field: "id", Message: "id cannot be empty"}
	}

	query := fmt.Sprintf(`
		query GetProjectMilestone($id: String!) {
			projectMilestone(id: $id) {
				%s
			}
		}
	`, milestoneDetailFields)

	var response struct {
		ProjectMilestone core.ProjectMilestone `json:"projectMilestone"`
	}
	variables := map[string]interface{}{"id": id}

	if err := mc.base.ExecuteRequest(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to get project milestone: %w", err)
	}
	if response.ProjectMilestone.ID == "" {
		return nil, &core.NotFoundError{ResourceType: "project milestone", ResourceID: id}
	}

	return &response.ProjectMilestone, nil
}

// Create creates a project milestone.
func (mc *Client) Create(input *core.CreateProjectMilestoneInput) (*core.ProjectMilestone, error) {
	if input == nil {
		return nil, &core.ValidationError{Field: "input", Message: "input cannot be nil"}
	}
	if input.Name == "" {
		return nil, &core.ValidationError{Field: "name", Message: "name cannot be empty"}
	}
	if input.ProjectID == "" {
		return nil, &core.ValidationError{Field: "projectID", Message: "projectID cannot be empty"}
	}

	mutation := fmt.Sprintf(`
		mutation CreateProjectMilestone($input: ProjectMilestoneCreateInput!) {
			projectMilestoneCreate(input: $input) {
				success
				projectMilestone {
					%s
				}
			}
		}
	`, milestoneFields)

	var response struct {
		ProjectMilestoneCreate struct {
			Success          bool                  `json:"success"`
			ProjectMilestone core.ProjectMilestone `json:"projectMilestone"`
		} `json:"projectMilestoneCreate"`
	}
	variables := map[string]interface{}{"input": buildCreateInput(input)}

	if err := mc.base.ExecuteRequest(mutation, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to create project milestone: %w", err)
	}
	if !response.ProjectMilestoneCreate.Success {
		return nil, fmt.Errorf("project milestone creation was not successful")
	}

	return &response.ProjectMilestoneCreate.ProjectMilestone, nil
}

// Update updates a project milestone.
func (mc *Client) Update(id string, input *core.UpdateProjectMilestoneInput) (*core.ProjectMilestone, error) {
	if id == "" {
		return nil, &core.ValidationError{Field: "id", Message: "id cannot be empty"}
	}
	if input == nil {
		return nil, &core.ValidationError{Field: "input", Message: "input cannot be nil"}
	}

	inputMap := buildUpdateInput(input)
	if len(inputMap) == 0 {
		return nil, &core.ValidationError{Field: "input", Message: "at least one field must be provided"}
	}

	mutation := fmt.Sprintf(`
		mutation UpdateProjectMilestone($id: String!, $input: ProjectMilestoneUpdateInput!) {
			projectMilestoneUpdate(id: $id, input: $input) {
				success
				projectMilestone {
					%s
				}
			}
		}
	`, milestoneFields)

	var response struct {
		ProjectMilestoneUpdate struct {
			Success          bool                  `json:"success"`
			ProjectMilestone core.ProjectMilestone `json:"projectMilestone"`
		} `json:"projectMilestoneUpdate"`
	}
	variables := map[string]interface{}{
		"id":    id,
		"input": inputMap,
	}

	if err := mc.base.ExecuteRequest(mutation, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to update project milestone: %w", err)
	}
	if !response.ProjectMilestoneUpdate.Success {
		return nil, fmt.Errorf("project milestone update was not successful")
	}

	return &response.ProjectMilestoneUpdate.ProjectMilestone, nil
}

// Delete deletes a project milestone.
func (mc *Client) Delete(id string) error {
	if id == "" {
		return &core.ValidationError{Field: "id", Message: "id cannot be empty"}
	}

	const mutation = `
		mutation DeleteProjectMilestone($id: String!) {
			projectMilestoneDelete(id: $id) {
				success
				entityId
			}
		}
	`

	var response struct {
		ProjectMilestoneDelete struct {
			Success  bool   `json:"success"`
			EntityID string `json:"entityId"`
		} `json:"projectMilestoneDelete"`
	}
	variables := map[string]interface{}{"id": id}

	if err := mc.base.ExecuteRequest(mutation, variables, &response); err != nil {
		return fmt.Errorf("failed to delete project milestone: %w", err)
	}
	if !response.ProjectMilestoneDelete.Success {
		return fmt.Errorf("project milestone deletion was not successful")
	}

	return nil
}

func buildCreateInput(input *core.CreateProjectMilestoneInput) map[string]interface{} {
	inputMap := map[string]interface{}{
		"name":      input.Name,
		"projectId": input.ProjectID,
	}
	if input.Description != "" {
		inputMap["description"] = input.Description
	}
	if input.TargetDate != "" {
		inputMap["targetDate"] = input.TargetDate
	}
	return inputMap
}

func buildUpdateInput(input *core.UpdateProjectMilestoneInput) map[string]interface{} {
	inputMap := make(map[string]interface{})
	if input.Name != nil {
		inputMap["name"] = *input.Name
	}
	if input.Description != nil {
		inputMap["description"] = *input.Description
	}
	if input.TargetDate != nil {
		inputMap["targetDate"] = *input.TargetDate
	}
	return inputMap
}
