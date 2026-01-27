package teams

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/linear/core"
)

// TeamClient handles all team and user-related operations for the Linear API.
// It uses the shared BaseClient for HTTP communication and focuses on
// organizational structure queries.
type Client struct {
	base *core.BaseClient
}

// NewTeamClient creates a new team client with the provided base client
func NewClient(base *core.BaseClient) *Client {
	return &Client{base: base}
}

// GetTeams retrieves all teams in the workspace
// Why: Teams are the primary organizational unit in Linear. Users need
// to discover available teams for issue creation and assignment.
func (tc *Client) GetTeams() ([]core.Team, error) {
	const query = `
		query GetTeams {
			teams {
				nodes {
					id
					name
					key
					description
					issueEstimationType
					issueEstimationAllowZero
					issueEstimationExtended
					defaultIssueEstimate
				}
			}
		}
	`
	
	var response struct {
		Teams struct {
			Nodes []core.Team `json:"nodes"`
		} `json:"teams"`
	}
	
	err := tc.base.ExecuteRequest(query, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}
	
	return response.Teams.Nodes, nil
}

// GetTeam retrieves a single team by ID with estimate settings
// Why: Users need to get details about a specific team including
// the estimate scale configuration for that team.
func (tc *Client) GetTeam(teamID string) (*core.Team, error) {
	const query = `
		query GetTeam($id: String!) {
			team(id: $id) {
				id
				name
				key
				description
				issueEstimationType
				issueEstimationAllowZero
				issueEstimationExtended
				defaultIssueEstimate
			}
		}
	`

	variables := map[string]interface{}{
		"id": teamID,
	}

	var response struct {
		Team *core.Team `json:"team"`
	}

	err := tc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	if response.Team == nil {
		return nil, fmt.Errorf("team not found: %s", teamID)
	}

	return response.Team, nil
}

// GetViewer retrieves information about the authenticated user
// Why: Users need to know their own identity and capabilities within
// the system. This is essential for determining permissions and context.
func (tc *Client) GetViewer() (*core.User, error) {
	const query = `
		query GetViewer {
			viewer {
				id
				name
				email
				displayName
				avatarUrl
				createdAt
				isMe
			}
		}
	`
	
	var response struct {
		Viewer core.User `json:"viewer"`
	}
	
	err := tc.base.ExecuteRequest(query, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get viewer: %w", err)
	}
	
	return &response.Viewer, nil
}

// ListUsers retrieves users based on the provided filter
// Why: Users need to discover team members, find assignees, and understand
// organizational structure. Filters allow focusing on specific subsets.
func (tc *Client) ListUsers(filter *core.UserFilter) ([]core.User, error) {
	// Set default pagination if not specified
	if filter == nil {
		filter = &core.UserFilter{}
	}
	if filter.First == 0 {
		filter.First = 10 // Reduced from 50 to minimize token usage
	}

	// Build the query based on filter options
	var query string
	variables := make(map[string]interface{})

	if filter.TeamID != "" {
		// Query users within a specific team
		query = `
			query GetTeamMembers($teamId: String!, $first: Int!, $after: String) {
				team(id: $teamId) {
					members(first: $first, after: $after) {
						nodes {
							id
							name
							displayName
							email
							avatarUrl
							active
							admin
							createdAt
						}
					}
				}
			}
		`
		variables["teamId"] = filter.TeamID
		variables["first"] = filter.First
		if filter.After != "" {
			variables["after"] = filter.After
		}

		var response struct {
			Team struct {
				Members struct {
					Nodes []core.User `json:"nodes"`
				} `json:"members"`
			} `json:"team"`
		}

		err := tc.base.ExecuteRequest(query, variables, &response)
		if err != nil {
			return nil, fmt.Errorf("failed to list team members: %w", err)
		}

		// Apply active filter if specified
		if filter.ActiveOnly != nil && *filter.ActiveOnly {
			activeUsers := make([]core.User, 0)
			for _, user := range response.Team.Members.Nodes {
				if user.Active {
					activeUsers = append(activeUsers, user)
				}
			}
			return activeUsers, nil
		}

		return response.Team.Members.Nodes, nil
	}

	// Query all users in the workspace
	if filter.ActiveOnly != nil && *filter.ActiveOnly {
		query = `
			query GetActiveUsers($first: Int!, $after: String) {
				users(first: $first, after: $after, filter: { active: { eq: true } }) {
					nodes {
						id
						name
						displayName
						email
						avatarUrl
						active
						admin
						createdAt
					}
				}
			}
		`
	} else {
		query = `
			query GetAllUsers($first: Int!, $after: String) {
				users(first: $first, after: $after) {
					nodes {
						id
						name
						displayName
						email
						avatarUrl
						active
						admin
						createdAt
					}
				}
			}
		`
	}

	variables["first"] = filter.First
	if filter.After != "" {
		variables["after"] = filter.After
	}

	var response struct {
		Users struct {
			Nodes []core.User `json:"nodes"`
		} `json:"users"`
	}

	err := tc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return response.Users.Nodes, nil
}

