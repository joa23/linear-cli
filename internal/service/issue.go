package service

import (
	"fmt"
	"sort"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear/identifiers"
	paginationutil "github.com/joa23/linear-cli/internal/linear/pagination"
	"github.com/joa23/linear-cli/internal/linear/core"
)

// IssueService handles issue-related operations
type IssueService struct {
	client    IssueClientOperations
	formatter *format.Formatter
}

// NewIssueService creates a new IssueService
func NewIssueService(client IssueClientOperations, formatter *format.Formatter) *IssueService {
	return &IssueService{
		client:    client,
		formatter: formatter,
	}
}

// SearchFilters represents filters for searching issues
type SearchFilters struct {
	TeamID     string
	ProjectID  string
	AssigneeID string
	CycleID    string
	StateIDs   []string
	LabelIDs   []string
	Priority   *int
	SearchTerm string
	Limit      int
	After      string
	Format     format.Format
}

// Get retrieves a single issue by identifier (e.g., "CEN-123")
func (s *IssueService) Get(identifier string, outputFormat format.Format) (string, error) {
	issue, err := s.client.GetIssue(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to get issue %s: %w", identifier, err)
	}

	return s.formatter.Issue(issue, outputFormat), nil
}

// GetWithOutput retrieves a single issue with new renderer architecture
func (s *IssueService) GetWithOutput(identifier string, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	issue, err := s.client.GetIssue(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to get issue %s: %w", identifier, err)
	}

	return s.formatter.RenderIssue(issue, verbosity, outputType), nil
}

// Search searches for issues with the given filters
func (s *IssueService) Search(filters *SearchFilters) (string, error) {
	if filters == nil {
		filters = &SearchFilters{}
	}

	// Set defaults
	if filters.Limit <= 0 {
		filters.Limit = 10
	}
	if filters.Format == "" {
		filters.Format = format.Compact
	}

	// Build Linear API filter
	linearFilters := &core.IssueSearchFilters{
		Limit:  filters.Limit,
		After:  filters.After,
		Format: core.ResponseFormat(filters.Format),
	}

	// Resolve team identifier if provided
	if filters.TeamID != "" {
		teamID, err := s.client.ResolveTeamIdentifier(filters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve team '%s': %w", filters.TeamID, err)
		}
		linearFilters.TeamID = teamID
	}

	// Resolve project identifier if provided (name or UUID)
	if filters.ProjectID != "" {
		projectID, err := s.client.ResolveProjectIdentifier(filters.ProjectID, linearFilters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve project '%s': %w", filters.ProjectID, err)
		}
		linearFilters.ProjectID = projectID
	}

	// Resolve assignee identifier if provided
	if filters.AssigneeID != "" {
		resolved, err := s.client.ResolveUserIdentifier(filters.AssigneeID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve user '%s': %w", filters.AssigneeID, err)
		}
		linearFilters.AssigneeID = resolved.ID
	}

	// Resolve cycle identifier if provided (requires team)
	if filters.CycleID != "" {
		if linearFilters.TeamID == "" {
			return "", fmt.Errorf("teamId is required to resolve cycleId")
		}
		cycleID, err := s.client.ResolveCycleIdentifier(filters.CycleID, linearFilters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve cycle '%s': %w", filters.CycleID, err)
		}
		linearFilters.CycleID = cycleID
	}

	// Resolve state names to IDs (requires team)
	if len(filters.StateIDs) > 0 {
		if linearFilters.TeamID == "" {
			return "", fmt.Errorf("--team is required when filtering by state")
		}
		resolvedStates, err := s.resolveStateIDs(filters.StateIDs, linearFilters.TeamID)
		if err != nil {
			return "", err
		}
		linearFilters.StateIDs = resolvedStates
	}

	// Resolve label names to IDs (requires team)
	if len(filters.LabelIDs) > 0 {
		if linearFilters.TeamID == "" {
			return "", fmt.Errorf("--team is required when filtering by labels")
		}
		resolvedLabels, err := s.resolveLabelIDs(filters.LabelIDs, linearFilters.TeamID)
		if err != nil {
			return "", err
		}
		linearFilters.LabelIDs = resolvedLabels
	}

	// Copy remaining filters
	linearFilters.Priority = filters.Priority
	linearFilters.SearchTerm = filters.SearchTerm

	// Execute search
	result, err := s.client.SearchIssues(linearFilters)
	if err != nil {
		return "", fmt.Errorf("failed to search issues: %w", err)
	}

	// Format output
	pagination := &format.Pagination{
		HasNextPage: result.HasNextPage,
		EndCursor:   result.EndCursor,
	}

	return s.formatter.IssueList(result.Issues, filters.Format, pagination), nil
}

