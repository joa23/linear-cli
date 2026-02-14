package issues

import (
	"errors"
	"fmt"
	"strings"

	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/joa23/linear-cli/internal/linear/guidance"
	"github.com/joa23/linear-cli/internal/linear/metadata"
	"github.com/joa23/linear-cli/internal/linear/validation"
)

// IssueClient handles all issue-related operations for the Linear API.
// It uses the shared BaseClient for HTTP communication and focuses solely
// on issue management functionality.
type Client struct {
	base *core.BaseClient
}

// NewIssueClient creates a new issue client with the provided base client
func NewClient(base *core.BaseClient) *Client {
	return &Client{base: base}
}

// CreateIssue creates a new issue in Linear
// Why: This is a core function for creating work items in Linear. It requires
// a team assignment and supports optional descriptions for detailed context.
func (ic *Client) CreateIssue(title, description, teamID string) (*core.Issue, error) {
	// Validate required inputs
	// Why: Title and teamID are mandatory fields in Linear. Empty values would
	// cause the API to reject the request, so we validate early for better errors.
	if title == "" {
		return nil, guidance.ValidationErrorWithExample("title", "cannot be empty",
			`linear_create_issue("Fix login bug", "Bug in authentication flow", "team-123")`)
	}
	if teamID == "" {
		return nil, guidance.ValidationErrorWithExample("teamID", "cannot be empty",
			`// First, get available teams
teams = linear_get_teams()
// Then use a team ID
linear_create_issue("Task title", "Description", teams[0].id)`)
	}

	// Validate length limits
	// Why: Linear enforces limits on title and description lengths
	if err := validation.ValidateStringLength(title, "title", validation.MaxTitleLength); err != nil {
		return nil, err
	}
	if description != "" {
		if err := validation.ValidateStringLength(description, "description", validation.MaxDescriptionLength); err != nil {
			return nil, err
		}
	}
	
	const mutation = `
		mutation CreateIssue($input: IssueCreateInput!) {
			issueCreate(input: $input) {
				success
				issue {
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
					createdAt
					updatedAt
					url
					project {
						id
						name
					}
					parent {
						id
						identifier
						title
					}
					children {
						nodes {
							id
							identifier
							title
							state {
								id
								name
							}
						}
					}
				}
			}
		}
	`
	
	// Build the input object
	// Why: Linear's API expects an input object with specific fields.
	// We conditionally include description only if provided to avoid
	// sending empty strings which might override defaults.
	input := map[string]interface{}{
		"title":  title,
		"teamId": teamID,
	}
	if description != "" {
		input["description"] = description
	}
	
	variables := map[string]interface{}{
		"input": input,
	}
	
	var response struct {
		IssueCreate struct {
			Success bool  `json:"success"`
			Issue   core.Issue `json:"issue"`
		} `json:"issueCreate"`
	}
	
	err := ic.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}
	
	if !response.IssueCreate.Success {
		return nil, fmt.Errorf("issue creation was not successful")
	}
	
	// Extract metadata from description if present
	// Why: We store metadata in issue descriptions as hidden markdown.
	// After creating an issue, we need to extract this metadata to populate
	// the Metadata field in our Issue struct for consistent access.
	if response.IssueCreate.Issue.Description != "" {
		metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.IssueCreate.Issue.Description)
		response.IssueCreate.Issue.Metadata = metadata
		response.IssueCreate.Issue.Description = cleanDesc
	}
	
	return &response.IssueCreate.Issue, nil
}

