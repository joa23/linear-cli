package linear

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joa23/linear-cli/internal/token"
)

// Client represents the main Linear API client that orchestrates all sub-clients.
// It provides a single entry point for all Linear API operations while delegating
// to specialized sub-clients for specific functionality.
type Client struct {
	// Base client with shared HTTP functionality
	base *BaseClient

	// Sub-clients for different domains
	Issues        *IssueClient
	Projects      *ProjectClient
	Comments      *CommentClient
	Teams         *TeamClient
	Notifications *NotificationClient
	Workflows     *WorkflowClient
	Attachments   *AttachmentClient
	Cycles        *CycleClient

	// Resolver for human-readable identifier translation
	resolver *Resolver

	// Direct access to API token for compatibility
	apiToken string
}

// NewClient creates a new Linear API client with all sub-clients initialized
func NewClient(apiToken string) *Client {
	// Create the base client with shared HTTP functionality
	base := NewBaseClient(apiToken)

	// Initialize the main client with all sub-clients
	client := &Client{
		base:          base,
		Issues:        NewIssueClient(base),
		Projects:      NewProjectClient(base),
		Comments:      NewCommentClient(base),
		Teams:         NewTeamClient(base),
		Notifications: NewNotificationClient(base),
		Workflows:     NewWorkflowClient(base),
		Attachments:   NewAttachmentClient(base),
		Cycles:        NewCycleClient(base),
		apiToken:      apiToken,
	}

	// Initialize resolver with the client
	client.resolver = NewResolver(client)

	return client
}

// NewClientWithTokenPath creates a new Linear API client with token loading
// It first tries to load token from the specified path, then falls back to env var
func NewClientWithTokenPath(tokenPath string) *Client {
	var apiToken string
	
	// Try to load from stored token first
	// Why: Users may have authenticated via OAuth and stored their token.
	// This provides persistence across sessions.
	storage := token.NewStorage(tokenPath)
	if storage.TokenExists() {
		if loadedToken, err := storage.LoadToken(); err == nil {
			apiToken = loadedToken
		}
	}
	
	// Fall back to environment variable if no stored token
	// Why: Environment variables are a common way to provide API tokens,
	// especially in CI/CD environments or containerized deployments.
	if apiToken == "" {
		apiToken = os.Getenv("LINEAR_API_TOKEN")
	}
	
	return NewClient(apiToken)
}

// NewDefaultClient creates a new Linear API client using default token path
func NewDefaultClient() *Client {
	return NewClientWithTokenPath(token.GetDefaultTokenPath())
}

// GetAPIToken returns the current API token
// Why: Some operations may need direct access to the token,
// such as checking authentication status.
func (c *Client) GetAPIToken() string {
	return c.apiToken
}

// GetHTTPClient returns the underlying HTTP client for testing purposes
func (c *Client) GetHTTPClient() *http.Client {
	return c.base.httpClient
}

// SetBase sets the base client (for testing purposes)
func (c *Client) SetBase(base *BaseClient) {
	c.base = base
}

// GetBase returns the base client (for testing purposes)
func (c *Client) GetBase() *BaseClient {
	return c.base
}

// TestConnection tests if the client can connect to Linear API
// Why: Users need to verify their authentication and network connectivity
// before attempting other operations.
func (c *Client) TestConnection() error {
	// Delegate to the Teams client to get viewer info as a connection test
	_, err := c.Teams.GetViewer()
	return err
}

// Direct method delegates for backward compatibility
// These methods maintain the existing API surface while delegating to sub-clients

// Issue operations
func (c *Client) CreateIssue(title, description, teamKeyOrName string) (*Issue, error) {
	// Resolve team name/key to UUID if needed
	teamID := teamKeyOrName
	if !isUUID(teamKeyOrName) {
		resolvedID, err := c.resolver.ResolveTeam(teamKeyOrName)
		if err != nil {
			return nil, err
		}
		teamID = resolvedID
	}

	return c.Issues.CreateIssue(title, description, teamID)
}