// SearchWithOutput searches for issues with new renderer architecture
func (s *IssueService) SearchWithOutput(filters *SearchFilters, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	if filters == nil {
		filters = &SearchFilters{}
	}

	// Set defaults
	if filters.Limit <= 0 {
		filters.Limit = 10
	}

	// Build Linear API filter
	linearFilters := &core.IssueSearchFilters{
		Limit: filters.Limit,
		After: filters.After,
	}

	// Resolve team identifier if provided
	if filters.TeamID != "" {
		teamID, err := s.client.ResolveTeamIdentifier(filters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve team '%s': %w", filters.TeamID, err)
		}
		linearFilters.TeamID = teamID
	}

	// Resolve project identifier if provided (name or UUID)
	if filters.ProjectID != "" {
		projectID, err := s.client.ResolveProjectIdentifier(filters.ProjectID, linearFilters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve project '%s': %w", filters.ProjectID, err)
		}
		linearFilters.ProjectID = projectID
	}

	// Resolve assignee identifier if provided
	if filters.AssigneeID != "" {
		resolved, err := s.client.ResolveUserIdentifier(filters.AssigneeID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve user '%s': %w", filters.AssigneeID, err)
		}
		linearFilters.AssigneeID = resolved.ID
	}

	// Resolve cycle identifier if provided (requires team)
	if filters.CycleID != "" {
		if linearFilters.TeamID == "" {
			return "", fmt.Errorf("teamId is required to resolve cycleId")
		}
		cycleID, err := s.client.ResolveCycleIdentifier(filters.CycleID, linearFilters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve cycle '%s': %w", filters.CycleID, err)
		}
		linearFilters.CycleID = cycleID
	}

	// Resolve state names to IDs (requires team)
	if len(filters.StateIDs) > 0 {
		if linearFilters.TeamID == "" {
			return "", fmt.Errorf("--team is required when filtering by state")
		}
		resolvedStates, err := s.resolveStateIDs(filters.StateIDs, linearFilters.TeamID)
		if err != nil {
			return "", err
		}
		linearFilters.StateIDs = resolvedStates
	}

	// Resolve label names to IDs (requires team)
	if len(filters.LabelIDs) > 0 {
		if linearFilters.TeamID == "" {
			return "", fmt.Errorf("--team is required when filtering by labels")
		}
		resolvedLabels, err := s.resolveLabelIDs(filters.LabelIDs, linearFilters.TeamID)
		if err != nil {
			return "", err
		}
		linearFilters.LabelIDs = resolvedLabels
	}

	// Copy remaining filters
	linearFilters.Priority = filters.Priority
	linearFilters.SearchTerm = filters.SearchTerm

	// Execute search
	result, err := s.client.SearchIssues(linearFilters)
	if err != nil {
		return "", fmt.Errorf("failed to search issues: %w", err)
	}

	// Format output with new renderer
	pagination := &format.Pagination{
		HasNextPage: result.HasNextPage,
		EndCursor:   result.EndCursor,
	}

	return s.formatter.RenderIssueList(result.Issues, verbosity, outputType, pagination), nil
}

// ListAssigned lists issues assigned to the current user
func (s *IssueService) ListAssigned(limit int, outputFormat format.Format) (string, error) {
	if limit <= 0 {
		limit = 10
	}

	issues, err := s.client.ListAssignedIssues(limit)
	if err != nil {
		return "", fmt.Errorf("failed to list assigned issues: %w", err)
	}

	return s.formatter.IssueList(issues, outputFormat, nil), nil
}

