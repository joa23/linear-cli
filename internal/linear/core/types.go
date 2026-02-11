package core

import (
	"encoding/json"
	"fmt"
)

// ResponseFormat specifies the level of detail in API responses
type ResponseFormat string

const (
	// FormatMinimal returns only essential fields (~50 tokens per issue)
	FormatMinimal ResponseFormat = "minimal"
	// FormatCompact returns commonly needed fields (~150 tokens per issue)
	FormatCompact ResponseFormat = "compact"
	// FormatFull returns all fields (~ 1500 tokens per issue)
	FormatFull ResponseFormat = "full"
)

// ParseResponseFormat parses a string into a ResponseFormat with validation
func ParseResponseFormat(s string) (ResponseFormat, error) {
	if s == "" {
		return FormatMinimal, nil
	}
	format := ResponseFormat(s)
	switch format {
	case FormatMinimal, FormatCompact, FormatFull:
		return format, nil
	default:
		return "", fmt.Errorf("invalid format '%s': must be 'minimal', 'compact', or 'full'", s)
	}
}

// User represents a Linear user with full information
type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	AvatarURL   string `json:"avatarUrl"`
	Active      bool   `json:"active"`
	Admin       bool   `json:"admin"`
	CreatedAt   string `json:"createdAt"`
	IsMe        bool   `json:"isMe"`
	Teams       []Team `json:"teams,omitempty"`
}

// UnmarshalJSON handles custom unmarshaling for User to support both
// direct teams array and nested teams.nodes structure from GraphQL
func (u *User) UnmarshalJSON(data []byte) error {
	type userAlias struct {
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		DisplayName string          `json:"displayName"`
		Email       string          `json:"email"`
		AvatarURL   string          `json:"avatarUrl"`
		Active      bool            `json:"active"`
		Admin       bool            `json:"admin"`
		CreatedAt   string          `json:"createdAt"`
		IsMe        bool            `json:"isMe"`
		Teams       json.RawMessage `json:"teams,omitempty"`
	}

	var alias userAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return fmt.Errorf("failed to unmarshal user: %w", err)
	}

	u.ID = alias.ID
	u.Name = alias.Name
	u.DisplayName = alias.DisplayName
	u.Email = alias.Email
	u.AvatarURL = alias.AvatarURL
	u.Active = alias.Active
	u.Admin = alias.Admin
	u.CreatedAt = alias.CreatedAt
	u.IsMe = alias.IsMe

	// Handle teams - can be either direct array or nested object with nodes
	if len(alias.Teams) > 0 {
		var teams []Team
		// Try unmarshaling as direct array first
		if err := json.Unmarshal(alias.Teams, &teams); err == nil {
			u.Teams = teams
		} else {
			// Try unmarshaling as object with nodes field
			var teamsConnection struct {
				Nodes []Team `json:"nodes"`
			}
			if err := json.Unmarshal(alias.Teams, &teamsConnection); err == nil {
				u.Teams = teamsConnection.Nodes
			} else {
				// If neither works, return the original error
				return fmt.Errorf("failed to decode response data: %w", err)
			}
		}
	}

	return nil
}

// Team represents a Linear team
type Team struct {
	ID                         string   `json:"id"`
	Name                       string   `json:"name"`
	Key                        string   `json:"key"`
	Description                string   `json:"description"`
	IssueEstimationType        string   `json:"issueEstimationType,omitempty"`        // notUsed, exponential, fibonacci, linear, tShirt
	IssueEstimationAllowZero   bool     `json:"issueEstimationAllowZero,omitempty"`   // Whether 0 is allowed as estimate
	IssueEstimationExtended    bool     `json:"issueEstimationExtended,omitempty"`    // Whether extended estimates are enabled
	DefaultIssueEstimate       *float64 `json:"defaultIssueEstimate,omitempty"`       // Default estimate for new issues
}

