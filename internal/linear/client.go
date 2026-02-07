package linear

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joa23/linear-cli/internal/config"
	"github.com/joa23/linear-cli/internal/linear/attachments"
	"github.com/joa23/linear-cli/internal/linear/comments"
	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/joa23/linear-cli/internal/linear/cycles"
	"github.com/joa23/linear-cli/internal/linear/identifiers"
	"github.com/joa23/linear-cli/internal/linear/issues"
	"github.com/joa23/linear-cli/internal/linear/projects"
	"github.com/joa23/linear-cli/internal/linear/teams"
	"github.com/joa23/linear-cli/internal/linear/users"
	"github.com/joa23/linear-cli/internal/linear/workflows"
	"github.com/joa23/linear-cli/internal/oauth"
	"github.com/joa23/linear-cli/internal/token"
)

// Client represents the main Linear API client that orchestrates all sub-clients.
// It provides a single entry point for all Linear API operations while delegating
// to specialized sub-clients for specific functionality.
type Client struct {
	// Base client with shared HTTP functionality
	base *core.BaseClient

	// Sub-clients for different domains
	Issues        *issues.Client
	Projects      *projects.Client
	Comments      *comments.Client
	Teams         *teams.Client
	Notifications *users.NotificationClient
	Workflows     *workflows.Client
	Attachments   *attachments.Client
	Cycles        *cycles.Client

	// Resolver for human-readable identifier translation
	resolver *Resolver

	// Direct access to API token for compatibility
	apiToken string

	// authMode stores how the user authenticated: "user", "agent", or "" (legacy token)
	// When "agent", "me" resolution uses delegateId instead of assigneeId
	authMode string
}

// NewClient creates a new Linear API client with all sub-clients initialized
func NewClient(apiToken string) *Client {
	return NewClientWithAuthMode(apiToken, "")
}

// NewClientWithAuthMode creates a new Linear API client with explicit auth mode.
// authMode should be "user" or "agent". Empty string defaults to email suffix detection
// for backward compatibility with existing tokens.
func NewClientWithAuthMode(apiToken string, authMode string) *Client {
	// Create the base client with shared HTTP functionality
	base := core.NewBaseClient(apiToken)

	// Initialize the main client with all sub-clients
	client := &Client{
		base:          base,
		Issues:        issues.NewClient(base),
		Projects:      projects.NewClient(base),
		Comments:      comments.NewClient(base),
		Teams:         teams.NewClient(base),
		Notifications: users.NewNotificationClient(base),
		Workflows:     workflows.NewClient(base),
		Attachments:   attachments.NewClient(base),
		Cycles:        cycles.NewClient(base),
		apiToken:      apiToken,
		authMode:      authMode,
	}

	// Initialize resolver with the client
	client.resolver = NewResolver(client)

	return client
}