// ListAssignedWithPagination lists assigned issues with offset-based pagination
func (s *IssueService) ListAssignedWithPagination(pagination *core.PaginationInput) (string, error) {
	// Validate and normalize pagination
	pagination = paginationutil.ValidatePagination(pagination)

	// Get viewer ID
	viewer, err := s.client.TeamClient().GetViewer()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	// Build filter - fetch enough to cover offset + page
	filter := &core.IssueFilter{
		AssigneeID: viewer.ID,
		First:      pagination.Start + pagination.Limit, // Fetch enough to skip to offset
	}

	// Use API-level sorting if supported (createdAt, updatedAt)
	// Note: Linear's orderBy doesn't support direction, always returns desc
	// For priority or asc direction, we'll do client-side sorting
	orderBy := paginationutil.MapSortField(pagination.Sort)
	if orderBy != "" && pagination.Direction == "desc" {
		filter.OrderBy = orderBy
	}

	// Execute query
	result, err := s.client.IssueClient().ListAllIssues(filter)
	if err != nil {
		return "", fmt.Errorf("failed to list assigned issues: %w", err)
	}

	// Apply client-side sorting if needed (priority or asc direction)
	if orderBy == "" || pagination.Direction == "asc" {
		sortIssues(result.Issues, pagination.Sort, pagination.Direction)
	}

	// Slice to offset range
	totalFetched := len(result.Issues)
	start := pagination.Start
	end := start + pagination.Limit

	if start > totalFetched {
		return "No issues found.", nil
	}
	if end > totalFetched {
		end = totalFetched
	}

	pageIssues := result.Issues[start:end]

	// Convert to display format
	issues := convertIssueDetails(pageIssues)

	// Build pagination metadata
	pageResult := &format.Pagination{
		Start:       pagination.Start,
		Limit:       pagination.Limit,
		Count:       len(issues),
		HasNextPage: end < totalFetched || result.HasNextPage,
		TotalCount:  result.TotalCount,
	}

	return s.formatter.IssueList(issues, format.Compact, pageResult), nil
}

// convertIssueDetails converts IssueWithDetails to Issue for formatting
func convertIssueDetails(details []core.IssueWithDetails) []core.Issue {
	issues := make([]core.Issue, len(details))
	for i, d := range details {
		priority := d.Priority
		issues[i] = core.Issue{
			ID:          d.ID,
			Identifier:  d.Identifier,
			Title:       d.Title,
			Description: d.Description,
			State: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: d.State.ID, Name: d.State.Name},
			Priority:  &priority,
			Assignee:  d.Assignee,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return issues
}

// sortIssues sorts issues by the specified field and direction
func sortIssues(issues []core.IssueWithDetails, sortBy, direction string) {
	sort.Slice(issues, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "priority":
			less = issues[i].Priority > issues[j].Priority // Higher priority first
		case "created":
			less = issues[i].CreatedAt < issues[j].CreatedAt
		case "updated":
			less = issues[i].UpdatedAt < issues[j].UpdatedAt
		default:
			less = issues[i].UpdatedAt < issues[j].UpdatedAt
		}

		if direction == "desc" {
			return !less
		}
		return less
	})
}

// CreateIssueInput represents input for creating an issue
type CreateIssueInput struct {
	Title       string
	Description string
	TeamID      string
	StateID     string
	AssigneeID  string
	ProjectID   string
	ParentID    string
	CycleID     string
	Priority    *int
	Estimate    *float64
	DueDate     string
	LabelIDs    []string
	DependsOn   []string // Issue identifiers this issue depends on (stored in metadata)
	BlockedBy   []string // Issue identifiers that block this issue (stored in metadata)
}