// EstimateScale represents the available estimate values for a team
type EstimateScale struct {
	Type            string    `json:"type"`            // notUsed, exponential, fibonacci, linear, tShirt
	AllowZero       bool      `json:"allowZero"`       // Whether 0 is allowed
	Extended        bool      `json:"extended"`        // Whether extended values are enabled
	DefaultEstimate *float64  `json:"defaultEstimate"` // Default value
	Values          []float64 `json:"values"`          // Available estimate values
	Labels          []string  `json:"labels"`          // Human-readable labels (for tShirt)
}

// GetEstimateScale returns the available estimate values based on team settings
func (t *Team) GetEstimateScale() *EstimateScale {
	scale := &EstimateScale{
		Type:            t.IssueEstimationType,
		AllowZero:       t.IssueEstimationAllowZero,
		Extended:        t.IssueEstimationExtended,
		DefaultEstimate: t.DefaultIssueEstimate,
	}

	// Define values based on estimation type
	switch t.IssueEstimationType {
	case "notUsed":
		scale.Values = []float64{}
		scale.Labels = []string{}
	case "exponential":
		if t.IssueEstimationExtended {
			scale.Values = []float64{1, 2, 4, 8, 16, 32, 64}
		} else {
			scale.Values = []float64{1, 2, 4, 8, 16}
		}
	case "fibonacci":
		if t.IssueEstimationExtended {
			scale.Values = []float64{1, 2, 3, 5, 8, 13, 21}
		} else {
			scale.Values = []float64{1, 2, 3, 5, 8}
		}
	case "linear":
		if t.IssueEstimationExtended {
			scale.Values = []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		} else {
			scale.Values = []float64{1, 2, 3, 4, 5}
		}
	case "tShirt":
		// T-shirt sizes map to Fibonacci values
		if t.IssueEstimationExtended {
			scale.Values = []float64{1, 2, 3, 5, 8, 13, 21}
			scale.Labels = []string{"XS", "S", "M", "L", "XL", "XXL", "XXXL"}
		} else {
			scale.Values = []float64{1, 2, 3, 5, 8}
			scale.Labels = []string{"XS", "S", "M", "L", "XL"}
		}
	default:
		// Default to fibonacci if unknown
		scale.Values = []float64{1, 2, 3, 5, 8}
	}

	// Add 0 if allowed
	if t.IssueEstimationAllowZero && len(scale.Values) > 0 {
		scale.Values = append([]float64{0}, scale.Values...)
		if len(scale.Labels) > 0 {
			scale.Labels = append([]string{"None"}, scale.Labels...)
		}
	}

	return scale
}

// WorkflowState represents a workflow state in Linear
type WorkflowState struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"` // backlog, unstarted, started, completed, canceled
	Color       string  `json:"color"`
	Position    float64 `json:"position"`
	Description string  `json:"description"`
	Team        *Team   `json:"team,omitempty"`
}

// Issue represents a Linear issue
type Issue struct {
	ID          string                 `json:"id"`
	Identifier  string                 `json:"identifier"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	State       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"state"`
	Project  *Project               `json:"project,omitempty"`
	Creator  *User                  `json:"creator,omitempty"`
	Assignee *User                  `json:"assignee,omitempty"`
	Delegate *User                  `json:"delegate,omitempty"` // Agent user delegated to work on issue (OAuth apps)
	Parent   *ParentIssue           `json:"parent,omitempty"`
	Children ChildrenNodes          `json:"children,omitempty"`
	Cycle    *CycleReference        `json:"cycle,omitempty"`
	Labels   *LabelConnection       `json:"labels,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Priority  *int                   `json:"priority,omitempty"`
	Estimate  *float64               `json:"estimate,omitempty"`
	DueDate   *string                `json:"dueDate,omitempty"`
	CreatedAt string                 `json:"createdAt"`
	UpdatedAt string                 `json:"updatedAt"`
	URL       string                 `json:"url"`
	
	// Attachment support
	AttachmentCount int                    `json:"attachmentCount"`  // Total number of attachments
	HasAttachments  bool                   `json:"hasAttachments"`   // Computed: AttachmentCount > 0
	Attachments     *AttachmentConnection  `json:"attachments,omitempty"`
	Comments        *CommentConnection     `json:"comments,omitempty"`
}

// AttachmentConnection represents a paginated collection of attachments
type AttachmentConnection struct {
	Nodes []Attachment `json:"nodes,omitempty"`
}

// CommentConnection represents a paginated collection of comments
type CommentConnection struct {
	Nodes []Comment `json:"nodes,omitempty"`
}

// IssueMinimal represents a minimal issue with only essential fields (~50 tokens)
// Use this for efficient browsing and list operations where full details aren't needed
type IssueMinimal struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	State      struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"state"`
	ParentIdentifier string `json:"parentIdentifier,omitempty"` // Parent issue identifier (e.g., "CEN-123")
}

