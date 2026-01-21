// Package service provides a service layer between interfaces (CLI/MCP) and
// the Linear client. It handles business logic, validation, identifier resolution,
// and response formatting.
package service

import (
	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
)

// Services holds all service instances
type Services struct {
	Issues   *IssueService
	Projects *ProjectService
	Cycles   *CycleService
	Teams    *TeamService
	Users    *UserService
}

// New creates all services with a shared Linear client and formatter
func New(client *linear.Client) *Services {
	formatter := format.New()

	return &Services{
		Issues:   NewIssueService(client, formatter),
		Projects: NewProjectService(client, formatter),
		Cycles:   NewCycleService(client, formatter),
		Teams:    NewTeamService(client, formatter),
		Users:    NewUserService(client, formatter),
	}
}

// Client returns the underlying Linear client for advanced operations
func (s *Services) Client() *linear.Client {
	return s.Issues.client
}
