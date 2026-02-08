package service

import (
	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear/core"
)

// IssueServiceInterface defines the contract for issue operations
type IssueServiceInterface interface {
	Get(identifier string, outputFormat format.Format) (string, error)
	GetWithOutput(identifier string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	Search(filters *SearchFilters) (string, error)
	SearchWithOutput(filters *SearchFilters, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	ListAssigned(limit int, outputFormat format.Format) (string, error)
	ListAssignedWithPagination(pagination *core.PaginationInput) (string, error)
	Create(input *CreateIssueInput) (string, error)
	Update(identifier string, input *UpdateIssueInput) (string, error)
	GetComments(identifier string) (string, error)
	AddComment(identifier, body string) (string, error)
	ReplyToComment(issueIdentifier, parentCommentID, body string) (*core.Comment, error)
	AddReaction(targetID, emoji string) error
	GetIssueID(identifier string) (string, error)
}

// CycleServiceInterface defines the contract for cycle operations
type CycleServiceInterface interface {
	Get(cycleIDOrNumber string, teamID string, outputFormat format.Format) (string, error)
	GetWithOutput(cycleIDOrNumber string, teamID string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	Search(filters *CycleFilters) (string, error)
	SearchWithOutput(filters *CycleFilters, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	Create(input *CreateCycleInput) (string, error)
	Analyze(input *AnalyzeInput) (string, error)
	AnalyzeWithOutput(input *AnalyzeInput, verbosity format.Verbosity, outputType format.OutputType) (string, error)
}

// ProjectServiceInterface defines the contract for project operations
type ProjectServiceInterface interface {
	Get(projectID string) (string, error)
	GetWithOutput(projectID string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	ListAll(limit int) (string, error)
	ListAllWithOutput(limit int, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	ListByTeam(teamID string, limit int) (string, error)
	ListByTeamWithOutput(teamID string, limit int, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	ListUserProjects(limit int) (string, error)
	ListUserProjectsWithOutput(limit int, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	Create(input *CreateProjectInput) (string, error)
	Update(projectID string, input *UpdateProjectInput) (string, error)
}

// SearchServiceInterface defines the contract for unified search
type SearchServiceInterface interface {
	Search(opts *SearchOptions) (string, error)
}

// TeamServiceInterface defines the contract for team operations
type TeamServiceInterface interface {
	Get(identifier string) (string, error)
	GetWithOutput(identifier string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	ListAll() (string, error)
	ListAllWithOutput(verbosity format.Verbosity, outputType format.OutputType) (string, error)
	GetLabels(identifier string) (string, error)
	GetLabelsWithOutput(identifier string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	GetWorkflowStates(identifier string) (string, error)
	GetWorkflowStatesWithOutput(identifier string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
}

// UserServiceInterface defines the contract for user operations
type UserServiceInterface interface {
	GetViewer() (string, error)
	GetViewerWithOutput(verbosity format.Verbosity, outputType format.OutputType) (string, error)
	Get(identifier string) (string, error)
	GetWithOutput(identifier string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	Search(filters *UserFilters) (string, error)
	SearchWithOutput(filters *UserFilters, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	ResolveByName(name string) (string, error)
}

// LabelServiceInterface defines the contract for label operations
type LabelServiceInterface interface {
	List(teamID string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	Create(input *core.CreateLabelInput) (string, error)
	Update(id string, input *core.UpdateLabelInput) (string, error)
	Delete(id string) (string, error)
}

// TaskExportServiceInterface defines the contract for task export operations
type TaskExportServiceInterface interface {
	Export(identifier string, outputFolder string, dryRun bool) (*ExportResult, error)
}

// Verify implementations satisfy interfaces (compile-time check)
var (
	_ IssueServiceInterface      = (*IssueService)(nil)
	_ CycleServiceInterface      = (*CycleService)(nil)
	_ ProjectServiceInterface    = (*ProjectService)(nil)
	_ SearchServiceInterface     = (*SearchService)(nil)
	_ TeamServiceInterface       = (*TeamService)(nil)
	_ UserServiceInterface       = (*UserService)(nil)
	_ LabelServiceInterface      = (*LabelService)(nil)
	_ TaskExportServiceInterface = (*TaskExportService)(nil)
)