// IssueCompact represents a compact issue with commonly needed fields (~150 tokens)
// Use this when you need more context than minimal but not full details
type IssueCompact struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	State      struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"state"`
	Assignee  *User    `json:"assignee,omitempty"`
	Priority  *int     `json:"priority,omitempty"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
	Parent    *struct {
		ID         string `json:"id"`
		Identifier string `json:"identifier"`
		Title      string `json:"title"`
	} `json:"parent,omitempty"`
	Children []struct {
		ID         string `json:"id"`
		Identifier string `json:"identifier"`
		Title      string `json:"title"`
	} `json:"children,omitempty"`
}

// ToMinimal converts a full Issue to IssueMinimal
func (i *Issue) ToMinimal() IssueMinimal {
	minimal := IssueMinimal{
		ID:         i.ID,
		Identifier: i.Identifier,
		Title:      i.Title,
		State:      i.State,
	}

	// Include parent identifier if present
	if i.Parent != nil {
		minimal.ParentIdentifier = i.Parent.Identifier
	}

	return minimal
}

// ToCompact converts a full Issue to IssueCompact
func (i *Issue) ToCompact() IssueCompact {
	compact := IssueCompact{
		ID:         i.ID,
		Identifier: i.Identifier,
		Title:      i.Title,
		State:      i.State,
		Assignee:   i.Assignee,
		Priority:   i.Priority,
		CreatedAt:  i.CreatedAt,
		UpdatedAt:  i.UpdatedAt,
	}

	// Include parent if present
	if i.Parent != nil {
		compact.Parent = &struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			Title      string `json:"title"`
		}{
			ID:         i.Parent.ID,
			Identifier: i.Parent.Identifier,
			Title:      i.Parent.Title,
		}
	}

	// Include children if present
	if i.Children.Nodes != nil && len(i.Children.Nodes) > 0 {
		compact.Children = make([]struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			Title      string `json:"title"`
		}, len(i.Children.Nodes))

		for idx, child := range i.Children.Nodes {
			compact.Children[idx] = struct {
				ID         string `json:"id"`
				Identifier string `json:"identifier"`
				Title      string `json:"title"`
			}{
				ID:         child.ID,
				Identifier: child.Identifier,
				Title:      child.Title,
			}
		}
	}

	return compact
}

// Attachment represents a file attachment on an issue
// Based on Linear's GraphQL schema research
type Attachment struct {
	ID               string                 `json:"id"`
	URL              string                 `json:"url"`
	Title            string                 `json:"title"`
	Subtitle         string                 `json:"subtitle,omitempty"`
	Filename         string                 `json:"filename,omitempty"`        // From UploadFile if available
	ContentType      string                 `json:"contentType,omitempty"`     // From UploadFile if available  
	Size             int64                  `json:"size,omitempty"`            // From UploadFile if available
	CreatedAt        string                 `json:"createdAt"`
	UpdatedAt        string                 `json:"updatedAt"`
	ArchivedAt       *string                `json:"archivedAt,omitempty"`
	Creator          *User                  `json:"creator,omitempty"`
	ExternalCreator  *ExternalUser          `json:"externalUserCreator,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`        // Custom metadata
	Source           map[string]interface{} `json:"source,omitempty"`          // Source information
	SourceType       string                 `json:"sourceType,omitempty"`
	GroupBySource    bool                   `json:"groupBySource"`
	Issue            *Issue                 `json:"issue,omitempty"`           // Parent issue
	OriginalIssue    *Issue                 `json:"originalIssue,omitempty"`   // If moved/copied
}

