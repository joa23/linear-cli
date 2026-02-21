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
	Issues     *IssueService
	Projects   *ProjectService
	Cycles     *CycleService
	Teams      *TeamService
	Users      *UserService
	Labels     *LabelService
	Search     *SearchService
	TaskExport  *TaskExportService
	Attachments *AttachmentService

	client *linear.Client // Store original client for backward compatibility
}

// New creates all services with a shared Linear client and formatter
func New(client *linear.Client) *Services {
	formatter := format.New()

	return &Services{
		Issues:     NewIssueService(client, formatter),
		Projects:   NewProjectService(client, formatter),
		Cycles:     NewCycleService(client, formatter),
		Teams:      NewTeamService(client, formatter),
		Users:      NewUserService(client, formatter),
		Labels:     NewLabelService(client, formatter),
		Search:     NewSearchService(client, formatter),
		TaskExport:  NewTaskExportService(client),
		Attachments: NewAttachmentService(client, formatter),
		client:      client,
	}
}

// Client returns the underlying Linear client for advanced operations
func (s *Services) Client() *linear.Client {
	return s.client
}