// ListUsersWithPagination retrieves users with pagination information
// Why: Large organizations may have many users, requiring pagination to
// efficiently retrieve and display user lists.
func (tc *Client) ListUsersWithPagination(filter *core.UserFilter) (*core.ListUsersResult, error) {
	// Set default pagination if not specified
	if filter == nil {
		filter = &core.UserFilter{}
	}
	if filter.First == 0 {
		filter.First = 10 // Reduced from 50 to minimize token usage
	}

	// Build the query based on filter options
	var query string
	variables := make(map[string]interface{})

	if filter.TeamID != "" {
		// Query users within a specific team with pagination
		query = `
			query GetTeamMembersWithPagination($teamId: String!, $first: Int!, $after: String) {
				team(id: $teamId) {
					members(first: $first, after: $after) {
						nodes {
							id
							name
							displayName
							email
							avatarUrl
							active
							admin
							createdAt
						}
						pageInfo {
							hasNextPage
							endCursor
						}
					}
				}
			}
		`
		variables["teamId"] = filter.TeamID
		variables["first"] = filter.First
		if filter.After != "" {
			variables["after"] = filter.After
		}

		var response struct {
			Team struct {
				Members struct {
					Nodes    []core.User `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"members"`
			} `json:"team"`
		}

		err := tc.base.ExecuteRequest(query, variables, &response)
		if err != nil {
			return nil, fmt.Errorf("failed to list team members with pagination: %w", err)
		}

		// Apply active filter if specified
		users := response.Team.Members.Nodes
		if filter.ActiveOnly != nil && *filter.ActiveOnly {
			activeUsers := make([]core.User, 0)
			for _, user := range users {
				if user.Active {
					activeUsers = append(activeUsers, user)
				}
			}
			users = activeUsers
		}

		return &core.ListUsersResult{
			Users:       users,
			HasNextPage: response.Team.Members.PageInfo.HasNextPage,
			EndCursor:   response.Team.Members.PageInfo.EndCursor,
		}, nil
	}

	// Query all users in the workspace with pagination
	if filter.ActiveOnly != nil && *filter.ActiveOnly {
		query = `
			query GetActiveUsersWithPagination($first: Int!, $after: String) {
				users(first: $first, after: $after, filter: { active: { eq: true } }) {
					nodes {
						id
						name
						displayName
						email
						avatarUrl
						active
						admin
						createdAt
					}
					pageInfo {
						hasNextPage
						endCursor
					}
				}
			}
		`
	} else {
		query = `
			query GetAllUsersWithPagination($first: Int!, $after: String) {
				users(first: $first, after: $after) {
					nodes {
						id
						name
						displayName
						email
						avatarUrl
						active
						admin
						createdAt
					}
					pageInfo {
						hasNextPage
						endCursor
					}
				}
			}
		`
	}

	variables["first"] = filter.First
	if filter.After != "" {
		variables["after"] = filter.After
	}

	var response struct {
		Users struct {
			Nodes    []core.User `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"users"`
	}

	err := tc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list users with pagination: %w", err)
	}

	return &core.ListUsersResult{
		Users:       response.Users.Nodes,
		HasNextPage: response.Users.PageInfo.HasNextPage,
		EndCursor:   response.Users.PageInfo.EndCursor,
	}, nil
}

// GetUser retrieves a specific user by ID with optional team memberships and preferences
// Why: Users need to look up specific team members by their unique ID to view
// profile details, team associations, and notification preferences.
func (tc *Client) GetUser(userID string) (*core.User, error) {
	const query = `
		query GetUser($userId: String!) {
			user(id: $userId) {
				id
				name
				displayName
				email
				avatarUrl
				active
				admin
				createdAt
				isMe
				teams {
					nodes {
						id
						name
						key
						description
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"userId": userID,
	}
	
	var response struct {
		User *core.User `json:"user"`
	}
	
	err := tc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if response.User == nil {
		return nil, fmt.Errorf("user not found")
	}
	
	return response.User, nil
}

// GetUserByEmail retrieves a user by their email address
// Why: Email is a common identifier for users. This allows finding users
// when only their email is known, useful for mentions and assignments.
func (tc *Client) GetUserByEmail(email string) (*core.User, error) {
	const query = `
		query GetUserByEmail($email: String!) {
			users(filter: { email: { eq: $email } }) {
				nodes {
					id
					name
					displayName
					email
					avatarUrl
					active
					admin
					createdAt
					isMe
					teams {
						nodes {
							id
							name
							key
							description
						}
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"email": email,
	}
	
	var response struct {
		Users struct {
			Nodes []core.User `json:"nodes"`
		} `json:"users"`
	}
	
	err := tc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	
	if len(response.Users.Nodes) == 0 {
		return nil, fmt.Errorf("user not found with email: %s", email)
	}
	
	if len(response.Users.Nodes) > 1 {
		return nil, fmt.Errorf("multiple users found with email: %s", email)
	}
	
	return &response.Users.Nodes[0], nil
}

// ListUsersWithDisplayNameFilter retrieves users by displayName filter with optional active status
// Why: Efficient server-side filtering for user name resolution. Uses displayName filter
// to avoid downloading all users when searching by name.
func (tc *Client) ListUsersWithDisplayNameFilter(displayName string, activeOnly *bool, limit int) ([]core.User, error) {
	// Build filter object
	filter := make(map[string]interface{})

	// Add displayName filter using contains for fuzzy matching
	filter["displayName"] = map[string]interface{}{
		"contains": displayName,
	}

	// Add active filter if specified
	if activeOnly != nil {
		filter["active"] = map[string]interface{}{
			"eq": *activeOnly,
		}
	}

	const query = `
		query ListUsersByDisplayName($filter: UserFilter!, $first: Int!) {
			users(filter: $filter, first: $first) {
				nodes {
					id
					name
					displayName
					email
					avatarUrl
					active
					admin
					createdAt
				}
			}
		}
	`

	variables := map[string]interface{}{
		"filter": filter,
		"first":  limit,
	}

	var response struct {
		Users struct {
			Nodes []core.User `json:"nodes"`
		} `json:"users"`
	}

	err := tc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list users by display name: %w", err)
	}

	return response.Users.Nodes, nil
}

// ListLabels retrieves all labels for a specific team
// Why: Labels are used to categorize and filter issues. Teams need to
// discover available labels for issue tagging and organization.
func (tc *Client) ListLabels(teamID string) ([]core.Label, error) {
	const query = `
		query GetTeamLabels($teamId: String!) {
			team(id: $teamId) {
				labels {
					nodes {
						id
						name
						color
						description
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"teamId": teamID,
	}
	
	var response struct {
		Team *struct {
			Labels struct {
				Nodes []core.Label `json:"nodes"`
			} `json:"labels"`
		} `json:"team"`
	}
	
	err := tc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	
	if response.Team == nil {
		return nil, fmt.Errorf("team not found")
	}
	
	return response.Team.Labels.Nodes, nil
}