// GetIssue retrieves a single issue by ID
// Why: This is the primary method for fetching detailed issue information.
// It automatically extracts metadata from the description for easy access.
func (ic *Client) GetIssue(issueID string) (*core.Issue, error) {
	// Validate input
	// Why: An empty issue ID would cause the query to fail. Early validation
	// provides clearer error messages than GraphQL errors.
	if issueID == "" {
		return nil, &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	
	const query = `
		query GetIssue($id: String!) {
			issue(id: $id) {
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
				delegate {
					id
					name
					email
				}
				priority
				estimate
				dueDate
				labels {
					nodes {
						id
						name
						color
					}
				}
				cycle {
					id
					number
					name
				}
				createdAt
				updatedAt
				url
				project {
					id
					name
				}
				parent {
					id
					identifier
					title
				}
				children {
					nodes {
						id
						identifier
						title
						state {
							id
							name
						}
					}
				}
				attachments(first: 50) {
					nodes {
						id
						url
						title
						subtitle
						createdAt
						sourceType
					}
				}
				comments(first: 50) {
					nodes {
						id
						body
						createdAt
						updatedAt
						user {
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
		"id": issueID,
	}
	
	var response struct {
		Issue struct {
			core.Issue
			Attachments struct {
				Nodes []struct {
					ID string `json:"id"`
				} `json:"nodes"`
			} `json:"attachments"`
		} `json:"issue"`
	}
	
	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	// Check if issue was found (null response means not found)
	// This can happen when using identifiers that don't exist
	if response.Issue.ID == "" {
		return nil, &core.NotFoundError{
			ResourceType: "issue",
			ResourceID:   issueID,
		}
	}

	// Extract metadata from description
	// Why: Metadata is stored in descriptions but should be easily accessible
	// as a separate field. This extraction ensures consistent metadata access
	// regardless of how the issue was retrieved.
	if response.Issue.Description != "" {
		metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.Issue.Description)
		response.Issue.Metadata = metadata
		response.Issue.Description = cleanDesc
	}
	
	// Set attachment fields
	// Why: AttachmentCount is computed from the nodes length, and HasAttachments
	// is a computed boolean for easy checking by agents
	response.Issue.AttachmentCount = len(response.Issue.Attachments.Nodes)
	response.Issue.HasAttachments = response.Issue.AttachmentCount > 0
	
	return &response.Issue.Issue, nil
}

// GetIssueWithProjectContext retrieves an issue with additional project information
// Why: When working within a project context, we need more project details like
// metadata and state. This method provides that extended information in one call.
func (ic *Client) GetIssueWithProjectContext(issueID string) (*core.Issue, error) {
	issue, err := ic.getIssueWithProjectContextInternal(issueID)
	if err != nil {
		// Check if it's a server error and try fallback
		var httpErr *core.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode >= 500 {
			// Try simplified query, then enrich with project data separately
			issue, err = ic.GetIssueSimplified(issueID)
			if err != nil {
				return nil, err
			}
			// Note: This simplified version won't have full project details,
			// but it's better than failing completely
			return issue, nil
		}
		return nil, err
	}
	return issue, nil
}

// getIssueWithProjectContextInternal is the internal implementation that can fail
func (ic *Client) getIssueWithProjectContextInternal(issueID string) (*core.Issue, error) {
	if issueID == "" {
		return nil, &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	
	const query = `
		query GetIssueWithProject($id: String!) {
			issue(id: $id) {
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
				delegate {
					id
					name
					email
				}
				priority
				estimate
				dueDate
				labels {
					nodes {
						id
						name
						color
					}
				}
				cycle {
					id
					number
					name
				}
				createdAt
				updatedAt
				url
				project {
					id
					name
					description
					state
					createdAt
					updatedAt
				}
				parent {
					id
					identifier
					title
				}
				children {
					nodes {
						id
						identifier
						title
						state {
							id
							name
						}
					}
				}
				attachments(first: 50) {
					nodes {
						id
						url
						title
						subtitle
						createdAt
						sourceType
					}
				}
				comments(first: 50) {
					nodes {
						id
						body
						createdAt
						updatedAt
						user {
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
		"id": issueID,
	}

	var response struct {
		Issue core.Issue `json:"issue"`
	}

	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue with project context: %w", err)
	}
	
	// Extract metadata from issue description
	if response.Issue.Description != "" {
		metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.Issue.Description)
		response.Issue.Metadata = metadata
		response.Issue.Description = cleanDesc
	}
	
	// Extract metadata from project description if project exists
	// Why: Projects can also have metadata. When fetching project context,
	// we want to ensure project metadata is also extracted and available.
	if response.Issue.Project != nil && response.Issue.Project.Description != "" {
		projectMetadata, cleanProjectDesc := metadata.ExtractMetadataFromDescription(response.Issue.Project.Description)
		response.Issue.Project.Metadata = projectMetadata
		response.Issue.Project.Description = cleanProjectDesc
	}
	
	return &response.Issue, nil
}

// GetIssueWithParentContext retrieves an issue with parent issue details
// Why: Understanding issue hierarchy is important for sub-tasks. This method
// provides parent context needed for proper sub-task management.
func (ic *Client) GetIssueWithParentContext(issueID string) (*core.Issue, error) {
	issue, err := ic.getIssueWithParentContextInternal(issueID)
	if err != nil {
		// Check if it's a server error and try fallback
		var httpErr *core.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode >= 500 {
			// Try simplified query, won't have full parent details
			issue, err = ic.GetIssueSimplified(issueID)
			if err != nil {
				return nil, err
			}
			return issue, nil
		}
		return nil, err
	}
	return issue, nil
}

// getIssueWithParentContextInternal is the internal implementation that can fail
func (ic *Client) getIssueWithParentContextInternal(issueID string) (*core.Issue, error) {
	if issueID == "" {
		return nil, &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	
	const query = `
		query GetIssueWithParent($id: String!) {
			issue(id: $id) {
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
				delegate {
					id
					name
					email
				}
				priority
				estimate
				dueDate
				labels {
					nodes {
						id
						name
						color
					}
				}
				cycle {
					id
					number
					name
				}
				createdAt
				updatedAt
				url
				project {
					id
					name
				}
				parent {
					id
					identifier
					title
					description
					state {
						id
						name
					}
				}
				children {
					nodes {
						id
						identifier
						title
						state {
							id
							name
						}
					}
				}
				attachments(first: 50) {
					nodes {
						id
						url
						title
						subtitle
						createdAt
						sourceType
					}
				}
				comments(first: 50) {
					nodes {
						id
						body
						createdAt
						updatedAt
						user {
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
		"id": issueID,
	}

	var response struct {
		Issue core.Issue `json:"issue"`
	}

	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue with parent context: %w", err)
	}
	
	// Extract metadata from issue description
	if response.Issue.Description != "" {
		metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.Issue.Description)
		response.Issue.Metadata = metadata
		response.Issue.Description = cleanDesc
	}
	
	// Extract metadata from parent description if parent exists
	// Why: Parent issues may contain metadata that provides context for
	// sub-tasks. Extracting it ensures complete metadata visibility.
	if response.Issue.Parent != nil && response.Issue.Parent.Description != "" {
		parentMetadata, cleanParentDesc := metadata.ExtractMetadataFromDescription(response.Issue.Parent.Description)
		response.Issue.Parent.Metadata = parentMetadata
		response.Issue.Parent.Description = cleanParentDesc
	}
	
	return &response.Issue, nil
}

// UpdateIssueState updates the workflow state of an issue
// Why: Moving issues through workflow states is a core project management
// action. This method provides that capability with proper validation.
func (ic *Client) UpdateIssueState(issueID, stateID string) error {
	// Validate inputs
	// Why: Both IDs are required for the mutation. Empty values would
	// cause confusing GraphQL errors, so we validate early.
	if issueID == "" {
		return &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	if stateID == "" {
		return &core.ValidationError{Field: "stateID", Message: "stateID cannot be empty"}
	}
	
	const mutation = `
		mutation UpdateIssueState($issueId: String!, $stateId: String!) {
			issueUpdate(
				id: $issueId,
				input: { stateId: $stateId }
			) {
				success
				issue {
					id
					state {
						id
						name
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"issueId": issueID,
		"stateId": stateID,
	}
	
	var response struct {
		IssueUpdate struct {
			Success bool `json:"success"`
			Issue   struct {
				ID    string `json:"id"`
				State struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"state"`
			} `json:"issue"`
		} `json:"issueUpdate"`
	}
	
	err := ic.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		// Check if this is a state ID not found error
		// Why: The Linear API returns specific error messages when state IDs
		// are invalid. We want to provide helpful guidance to users.
		if strings.Contains(err.Error(), "Entity not found in validateAccess: stateId") || 
		   strings.Contains(err.Error(), "does not exist") && strings.Contains(err.Error(), "state") {
			return guidance.InvalidStateIDError(stateID, err)
		}
		return guidance.EnhanceGenericError("update issue state", err)
	}
	
	if !response.IssueUpdate.Success {
		return guidance.OperationFailedError("Update issue state", "issue", []string{
			"Verify the issue ID exists using linear_get_issue",
			"Check if the state transition is allowed from current state",
			"Ensure you have permission to update this issue",
		})
	}
	
	return nil
}

// AssignIssue assigns or unassigns an issue to/from a user
// Why: Issue assignment is crucial for workload management. Passing an empty
// assigneeID unassigns the issue, providing flexibility in one method.
func (ic *Client) AssignIssue(issueID, assigneeID string) error {
	if issueID == "" {
		return &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	
	const mutation = `
		mutation AssignIssue($issueId: String!, $assigneeId: String) {
			issueUpdate(
				id: $issueId,
				input: { assigneeId: $assigneeId }
			) {
				success
				issue {
					id
					assignee {
						id
						name
						email
					}
				}
			}
		}
	`

	// Handle nullable assignee ID
	// Why: Linear accepts nullable strings directly. An empty string means
	// "unassign" which we represent as null in the GraphQL mutation.
	var assigneeInput interface{}
	if assigneeID == "" {
		assigneeInput = nil
	} else {
		assigneeInput = assigneeID
	}

	variables := map[string]interface{}{
		"issueId":    issueID,
		"assigneeId": assigneeInput,
	}
	
	var response struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}
	
	err := ic.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to assign issue: %w", err)
	}
	
	if !response.IssueUpdate.Success {
		return fmt.Errorf("issue assignment was not successful")
	}
	
	return nil
}

// ListAssignedIssues retrieves issues assigned to the authenticated user
// Why: Users need to see their workload. This method provides a focused view
// of assigned work with configurable result limits.
func (ic *Client) ListAssignedIssues(limit int) ([]core.Issue, error) {
	// Default limit if not specified
	// Why: We need some limit to prevent overwhelming responses. 50 is a
	// reasonable default that balances completeness with performance.
	if limit <= 0 {
		limit = 50
	}
	
	const query = `
		query ListAssignedIssues($filter: IssueFilter, $first: Int) {
			issues(filter: $filter, first: $first) {
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
					createdAt
					updatedAt
					url
					project {
						id
						name
					}
					parent {
						id
						identifier
						title
					}
					children {
						nodes {
							id
							identifier
							title
							state {
								id
									name
							}
						}
					}
				}
			}
		}
	`
	
	// Filter for issues assigned to the current user
	// Why: The "me" identifier is Linear's way of referring to the
	// authenticated user without needing to know their specific ID.
	filter := map[string]interface{}{
		"assignee": map[string]interface{}{
			"isMe": map[string]interface{}{
				"eq": true,
			},
		},
	}
	
	variables := map[string]interface{}{
		"filter": filter,
		"first":  limit,
	}
	
	var response struct {
		Issues struct {
			Nodes []core.Issue `json:"nodes"`
		} `json:"issues"`
	}
	
	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list assigned issues: %w", err)
	}
	
	// Extract metadata from descriptions
	// Why: Each issue might have metadata. We extract it here to ensure
	// consistent metadata access across all retrieval methods.
	for i := range response.Issues.Nodes {
		if response.Issues.Nodes[i].Description != "" {
			metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.Issues.Nodes[i].Description)
			response.Issues.Nodes[i].Metadata = metadata
			response.Issues.Nodes[i].Description = cleanDesc
		}
	}
	
	return response.Issues.Nodes, nil
}

// SearchIssuesEnhanced searches for issues with advanced filtering options
// Why: Users need flexible search capabilities to find issues based on multiple
// criteria like team, state, labels, assignee, priority, and date ranges.
func (ic *Client) SearchIssuesEnhanced(filters *core.IssueSearchFilters) (*core.IssueSearchResult, error) {
	// Default limit if not specified
	if filters == nil {
		filters = &core.IssueSearchFilters{}
	}
	if filters.Limit <= 0 {
		filters.Limit = 10 // Reduced from 50 to minimize token usage
	}
	
	const query = `
		query SearchIssuesEnhanced($filter: IssueFilter, $first: Int, $after: String, $includeArchived: Boolean, $orderBy: PaginationOrderBy) {
			issues(filter: $filter, first: $first, after: $after, includeArchived: $includeArchived, orderBy: $orderBy) {
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
					labels {
						nodes {
							id
							name
							color
						}
					}
					priority
					createdAt
					updatedAt
					url
					project {
						id
						name
					}
					cycle {
						id
						name
						number
					}
					parent {
						id
						identifier
						title
					}
					children {
						nodes {
							id
							identifier
							title
							state {
								id
								name
							}
						}
					}
				}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	`

	// Build filter object
	filter := make(map[string]interface{})
	
	// Team filter
	if filters.TeamID != "" {
		// Linear's team filter requires IDComparator format
		filter["team"] = map[string]interface{}{
			"id": map[string]interface{}{
				"eq": filters.TeamID,
			},
		}
	}

	// Project filter
	if filters.ProjectID != "" {
		filter["project"] = map[string]interface{}{
			"id": map[string]interface{}{
				"eq": filters.ProjectID,
			},
		}
	}

	// Identifier filter (e.g., "CEN-123")
	// Note: Use "identifier" field for issue identifiers, not "id"
	if filters.Identifier != "" {
		filter["identifier"] = map[string]interface{}{
			"eq": filters.Identifier,
		}
	}

	// State filters
	if len(filters.StateIDs) > 0 {
		filter["state"] = map[string]interface{}{
			"id": map[string]interface{}{
				"in": filters.StateIDs,
			},
		}
	}
	
	// Label filters (include and/or exclude)
	hasIncludeLabels := len(filters.LabelIDs) > 0
	hasExcludeLabels := len(filters.ExcludeLabelIDs) > 0
	if hasIncludeLabels && hasExcludeLabels {
		// Both include and exclude: use "and" since both target the "labels" key
		filter["and"] = []interface{}{
			map[string]interface{}{
				"labels": map[string]interface{}{
					"some": map[string]interface{}{
						"id": map[string]interface{}{"in": filters.LabelIDs},
					},
				},
			},
			map[string]interface{}{
				"labels": map[string]interface{}{
					"every": map[string]interface{}{
						"id": map[string]interface{}{"nin": filters.ExcludeLabelIDs},
					},
				},
			},
		}
	} else if hasIncludeLabels {
		filter["labels"] = map[string]interface{}{
			"some": map[string]interface{}{
				"id": map[string]interface{}{"in": filters.LabelIDs},
			},
		}
	} else if hasExcludeLabels {
		filter["labels"] = map[string]interface{}{
			"every": map[string]interface{}{
				"id": map[string]interface{}{"nin": filters.ExcludeLabelIDs},
			},
		}
	}
	
	// Assignee filter
	if filters.AssigneeID != "" {
		filter["assignee"] = map[string]interface{}{
			"id": map[string]interface{}{
				"eq": filters.AssigneeID,
			},
		}
	}
	
	// Priority filter
	if filters.Priority != nil {
		filter["priority"] = map[string]interface{}{
			"eq": *filters.Priority,
		}
	}

	// Project filter
	if filters.ProjectID != "" {
		filter["project"] = map[string]interface{}{
			"id": map[string]interface{}{
				"eq": filters.ProjectID,
			},
		}
	}

	// Cycle filter
	if filters.CycleID != "" {
		filter["cycle"] = map[string]interface{}{
			"id": map[string]interface{}{
				"eq": filters.CycleID,
			},
		}
	}

	// Search term
	if filters.SearchTerm != "" {
		filter["searchableContent"] = map[string]interface{}{
			"contains": filters.SearchTerm,
		}
	}

	// Date filters
	if filters.CreatedAfter != "" {
		if _, ok := filter["createdAt"]; !ok {
			filter["createdAt"] = make(map[string]interface{})
		}
		filter["createdAt"].(map[string]interface{})["gte"] = filters.CreatedAfter
	}
	if filters.CreatedBefore != "" {
		if _, ok := filter["createdAt"]; !ok {
			filter["createdAt"] = make(map[string]interface{})
		}
		filter["createdAt"].(map[string]interface{})["lte"] = filters.CreatedBefore
	}
	if filters.UpdatedAfter != "" {
		if _, ok := filter["updatedAt"]; !ok {
			filter["updatedAt"] = make(map[string]interface{})
		}
		filter["updatedAt"].(map[string]interface{})["gte"] = filters.UpdatedAfter
	}
	if filters.UpdatedBefore != "" {
		if _, ok := filter["updatedAt"]; !ok {
			filter["updatedAt"] = make(map[string]interface{})
		}
		filter["updatedAt"].(map[string]interface{})["lte"] = filters.UpdatedBefore
	}
	
	variables := map[string]interface{}{
		"first": filters.Limit,
	}

	// Only add filter if it has content
	if len(filter) > 0 {
		variables["filter"] = filter
	}

	// Add pagination cursor if provided
	if filters.After != "" {
		variables["after"] = filters.After
	}

	// Always include the includeArchived parameter (defaults to false)
	variables["includeArchived"] = filters.IncludeArchived
	
	// Add orderBy if specified
	if filters.OrderBy != "" {
		variables["orderBy"] = filters.OrderBy
	}

	var response struct {
		Issues struct {
			Nodes    []core.Issue `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"issues"`
	}
	
	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}
	
	return &core.IssueSearchResult{
		Issues:      response.Issues.Nodes,
		HasNextPage: response.Issues.PageInfo.HasNextPage,
		EndCursor:   response.Issues.PageInfo.EndCursor,
	}, nil
}

// BatchUpdateIssues updates multiple issues with the same changes
// Why: Bulk operations are common in issue management, such as moving multiple
// issues to a new state, assigning them to someone, or applying labels in bulk.
func (ic *Client) BatchUpdateIssues(issueIDs []string, update core.BatchIssueUpdate) (*core.BatchIssueUpdateResult, error) {
	if len(issueIDs) == 0 {
		return nil, fmt.Errorf("no issue IDs provided")
	}
	
	const mutation = `
		mutation BatchUpdateIssues($issueIds: [String!]!, $input: IssueUpdateInput!) {
			issueBatchUpdate(ids: $issueIds, input: $input) {
				success
				updatedIssues: issues {
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
					labels {
						nodes {
							id
							name
							color
						}
					}
					priority
					project {
						id
						name
					}
				}
			}
		}
	`
	
	// Build the update input
	input := make(map[string]interface{})
	
	if update.StateID != "" {
		input["stateId"] = update.StateID
	}
	if update.AssigneeID != "" {
		input["assigneeId"] = update.AssigneeID
	}
	if len(update.LabelIDs) > 0 {
		input["labelIds"] = update.LabelIDs
	}
	if update.Priority != nil {
		input["priority"] = *update.Priority
	}
	if update.ProjectID != "" {
		input["projectId"] = update.ProjectID
	}
	
	if len(input) == 0 {
		return nil, fmt.Errorf("no update fields provided")
	}
	
	variables := map[string]interface{}{
		"issueIds": issueIDs,
		"input":    input,
	}
	
	var response struct {
		IssueBatchUpdate core.BatchIssueUpdateResult `json:"issueBatchUpdate"`
	}
	
	err := ic.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to batch update issues: %w", err)
	}
	
	return &response.IssueBatchUpdate, nil
}

// GetIssueWithBestContext retrieves an issue with the most appropriate context
// Why: This intelligently determines whether to fetch parent or project context
// based on the issue's actual relationships, avoiding unnecessary API calls.
func (ic *Client) GetIssueWithBestContext(issueID string) (*core.Issue, error) {
	// Validate input
	if issueID == "" {
		return nil, &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	
	// First, get basic issue info to determine what context to fetch
	issue, err := ic.GetIssue(issueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}
	
	// Determine the best context based on what the issue has
	if issue.Parent != nil && issue.Parent.ID != "" {
		// Issue has a parent - fetch with parent context for sibling information
		parentContextIssue, err := ic.GetIssueWithParentContext(issueID)
		if err != nil {
			// If parent context fails, fall back to what we already have
			return issue, nil
		}
		return parentContextIssue, nil
		
	} else if issue.Project != nil && issue.Project.ID != "" {
		// Issue has a project but no parent - fetch with project context
		projectContextIssue, err := ic.GetIssueWithProjectContext(issueID)
		if err != nil {
			// If project context fails, fall back to what we already have
			return issue, nil
		}
		return projectContextIssue, nil
	}
	
	// Standalone issue - we already have all the data we need
	return issue, nil
}

// GetSubIssues retrieves all sub-issues for a given parent issue
// Why: Sub-task management requires fetching all children of an issue.
// This dedicated method provides that functionality efficiently.
func (ic *Client) GetSubIssues(parentIssueID string) ([]core.SubIssue, error) {
	if parentIssueID == "" {
		return nil, &core.ValidationError{Field: "parentIssueID", Message: "parentIssueID cannot be empty"}
	}
	
	const query = `
		query GetSubIssues($id: String!) {
			issue(id: $id) {
				children {
					nodes {
						id
						identifier
						title
						state {
							id
							name
						}
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"id": parentIssueID,
	}
	
	var response struct {
		Issue struct {
			Children struct {
				Nodes []core.SubIssue `json:"nodes"`
			} `json:"children"`
		} `json:"issue"`
	}
	
	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-issues: %w", err)
	}
	
	return response.Issue.Children.Nodes, nil
}

// UpdateIssueDescription updates an issue's description while preserving metadata
// Why: Descriptions may contain both user content and hidden metadata. This method
// ensures metadata is preserved when users update descriptions.
func (ic *Client) UpdateIssueDescription(issueID, newDescription string) error {
	if issueID == "" {
		return &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	
	// First, get the current issue to preserve metadata
	// Why: We need to extract existing metadata before updating the description
	// to ensure we don't lose any stored metadata during the update.
	issue, err := ic.GetIssue(issueID)
	if err != nil {
		return fmt.Errorf("failed to get current issue: %w", err)
	}
	
	// Preserve existing metadata
	// Why: The issue.Metadata field contains the extracted metadata from the
	// current description. We need to inject this back into the new description.
	descriptionWithMetadata := newDescription
	if issue.Metadata != nil && len(issue.Metadata) > 0 {
		descriptionWithMetadata = metadata.InjectMetadataIntoDescription(newDescription, issue.Metadata)
	}
	
	const mutation = `
		mutation UpdateIssueDescription($issueId: String!, $description: String!) {
			issueUpdate(
				id: $issueId,
				input: { description: $description }
			) {
				success
			}
		}
	`
	
	variables := map[string]interface{}{
		"issueId":     issueID,
		"description": descriptionWithMetadata,
	}
	
	var response struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}
	
	err = ic.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to update issue description: %w", err)
	}
	
	if !response.IssueUpdate.Success {
		return fmt.Errorf("issue description update was not successful")
	}
	
	return nil
}

// UpdateIssueMetadataKey updates a specific metadata key for an issue
// Why: Granular metadata updates are more efficient than replacing all metadata.
// This method allows updating individual keys without affecting others.
func (ic *Client) UpdateIssueMetadataKey(issueID, key string, value interface{}) error {
	if issueID == "" {
		return &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	if key == "" {
		return &core.ValidationError{Field: "key", Message: "key cannot be empty"}
	}
	if !validation.IsValidMetadataKey(key) {
		return &core.ValidationError{Field: "key", Value: key, Reason: "must be alphanumeric with underscores or hyphens, starting with letter or underscore"}
	}

	// Get current issue to access existing metadata
	// Why: We need to merge the new key-value with existing metadata
	// to avoid losing other metadata entries during the update.
	issue, err := ic.GetIssue(issueID)
	if err != nil {
		return fmt.Errorf("failed to get current issue: %w", err)
	}
	
	// Initialize metadata if needed and update the key
	// Why: The issue might not have any metadata yet. We initialize
	// it as an empty map if needed before adding the new key.
	if issue.Metadata == nil {
		issue.Metadata = make(map[string]interface{})
	}
	issue.Metadata[key] = value
	
	// Update the description with new metadata
	// Why: Metadata is stored in the description field. We need to
	// inject the updated metadata back into the description.
	descriptionWithMetadata := metadata.InjectMetadataIntoDescription(issue.Description, issue.Metadata)
	
	const mutation = `
		mutation UpdateIssueDescription($issueId: String!, $description: String!) {
			issueUpdate(
				id: $issueId,
				input: { description: $description }
			) {
				success
			}
		}
	`
	
	variables := map[string]interface{}{
		"issueId":     issueID,
		"description": descriptionWithMetadata,
	}
	
	var response struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}
	
	err = ic.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to update issue metadata: %w", err)
	}
	
	if !response.IssueUpdate.Success {
		return fmt.Errorf("issue metadata update was not successful")
	}
	
	return nil
}

// RemoveIssueMetadataKey removes a specific metadata key from an issue
// Why: Sometimes metadata keys become obsolete or need to be cleaned up.
// This method provides that capability without affecting other metadata.
func (ic *Client) RemoveIssueMetadataKey(issueID, key string) error {
	if issueID == "" {
		return &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	if key == "" {
		return &core.ValidationError{Field: "key", Message: "key cannot be empty"}
	}
	
	// Get current issue
	issue, err := ic.GetIssue(issueID)
	if err != nil {
		return fmt.Errorf("failed to get current issue: %w", err)
	}
	
	// Remove the key if metadata exists
	// Why: We only proceed if there's metadata and the key exists.
	// No need to update if there's nothing to remove.
	if issue.Metadata == nil || len(issue.Metadata) == 0 {
		// No metadata to remove from, nothing to do
		return nil
	}

	// Check if key exists before attempting removal
	if _, exists := issue.Metadata[key]; !exists {
		// Key doesn't exist, nothing to do
		return nil
	}

	delete(issue.Metadata, key)

	// Update description with modified metadata
	// Why: After removing the key, we need to update the description
	// with the remaining metadata, or remove metadata entirely if empty.
	var descriptionWithMetadata string
	if len(issue.Metadata) > 0 {
		descriptionWithMetadata = metadata.InjectMetadataIntoDescription(issue.Description, issue.Metadata)
	} else {
		// No metadata left, just use the clean description
		descriptionWithMetadata = issue.Description
	}

	const mutation = `
		mutation UpdateIssueDescription($issueId: String!, $description: String!) {
			issueUpdate(
				id: $issueId,
				input: { description: $description }
			) {
				success
			}
		}
	`

	variables := map[string]interface{}{
		"issueId":     issueID,
		"description": descriptionWithMetadata,
	}

	var response struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}

	err = ic.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to update issue description: %w", err)
	}

	if !response.IssueUpdate.Success {
		return fmt.Errorf("issue metadata removal was not successful")
	}

	return nil
}

// GetIssueSimplified retrieves a single issue by ID with reduced query complexity
// Why: The full GetIssue query can sometimes hit server complexity limits for
// issues with many children or complex data. This simplified version excludes
// children nodes to reduce query complexity.
func (ic *Client) GetIssueSimplified(issueID string) (*core.Issue, error) {
	if issueID == "" {
		return nil, &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	
	const query = `
		query GetIssueSimplified($id: String!) {
			issue(id: $id) {
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
				delegate {
					id
					name
					email
				}
				priority
				estimate
				dueDate
				labels {
					nodes {
						id
						name
						color
					}
				}
				cycle {
					id
					number
					name
				}
				createdAt
				updatedAt
				url
				project {
					id
					name
				}
				parent {
					id
					identifier
					title
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"id": issueID,
	}
	
	var response struct {
		Issue core.Issue `json:"issue"`
	}
	
	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue (simplified): %w", err)
	}
	
	// Extract metadata from description
	if response.Issue.Description != "" {
		metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.Issue.Description)
		response.Issue.Metadata = metadata
		response.Issue.Description = cleanDesc
	}
	
	// Initialize empty children to maintain consistency
	response.Issue.Children.Nodes = []core.SubIssue{}
	
	return &response.Issue, nil
}

// GetIssueWithFallback attempts to get an issue with full details, falling back
// to simplified query if the full query fails with a server error.
// Why: This provides resilience against server-side complexity limits while
// still attempting to get full data when possible.
func (ic *Client) GetIssueWithFallback(issueID string) (*core.Issue, error) {
	// First try the full query
	issue, err := ic.GetIssue(issueID)
	if err == nil {
		return issue, nil
	}
	
	// Check if it's a server error (500) or complexity error
	// Need to unwrap the error to check for HTTPError
	var httpErr *core.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode >= 500 {
		// Try simplified query
		return ic.GetIssueSimplified(issueID)
	}
	
	// For other errors, return the original error
	return nil, err
}

// UpdateIssue updates an issue with the provided fields
// Why: Issues need to be updated with various fields like title, description, priority, etc.
// This method provides a flexible way to update any combination of fields while preserving
// existing data like metadata.
func (ic *Client) UpdateIssue(issueID string, input core.UpdateIssueInput) (*core.Issue, error) {
	// Validate inputs
	if issueID == "" {
		return nil, &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	
	// Check if there are any fields to update
	if !hasFieldsToUpdate(input) {
		return nil, &core.ValidationError{Field: "input", Message: "no fields to update"}
	}
	
	// Validate priority if provided
	if input.Priority != nil && (*input.Priority < 0 || *input.Priority > 4) {
		return nil, &core.ValidationError{Field: "priority", Message: fmt.Sprintf("invalid priority value: %d (must be between 0-4)", *input.Priority)}
	}
	
	// If updating description, preserve existing metadata
	if input.Description != nil {
		issue, err := ic.GetIssue(issueID)
		if err != nil {
			return nil, fmt.Errorf("failed to get current issue for metadata preservation: %w", err)
		}
		
		// Preserve metadata in the new description
		if issue.Metadata != nil && len(issue.Metadata) > 0 {
			descWithMetadata := metadata.InjectMetadataIntoDescription(*input.Description, issue.Metadata)
			input.Description = &descWithMetadata
		}
	}
	
	// Build the GraphQL mutation
	const mutation = `
		mutation UpdateIssue($issueId: String!, $input: IssueUpdateInput!) {
			issueUpdate(
				id: $issueId,
				input: $input
			) {
				success
				issue {
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
					delegate {
						id
						name
						email
					}
					project {
						id
						name
					}
					createdAt
					updatedAt
					url
					priority
					estimate
					dueDate
					parent {
						id
						identifier
						title
					}
					children {
						nodes {
							id
							identifier
							title
							state {
								id
								name
							}
						}
					}
				}
			}
		}
	`
	
	// Build the input object
	updateInput := buildUpdateInput(input)
	
	variables := map[string]interface{}{
		"issueId": issueID,
		"input":   updateInput,
	}
	
	var response struct {
		IssueUpdate struct {
			Success bool  `json:"success"`
			Issue   core.Issue `json:"issue"`
		} `json:"issueUpdate"`
	}
	
	err := ic.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to update issue: %w", err)
	}
	
	if !response.IssueUpdate.Success {
		return nil, fmt.Errorf("issue update was not successful")
	}
	
	// Extract metadata from description if present
	if response.IssueUpdate.Issue.Description != "" {
		metadata, cleanDesc := metadata.ExtractMetadataFromDescription(response.IssueUpdate.Issue.Description)
		response.IssueUpdate.Issue.Metadata = metadata
		response.IssueUpdate.Issue.Description = cleanDesc
	}
	
	return &response.IssueUpdate.Issue, nil
}

// hasFieldsToUpdate checks if the input has any fields to update
func hasFieldsToUpdate(input core.UpdateIssueInput) bool {
	return input.Title != nil ||
		input.Description != nil ||
		input.Priority != nil ||
		input.Estimate != nil ||
		input.DueDate != nil ||
		input.StateID != nil ||
		input.AssigneeID != nil ||
		input.DelegateID != nil ||
		input.ProjectID != nil ||
		input.ParentID != nil ||
		input.TeamID != nil ||
		input.CycleID != nil ||
		len(input.LabelIDs) > 0
}

// buildUpdateInput builds the GraphQL input object from UpdateIssueInput
func buildUpdateInput(input core.UpdateIssueInput) map[string]interface{} {
	updateInput := make(map[string]interface{})

	if input.Title != nil {
		updateInput["title"] = *input.Title
	}
	if input.Description != nil {
		updateInput["description"] = *input.Description
	}
	if input.Priority != nil {
		updateInput["priority"] = *input.Priority
	}
	if input.Estimate != nil {
		updateInput["estimate"] = *input.Estimate
	}
	if input.DueDate != nil {
		updateInput["dueDate"] = *input.DueDate
	}
	if input.StateID != nil {
		updateInput["stateId"] = *input.StateID
	}
	if input.ProjectID != nil {
		updateInput["projectId"] = *input.ProjectID
	}
	if input.ParentID != nil {
		updateInput["parentId"] = *input.ParentID
	}
	if input.TeamID != nil {
		updateInput["teamId"] = *input.TeamID
	}
	if len(input.LabelIDs) > 0 {
		updateInput["labelIds"] = input.LabelIDs
	}
	if input.CycleID != nil {
		updateInput["cycleId"] = *input.CycleID
	}

	// Handle assignee ID with nullable input (for human users)
	if input.AssigneeID != nil {
		if *input.AssigneeID == "" {
			// Unassign - use null
			updateInput["assigneeId"] = nil
		} else {
			// Assign to user - just pass the string directly
			updateInput["assigneeId"] = *input.AssigneeID
		}
	}

	// Handle delegate ID with nullable input (for OAuth applications)
	if input.DelegateID != nil {
		if *input.DelegateID == "" {
			// Remove delegation - use null
			updateInput["delegateId"] = nil
		} else {
			// Delegate to application
			updateInput["delegateId"] = *input.DelegateID
		}
	}

	return updateInput
}

// parseLinearIdentifier extracts the numeric part from a Linear issue identifier
// Why: Linear identifiers like "ENG-123" need to be parsed to extract the
// issue number for certain operations. This helper provides that functionality.
func parseLinearIdentifier(identifier string) string {
	parts := strings.Split(identifier, "-")
	if len(parts) > 1 {
		return parts[1]
	}
	return identifier
}

// ListAllIssues retrieves issues with comprehensive filtering, pagination, and sorting options
// Why: Users need flexible ways to query issues across teams, projects, states, etc.
// This method provides a powerful search interface with metadata support.
func (ic *Client) ListAllIssues(filter *core.IssueFilter) (*core.ListAllIssuesResult, error) {
	// Validate required fields
	if filter == nil {
		return nil, &core.ValidationError{Field: "filter", Message: "filter cannot be nil"}
	}
	if filter.First <= 0 {
		filter.First = 10 // Reduced from 50 to minimize token usage
	}
	if filter.First > 250 {
		filter.First = 250 // Cap at Linear's maximum
	}

	const query = `
		query ListAllIssues($first: Int!, $after: String, $filter: IssueFilter, $orderBy: PaginationOrderBy) {
			issues(first: $first, after: $after, filter: $filter, orderBy: $orderBy) {
				nodes {
					id
					identifier
					title
					description
					priority
					createdAt
					updatedAt
					state {
						id
						name
						type
						color
						position
						description
						team {
							id
							name
							key
						}
					}
					assignee {
						id
						name
						displayName
						email
						avatarUrl
						active
						createdAt
						isMe
					}
					labels {
						nodes {
							id
							name
							color
						}
					}
					project {
						id
						name
						description
						state
						createdAt
						updatedAt
					}
					team {
						id
						name
						key
						description
					}
				}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	`

	variables := map[string]interface{}{
		"first": filter.First,
	}
	if filter.After != "" {
		variables["after"] = filter.After
	}

	// Build filter object if any filters are specified
	if hasFilters(filter) {
		variables["filter"] = buildFilterObject(filter)
	}

	// Add orderBy if specified
	if filter.OrderBy != "" {
		orderByObj := buildOrderByObject(filter.OrderBy, filter.Direction)
		variables["orderBy"] = orderByObj
	}

	var response struct {
		Issues struct {
			Nodes []struct {
				ID          string        `json:"id"`
				Identifier  string        `json:"identifier"`
				Title       string        `json:"title"`
				Description string        `json:"description"`
				Priority    int           `json:"priority"`
				CreatedAt   string        `json:"createdAt"`
				UpdatedAt   string        `json:"updatedAt"`
				State       core.WorkflowState `json:"state"`
				Assignee    *core.User         `json:"assignee"`
				Labels      struct {
					Nodes []core.Label `json:"nodes"`
				} `json:"labels"`
				Project *core.Project `json:"project"`
				Team core.Team     `json:"team"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"issues"`
	}

	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list all issues: %w", err)
	}

	// Convert response to result type
	result := &core.ListAllIssuesResult{
		Issues:      make([]core.IssueWithDetails, 0, len(response.Issues.Nodes)),
		HasNextPage: response.Issues.PageInfo.HasNextPage,
		EndCursor:   response.Issues.PageInfo.EndCursor,
		TotalCount:  len(response.Issues.Nodes), // Note: Linear issues endpoint doesn't return totalCount
	}

	// Process each issue
	for _, node := range response.Issues.Nodes {
		issue := core.IssueWithDetails{
			ID:          node.ID,
			Identifier:  node.Identifier,
			Title:       node.Title,
			Description: node.Description,
			Priority:    node.Priority,
			CreatedAt:   node.CreatedAt,
			UpdatedAt:   node.UpdatedAt,
			State:       node.State,
			Assignee:    node.Assignee,
			Labels:      node.Labels.Nodes,
			Project:     node.Project,
			Team:        node.Team,
		}

		// Extract metadata from description
		if issue.Description != "" {
			metadata, cleanDesc := metadata.ExtractMetadataFromDescription(issue.Description)
			if len(metadata) > 0 {
				issue.Metadata = &metadata
			}
			issue.Description = cleanDesc
		}

		// Extract metadata from project description if present
		if issue.Project != nil && issue.Project.Description != "" {
			projectMetadata, cleanProjectDesc := metadata.ExtractMetadataFromDescription(issue.Project.Description)
			issue.Project.Metadata = projectMetadata
			issue.Project.Description = cleanProjectDesc
		}

		result.Issues = append(result.Issues, issue)
	}

	return result, nil
}

// hasFilters checks if any filters are specified
func hasFilters(filter *core.IssueFilter) bool {
	return len(filter.StateIDs) > 0 ||
		filter.AssigneeID != "" ||
		len(filter.LabelIDs) > 0 ||
		filter.ProjectID != "" ||
		filter.TeamID != ""
}

// buildFilterObject constructs the GraphQL filter object
func buildFilterObject(filter *core.IssueFilter) map[string]interface{} {
	filterObj := make(map[string]interface{})

	// State filter
	if len(filter.StateIDs) > 0 {
		filterObj["state"] = map[string]interface{}{
			"id": map[string]interface{}{
				"in": filter.StateIDs,
			},
		}
	}

	// Assignee filter
	if filter.AssigneeID != "" {
		filterObj["assignee"] = map[string]interface{}{
			"id": map[string]interface{}{
				"eq": filter.AssigneeID,
			},
		}
	}

	// Label filter
	if len(filter.LabelIDs) > 0 {
		filterObj["labels"] = map[string]interface{}{
			"id": map[string]interface{}{
				"in": filter.LabelIDs,
			},
		}
	}

	// Exclude label filter
	if len(filter.ExcludeLabelIDs) > 0 {
		excludeFilter := map[string]interface{}{
			"every": map[string]interface{}{
				"id": map[string]interface{}{"nin": filter.ExcludeLabelIDs},
			},
		}
		if existing, ok := filterObj["labels"]; ok {
			// Combine with existing include filter using "and"
			delete(filterObj, "labels")
			filterObj["and"] = []interface{}{
				map[string]interface{}{"labels": existing},
				map[string]interface{}{"labels": excludeFilter},
			}
		} else {
			filterObj["labels"] = excludeFilter
		}
	}

	// Project filter
	if filter.ProjectID != "" {
		filterObj["project"] = map[string]interface{}{
			"id": map[string]interface{}{
				"eq": filter.ProjectID,
			},
		}
	}

	// Team filter
	if filter.TeamID != "" {
		filterObj["team"] = map[string]interface{}{
			"id": map[string]interface{}{
				"eq": filter.TeamID,
			},
		}
	}

	return filterObj
}

// buildOrderByObject constructs the GraphQL orderBy object
func buildOrderByObject(field, direction string) map[string]interface{} {
	// Map user-friendly field names to Linear's GraphQL enum values
	fieldMap := map[string]string{
		"createdAt": "CreatedAt",
		"updatedAt": "UpdatedAt",
		"priority":  "PriorityDesc", // Linear uses PriorityDesc for priority sorting
	}

	graphQLField, ok := fieldMap[field]
	if !ok {
		// Default to CreatedAt if invalid field
		graphQLField = "CreatedAt"
	}

	// Map direction to GraphQL enum
	graphQLDirection := "DESCENDING" // Default
	if direction == "asc" {
		graphQLDirection = "ASCENDING"
	}

	return map[string]interface{}{
		"field":     graphQLField,
		"direction": graphQLDirection,
	}
}

// ListIssueAttachments retrieves all attachments for a specific issue
// Why: Agents need to access UI specs, mockups, and screenshots attached to issues
// for implementation guidance. This method provides comprehensive attachment metadata
// including filename, size, content type, and URLs.
func (ic *Client) ListIssueAttachments(issueID string) ([]core.Attachment, error) {
	// Validate input
	if issueID == "" {
		return nil, guidance.ValidationErrorWithExample("issueID", "cannot be empty", 
			`// First get an issue ID
issue = linear_get_issue("some-issue-id")
// Then list its attachments
attachments = linear_list_issue_attachments(issue.id)`)
	}

	const query = `
		query GetIssueAttachments($issueId: String!) {
			issue(id: $issueId) {
				attachments {
					nodes {
						id
						url
						title
						subtitle
						createdAt
						updatedAt
						archivedAt
						metadata
						source
						sourceType
						groupBySource
						creator {
							id
							name
							email
							displayName
						}
						externalUserCreator {
							id
							name
							displayName
							email
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"issueId": issueID,
	}

	var response struct {
		Issue struct {
			Attachments struct {
				Nodes []core.Attachment `json:"nodes"`
			} `json:"attachments"`
		} `json:"issue"`
	}

	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list issue attachments: %w", err)
	}

	return response.Issue.Attachments.Nodes, nil
}

// GetIssueWithRelations retrieves an issue with its blocking/blocked-by relations
// for dependency graph visualization.
func (ic *Client) GetIssueWithRelations(issueID string) (*core.IssueWithRelations, error) {
	if issueID == "" {
		return nil, &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}

	const query = `
		query GetIssueWithRelations($id: String!) {
			issue(id: $id) {
				id
				identifier
				title
				state {
					id
					name
				}
				relations {
					nodes {
						id
						type
						relatedIssue {
							id
							identifier
							title
							state {
								id
								name
							}
						}
					}
				}
				inverseRelations {
					nodes {
						id
						type
						issue {
							id
							identifier
							title
							state {
								id
								name
							}
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id": issueID,
	}

	var response struct {
		Issue core.IssueWithRelations `json:"issue"`
	}

	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue relations: %w", err)
	}

	return &response.Issue, nil
}

// CreateRelation creates a relation between two issues using Linear's native issueRelationCreate mutation.
// For "blocks" relations: issueID is the blocker, relatedIssueID is the issue being blocked.
// Both parameters accept Linear identifiers (e.g., "CEN-123") directly.
func (ic *Client) CreateRelation(issueID, relatedIssueID string, relationType core.IssueRelationType) error {
	if issueID == "" {
		return &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	if relatedIssueID == "" {
		return &core.ValidationError{Field: "relatedIssueID", Message: "relatedIssueID cannot be empty"}
	}

	const mutation = `
		mutation CreateIssueRelation($issueId: String!, $relatedIssueId: String!, $type: IssueRelationType!) {
			issueRelationCreate(input: {
				issueId: $issueId,
				relatedIssueId: $relatedIssueId,
				type: $type
			}) {
				success
			}
		}
	`

	variables := map[string]interface{}{
		"issueId":        issueID,
		"relatedIssueId": relatedIssueID,
		"type":           string(relationType),
	}

	var response struct {
		IssueRelationCreate struct {
			Success bool `json:"success"`
		} `json:"issueRelationCreate"`
	}

	err := ic.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to create issue relation: %w", err)
	}

	if !response.IssueRelationCreate.Success {
		return fmt.Errorf("issue relation creation was not successful")
	}

	return nil
}

// GetTeamIssuesWithRelations retrieves all issues for a team with their relations
// for building a complete dependency graph.
func (ic *Client) GetTeamIssuesWithRelations(teamID string, limit int) ([]core.IssueWithRelations, error) {
	if teamID == "" {
		return nil, &core.ValidationError{Field: "teamID", Message: "teamID cannot be empty"}
	}
	if limit <= 0 {
		limit = 100
	}

	const query = `
		query GetTeamIssuesWithRelations($teamId: String!, $first: Int!) {
			issues(filter: { team: { key: { eq: $teamId } } }, first: $first) {
				nodes {
					id
					identifier
					title
					state {
						id
						name
					}
					project {
						id
						name
					}
					relations {
						nodes {
							id
							type
							relatedIssue {
								id
								identifier
								title
								state {
									id
									name
								}
							}
						}
					}
					inverseRelations {
						nodes {
							id
							type
							issue {
								id
								identifier
								title
								state {
									id
									name
								}
							}
						}
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
		Issues struct {
			Nodes []core.IssueWithRelations `json:"nodes"`
		} `json:"issues"`
	}

	err := ic.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get team issues with relations: %w", err)
	}

	return response.Issues.Nodes, nil
}