// ExternalUser represents a user from external systems (like Slack)
type ExternalUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email,omitempty"`
}

// ChildrenNodes represents the children structure from GraphQL
type ChildrenNodes struct {
	Nodes []SubIssue `json:"nodes"`
}

// SubIssue represents a minimal sub-issue with its state
type SubIssue struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	State      struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"state"`
}

// ParentIssue represents a parent issue
type ParentIssue struct {
	ID          string                 `json:"id"`
	Identifier  string                 `json:"identifier"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	State       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"state"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Project represents a Linear project
type Project struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`         // Short description (255 char limit)
	Content     string                 `json:"content,omitempty"`   // Long markdown content (no limit)
	State       string                 `json:"state"`               // planned, started, completed, etc.
	Issues      *IssueConnection       `json:"issues,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
}

// IssueConnection represents the GraphQL connection for issues
type IssueConnection struct {
	Nodes []ProjectIssue `json:"nodes"`
}

// GetIssues returns the issues from a project, handling nil Issues field
func (p *Project) GetIssues() []ProjectIssue {
	if p.Issues == nil {
		return []ProjectIssue{}
	}
	return p.Issues.Nodes
}

// ProjectIssue represents a minimal issue in a project context
type ProjectIssue struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	State      struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"state"`
	Assignee *User `json:"assignee,omitempty"`
}

// Cycle represents a Linear cycle (sprint/iteration)
type Cycle struct {
	ID                         string   `json:"id"`
	Name                       string   `json:"name"`
	Number                     int      `json:"number"`
	Description                string   `json:"description,omitempty"`
	StartsAt                   string   `json:"startsAt"`
	EndsAt                     string   `json:"endsAt"`
	CompletedAt                *string  `json:"completedAt,omitempty"`
	Progress                   float64  `json:"progress"`
	Team                       *Team    `json:"team,omitempty"`
	IsActive                   bool     `json:"isActive"`
	IsFuture                   bool     `json:"isFuture"`
	IsPast                     bool     `json:"isPast"`
	IsNext                     bool     `json:"isNext"`
	IsPrevious                 bool     `json:"isPrevious"`
	ScopeHistory               []int    `json:"scopeHistory,omitempty"`
	CompletedScopeHistory      []int    `json:"completedScopeHistory,omitempty"`
	CompletedIssueCountHistory []int    `json:"completedIssueCountHistory,omitempty"`
	InProgressScopeHistory     []int    `json:"inProgressScopeHistory,omitempty"`
	IssueCountHistory          []int    `json:"issueCountHistory,omitempty"`
	CreatedAt                  string   `json:"createdAt"`
	UpdatedAt                  string   `json:"updatedAt"`
	ArchivedAt                 *string  `json:"archivedAt,omitempty"`
	AutoArchivedAt             *string  `json:"autoArchivedAt,omitempty"`
}

// CycleMinimal represents a minimal cycle (~30 tokens)
// Use this for efficient browsing and list operations
type CycleMinimal struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Number   int    `json:"number"`
	IsActive bool   `json:"isActive"`
}

// CycleCompact represents a compact cycle (~80 tokens)
// Use this when you need more context than minimal but not full details
type CycleCompact struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Number   int     `json:"number"`
	StartsAt string  `json:"startsAt"`
	EndsAt   string  `json:"endsAt"`
	Progress float64 `json:"progress"`
	IsActive bool    `json:"isActive"`
	IsFuture bool    `json:"isFuture"`
	IsPast   bool    `json:"isPast"`
}

// CycleReference represents a minimal cycle reference for Issue.Cycle field
type CycleReference struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Number int    `json:"number"`
}

// ToMinimal converts a full Cycle to CycleMinimal
func (c *Cycle) ToMinimal() CycleMinimal {
	return CycleMinimal{
		ID:       c.ID,
		Name:     c.Name,
		Number:   c.Number,
		IsActive: c.IsActive,
	}
}

// ToCompact converts a full Cycle to CycleCompact
func (c *Cycle) ToCompact() CycleCompact {
	return CycleCompact{
		ID:       c.ID,
		Name:     c.Name,
		Number:   c.Number,
		StartsAt: c.StartsAt,
		EndsAt:   c.EndsAt,
		Progress: c.Progress,
		IsActive: c.IsActive,
		IsFuture: c.IsFuture,
		IsPast:   c.IsPast,
	}
}

// CycleFilter represents filter options for listing cycles
type CycleFilter struct {
	TeamID          string         `json:"teamId,omitempty"`
	IsActive        *bool          `json:"isActive,omitempty"`
	IsFuture        *bool          `json:"isFuture,omitempty"`
	IsPast          *bool          `json:"isPast,omitempty"`
	IncludeArchived bool           `json:"includeArchived,omitempty"`
	Limit           int            `json:"limit"`
	After           string         `json:"after,omitempty"`
	Format          ResponseFormat `json:"format,omitempty"`
}

// CycleSearchResult represents the result of searching cycles with pagination
type CycleSearchResult struct {
	Cycles      []Cycle `json:"cycles"`
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   string  `json:"endCursor"`
}

// CreateCycleInput represents the input for creating a cycle
type CreateCycleInput struct {
	TeamID      string `json:"teamId"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	StartsAt    string `json:"startsAt"`
	EndsAt      string `json:"endsAt"`
}