// GetIssue retrieves an issue with the best context automatically determined
// This is the preferred method for getting issues as it intelligently chooses
// whether to include parent or project context based on the issue's relationships.
func (c *Client) GetIssue(identifierOrID string) (*Issue, error) {
	// Resolve identifier to UUID if needed
	// Linear's issue(id:) query accepts UUIDs but not identifiers,
	// so we use SearchIssuesEnhanced with identifier filter for identifiers
	issueID := identifierOrID
	if isIssueIdentifier(identifierOrID) {
		resolvedID, err := c.resolver.ResolveIssue(identifierOrID)
		if err != nil {
			return nil, err
		}
		issueID = resolvedID
	}

	return c.Issues.GetIssueWithBestContext(issueID)
}

// GetIssueBasic retrieves basic issue information without additional context
// Use this when you only need basic issue data without parent/project details.
func (c *Client) GetIssueBasic(issueID string) (*Issue, error) {
	return c.Issues.GetIssue(issueID)
}

// DEPRECATED: Use GetIssue() instead, which automatically determines the best context
func (c *Client) GetIssueWithProjectContext(issueID string) (*Issue, error) {
	return c.Issues.GetIssueWithProjectContext(issueID)
}

// DEPRECATED: Use GetIssue() instead, which automatically determines the best context
func (c *Client) GetIssueWithParentContext(issueID string) (*Issue, error) {
	return c.Issues.GetIssueWithParentContext(issueID)
}

func (c *Client) UpdateIssueState(identifierOrID, stateID string) error {
	// Resolve issue identifier to UUID if needed
	issueID := identifierOrID
	if isIssueIdentifier(identifierOrID) {
		resolvedID, err := c.resolver.ResolveIssue(identifierOrID)
		if err != nil {
			return err
		}
		issueID = resolvedID
	}

	return c.Issues.UpdateIssueState(issueID, stateID)
}

func (c *Client) AssignIssue(identifierOrID, assigneeNameOrEmail string) error {
	// Resolve issue identifier to UUID if needed
	issueID := identifierOrID
	if isIssueIdentifier(identifierOrID) {
		resolvedID, err := c.resolver.ResolveIssue(identifierOrID)
		if err != nil {
			return err
		}
		issueID = resolvedID
	}

	// Resolve assignee name/email to UUID if needed
	// Empty string is allowed for unassignment
	assigneeID := assigneeNameOrEmail
	if assigneeNameOrEmail != "" && !isUUID(assigneeNameOrEmail) {
		resolvedID, err := c.resolver.ResolveUser(assigneeNameOrEmail)
		if err != nil {
			return err
		}
		assigneeID = resolvedID
	}

	return c.Issues.AssignIssue(issueID, assigneeID)
}

func (c *Client) ListAssignedIssues(userID string) ([]Issue, error) {
	// Validate userID even though the new implementation uses the authenticated user
	// This maintains backward compatibility with validation expectations
	if userID == "" {
		return nil, &ValidationError{Field: "userID", Value: userID, Message: "cannot be empty"}
	}
	// Use default limit of 50 since the wrapper doesn't expose limit parameter
	return c.Issues.ListAssignedIssues(50)
}

func (c *Client) GetSubIssues(parentIssueID string) ([]SubIssue, error) {
	return c.Issues.GetSubIssues(parentIssueID)
}

func (c *Client) UpdateIssueDescription(issueID, newDescription string) error {
	return c.Issues.UpdateIssueDescription(issueID, newDescription)
}

func (c *Client) UpdateIssueMetadataKey(issueID, key string, value interface{}) error {
	return c.Issues.UpdateIssueMetadataKey(issueID, key, value)
}

func (c *Client) RemoveIssueMetadataKey(issueID, key string) error {
	return c.Issues.RemoveIssueMetadataKey(issueID, key)
}

// GetIssueSimplified retrieves basic issue information using a simplified query
// Use this as a fallback when the full context queries fail due to server issues.
func (c *Client) GetIssueSimplified(issueID string) (*Issue, error) {
	return c.Issues.GetIssueSimplified(issueID)
}

