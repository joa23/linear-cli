package service

import (
	"github.com/joa23/linear-cli/internal/linear/comments"
	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/joa23/linear-cli/internal/linear/cycles"
	"github.com/joa23/linear-cli/internal/linear/issues"
	"github.com/joa23/linear-cli/internal/linear/projects"
	"github.com/joa23/linear-cli/internal/linear/teams"
	"github.com/joa23/linear-cli/internal/linear/workflows"
)

// IssueClientOperations defines the minimal interface needed by IssueService.
// This follows the "consumer defines interface" pattern for dependency injection,
// enabling mock implementations for unit testing.
type IssueClientOperations interface {
	// Smart resolver-aware methods (kept in Phase 2)
	CreateIssue(title, description, teamKeyOrName string) (*core.Issue, error)
	GetIssue(identifierOrID string) (*core.Issue, error)
	UpdateIssue(identifierOrID string, input core.UpdateIssueInput) (*core.Issue, error)
	UpdateIssueState(identifierOrID, stateID string) error
	AssignIssue(identifierOrID, assigneeNameOrEmail string) error
	ListAssignedIssues(limit int) ([]core.Issue, error)
	SearchIssues(filters *core.IssueSearchFilters) (*core.IssueSearchResult, error)

	// Resolver operations
	ResolveTeamIdentifier(keyOrName string) (string, error)
	ResolveUserIdentifier(nameOrEmail string) (string, error)
	ResolveCycleIdentifier(numberOrNameOrID, teamID string) (string, error)

	// Metadata operations (kept in Phase 2)
	UpdateIssueMetadataKey(issueID, key string, value interface{}) error

	// Sub-client access (Phase 2 - use sub-clients directly)
	CommentClient() *comments.Client
	WorkflowClient() *workflows.Client
	IssueClient() *issues.Client
	TeamClient() *teams.Client
}

// CycleClientOperations defines the minimal interface needed by CycleService
type CycleClientOperations interface {
	// Smart resolver-aware methods (kept in Phase 2)
	ListCycles(filter *core.CycleFilter) (*core.CycleSearchResult, error)
	GetActiveCycle(teamKeyOrName string) (*core.Cycle, error)
	CreateCycle(input *core.CreateCycleInput) (*core.Cycle, error)

	// Resolver operations
	ResolveTeamIdentifier(keyOrName string) (string, error)
	ResolveCycleIdentifier(numberOrNameOrID, teamID string) (string, error)
	ResolveUserIdentifier(nameOrEmail string) (string, error)

	// Sub-client access (Phase 2 - use sub-clients directly)
	CycleClient() *cycles.Client
}

// ProjectClientOperations defines the minimal interface needed by ProjectService
type ProjectClientOperations interface {
	// Smart resolver-aware methods (kept in Phase 2)
	CreateProject(name, description, teamKeyOrName string) (*core.Project, error)

	// Resolver operations
	ResolveTeamIdentifier(keyOrName string) (string, error)
	ResolveUserIdentifier(nameOrEmail string) (string, error)

	// Sub-client access (Phase 2 - use sub-clients directly)
	ProjectClient() *projects.Client
	TeamClient() *teams.Client
}

// UserClientOperations defines the minimal interface needed by UserService
type UserClientOperations interface {
	// Smart resolver-aware methods (kept in Phase 2)
	GetUser(idOrEmail string) (*core.User, error)

	// Resolver operations
	ResolveUserIdentifier(nameOrEmail string) (string, error)
	ResolveTeamIdentifier(keyOrName string) (string, error)

	// Sub-client access (Phase 2 - use sub-clients directly)
	TeamClient() *teams.Client
}

// SearchClientOperations defines the minimal interface needed by SearchService
type SearchClientOperations interface {
	// Smart resolver-aware methods (kept in Phase 2)
	SearchIssues(filters *core.IssueSearchFilters) (*core.IssueSearchResult, error)
	ListCycles(filter *core.CycleFilter) (*core.CycleSearchResult, error)

	// Resolver operations
	ResolveTeamIdentifier(keyOrName string) (string, error)
	ResolveUserIdentifier(nameOrEmail string) (string, error)
	ResolveCycleIdentifier(numberOrNameOrID, teamID string) (string, error)

	// Sub-client access (Phase 2 - use sub-clients directly)
	IssueClient() *issues.Client
	ProjectClient() *projects.Client
	TeamClient() *teams.Client
}