// UpdateCycleInput represents the input for updating a cycle
type UpdateCycleInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	StartsAt    *string `json:"startsAt,omitempty"`
	EndsAt      *string `json:"endsAt,omitempty"`
	CompletedAt *string `json:"completedAt,omitempty"`
}

// Comment represents a Linear comment
type Comment struct {
	ID        string         `json:"id"`
	Body      string         `json:"body"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt string         `json:"updatedAt"`
	User      User           `json:"user"`
	Issue     CommentIssue   `json:"issue"`
	Parent    *CommentParent `json:"parent,omitempty"`
}

// CommentIssue represents the issue a comment belongs to
type CommentIssue struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
}

// CommentParent represents the parent of a comment reply
type CommentParent struct {
	ID string `json:"id"`
}

// CommentWithReplies represents a comment with all its replies
type CommentWithReplies struct {
	Comment Comment   `json:"comment"`
	Replies []Comment `json:"replies"`
}

// Notification represents a Linear notification
type Notification struct {
	ID             string                `json:"id"`
	Type           string                `json:"type"`
	CreatedAt      string                `json:"createdAt"`
	ReadAt         *string               `json:"readAt,omitempty"`
	ArchivedAt     *string               `json:"archivedAt,omitempty"`
	SnoozedUntilAt *string               `json:"snoozedUntilAt,omitempty"`
	User           *User                 `json:"user,omitempty"`
	Issue          *NotificationIssue    `json:"issue,omitempty"`
	Comment        *NotificationComment  `json:"comment,omitempty"`
}

// NotificationIssue represents issue info in a notification
type NotificationIssue struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
}

// NotificationComment represents comment info in a notification
type NotificationComment struct {
	ID   string `json:"id"`
	Body string `json:"body"`
}

// UpdateIssueInput represents the input for updating an issue
// All fields are optional to support partial updates
type UpdateIssueInput struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	Priority    *int     `json:"priority,omitempty"`    // 0 = No priority, 1 = Urgent, 2 = High, 3 = Medium, 4 = Low
	Estimate    *float64 `json:"estimate,omitempty"`    // Story points estimate
	DueDate     *string  `json:"dueDate,omitempty"`     // ISO 8601 date format
	StateID     *string  `json:"stateId,omitempty"`     // Workflow state ID
	AssigneeID  *string  `json:"assigneeId,omitempty"`  // User ID to assign to (for human users)
	DelegateID  *string  `json:"delegateId,omitempty"`  // Application ID to delegate to (for OAuth apps)
	ProjectID   *string  `json:"projectId,omitempty"`   // Project ID to move issue to
	ParentID    *string  `json:"parentId,omitempty"`    // Parent issue ID for sub-issues
	TeamID      *string  `json:"teamId,omitempty"`      // Team ID to move issue to
	CycleID     *string  `json:"cycleId,omitempty"`     // Cycle ID to move issue to
	LabelIDs    []string `json:"labelIds,omitempty"`    // Label IDs to apply
}

// IssueFilter represents filter options for listing issues
type IssueFilter struct {
	// Pagination
	First int    `json:"first"`           // Number of items to fetch (required, max 250)
	After string `json:"after,omitempty"` // Cursor for pagination

	// Filters
	StateIDs   []string `json:"stateIds,omitempty"`   // Filter by workflow state IDs
	AssigneeID string   `json:"assigneeId,omitempty"` // Filter by assignee user ID
	LabelIDs   []string `json:"labelIds,omitempty"`   // Filter by label IDs
	ExcludeLabelIDs []string `json:"excludeLabelIds,omitempty"` // Exclude issues with these label IDs
	ProjectID  string   `json:"projectId,omitempty"`  // Filter by project ID
	TeamID     string   `json:"teamId,omitempty"`     // Filter by team ID

	// Sorting
	OrderBy   string `json:"orderBy,omitempty"`   // Sort field: "createdAt", "updatedAt", "priority"
	Direction string `json:"direction,omitempty"` // Sort direction: "asc" or "desc"

	// Response format (minimal, compact, or full)
	Format ResponseFormat `json:"format,omitempty"`
}

// ListAllIssuesResult represents the result of listing all issues
type ListAllIssuesResult struct {
	Issues      []IssueWithDetails `json:"issues"`
	HasNextPage bool               `json:"hasNextPage"`
	EndCursor   string             `json:"endCursor"`
	TotalCount  int                `json:"totalCount"`
}

// IssueWithDetails represents an issue with full details including metadata
type IssueWithDetails struct {
	ID          string                  `json:"id"`
	Identifier  string                  `json:"identifier"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Priority    int                     `json:"priority"`
	CreatedAt   string                  `json:"createdAt"`
	UpdatedAt   string                  `json:"updatedAt"`
	State       WorkflowState           `json:"state"`
	Assignee    *User                   `json:"assignee,omitempty"`
	Labels      []Label                 `json:"labels"`
	Project     *Project                `json:"project,omitempty"`
	Team        Team                    `json:"team"`
	Metadata    *map[string]interface{} `json:"metadata,omitempty"`
}

