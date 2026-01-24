package cli

import (
	"github.com/joa23/linear-cli/internal/linear"
	"github.com/joa23/linear-cli/internal/service"
)

// Dependencies holds all injectable dependencies for CLI commands
// This enables testing by allowing mock implementations to be injected
type Dependencies struct {
	// Client is the Linear API client
	Client *linear.Client

	// Services provide business logic and formatting
	Issues   service.IssueServiceInterface
	Cycles   service.CycleServiceInterface
	Projects service.ProjectServiceInterface
	Search   service.SearchServiceInterface
	Teams    service.TeamServiceInterface
	Users    service.UserServiceInterface
}

// NewDependencies creates dependencies with real implementations
func NewDependencies(client *linear.Client) *Dependencies {
	services := service.New(client)

	return &Dependencies{
		Client:   client,
		Issues:   services.Issues,
		Cycles:   services.Cycles,
		Projects: services.Projects,
		Search:   services.Search,
		Teams:    services.Teams,
		Users:    services.Users,
	}
}