func (c *Client) UpdateIssue(identifierOrID string, input UpdateIssueInput) (*Issue, error) {
	// Resolve issue identifier to UUID if needed
	issueID := identifierOrID
	if isIssueIdentifier(identifierOrID) {
		resolvedID, err := c.resolver.ResolveIssue(identifierOrID)
		if err != nil {
			return nil, err
		}
		issueID = resolvedID
	}

	// Resolve AssigneeID (name/email to UUID)
	if input.AssigneeID != nil && *input.AssigneeID != "" && !isUUID(*input.AssigneeID) {
		resolvedID, err := c.resolver.ResolveUser(*input.AssigneeID)
		if err != nil {
			return nil, err
		}
		input.AssigneeID = &resolvedID
	}

	// Resolve ParentID (identifier to UUID)
	if input.ParentID != nil && *input.ParentID != "" && isIssueIdentifier(*input.ParentID) {
		resolvedID, err := c.resolver.ResolveIssue(*input.ParentID)
		if err != nil {
			return nil, err
		}
		input.ParentID = &resolvedID
	}

	// Resolve TeamID (name/key to UUID)
	if input.TeamID != nil && *input.TeamID != "" && !isUUID(*input.TeamID) {
		resolvedID, err := c.resolver.ResolveTeam(*input.TeamID)
		if err != nil {
			return nil, err
		}
		input.TeamID = &resolvedID
	}

	// Note: StateID resolution would require knowing the team ID
	// We'll handle this separately in AssignIssue and UpdateIssueState methods

	return c.Issues.UpdateIssue(issueID, input)
}

func (c *Client) ListAllIssues(filter *IssueFilter) (*ListAllIssuesResult, error) {
	return c.Issues.ListAllIssues(filter)
}

// Project operations
func (c *Client) CreateProject(name, description, teamKeyOrName string) (*Project, error) {
	// Resolve team name/key to UUID if needed
	teamID := teamKeyOrName
	if !isUUID(teamKeyOrName) {
		resolvedID, err := c.resolver.ResolveTeam(teamKeyOrName)
		if err != nil {
			return nil, err
		}
		teamID = resolvedID
	}

	return c.Projects.CreateProject(name, description, teamID)
}

func (c *Client) GetProject(projectID string) (*Project, error) {
	return c.Projects.GetProject(projectID)
}

func (c *Client) ListAllProjects() ([]Project, error) {
	return c.Projects.ListAllProjects(0) // 0 means use default limit
}

func (c *Client) ListUserProjects(userID string) ([]Project, error) {
	return c.Projects.ListUserProjects(userID, 0) // 0 means use default limit
}

func (c *Client) UpdateProjectState(projectID, state string) error {
	return c.Projects.UpdateProjectState(projectID, state)
}

func (c *Client) UpdateProjectDescription(projectID, newDescription string) error {
	return c.Projects.UpdateProjectDescription(projectID, newDescription)
}

func (c *Client) UpdateProjectMetadataKey(projectID, key string, value interface{}) error {
	return c.Projects.UpdateProjectMetadataKey(projectID, key, value)
}

func (c *Client) RemoveProjectMetadataKey(projectID, key string) error {
	return c.Projects.RemoveProjectMetadataKey(projectID, key)
}

// Cycle operations
func (c *Client) GetCycle(cycleID string) (*Cycle, error) {
	return c.Cycles.GetCycle(cycleID)
}

func (c *Client) GetActiveCycle(teamKeyOrName string) (*Cycle, error) {
	// Resolve team name/key to UUID if needed
	teamID := teamKeyOrName
	if !isUUID(teamKeyOrName) {
		resolvedID, err := c.resolver.ResolveTeam(teamKeyOrName)
		if err != nil {
			return nil, err
		}
		teamID = resolvedID
	}
	return c.Cycles.GetActiveCycle(teamID)
}

func (c *Client) ListCycles(filter *CycleFilter) (*CycleSearchResult, error) {
	// Resolve team name/key to UUID if needed
	if filter != nil && filter.TeamID != "" && !isUUID(filter.TeamID) {
		resolvedID, err := c.resolver.ResolveTeam(filter.TeamID)
		if err != nil {
			return nil, err
		}
		filter.TeamID = resolvedID
	}
	return c.Cycles.ListCycles(filter)
}

func (c *Client) CreateCycle(input *CreateCycleInput) (*Cycle, error) {
	// Resolve team name/key to UUID if needed
	if input != nil && input.TeamID != "" && !isUUID(input.TeamID) {
		resolvedID, err := c.resolver.ResolveTeam(input.TeamID)
		if err != nil {
			return nil, err
		}
		input.TeamID = resolvedID
	}
	return c.Cycles.CreateCycle(input)
}

func (c *Client) UpdateCycle(cycleID string, input *UpdateCycleInput) (*Cycle, error) {
	return c.Cycles.UpdateCycle(cycleID, input)
}

func (c *Client) ArchiveCycle(cycleID string) error {
	return c.Cycles.ArchiveCycle(cycleID)
}