// NewClientWithTokenPath creates a new Linear API client with token loading.
// It intelligently selects between static and refreshing token providers based on:
// - Whether a refresh token is available
// - Whether OAuth credentials are configured
func NewClientWithTokenPath(tokenPath string) *Client {
	storage := token.NewStorage(tokenPath)
	var provider token.TokenProvider
	var apiToken string // For backward compatibility

	// Try to load from stored token first
	if storage.TokenExists() {
		tokenData, err := storage.LoadTokenData()
		if err == nil {
			apiToken = tokenData.AccessToken

			// Check if OAuth credentials available for refresh
			cfgManager := config.NewManager("")
			cfg, _ := cfgManager.Load()

			clientID := cfg.Linear.ClientID
			clientSecret := cfg.Linear.ClientSecret

			// Also check env vars
			if clientID == "" {
				clientID = os.Getenv("LINEAR_CLIENT_ID")
			}
			if clientSecret == "" {
				clientSecret = os.Getenv("LINEAR_CLIENT_SECRET")
			}

			// If OAuth credentials available AND token has refresh capability
			if clientID != "" && clientSecret != "" && tokenData.RefreshToken != "" {
				oauthHandler := oauth.NewHandlerWithClient(clientID, clientSecret, core.GetSharedHTTPClient())
				refresherAdapter := oauth.NewRefresherAdapter(oauthHandler)
				refresher := token.NewRefresher(storage, refresherAdapter)
				provider = token.NewRefreshingProvider(refresher)
			} else {
				// No refresh capability - use static provider
				provider = token.NewStaticProvider(tokenData.AccessToken)
			}
		}
	}

	// Fall back to environment variable if no stored token
	if provider == nil {
		apiToken = os.Getenv("LINEAR_API_TOKEN")
		if apiToken == "" {
			// No token available at all
			return nil
		}
		provider = token.NewStaticProvider(apiToken)
	}

	// Create base client with provider
	base := core.NewBaseClientWithProvider(provider)

	// Initialize the main client with all sub-clients
	client := &Client{
		base:          base,
		Issues:        issues.NewClient(base),
		Projects:      projects.NewClient(base),
		Comments:      comments.NewClient(base),
		Teams:         teams.NewClient(base),
		Notifications: users.NewNotificationClient(base),
		Workflows:     workflows.NewClient(base),
		Attachments:   attachments.NewClient(base),
		Cycles:        cycles.NewClient(base),
		apiToken:      apiToken,
	}

	// Initialize resolver with the client
	client.resolver = NewResolver(client)

	return client
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
	return c.base.HTTPClient
}

// SetBase sets the base client (for testing purposes)
func (c *Client) SetBase(base *core.BaseClient) {
	c.base = base
}

// GetBase returns the base client (for testing purposes)
func (c *Client) GetBase() *core.BaseClient {
	return c.base
}

// IsAgentMode returns whether the client is authenticated as an OAuth application
// When true, "me" resolution will use delegateId instead of assigneeId
func (c *Client) IsAgentMode() bool {
	return c.authMode == "agent"
}

// GetAuthMode returns the authentication mode: "user", "agent", or "" (legacy token)
// Empty string indicates a legacy token without explicit auth mode
func (c *Client) GetAuthMode() string {
	return c.authMode
}

// TestConnection tests if the client can connect to Linear API
// Why: Users need to verify their authentication and network connectivity
// before attempting other operations.
func (c *Client) TestConnection() error {
	// Delegate to the Teams client to get viewer info as a connection test
	_, err := c.Teams.GetViewer()
	return err
}

// Sub-client accessor methods for service layer
// These provide interface-based access to sub-clients

func (c *Client) CommentClient() *comments.Client {
	return c.Comments
}

func (c *Client) WorkflowClient() *workflows.Client {
	return c.Workflows
}

func (c *Client) IssueClient() *issues.Client {
	return c.Issues
}

func (c *Client) CycleClient() *cycles.Client {
	return c.Cycles
}

func (c *Client) ProjectClient() *projects.Client {
	return c.Projects
}

func (c *Client) TeamClient() *teams.Client {
	return c.Teams
}

// Direct method delegates for backward compatibility
// These methods maintain the existing API surface while delegating to sub-clients

