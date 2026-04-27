package service

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/pkg/linear/core"
)

// UserService handles user-related operations
type UserService struct {
	client    UserClientOperations
	formatter *format.Formatter
}

// NewUserService creates a new UserService
func NewUserService(client UserClientOperations, formatter *format.Formatter) *UserService {
	return &UserService{
		client:    client,
		formatter: formatter,
	}
}

// UserFilters represents filters for searching users
type UserFilters struct {
	TeamID     string
	ActiveOnly *bool
	Limit      int
	After      string
}

// GetViewer returns the current authenticated user (legacy method)
func (s *UserService) GetViewer() (string, error) {
	viewer, err := s.client.TeamClient().GetViewer()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	return s.formatter.User(viewer), nil
}

// GetViewerWithOutput returns the current user with new renderer architecture
func (s *UserService) GetViewerWithOutput(verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	viewer, err := s.client.TeamClient().GetViewer()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	return s.formatter.RenderUser(viewer, verbosity, outputType), nil
}

// Get retrieves a single user by identifier (email, name, or ID) (legacy method)
func (s *UserService) Get(identifier string) (string, error) {
	// Resolve user identifier
	resolved, err := s.client.ResolveUserIdentifier(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to resolve user '%s': %w", identifier, err)
	}

	user, err := s.client.GetUser(resolved.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	return s.formatter.User(user), nil
}

// GetWithOutput retrieves a single user with new renderer architecture
func (s *UserService) GetWithOutput(identifier string, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	// Resolve user identifier
	resolved, err := s.client.ResolveUserIdentifier(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to resolve user '%s': %w", identifier, err)
	}

	user, err := s.client.GetUser(resolved.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	return s.formatter.RenderUser(user, verbosity, outputType), nil
}

// Search searches for users with the given filters (legacy method)
func (s *UserService) Search(filters *UserFilters) (string, error) {
	if filters == nil {
		filters = &UserFilters{}
	}

	// Set defaults
	if filters.Limit <= 0 {
		filters.Limit = 50
	}

	// Build Linear API filter
	linearFilters := &core.UserFilter{
		First:      filters.Limit,
		After:      filters.After,
		ActiveOnly: filters.ActiveOnly,
	}

	// Resolve team identifier if provided
	if filters.TeamID != "" {
		teamID, err := s.client.ResolveTeamIdentifier(filters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve team '%s': %w", filters.TeamID, err)
		}
		linearFilters.TeamID = teamID
	}

	// Execute search with pagination
	result, err := s.client.TeamClient().ListUsersWithPagination(linearFilters)
	if err != nil {
		return "", fmt.Errorf("failed to list users: %w", err)
	}

	// Format output
	pagination := &format.Pagination{
		HasNextPage: result.HasNextPage,
		EndCursor:   result.EndCursor,
	}

	return s.formatter.UserList(result.Users, pagination), nil
}

// SearchWithOutput searches for users with new renderer architecture
func (s *UserService) SearchWithOutput(filters *UserFilters, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	if filters == nil {
		filters = &UserFilters{}
	}

	// Set defaults
	if filters.Limit <= 0 {
		filters.Limit = 50
	}

	// Build Linear API filter
	linearFilters := &core.UserFilter{
		First:      filters.Limit,
		After:      filters.After,
		ActiveOnly: filters.ActiveOnly,
	}

	// Resolve team identifier if provided
	if filters.TeamID != "" {
		teamID, err := s.client.ResolveTeamIdentifier(filters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve team '%s': %w", filters.TeamID, err)
		}
		linearFilters.TeamID = teamID
	}

	// Execute search with pagination
	result, err := s.client.TeamClient().ListUsersWithPagination(linearFilters)
	if err != nil {
		return "", fmt.Errorf("failed to list users: %w", err)
	}

	// Format output with new renderer
	// TODO: Add pagination support to RenderUserList
	return s.formatter.RenderUserList(result.Users, verbosity, outputType), nil
}

// ResolveByName resolves a user by name and returns their ID
// Supports:
// - "me" - returns current authenticated user
// - Email addresses: "john@company.com"
// - Display names: "John Doe"
// - Partial names: "John" (errors if ambiguous with suggestions)
//
// Returns error with multiple user names if ambiguous
func (s *UserService) ResolveByName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("user name cannot be empty")
	}

	// Handle "me" as current user
	if name == "me" {
		viewer, err := s.client.TeamClient().GetViewer()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %w", err)
		}
		return viewer.ID, nil
	}

	// Use the client's resolver for name/email resolution
	resolved, err := s.client.ResolveUserIdentifier(name)
	if err != nil {
		return "", err
	}

	return resolved.ID, nil
}
