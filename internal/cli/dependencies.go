package cli

import (
	"github.com/joa23/linear-cli/internal/service"
	"github.com/joa23/linear-cli/pkg/linear"
)

// Dependencies holds all injectable dependencies for CLI commands
// This enables testing by allowing mock implementations to be injected
type Dependencies struct {
	// Client is the Linear API client
	Client *linear.Client

	// Services provide business logic and formatting
	Issues     service.IssueServiceInterface
	Cycles     service.CycleServiceInterface
	Projects   service.ProjectServiceInterface
	Milestones service.MilestoneServiceInterface
	Search     service.SearchServiceInterface
	Teams      service.TeamServiceInterface
	Users      service.UserServiceInterface
	Labels     service.LabelServiceInterface
	TaskExport  service.TaskExportServiceInterface
	Attachments service.AttachmentServiceInterface
	IssueExport service.IssueExportServiceInterface
}

// NewDependencies creates dependencies with real implementations
func NewDependencies(client *linear.Client) *Dependencies {
	services := service.New(client)

	return &Dependencies{
		Client:     client,
		Issues:     services.Issues,
		Cycles:     services.Cycles,
		Projects:   services.Projects,
		Milestones: services.Milestones,
		Search:     services.Search,
		Teams:      services.Teams,
		Users:      services.Users,
		Labels:     services.Labels,
		TaskExport:  services.TaskExport,
		Attachments: services.Attachments,
		IssueExport: services.IssueExport,
	}
}