// Label represents a Linear label
type Label struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// LabelConnection represents a connection to labels
type LabelConnection struct {
	Nodes []Label `json:"nodes"`
}

// UserFilter represents filter options for listing users
type UserFilter struct {
	// Filter by team membership
	TeamID string `json:"teamId,omitempty"`
	
	// Filter by active status (nil means include all)
	ActiveOnly *bool `json:"activeOnly,omitempty"`
	
	// Pagination
	First int    `json:"first"`           // Number of items to fetch (default 50, max 250)
	After string `json:"after,omitempty"` // Cursor for pagination
}

// ListUsersResult represents the result of listing users with pagination
type ListUsersResult struct {
	Users       []User `json:"users"`
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// IssueSearchFilters represents enhanced filter options for searching issues
type IssueSearchFilters struct {
	// Team filter
	TeamID string `json:"teamId,omitempty"`

	// Project filter
	ProjectID string `json:"projectId,omitempty"`

	// Identifier filter (e.g., "CEN-123")
	Identifier string `json:"identifier,omitempty"`

	// State filters
	StateIDs []string `json:"stateIds,omitempty"`

	// Label filters
	LabelIDs []string `json:"labelIds,omitempty"`
	ExcludeLabelIDs []string `json:"excludeLabelIds,omitempty"`

	// Assignee filter
	AssigneeID string `json:"assigneeId,omitempty"`

	// Cycle filter
	CycleID string `json:"cycleId,omitempty"`

	// Priority filter (0-4, where 0 is no priority, 1 is urgent, 4 is low)
	Priority *int `json:"priority,omitempty"`

	// Text search
	SearchTerm string `json:"searchTerm,omitempty"`

	// Include archived issues
	IncludeArchived bool `json:"includeArchived,omitempty"`

	// Date filters
	CreatedAfter  string `json:"createdAfter,omitempty"`
	CreatedBefore string `json:"createdBefore,omitempty"`
	UpdatedAfter  string `json:"updatedAfter,omitempty"`
	UpdatedBefore string `json:"updatedBefore,omitempty"`

	// Sorting
	OrderBy string `json:"orderBy,omitempty"` // Sort field: "createdAt", "updatedAt"

	// Pagination
	Limit int    `json:"limit"`
	After string `json:"after,omitempty"`

	// Response format (minimal, compact, or full)
	Format ResponseFormat `json:"format,omitempty"`
}

// IssueSearchResult represents the result of searching issues with pagination
type IssueSearchResult struct {
	Issues      []Issue `json:"issues"`
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   string  `json:"endCursor"`
}

// BatchIssueUpdate represents update fields for batch operations
type BatchIssueUpdate struct {
	StateID     string   `json:"stateId,omitempty"`
	AssigneeID  string   `json:"assigneeId,omitempty"`
	LabelIDs    []string `json:"labelIds,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	ProjectID   string   `json:"projectId,omitempty"`
}

// BatchIssueUpdateResult represents the result of a batch update operation
type BatchIssueUpdateResult struct {
	Success       bool    `json:"success"`
	UpdatedIssues []Issue `json:"updatedIssues"`
}

// IssueRelationType represents the type of relationship between issues
type IssueRelationType string

const (
	// RelationBlocks indicates this issue blocks another
	RelationBlocks IssueRelationType = "blocks"
	// RelationDuplicate indicates this issue is a duplicate
	RelationDuplicate IssueRelationType = "duplicate"
	// RelationRelated indicates a general relationship
	RelationRelated IssueRelationType = "related"
)

// IssueRelation represents a dependency relationship between two issues
type IssueRelation struct {
	ID           string            `json:"id"`
	Type         IssueRelationType `json:"type"`
	Issue        *IssueMinimal     `json:"issue"`
	RelatedIssue *IssueMinimal     `json:"relatedIssue"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
}

// IssueRelationConnection represents a collection of issue relations
type IssueRelationConnection struct {
	Nodes []IssueRelation `json:"nodes"`
}

// IssueWithRelations extends Issue with relation information
type IssueWithRelations struct {
	ID               string                  `json:"id"`
	Identifier       string                  `json:"identifier"`
	Title            string                  `json:"title"`
	State            struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"state"`
	Project          *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"project"`
	Relations        IssueRelationConnection `json:"relations"`
	InverseRelations IssueRelationConnection `json:"inverseRelations"`
}

// PaginationInput represents offset-based pagination parameters
type PaginationInput struct {
	Start     int    `json:"start"`     // Starting position (0-indexed)
	Limit     int    `json:"limit"`     // Number of items per page
	Sort      string `json:"sort"`      // Sort field: priority|created|updated
	Direction string `json:"direction"` // Sort direction: asc|desc
}