// Create creates a new issue
func (s *IssueService) Create(input *CreateIssueInput) (string, error) {
	if input.Title == "" {
		return "", fmt.Errorf("title is required")
	}
	if input.TeamID == "" {
		return "", fmt.Errorf("teamId is required")
	}

	// Resolve team identifier
	teamID, err := s.client.ResolveTeamIdentifier(input.TeamID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", input.TeamID, err)
	}

	// Create the issue
	issue, err := s.client.CreateIssue(input.Title, input.Description, teamID)
	if err != nil {
		return "", fmt.Errorf("failed to create issue: %w", err)
	}

	// Update with additional fields if provided
	updateInput := core.UpdateIssueInput{}
	needsUpdate := false

	if input.StateID != "" {
		// Resolve state name to ID if needed
		stateID, err := s.resolveStateID(input.StateID, teamID)
		if err != nil {
			return "", fmt.Errorf("could not resolve state '%s': %w\n\nRun 'linear onboard' to see valid states for your teams", input.StateID, err)
		}
		updateInput.StateID = &stateID
		needsUpdate = true
	}
	if input.AssigneeID != "" {
		// Resolve user identifier
		resolved, err := s.client.ResolveUserIdentifier(input.AssigneeID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve user '%s': %w", input.AssigneeID, err)
		}
		// Use delegateId for OAuth applications, assigneeId for human users
		if resolved.IsApplication {
			updateInput.DelegateID = &resolved.ID
		} else {
			updateInput.AssigneeID = &resolved.ID
		}
		needsUpdate = true
	}
	if input.Priority != nil {
		updateInput.Priority = input.Priority
		needsUpdate = true
	}
	if input.Estimate != nil {
		updateInput.Estimate = input.Estimate
		needsUpdate = true
	}
	if input.DueDate != "" {
		updateInput.DueDate = &input.DueDate
		needsUpdate = true
	}
	if input.ParentID != "" {
		updateInput.ParentID = &input.ParentID
		needsUpdate = true
	}
	if input.ProjectID != "" {
		projectID, err := s.client.ResolveProjectIdentifier(input.ProjectID, teamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve project '%s': %w", input.ProjectID, err)
		}
		updateInput.ProjectID = &projectID
		needsUpdate = true
	}
	if input.CycleID != "" {
		// Resolve cycle identifier
		cycleID, err := s.client.ResolveCycleIdentifier(input.CycleID, teamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve cycle '%s': %w", input.CycleID, err)
		}
		updateInput.CycleID = &cycleID
		needsUpdate = true
	}
	if len(input.LabelIDs) > 0 {
		// Resolve label names to IDs
		resolvedLabelIDs := make([]string, 0, len(input.LabelIDs))
		for _, labelName := range input.LabelIDs {
			labelID, err := s.client.ResolveLabelIdentifier(labelName, teamID)
			if err != nil {
				return "", fmt.Errorf("failed to resolve label '%s': %w", labelName, err)
			}
			resolvedLabelIDs = append(resolvedLabelIDs, labelID)
		}
		updateInput.LabelIDs = resolvedLabelIDs
		needsUpdate = true
	}

	if needsUpdate {
		issue, err = s.client.UpdateIssue(issue.ID, updateInput)
		if err != nil {
			return "", fmt.Errorf("failed to update issue after creation: %w", err)
		}
	}

	// Store dependencies in metadata if provided
	if len(input.DependsOn) > 0 {
		if err := s.client.UpdateIssueMetadataKey(issue.ID, "dependencies", input.DependsOn); err != nil {
			return "", fmt.Errorf("failed to set dependencies metadata: %w", err)
		}
	}
	if len(input.BlockedBy) > 0 {
		if err := s.client.UpdateIssueMetadataKey(issue.ID, "blocked_by", input.BlockedBy); err != nil {
			return "", fmt.Errorf("failed to set blocked_by metadata: %w", err)
		}
	}

	// Re-fetch the issue to get updated state with metadata
	if len(input.DependsOn) > 0 || len(input.BlockedBy) > 0 {
		issue, err = s.client.GetIssue(issue.Identifier)
		if err != nil {
			// Not fatal - just return what we have
			return s.formatter.Issue(issue, format.Full), nil
		}
	}

	return s.formatter.Issue(issue, format.Full), nil
}

// UpdateIssueInput represents input for updating an issue
type UpdateIssueInput struct {
	Title       *string
	Description *string
	StateID     *string
	AssigneeID  *string
	ProjectID   *string
	ParentID    *string
	TeamID      *string
	CycleID     *string
	Priority    *int
	Estimate    *float64
	DueDate     *string
	LabelIDs    []string
	DependsOn   []string // Issue identifiers this issue depends on (stored in metadata)
	BlockedBy   []string // Issue identifiers that block this issue (stored in metadata)
}

