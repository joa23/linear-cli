package service

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
)

// IssueService handles issue-related operations
type IssueService struct {
	client    *linear.Client
	formatter *format.Formatter
}

// NewIssueService creates a new IssueService
func NewIssueService(client *linear.Client, formatter *format.Formatter) *IssueService {
	return &IssueService{
		client:    client,
		formatter: formatter,
	}
}

// SearchFilters represents filters for searching issues
type SearchFilters struct {
	TeamID     string
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
	linearFilters := &linear.IssueSearchFilters{
		Limit:  filters.Limit,
		After:  filters.After,
		Format: linear.ResponseFormat(filters.Format),
	}

	// Resolve team identifier if provided
	if filters.TeamID != "" {
		teamID, err := s.client.ResolveTeamIdentifier(filters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve team '%s': %w", filters.TeamID, err)
		}
		linearFilters.TeamID = teamID
	}

	// Resolve assignee identifier if provided
	if filters.AssigneeID != "" {
		userID, err := s.client.ResolveUserIdentifier(filters.AssigneeID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve user '%s': %w", filters.AssigneeID, err)
		}
		linearFilters.AssigneeID = userID
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

	// Copy remaining filters
	linearFilters.StateIDs = filters.StateIDs
	linearFilters.LabelIDs = filters.LabelIDs
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

// ListAssigned lists issues assigned to the current user
func (s *IssueService) ListAssigned(limit int, outputFormat format.Format) (string, error) {
	if limit <= 0 {
		limit = 10
	}

	issues, err := s.client.Issues.ListAssignedIssues(limit)
	if err != nil {
		return "", fmt.Errorf("failed to list assigned issues: %w", err)
	}

	return s.formatter.IssueList(issues, outputFormat, nil), nil
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
	updateInput := linear.UpdateIssueInput{}
	needsUpdate := false

	if input.StateID != "" {
		updateInput.StateID = &input.StateID
		needsUpdate = true
	}
	if input.AssigneeID != "" {
		// Resolve user identifier
		userID, err := s.client.ResolveUserIdentifier(input.AssigneeID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve user '%s': %w", input.AssigneeID, err)
		}
		updateInput.AssigneeID = &userID
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
		updateInput.ProjectID = &input.ProjectID
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
		updateInput.LabelIDs = input.LabelIDs
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
	linearInput := linear.UpdateIssueInput{
		Title:       input.Title,
		Description: input.Description,
		Priority:    input.Priority,
		Estimate:    input.Estimate,
		DueDate:     input.DueDate,
	}

	// Resolve state if provided
	if input.StateID != nil {
		linearInput.StateID = input.StateID
	}

	// Resolve assignee if provided
	if input.AssigneeID != nil {
		if *input.AssigneeID == "" {
			// Empty string means unassign
			linearInput.AssigneeID = input.AssigneeID
		} else {
			userID, err := s.client.ResolveUserIdentifier(*input.AssigneeID)
			if err != nil {
				return "", fmt.Errorf("failed to resolve user '%s': %w", *input.AssigneeID, err)
			}
			linearInput.AssigneeID = &userID
		}
	}

	if input.ProjectID != nil {
		linearInput.ProjectID = input.ProjectID
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
		// Need team ID for cycle resolution
		teamID := ""
		if input.TeamID != nil {
			teamID, _ = s.client.ResolveTeamIdentifier(*input.TeamID)
		}
		cycleID, err := s.client.ResolveCycleIdentifier(*input.CycleID, teamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve cycle '%s': %w", *input.CycleID, err)
		}
		linearInput.CycleID = &cycleID
	}
	if len(input.LabelIDs) > 0 {
		linearInput.LabelIDs = input.LabelIDs
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

	comment, err := s.client.Comments.CreateComment(issue.ID, body)
	if err != nil {
		return "", fmt.Errorf("failed to create comment: %w", err)
	}

	return s.formatter.Comment(comment), nil
}

// ReplyToComment replies to an existing comment
func (s *IssueService) ReplyToComment(issueIdentifier, parentCommentID, body string) (*linear.Comment, error) {
	issue, err := s.client.GetIssue(issueIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", issueIdentifier, err)
	}

	comment, err := s.client.Comments.CreateCommentReply(issue.ID, parentCommentID, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create reply: %w", err)
	}

	return comment, nil
}

// AddReaction adds a reaction to an issue or comment
func (s *IssueService) AddReaction(targetID, emoji string) error {
	return s.client.Comments.AddReaction(targetID, emoji)
}

// GetIssueID resolves an issue identifier to its UUID
func (s *IssueService) GetIssueID(identifier string) (string, error) {
	issue, err := s.client.GetIssue(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to get issue %s: %w", identifier, err)
	}
	return issue.ID, nil
}
