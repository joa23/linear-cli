package service

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
)

// UserService handles user-related operations
type UserService struct {
	client    *linear.Client
	formatter *format.Formatter
}

// NewUserService creates a new UserService
func NewUserService(client *linear.Client, formatter *format.Formatter) *UserService {
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

// GetViewer returns the current authenticated user
func (s *UserService) GetViewer() (string, error) {
	viewer, err := s.client.GetViewer()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	return s.formatter.User(viewer), nil
}

// Get retrieves a single user by identifier (email, name, or ID)
func (s *UserService) Get(identifier string) (string, error) {
	// Resolve user identifier
	userID, err := s.client.ResolveUserIdentifier(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to resolve user '%s': %w", identifier, err)
	}

	user, err := s.client.Teams.GetUser(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	return s.formatter.User(user), nil
}

// Search searches for users with the given filters
func (s *UserService) Search(filters *UserFilters) (string, error) {
	if filters == nil {
		filters = &UserFilters{}
	}

	// Set defaults
	if filters.Limit <= 0 {
		filters.Limit = 50
	}

	// Build Linear API filter
	linearFilters := &linear.UserFilter{
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
	result, err := s.client.ListUsersWithPagination(linearFilters)
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
		viewer, err := s.client.GetViewer()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %w", err)
		}
		return viewer.ID, nil
	}

	// Use the client's resolver for name/email resolution
	userID, err := s.client.ResolveUserIdentifier(name)
	if err != nil {
		return "", err
	}

	return userID, nil
}
