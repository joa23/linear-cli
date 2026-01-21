package linear

import (
	"fmt"
)

// CycleClient handles all cycle-related operations for the Linear API.
// It uses the shared BaseClient for HTTP communication and focuses on
// cycle (sprint/iteration) management functionality.
type CycleClient struct {
	base *BaseClient
}

// NewCycleClient creates a new cycle client with the provided base client
func NewCycleClient(base *BaseClient) *CycleClient {
	return &CycleClient{base: base}
}

// GetCycle retrieves a single cycle by ID
// Why: This is the primary method for fetching detailed cycle information
// including progress metrics and state indicators.
func (cc *CycleClient) GetCycle(cycleID string) (*Cycle, error) {
	if cycleID == "" {
		return nil, &ValidationError{Field: "cycleID", Message: "cycleID cannot be empty"}
	}

	const query = `
		query GetCycle($id: String!) {
			cycle(id: $id) {
				id
				name
				number
				description
				startsAt
				endsAt
				completedAt
				progress
				team {
					id
					name
					key
				}
				isActive
				isFuture
				isPast
				isNext
				isPrevious
				scopeHistory
				completedScopeHistory
				completedIssueCountHistory
				inProgressScopeHistory
				issueCountHistory
				createdAt
				updatedAt
				archivedAt
				autoArchivedAt
			}
		}
	`

	variables := map[string]interface{}{
		"id": cycleID,
	}

	var response struct {
		Cycle Cycle `json:"cycle"`
	}

	err := cc.base.executeRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get cycle: %w", err)
	}

	if response.Cycle.ID == "" {
		return nil, &NotFoundError{ResourceType: "cycle", ResourceID: cycleID}
	}

	return &response.Cycle, nil
}

// ListCycles retrieves cycles with optional filtering
// Why: Users need to discover and browse cycles by team, status (active/future/past),
// with pagination support for large cycle histories.
func (cc *CycleClient) ListCycles(filter *CycleFilter) (*CycleSearchResult, error) {
	if filter == nil {
		filter = &CycleFilter{}
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}

	const query = `
		query ListCycles($filter: CycleFilter, $first: Int, $after: String) {
			cycles(filter: $filter, first: $first, after: $after) {
				nodes {
					id
					name
					number
					description
					startsAt
					endsAt
					completedAt
					progress
					team {
						id
						name
						key
					}
					isActive
					isFuture
					isPast
					isNext
					isPrevious
					createdAt
					updatedAt
					archivedAt
				}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	`

	// Build filter object for GraphQL
	// Why: Linear's GraphQL API expects specific filter structure
	// with nested comparators for boolean fields.
	filterObj := make(map[string]interface{})
	if filter.TeamID != "" {
		filterObj["team"] = map[string]interface{}{
			"id": map[string]interface{}{
				"eq": filter.TeamID,
			},
		}
	}
	if filter.IsActive != nil {
		filterObj["isActive"] = map[string]interface{}{"eq": *filter.IsActive}
	}
	if filter.IsFuture != nil {
		filterObj["isFuture"] = map[string]interface{}{"eq": *filter.IsFuture}
	}
	if filter.IsPast != nil {
		filterObj["isPast"] = map[string]interface{}{"eq": *filter.IsPast}
	}

	variables := map[string]interface{}{
		"first": filter.Limit,
	}
	if len(filterObj) > 0 {
		variables["filter"] = filterObj
	}
	if filter.After != "" {
		variables["after"] = filter.After
	}

	var response struct {
		Cycles struct {
			Nodes    []Cycle `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"cycles"`
	}

	err := cc.base.executeRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list cycles: %w", err)
	}

	return &CycleSearchResult{
		Cycles:      response.Cycles.Nodes,
		HasNextPage: response.Cycles.PageInfo.HasNextPage,
		EndCursor:   response.Cycles.PageInfo.EndCursor,
	}, nil
}

// GetActiveCycle retrieves the current active cycle for a team
// Why: Most common use case is finding what cycle is currently active
// for sprint planning and issue assignment.
func (cc *CycleClient) GetActiveCycle(teamID string) (*Cycle, error) {
	if teamID == "" {
		return nil, &ValidationError{Field: "teamID", Message: "teamID cannot be empty"}
	}

	isActive := true
	result, err := cc.ListCycles(&CycleFilter{
		TeamID:   teamID,
		IsActive: &isActive,
		Limit:    1,
	})
	if err != nil {
		return nil, err
	}

	if len(result.Cycles) == 0 {
		return nil, &NotFoundError{ResourceType: "active cycle", ResourceID: teamID}
	}

	return &result.Cycles[0], nil
}