// Update updates an existing issue
func (s *IssueService) Update(identifier string, input *UpdateIssueInput) (string, error) {
	// Get existing issue to get its ID
	issue, err := s.client.GetIssue(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to get issue %s: %w", identifier, err)
	}

	// Build update input
	linearInput := core.UpdateIssueInput{
		Title:       input.Title,
		Description: input.Description,
		Priority:    input.Priority,
		Estimate:    input.Estimate,
		DueDate:     input.DueDate,
	}

	// Resolve state if provided
	if input.StateID != nil {
		// Extract team key from issue identifier and resolve to team ID
		teamKey, _, err := identifiers.ParseIssueIdentifier(issue.Identifier)
		if err != nil {
			return "", fmt.Errorf("invalid issue identifier '%s': %w", issue.Identifier, err)
		}
		teamID, err := s.client.ResolveTeamIdentifier(teamKey)
		if err != nil {
			return "", fmt.Errorf("could not resolve team '%s': %w", teamKey, err)
		}

		stateID, err := s.resolveStateID(*input.StateID, teamID)
		if err != nil {
			return "", fmt.Errorf("could not resolve state '%s': %w\n\nRun 'linear onboard' to see valid states for your teams", *input.StateID, err)
		}
		linearInput.StateID = &stateID
	}

	// Resolve assignee if provided
	if input.AssigneeID != nil {
		if *input.AssigneeID == "" {
			// Empty string means unassign
			linearInput.AssigneeID = input.AssigneeID
		} else {
			resolved, err := s.client.ResolveUserIdentifier(*input.AssigneeID)
			if err != nil {
				return "", fmt.Errorf("failed to resolve user '%s': %w", *input.AssigneeID, err)
			}
			// Use delegateId for OAuth applications, assigneeId for human users
			if resolved.IsApplication {
				linearInput.DelegateID = &resolved.ID
				empty := ""
				linearInput.AssigneeID = &empty // Clear existing assignee when delegating
			} else {
				linearInput.AssigneeID = &resolved.ID
			}
		}
	}

	if input.ProjectID != nil {
		// Resolve project name/ID to UUID
		var resolveTeamID string
		if input.TeamID != nil && *input.TeamID != "" {
			resolveTeamID, err = s.client.ResolveTeamIdentifier(*input.TeamID)
			if err != nil {
				return "", fmt.Errorf("failed to resolve team '%s': %w", *input.TeamID, err)
			}
		} else {
			teamKey, _, parseErr := identifiers.ParseIssueIdentifier(issue.Identifier)
			if parseErr == nil {
				resolveTeamID, _ = s.client.ResolveTeamIdentifier(teamKey)
			}
		}
		projectID, err := s.client.ResolveProjectIdentifier(*input.ProjectID, resolveTeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve project '%s': %w", *input.ProjectID, err)
		}
		linearInput.ProjectID = &projectID
	}
	if input.ParentID != nil {
		linearInput.ParentID = input.ParentID
	}
	if input.TeamID != nil {
		teamID, err := s.client.ResolveTeamIdentifier(*input.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve team '%s': %w", *input.TeamID, err)
		}
		linearInput.TeamID = &teamID
	}
	if input.CycleID != nil {
		// Resolve team ID with proper hierarchy:
		// 1. Explicit team from input (--team flag or .linear.yaml)
		// 2. Fallback to extracting from issue identifier
		var teamID string
		var err error

		if input.TeamID != nil && *input.TeamID != "" {
			// Use explicit team (from --team flag or .linear.yaml)
			teamID, err = s.client.ResolveTeamIdentifier(*input.TeamID)
			if err != nil {
				return "", fmt.Errorf("could not resolve team '%s': %w", *input.TeamID, err)
			}
		} else {
			// Fallback: extract from issue identifier
			teamKey, _, err := identifiers.ParseIssueIdentifier(issue.Identifier)
			if err != nil {
				return "", fmt.Errorf("invalid issue identifier '%s': %w. Use --team flag or run 'linear init'", issue.Identifier, err)
			}
			teamID, err = s.client.ResolveTeamIdentifier(teamKey)
			if err != nil {
				return "", fmt.Errorf("could not resolve team '%s': %w", teamKey, err)
			}
		}

		// Now resolve cycle with team context
		cycleID, err := s.client.ResolveCycleIdentifier(*input.CycleID, teamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve cycle '%s': %w", *input.CycleID, err)
		}
		linearInput.CycleID = &cycleID
	}
	if len(input.LabelIDs) > 0 {
		// Resolve team ID for label resolution
		var teamIDForLabels string
		var err error

		if input.TeamID != nil && *input.TeamID != "" {
			teamIDForLabels, err = s.client.ResolveTeamIdentifier(*input.TeamID)
			if err != nil {
				return "", fmt.Errorf("could not resolve team '%s': %w", *input.TeamID, err)
			}
		} else {
			// Extract from issue identifier
			teamKey, _, err := identifiers.ParseIssueIdentifier(issue.Identifier)
			if err != nil {
				return "", fmt.Errorf("invalid issue identifier '%s': %w", issue.Identifier, err)
			}
			teamIDForLabels, err = s.client.ResolveTeamIdentifier(teamKey)
			if err != nil {
				return "", fmt.Errorf("could not resolve team '%s': %w", teamKey, err)
			}
		}

		// Resolve label names to IDs
		resolvedLabelIDs := make([]string, 0, len(input.LabelIDs))
		for _, labelName := range input.LabelIDs {
			labelID, err := s.client.ResolveLabelIdentifier(labelName, teamIDForLabels)
			if err != nil {
				return "", fmt.Errorf("failed to resolve label '%s': %w", labelName, err)
			}
			resolvedLabelIDs = append(resolvedLabelIDs, labelID)
		}
		linearInput.LabelIDs = resolvedLabelIDs
	}

	// Perform update
	updatedIssue, err := s.client.UpdateIssue(issue.ID, linearInput)
	if err != nil {
		return "", fmt.Errorf("failed to update issue: %w", err)
	}

	// Update dependencies metadata if provided
	if len(input.DependsOn) > 0 {
		if err := s.client.UpdateIssueMetadataKey(issue.ID, "dependencies", input.DependsOn); err != nil {
			return "", fmt.Errorf("failed to set dependencies metadata: %w", err)
		}
	}
	if len(input.BlockedBy) > 0 {
		if err := s.client.UpdateIssueMetadataKey(issue.ID, "blocked_by", input.BlockedBy); err != nil {
			return "", fmt.Errorf("failed to set blocked_by metadata: %w", err)
		}
	}

	// Re-fetch if dependencies were updated
	if len(input.DependsOn) > 0 || len(input.BlockedBy) > 0 {
		updatedIssue, err = s.client.GetIssue(updatedIssue.Identifier)
		if err != nil {
			// Not fatal - just return what we have
			return s.formatter.Issue(updatedIssue, format.Full), nil
		}
	}

	return s.formatter.Issue(updatedIssue, format.Full), nil
}