// Issue operations
func (c *Client) CreateIssue(title, description, teamKeyOrName string) (*core.Issue, error) {
	// Resolve team name/key to UUID if needed
	teamID := teamKeyOrName
	if !identifiers.IsUUID(teamKeyOrName) {
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
func (c *Client) GetIssue(identifierOrID string) (*core.Issue, error) {
	// Resolve identifier to UUID if needed
	// Linear's issue(id:) query accepts UUIDs but not identifiers,
	// so we use SearchIssuesEnhanced with identifier filter for identifiers
	issueID := identifierOrID
	if identifiers.IsIssueIdentifier(identifierOrID) {
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
func (c *Client) GetIssueBasic(issueID string) (*core.Issue, error) {
	return c.Issues.GetIssue(issueID)
}

// DEPRECATED: Use GetIssue() instead, which automatically determines the best context
func (c *Client) GetIssueWithProjectContext(issueID string) (*core.Issue, error) {
	return c.Issues.GetIssueWithProjectContext(issueID)
}

// DEPRECATED: Use GetIssue() instead, which automatically determines the best context
func (c *Client) GetIssueWithParentContext(issueID string) (*core.Issue, error) {
	return c.Issues.GetIssueWithParentContext(issueID)
}

func (c *Client) UpdateIssueState(identifierOrID, stateID string) error {
	// Resolve issue identifier to UUID if needed
	issueID := identifierOrID
	if identifiers.IsIssueIdentifier(identifierOrID) {
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
	if identifiers.IsIssueIdentifier(identifierOrID) {
		resolvedID, err := c.resolver.ResolveIssue(identifierOrID)
		if err != nil {
			return err
		}
		issueID = resolvedID
	}

	// Resolve assignee name/email to UUID if needed
	// Empty string is allowed for unassignment
	if assigneeNameOrEmail == "" {
		return c.Issues.AssignIssue(issueID, "")
	}

	if identifiers.IsUUID(assigneeNameOrEmail) {
		return c.Issues.AssignIssue(issueID, assigneeNameOrEmail)
	}

	resolved, err := c.resolver.ResolveUser(assigneeNameOrEmail)
	if err != nil {
		return err
	}

	// For OAuth applications, use UpdateIssue with delegateId
	if resolved.IsApplication {
		input := core.UpdateIssueInput{
			DelegateID: &resolved.ID,
		}
		_, err := c.Issues.UpdateIssue(issueID, input)
		return err
	}

	return c.Issues.AssignIssue(issueID, resolved.ID)
}

func (c *Client) ListAssignedIssues(limit int) ([]core.Issue, error) {
	if limit <= 0 {
		limit = 50 // default limit
	}
	return c.Issues.ListAssignedIssues(limit)
}

func (c *Client) GetSubIssues(parentIssueID string) ([]core.SubIssue, error) {
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
func (c *Client) GetIssueSimplified(issueID string) (*core.Issue, error) {
	return c.Issues.GetIssueSimplified(issueID)
}

func (c *Client) GetIssueWithRelations(identifier string) (*core.IssueWithRelations, error) {
	return c.Issues.GetIssueWithRelations(identifier)
}

func (c *Client) UpdateIssue(identifierOrID string, input core.UpdateIssueInput) (*core.Issue, error) {
	// Resolve issue identifier to UUID if needed
	issueID := identifierOrID
	if identifiers.IsIssueIdentifier(identifierOrID) {
		resolvedID, err := c.resolver.ResolveIssue(identifierOrID)
		if err != nil {
			return nil, err
		}
		issueID = resolvedID
	}

	// Resolve AssigneeID (name/email to UUID)
	// Use delegateId for OAuth applications, assigneeId for human users
	if input.AssigneeID != nil && *input.AssigneeID != "" && !identifiers.IsUUID(*input.AssigneeID) {
		resolved, err := c.resolver.ResolveUser(*input.AssigneeID)
		if err != nil {
			return nil, err
		}
		if resolved.IsApplication {
			input.DelegateID = &resolved.ID
			input.AssigneeID = nil // Clear assigneeId when using delegateId
		} else {
			input.AssigneeID = &resolved.ID
		}
	}

	// Resolve ParentID (identifier to UUID)
	if input.ParentID != nil && *input.ParentID != "" && identifiers.IsIssueIdentifier(*input.ParentID) {
		resolvedID, err := c.resolver.ResolveIssue(*input.ParentID)
		if err != nil {
			return nil, err
		}
		input.ParentID = &resolvedID
	}

	// Resolve TeamID (name/key to UUID)
	if input.TeamID != nil && *input.TeamID != "" && !identifiers.IsUUID(*input.TeamID) {
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

func (c *Client) ListAllIssues(filter *core.IssueFilter) (*core.ListAllIssuesResult, error) {
	return c.Issues.ListAllIssues(filter)
}

// Project operations
func (c *Client) CreateProject(name, description, teamKeyOrName string) (*core.Project, error) {
	// Resolve team name/key to UUID if needed
	teamID := teamKeyOrName
	if !identifiers.IsUUID(teamKeyOrName) {
		resolvedID, err := c.resolver.ResolveTeam(teamKeyOrName)
		if err != nil {
			return nil, err
		}
		teamID = resolvedID
	}

	return c.Projects.CreateProject(name, description, teamID)
}

func (c *Client) GetProject(projectID string) (*core.Project, error) {
	return c.Projects.GetProject(projectID)
}

func (c *Client) ListAllProjects(limit int) ([]core.Project, error) {
	if limit <= 0 {
		limit = 50 // default limit
	}
	return c.Projects.ListAllProjects(limit)
}

func (c *Client) ListByTeam(teamID string, limit int) ([]core.Project, error) {
	if limit <= 0 {
		limit = 50 // default limit
	}
	return c.Projects.ListByTeam(teamID, limit)
}

func (c *Client) ListUserProjects(userID string, limit int) ([]core.Project, error) {
	if limit <= 0 {
		limit = 50 // default limit
	}
	return c.Projects.ListUserProjects(userID, limit)
}

func (c *Client) UpdateProject(projectID string, input interface{}) (*core.Project, error) {
	// Convert interface{} to the actual type expected by Projects client
	// This is a temporary solution for Phase 1 to maintain flexibility
	return c.Projects.UpdateProject(projectID, input.(projects.UpdateProjectInput))
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
func (c *Client) GetCycle(cycleID string) (*core.Cycle, error) {
	return c.Cycles.GetCycle(cycleID)
}

func (c *Client) GetActiveCycle(teamKeyOrName string) (*core.Cycle, error) {
	// Resolve team name/key to UUID if needed
	teamID := teamKeyOrName
	if !identifiers.IsUUID(teamKeyOrName) {
		resolvedID, err := c.resolver.ResolveTeam(teamKeyOrName)
		if err != nil {
			return nil, err
		}
		teamID = resolvedID
	}
	return c.Cycles.GetActiveCycle(teamID)
}

func (c *Client) ListCycles(filter *core.CycleFilter) (*core.CycleSearchResult, error) {
	// Resolve team name/key to UUID if needed
	if filter != nil && filter.TeamID != "" && !identifiers.IsUUID(filter.TeamID) {
		resolvedID, err := c.resolver.ResolveTeam(filter.TeamID)
		if err != nil {
			return nil, err
		}
		filter.TeamID = resolvedID
	}
	return c.Cycles.ListCycles(filter)
}

func (c *Client) CreateCycle(input *core.CreateCycleInput) (*core.Cycle, error) {
	// Resolve team name/key to UUID if needed
	if input != nil && input.TeamID != "" && !identifiers.IsUUID(input.TeamID) {
		resolvedID, err := c.resolver.ResolveTeam(input.TeamID)
		if err != nil {
			return nil, err
		}
		input.TeamID = resolvedID
	}
	return c.Cycles.CreateCycle(input)
}

func (c *Client) UpdateCycle(cycleID string, input *core.UpdateCycleInput) (*core.Cycle, error) {
	return c.Cycles.UpdateCycle(cycleID, input)
}

func (c *Client) ArchiveCycle(cycleID string) error {
	return c.Cycles.ArchiveCycle(cycleID)
}

func (c *Client) GetCycleIssues(cycleID string, limit int) ([]core.Issue, error) {
	return c.Cycles.GetCycleIssues(cycleID, limit)
}

// Comment operations
func (c *Client) CreateComment(issueID, body string) (*core.Comment, error) {
	return c.Comments.CreateComment(issueID, body)
}

func (c *Client) CreateCommentReply(issueID, parentID, body string) (*core.Comment, error) {
	return c.Comments.CreateCommentReply(issueID, parentID, body)
}

func (c *Client) GetCommentWithReplies(commentID string) (*core.CommentWithReplies, error) {
	return c.Comments.GetCommentWithReplies(commentID)
}

func (c *Client) AddReaction(targetID, emoji string) error {
	return c.Comments.AddReaction(targetID, emoji)
}

// Team operations
func (c *Client) GetTeams() ([]core.Team, error) {
	return c.Teams.GetTeams()
}

func (c *Client) GetTeam(keyOrName string) (*core.Team, error) {
	// Resolve team key/name to UUID if needed
	teamID := keyOrName
	if !identifiers.IsUUID(keyOrName) {
		resolvedID, err := c.resolver.ResolveTeam(keyOrName)
		if err != nil {
			return nil, err
		}
		teamID = resolvedID
	}
	return c.Teams.GetTeam(teamID)
}

func (c *Client) GetTeamEstimateScale(keyOrName string) (*core.EstimateScale, error) {
	team, err := c.GetTeam(keyOrName)
	if err != nil {
		return nil, err
	}
	return team.GetEstimateScale(), nil
}

func (c *Client) GetViewer() (*core.User, error) {
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
func (c *Client) GetNotifications(includeRead bool, limit int) ([]core.Notification, error) {
	return c.Notifications.GetNotifications(includeRead, limit)
}

func (c *Client) MarkNotificationAsRead(notificationID string) error {
	return c.Notifications.MarkNotificationRead(notificationID)
}

// Workflow operations
func (c *Client) GetWorkflowStates(teamID string) ([]core.WorkflowState, error) {
	return c.Workflows.GetWorkflowStates(teamID)
}

func (c *Client) GetWorkflowStateByName(teamID, stateName string) (*core.WorkflowState, error) {
	return c.Workflows.GetWorkflowStateByName(teamID, stateName)
}

// User operations
func (c *Client) ListUsers(filter *core.UserFilter) ([]core.User, error) {
	return c.Teams.ListUsers(filter)
}

func (c *Client) ListUsersWithPagination(filter *core.UserFilter) (*core.ListUsersResult, error) {
	return c.Teams.ListUsersWithPagination(filter)
}

func (c *Client) GetUser(idOrEmail string) (*core.User, error) {
	// First try to resolve as email or name
	var userIDStr string
	resolved, err := c.resolver.ResolveUser(idOrEmail)
	if err != nil {
		// If resolution fails, assume it's already a UUID
		userIDStr = idOrEmail
	} else {
		userIDStr = resolved.ID
	}

	// Get user by listing with no filters and finding the matching ID
	users, err := c.ListUsers(&core.UserFilter{First: 250})
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.ID == userIDStr || user.Email == idOrEmail {
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

func (c *Client) ResolveUserIdentifier(nameOrEmail string) (*ResolvedUser, error) {
	return c.resolver.ResolveUser(nameOrEmail)
}

func (c *Client) ResolveCycleIdentifier(numberOrNameOrID string, teamID string) (string, error) {
	return c.resolver.ResolveCycle(numberOrNameOrID, teamID)
}

func (c *Client) ResolveLabelIdentifier(labelName string, teamID string) (string, error) {
	return c.resolver.ResolveLabel(labelName, teamID)
}

func (c *Client) ResolveProjectIdentifier(nameOrID string, teamID string) (string, error) {
	return c.resolver.ResolveProject(nameOrID, teamID)
}

// Issue search operations
func (c *Client) SearchIssues(filters *core.IssueSearchFilters) (*core.IssueSearchResult, error) {
	return c.Issues.SearchIssuesEnhanced(filters)
}