func (c *Client) GetCycleIssues(cycleID string, limit int) ([]Issue, error) {
	return c.Cycles.GetCycleIssues(cycleID, limit)
}

// Comment operations
func (c *Client) CreateComment(issueID, body string) (string, error) {
	comment, err := c.Comments.CreateComment(issueID, body)
	if err != nil {
		return "", err
	}
	return comment.ID, nil
}

func (c *Client) CreateCommentReply(issueID, parentID, body string) (string, error) {
	comment, err := c.Comments.CreateCommentReply(issueID, parentID, body)
	if err != nil {
		return "", err
	}
	return comment.ID, nil
}

func (c *Client) GetCommentWithReplies(commentID string) (*CommentWithReplies, error) {
	return c.Comments.GetCommentWithReplies(commentID)
}

func (c *Client) AddReaction(targetID, emoji string) error {
	return c.Comments.AddReaction(targetID, emoji)
}

// Team operations
func (c *Client) GetTeams() ([]Team, error) {
	return c.Teams.GetTeams()
}

func (c *Client) GetTeam(keyOrName string) (*Team, error) {
	// Resolve team key/name to UUID if needed
	teamID := keyOrName
	if !isUUID(keyOrName) {
		resolvedID, err := c.resolver.ResolveTeam(keyOrName)
		if err != nil {
			return nil, err
		}
		teamID = resolvedID
	}
	return c.Teams.GetTeam(teamID)
}

func (c *Client) GetTeamEstimateScale(keyOrName string) (*EstimateScale, error) {
	team, err := c.GetTeam(keyOrName)
	if err != nil {
		return nil, err
	}
	return team.GetEstimateScale(), nil
}

func (c *Client) GetViewer() (*User, error) {
	return c.Teams.GetViewer()
}

func (c *Client) GetAppUserID() (string, error) {
	viewer, err := c.Teams.GetViewer()
	if err != nil {
		return "", err
	}
	return viewer.ID, nil
}

// Notification operations
func (c *Client) GetNotifications(includeRead bool, limit int) ([]Notification, error) {
	return c.Notifications.GetNotifications(includeRead, limit)
}

func (c *Client) MarkNotificationAsRead(notificationID string) error {
	return c.Notifications.MarkNotificationRead(notificationID)
}

// Workflow operations
func (c *Client) GetWorkflowStates(teamID string) ([]WorkflowState, error) {
	return c.Workflows.GetWorkflowStates(teamID)
}

func (c *Client) GetWorkflowStateByName(teamID, stateName string) (*WorkflowState, error) {
	return c.Workflows.GetWorkflowStateByName(teamID, stateName)
}

// User operations
func (c *Client) ListUsers(filter *UserFilter) ([]User, error) {
	return c.Teams.ListUsers(filter)
}

func (c *Client) ListUsersWithPagination(filter *UserFilter) (*ListUsersResult, error) {
	return c.Teams.ListUsersWithPagination(filter)
}

func (c *Client) GetUser(idOrEmail string) (*User, error) {
	// First try to resolve as email or name
	userID, err := c.resolver.ResolveUser(idOrEmail)
	if err != nil {
		// If resolution fails, assume it's already a UUID
		userID = idOrEmail
	}

	// Get user by listing with no filters and finding the matching ID
	users, err := c.ListUsers(&UserFilter{First: 250})
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.ID == userID || user.Email == idOrEmail {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user not found: %s", idOrEmail)
}

// Resolver operations (expose resolver functionality)
func (c *Client) ResolveTeamIdentifier(keyOrName string) (string, error) {
	return c.resolver.ResolveTeam(keyOrName)
}

func (c *Client) ResolveIssueIdentifier(identifier string) (string, error) {
	return c.resolver.ResolveIssue(identifier)
}

func (c *Client) ResolveUserIdentifier(nameOrEmail string) (string, error) {
	return c.resolver.ResolveUser(nameOrEmail)
}

func (c *Client) ResolveCycleIdentifier(numberOrNameOrID string, teamID string) (string, error) {
	return c.resolver.ResolveCycle(numberOrNameOrID, teamID)
}

// Issue search operations
func (c *Client) SearchIssues(filters *IssueSearchFilters) (*IssueSearchResult, error) {
	return c.Issues.SearchIssuesEnhanced(filters)
}