// GetComments returns comments for an issue
func (s *IssueService) GetComments(identifier string) (string, error) {
	issue, err := s.client.GetIssue(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to get issue %s: %w", identifier, err)
	}

	if issue.Comments == nil || len(issue.Comments.Nodes) == 0 {
		return "No comments found.", nil
	}

	return s.formatter.CommentList(issue.Comments.Nodes, nil), nil
}

// AddComment adds a comment to an issue
func (s *IssueService) AddComment(identifier, body string) (string, error) {
	issue, err := s.client.GetIssue(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to get issue %s: %w", identifier, err)
	}

	comment, err := s.client.CommentClient().CreateComment(issue.ID, body)
	if err != nil {
		return "", fmt.Errorf("failed to create comment: %w", err)
	}

	return s.formatter.Comment(comment), nil
}

// ReplyToComment replies to an existing comment
func (s *IssueService) ReplyToComment(issueIdentifier, parentCommentID, body string) (*core.Comment, error) {
	issue, err := s.client.GetIssue(issueIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", issueIdentifier, err)
	}

	comment, err := s.client.CommentClient().CreateCommentReply(issue.ID, parentCommentID, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create reply: %w", err)
	}

	return comment, nil
}

// AddReaction adds a reaction to an issue or comment
func (s *IssueService) AddReaction(targetID, emoji string) error {
	return s.client.CommentClient().AddReaction(targetID, emoji)
}

// GetIssueID resolves an issue identifier to its UUID
func (s *IssueService) GetIssueID(identifier string) (string, error) {
	issue, err := s.client.GetIssue(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to get issue %s: %w", identifier, err)
	}
	return issue.ID, nil
}

// resolveStateID resolves a state name to a valid state ID
func (s *IssueService) resolveStateID(stateName, teamID string) (string, error) {
	// Always resolve by name - no UUID support
	state, err := s.client.WorkflowClient().GetWorkflowStateByName(teamID, stateName)
	if err != nil {
		return "", fmt.Errorf("state '%s' not found in team workflow: %w", stateName, err)
	}
	if state == nil {
		return "", fmt.Errorf("state '%s' not found in team workflow", stateName)
	}

	return state.ID, nil
}

// resolveStateIDs resolves a list of state names to state IDs
func (s *IssueService) resolveStateIDs(stateNames []string, teamID string) ([]string, error) {
	resolved := make([]string, 0, len(stateNames))
	for _, name := range stateNames {
		id, err := s.resolveStateID(name, teamID)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, id)
	}
	return resolved, nil
}

// resolveLabelIDs resolves a list of label names to label IDs
func (s *IssueService) resolveLabelIDs(labelNames []string, teamID string) ([]string, error) {
	resolved := make([]string, 0, len(labelNames))
	for _, name := range labelNames {
		id, err := s.client.ResolveLabelIdentifier(name, teamID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve label '%s': %w", name, err)
		}
		resolved = append(resolved, id)
	}
	return resolved, nil
}