// GetCycleIssues retrieves issues for a specific cycle
// Why: Users need to see all issues in a cycle for sprint planning,
// progress tracking, and workload analysis.
func (cc *CycleClient) GetCycleIssues(cycleID string, limit int) ([]Issue, error) {
	if cycleID == "" {
		return nil, &ValidationError{Field: "cycleID", Message: "cycleID cannot be empty"}
	}
	if limit <= 0 {
		limit = 50
	}

	const query = `
		query GetCycleIssues($cycleId: String!, $first: Int) {
			cycle(id: $cycleId) {
				issues(first: $first) {
					nodes {
						id
						identifier
						title
						description
						state {
							id
							name
						}
						assignee {
							id
							name
							email
						}
						priority
						estimate
						createdAt
						updatedAt
						url
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"cycleId": cycleID,
		"first":   limit,
	}

	var response struct {
		Cycle struct {
			Issues struct {
				Nodes []Issue `json:"nodes"`
			} `json:"issues"`
		} `json:"cycle"`
	}

	err := cc.base.executeRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get cycle issues: %w", err)
	}

	return response.Cycle.Issues.Nodes, nil
}

// CreateCycle creates a new cycle in Linear
// Why: Teams need to create new sprints/iterations for planning work.
func (cc *CycleClient) CreateCycle(input *CreateCycleInput) (*Cycle, error) {
	if input == nil {
		return nil, &ValidationError{Field: "input", Message: "input cannot be nil"}
	}
	if input.TeamID == "" {
		return nil, &ValidationError{Field: "teamId", Message: "teamId cannot be empty"}
	}
	if input.StartsAt == "" {
		return nil, &ValidationError{Field: "startsAt", Message: "startsAt cannot be empty"}
	}
	if input.EndsAt == "" {
		return nil, &ValidationError{Field: "endsAt", Message: "endsAt cannot be empty"}
	}

	const mutation = `
		mutation CreateCycle($input: CycleCreateInput!) {
			cycleCreate(input: $input) {
				success
				cycle {
					id
					name
					number
					description
					startsAt
					endsAt
					completedAt
					progress
					team {
						id
						name
						key
					}
					isActive
					isFuture
					isPast
					isNext
					isPrevious
					createdAt
					updatedAt
				}
			}
		}
	`

	// Build input object
	inputObj := map[string]interface{}{
		"teamId":   input.TeamID,
		"startsAt": input.StartsAt,
		"endsAt":   input.EndsAt,
	}
	if input.Name != "" {
		inputObj["name"] = input.Name
	}
	if input.Description != "" {
		inputObj["description"] = input.Description
	}

	variables := map[string]interface{}{
		"input": inputObj,
	}

	var response struct {
		CycleCreate struct {
			Success bool  `json:"success"`
			Cycle   Cycle `json:"cycle"`
		} `json:"cycleCreate"`
	}

	err := cc.base.executeRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create cycle: %w", err)
	}

	if !response.CycleCreate.Success {
		return nil, fmt.Errorf("cycle creation was not successful")
	}

	return &response.CycleCreate.Cycle, nil
}

// UpdateCycle updates an existing cycle
// Why: Teams need to modify cycle dates, names, or mark cycles as completed.
func (cc *CycleClient) UpdateCycle(cycleID string, input *UpdateCycleInput) (*Cycle, error) {
	if cycleID == "" {
		return nil, &ValidationError{Field: "cycleID", Message: "cycleID cannot be empty"}
	}
	if input == nil {
		return nil, &ValidationError{Field: "input", Message: "input cannot be nil"}
	}

	const mutation = `
		mutation UpdateCycle($id: String!, $input: CycleUpdateInput!) {
			cycleUpdate(id: $id, input: $input) {
				success
				cycle {
					id
					name
					number
					description
					startsAt
					endsAt
					completedAt
					progress
					team {
						id
						name
						key
					}
					isActive
					isFuture
					isPast
					isNext
					isPrevious
					createdAt
					updatedAt
				}
			}
		}
	`

	// Build input object with only provided fields
	inputObj := make(map[string]interface{})
	if input.Name != nil {
		inputObj["name"] = *input.Name
	}
	if input.Description != nil {
		inputObj["description"] = *input.Description
	}
	if input.StartsAt != nil {
		inputObj["startsAt"] = *input.StartsAt
	}
	if input.EndsAt != nil {
		inputObj["endsAt"] = *input.EndsAt
	}
	if input.CompletedAt != nil {
		inputObj["completedAt"] = *input.CompletedAt
	}

	variables := map[string]interface{}{
		"id":    cycleID,
		"input": inputObj,
	}

	var response struct {
		CycleUpdate struct {
			Success bool  `json:"success"`
			Cycle   Cycle `json:"cycle"`
		} `json:"cycleUpdate"`
	}

	err := cc.base.executeRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to update cycle: %w", err)
	}

	if !response.CycleUpdate.Success {
		return nil, fmt.Errorf("cycle update was not successful")
	}

	return &response.CycleUpdate.Cycle, nil
}

// ArchiveCycle archives a cycle
// Why: Completed or obsolete cycles should be archived to keep the workspace clean.
func (cc *CycleClient) ArchiveCycle(cycleID string) error {
	if cycleID == "" {
		return &ValidationError{Field: "cycleID", Message: "cycleID cannot be empty"}
	}

	const mutation = `
		mutation ArchiveCycle($id: String!) {
			cycleArchive(id: $id) {
				success
			}
		}
	`

	variables := map[string]interface{}{
		"id": cycleID,
	}

	var response struct {
		CycleArchive struct {
			Success bool `json:"success"`
		} `json:"cycleArchive"`
	}

	err := cc.base.executeRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to archive cycle: %w", err)
	}

	if !response.CycleArchive.Success {
		return fmt.Errorf("cycle archival was not successful")
	}

	return nil
}
