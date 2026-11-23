// Package initiatives provides the GraphQL client for Linear initiatives:
// collections of projects with an owner, status, health, and target date.
package initiatives

import (
	"fmt"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// Client handles initiative-related operations
type Client struct {
	base *core.BaseClient
}

// NewClient creates a new initiatives client
func NewClient(base *core.BaseClient) *Client {
	return &Client{base: base}
}

// initiativeListFields omits content, sub-initiatives, and updates to keep list queries cheap.
// projects(first: 50) { nodes { id } } only feeds the project count shown in lists.
const initiativeListFields = `
	id
	slugId
	name
	description
	icon
	color
	url
	status
	health
	targetDate
	createdAt
	updatedAt
	archivedAt
	owner { id name displayName email }
	parentInitiative { id slugId name }
	projects(first: 50, includeSubInitiatives: false) { nodes { id } }
`

// initiativeFields is the full selection set used by get/create/update.
// Projects are the directly linked ones (includeSubInitiatives: false) so that the list
// matches what add-project/remove-project manage.
const initiativeFields = `
	id
	slugId
	name
	description
	content
	icon
	color
	url
	status
	health
	healthUpdatedAt
	targetDate
	startedAt
	completedAt
	createdAt
	updatedAt
	archivedAt
	owner { id name displayName email }
	creator { id name displayName }
	parentInitiative { id slugId name status }
	subInitiatives(first: 50) {
		nodes { id slugId name status health targetDate }
	}
	projects(first: 100, includeSubInitiatives: false) {
		nodes {
			id
			name
			slugId
			url
			progress
			targetDate
			status { name type }
			lead { id name displayName }
		}
	}
	initiativeUpdates(first: 5, orderBy: createdAt) {
		nodes { ` + initiativeUpdateFields + ` }
	}
`

// initiativeUpdateFields is the selection set for a status update.
const initiativeUpdateFields = `
	id
	body
	health
	url
	createdAt
	updatedAt
	editedAt
	user { id name displayName }
`

// ListFilter holds filter criteria for listing initiatives. OwnerID must be a UUID.
type ListFilter struct {
	Status          string // Planned, Active, Completed (exact enum value)
	OwnerID         string
	Query           string // Case-insensitive name substring match
	IncludeArchived bool
}

// ListResult holds a page of initiatives.
type ListResult struct {
	Initiatives []core.Initiative
	HasNextPage bool
	EndCursor   string
}

// CreateInput holds parameters for creating an initiative.
type CreateInput struct {
	Name        string  // Required
	Description string  // "" = omit
	Content     *string // nil = omit
	OwnerID     string
	Status      string // InitiativeStatus enum value
	TargetDate  string // YYYY-MM-DD
	Icon        string
	Color       string
}

// UpdateInput holds parameters for updating an initiative. nil fields are not sent.
// For OwnerID and TargetDate a pointer to "" is sent as null, which clears the value.
type UpdateInput struct {
	Name        *string
	Description *string
	Content     *string
	OwnerID     *string
	Status      *string
	TargetDate  *string
	Icon        *string
	Color       *string
}

// StatusUpdateInput holds parameters for posting an initiative status update.
type StatusUpdateInput struct {
	InitiativeID string // Required (UUID)
	Body         string
	Health       string // onTrack, atRisk, offTrack ("" = omit)
}

// buildFilter converts a ListFilter into a GraphQL InitiativeFilter map.
// Returns nil when no criteria are set so the variable is omitted entirely.
func buildFilter(f *ListFilter) map[string]interface{} {
	if f == nil {
		return nil
	}
	filter := map[string]interface{}{}
	if f.Status != "" {
		filter["status"] = map[string]interface{}{"eq": f.Status}
	}
	if f.OwnerID != "" {
		filter["owner"] = map[string]interface{}{"id": map[string]interface{}{"eq": f.OwnerID}}
	}
	if f.Query != "" {
		filter["name"] = map[string]interface{}{"containsIgnoreCase": f.Query}
	}
	if len(filter) == 0 {
		return nil
	}
	return filter
}

// ListInitiatives returns initiatives matching the filter, most recently updated first.
func (ic *Client) ListInitiatives(filter *ListFilter, limit int) (*ListResult, error) {
	const query = `
		query Initiatives($first: Int!, $filter: InitiativeFilter, $includeArchived: Boolean) {
			initiatives(first: $first, filter: $filter, includeArchived: $includeArchived, orderBy: updatedAt) {
				nodes {` + initiativeListFields + `}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	`

	if limit <= 0 {
		limit = 50
	}

	variables := map[string]interface{}{
		"first": limit,
	}
	if f := buildFilter(filter); f != nil {
		variables["filter"] = f
	}
	if filter != nil && filter.IncludeArchived {
		variables["includeArchived"] = true
	}

	var response struct {
		Initiatives struct {
			Nodes    []core.Initiative `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"initiatives"`
	}

	if err := ic.base.ExecuteRequest(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to list initiatives: %w", err)
	}

	return &ListResult{
		Initiatives: response.Initiatives.Nodes,
		HasNextPage: response.Initiatives.PageInfo.HasNextPage,
		EndCursor:   response.Initiatives.PageInfo.EndCursor,
	}, nil
}

// GetInitiative fetches an initiative by UUID with projects, sub-initiatives, and recent updates.
func (ic *Client) GetInitiative(id string) (*core.Initiative, error) {
	if id == "" {
		return nil, &core.ValidationError{Field: "id", Message: "initiative ID is required"}
	}

	const query = `
		query Initiative($id: String!) {
			initiative(id: $id) {` + initiativeFields + `}
		}
	`

	var response struct {
		Initiative core.Initiative `json:"initiative"`
	}

	if err := ic.base.ExecuteRequest(query, map[string]interface{}{"id": id}, &response); err != nil {
		return nil, fmt.Errorf("failed to get initiative: %w", err)
	}

	if response.Initiative.ID == "" {
		return nil, &core.NotFoundError{ResourceType: "initiative", ResourceID: id}
	}

	return &response.Initiative, nil
}

// GetInitiativeBySlug fetches an initiative by its slugId (the trailing token of its Linear URL).
func (ic *Client) GetInitiativeBySlug(slug string) (*core.Initiative, error) {
	if slug == "" {
		return nil, &core.ValidationError{Field: "slug", Message: "initiative slug is required"}
	}

	const query = `
		query InitiativeBySlug($slug: String!) {
			initiatives(first: 1, filter: { slugId: { eq: $slug } }, includeArchived: true) {
				nodes {` + initiativeFields + `}
			}
		}
	`

	var response struct {
		Initiatives struct {
			Nodes []core.Initiative `json:"nodes"`
		} `json:"initiatives"`
	}

	if err := ic.base.ExecuteRequest(query, map[string]interface{}{"slug": slug}, &response); err != nil {
		return nil, fmt.Errorf("failed to get initiative: %w", err)
	}

	if len(response.Initiatives.Nodes) == 0 {
		return nil, &core.NotFoundError{ResourceType: "initiative", ResourceID: slug}
	}

	return &response.Initiatives.Nodes[0], nil
}

// FindInitiativesByName returns initiatives whose name equals name (case-insensitive),
// including archived ones. Only id, slugId, name, and status are populated.
func (ic *Client) FindInitiativesByName(name string) ([]core.Initiative, error) {
	if name == "" {
		return nil, &core.ValidationError{Field: "name", Message: "initiative name is required"}
	}

	const query = `
		query InitiativesByName($name: String!) {
			initiatives(first: 10, filter: { name: { eqIgnoreCase: $name } }, includeArchived: true) {
				nodes { id slugId name status archivedAt }
			}
		}
	`

	var response struct {
		Initiatives struct {
			Nodes []core.Initiative `json:"nodes"`
		} `json:"initiatives"`
	}

	if err := ic.base.ExecuteRequest(query, map[string]interface{}{"name": name}, &response); err != nil {
		return nil, fmt.Errorf("failed to look up initiative by name: %w", err)
	}

	return response.Initiatives.Nodes, nil
}

// buildCreateInput converts CreateInput into the GraphQL InitiativeCreateInput map.
func buildCreateInput(input *CreateInput) map[string]interface{} {
	m := map[string]interface{}{
		"name": input.Name,
	}
	setIf := func(key, v string) {
		if v != "" {
			m[key] = v
		}
	}
	setIf("description", input.Description)
	if input.Content != nil {
		m["content"] = *input.Content
	}
	setIf("ownerId", input.OwnerID)
	setIf("status", input.Status)
	setIf("targetDate", input.TargetDate)
	setIf("icon", input.Icon)
	setIf("color", input.Color)
	return m
}

// CreateInitiative creates a new initiative.
func (ic *Client) CreateInitiative(input *CreateInput) (*core.Initiative, error) {
	if input == nil || input.Name == "" {
		return nil, &core.ValidationError{Field: "name", Message: "initiative name is required"}
	}

	const mutation = `
		mutation InitiativeCreate($input: InitiativeCreateInput!) {
			initiativeCreate(input: $input) {
				success
				initiative {` + initiativeFields + `}
			}
		}
	`

	var response struct {
		InitiativeCreate struct {
			Success    bool            `json:"success"`
			Initiative core.Initiative `json:"initiative"`
		} `json:"initiativeCreate"`
	}

	if err := ic.base.ExecuteRequest(mutation, map[string]interface{}{"input": buildCreateInput(input)}, &response); err != nil {
		return nil, fmt.Errorf("failed to create initiative: %w", err)
	}

	if !response.InitiativeCreate.Success {
		return nil, fmt.Errorf("initiativeCreate returned success=false")
	}

	return &response.InitiativeCreate.Initiative, nil
}

// buildUpdateInput converts UpdateInput into the GraphQL InitiativeUpdateInput map.
// Only non-nil fields are included.
func buildUpdateInput(input *UpdateInput) map[string]interface{} {
	m := map[string]interface{}{}
	if input == nil {
		return m
	}
	set := func(key string, v *string) {
		if v != nil {
			m[key] = *v
		}
	}
	// Nullable references: an explicit "" clears the value (sent as null).
	setNullable := func(key string, v *string) {
		if v == nil {
			return
		}
		if *v == "" {
			m[key] = nil
			return
		}
		m[key] = *v
	}
	set("name", input.Name)
	set("description", input.Description)
	set("content", input.Content)
	setNullable("ownerId", input.OwnerID)
	set("status", input.Status)
	setNullable("targetDate", input.TargetDate)
	set("icon", input.Icon)
	set("color", input.Color)
	return m
}

// UpdateInitiative updates an existing initiative by UUID.
func (ic *Client) UpdateInitiative(id string, input *UpdateInput) (*core.Initiative, error) {
	if id == "" {
		return nil, &core.ValidationError{Field: "id", Message: "initiative ID is required"}
	}

	inputMap := buildUpdateInput(input)
	if len(inputMap) == 0 {
		return nil, &core.ValidationError{Field: "input", Message: "no fields to update"}
	}

	const mutation = `
		mutation InitiativeUpdate($id: String!, $input: InitiativeUpdateInput!) {
			initiativeUpdate(id: $id, input: $input) {
				success
				initiative {` + initiativeFields + `}
			}
		}
	`

	variables := map[string]interface{}{
		"id":    id,
		"input": inputMap,
	}

	var response struct {
		InitiativeUpdate struct {
			Success    bool            `json:"success"`
			Initiative core.Initiative `json:"initiative"`
		} `json:"initiativeUpdate"`
	}

	if err := ic.base.ExecuteRequest(mutation, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to update initiative: %w", err)
	}

	if !response.InitiativeUpdate.Success {
		return nil, fmt.Errorf("initiativeUpdate returned success=false")
	}

	return &response.InitiativeUpdate.Initiative, nil
}

// ArchiveInitiative archives an initiative by UUID.
func (ic *Client) ArchiveInitiative(id string) error {
	if id == "" {
		return &core.ValidationError{Field: "id", Message: "initiative ID is required"}
	}

	const mutation = `
		mutation InitiativeArchive($id: String!) {
			initiativeArchive(id: $id) {
				success
			}
		}
	`

	var response struct {
		InitiativeArchive struct {
			Success bool `json:"success"`
		} `json:"initiativeArchive"`
	}

	if err := ic.base.ExecuteRequest(mutation, map[string]interface{}{"id": id}, &response); err != nil {
		return fmt.Errorf("failed to archive initiative: %w", err)
	}
	if !response.InitiativeArchive.Success {
		return fmt.Errorf("initiativeArchive returned success=false")
	}
	return nil
}

// UnarchiveInitiative restores an archived initiative by UUID.
func (ic *Client) UnarchiveInitiative(id string) error {
	if id == "" {
		return &core.ValidationError{Field: "id", Message: "initiative ID is required"}
	}

	const mutation = `
		mutation InitiativeUnarchive($id: String!) {
			initiativeUnarchive(id: $id) {
				success
			}
		}
	`

	var response struct {
		InitiativeUnarchive struct {
			Success bool `json:"success"`
		} `json:"initiativeUnarchive"`
	}

	if err := ic.base.ExecuteRequest(mutation, map[string]interface{}{"id": id}, &response); err != nil {
		return fmt.Errorf("failed to unarchive initiative: %w", err)
	}
	if !response.InitiativeUnarchive.Success {
		return fmt.Errorf("initiativeUnarchive returned success=false")
	}
	return nil
}

// DeleteInitiative moves an initiative to trash by UUID.
func (ic *Client) DeleteInitiative(id string) error {
	if id == "" {
		return &core.ValidationError{Field: "id", Message: "initiative ID is required"}
	}

	const mutation = `
		mutation InitiativeDelete($id: String!) {
			initiativeDelete(id: $id) {
				success
			}
		}
	`

	var response struct {
		InitiativeDelete struct {
			Success bool `json:"success"`
		} `json:"initiativeDelete"`
	}

	if err := ic.base.ExecuteRequest(mutation, map[string]interface{}{"id": id}, &response); err != nil {
		return fmt.Errorf("failed to delete initiative: %w", err)
	}
	if !response.InitiativeDelete.Success {
		return fmt.Errorf("initiativeDelete returned success=false")
	}
	return nil
}

// AddProject links a project to an initiative. Both IDs must be UUIDs.
func (ic *Client) AddProject(initiativeID, projectID string) (*core.InitiativeToProject, error) {
	if initiativeID == "" {
		return nil, &core.ValidationError{Field: "initiativeId", Message: "initiative ID is required"}
	}
	if projectID == "" {
		return nil, &core.ValidationError{Field: "projectId", Message: "project ID is required"}
	}

	const mutation = `
		mutation InitiativeToProjectCreate($input: InitiativeToProjectCreateInput!) {
			initiativeToProjectCreate(input: $input) {
				success
				initiativeToProject {
					id
					initiative { id slugId name }
					project { id name }
				}
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"initiativeId": initiativeID,
			"projectId":    projectID,
		},
	}

	var response struct {
		InitiativeToProjectCreate struct {
			Success             bool                     `json:"success"`
			InitiativeToProject core.InitiativeToProject `json:"initiativeToProject"`
		} `json:"initiativeToProjectCreate"`
	}

	if err := ic.base.ExecuteRequest(mutation, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to add project to initiative: %w", err)
	}
	if !response.InitiativeToProjectCreate.Success {
		return nil, fmt.Errorf("initiativeToProjectCreate returned success=false")
	}

	return &response.InitiativeToProjectCreate.InitiativeToProject, nil
}

// FindProjectLink returns the InitiativeToProject link ID joining initiativeID and projectID,
// or a NotFoundError if the project is not linked to the initiative. Both IDs must be UUIDs.
func (ic *Client) FindProjectLink(initiativeID, projectID string) (string, error) {
	if initiativeID == "" || projectID == "" {
		return "", &core.ValidationError{Field: "id", Message: "initiative and project IDs are required"}
	}

	// initiativeToProjects has no filter argument at the root, so walk the project's links.
	const query = `
		query ProjectInitiativeLinks($projectId: String!) {
			project(id: $projectId) {
				initiativeToProjects(first: 100) {
					nodes {
						id
						initiative { id }
					}
				}
			}
		}
	`

	var response struct {
		Project struct {
			InitiativeToProjects struct {
				Nodes []core.InitiativeToProject `json:"nodes"`
			} `json:"initiativeToProjects"`
		} `json:"project"`
	}

	if err := ic.base.ExecuteRequest(query, map[string]interface{}{"projectId": projectID}, &response); err != nil {
		return "", fmt.Errorf("failed to look up project's initiatives: %w", err)
	}

	for _, link := range response.Project.InitiativeToProjects.Nodes {
		if link.Initiative != nil && link.Initiative.ID == initiativeID {
			return link.ID, nil
		}
	}

	return "", &core.NotFoundError{ResourceType: "initiative-project link", ResourceID: projectID}
}

// RemoveProjectLink deletes an InitiativeToProject link by its ID.
func (ic *Client) RemoveProjectLink(linkID string) error {
	if linkID == "" {
		return &core.ValidationError{Field: "id", Message: "link ID is required"}
	}

	const mutation = `
		mutation InitiativeToProjectDelete($id: String!) {
			initiativeToProjectDelete(id: $id) {
				success
			}
		}
	`

	var response struct {
		InitiativeToProjectDelete struct {
			Success bool `json:"success"`
		} `json:"initiativeToProjectDelete"`
	}

	if err := ic.base.ExecuteRequest(mutation, map[string]interface{}{"id": linkID}, &response); err != nil {
		return fmt.Errorf("failed to remove project from initiative: %w", err)
	}
	if !response.InitiativeToProjectDelete.Success {
		return fmt.Errorf("initiativeToProjectDelete returned success=false")
	}
	return nil
}

// ListUpdates returns the most recent status updates for an initiative (newest first).
func (ic *Client) ListUpdates(initiativeID string, limit int) ([]core.InitiativeUpdate, error) {
	if initiativeID == "" {
		return nil, &core.ValidationError{Field: "id", Message: "initiative ID is required"}
	}
	if limit <= 0 {
		limit = 10
	}

	const query = `
		query InitiativeUpdates($id: String!, $first: Int!) {
			initiative(id: $id) {
				initiativeUpdates(first: $first, orderBy: createdAt) {
					nodes {` + initiativeUpdateFields + `}
				}
			}
		}
	`

	var response struct {
		Initiative struct {
			InitiativeUpdates core.InitiativeUpdateConnection `json:"initiativeUpdates"`
		} `json:"initiative"`
	}

	variables := map[string]interface{}{"id": initiativeID, "first": limit}
	if err := ic.base.ExecuteRequest(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to list initiative updates: %w", err)
	}

	return response.Initiative.InitiativeUpdates.Nodes, nil
}

// buildStatusUpdateInput converts StatusUpdateInput into the GraphQL InitiativeUpdateCreateInput map.
func buildStatusUpdateInput(input *StatusUpdateInput) map[string]interface{} {
	m := map[string]interface{}{
		"initiativeId": input.InitiativeID,
		"body":         input.Body,
	}
	if input.Health != "" {
		m["health"] = input.Health
	}
	return m
}

// CreateUpdate posts a status update on an initiative.
func (ic *Client) CreateUpdate(input *StatusUpdateInput) (*core.InitiativeUpdate, error) {
	if input == nil || input.InitiativeID == "" {
		return nil, &core.ValidationError{Field: "initiativeId", Message: "initiative ID is required"}
	}

	const mutation = `
		mutation InitiativeUpdateCreate($input: InitiativeUpdateCreateInput!) {
			initiativeUpdateCreate(input: $input) {
				success
				initiativeUpdate {` + initiativeUpdateFields + `
					initiative { id slugId name }
				}
			}
		}
	`

	var response struct {
		InitiativeUpdateCreate struct {
			Success          bool                  `json:"success"`
			InitiativeUpdate core.InitiativeUpdate `json:"initiativeUpdate"`
		} `json:"initiativeUpdateCreate"`
	}

	if err := ic.base.ExecuteRequest(mutation, map[string]interface{}{"input": buildStatusUpdateInput(input)}, &response); err != nil {
		return nil, fmt.Errorf("failed to create initiative update: %w", err)
	}
	if !response.InitiativeUpdateCreate.Success {
		return nil, fmt.Errorf("initiativeUpdateCreate returned success=false")
	}

	return &response.InitiativeUpdateCreate.InitiativeUpdate, nil
}
