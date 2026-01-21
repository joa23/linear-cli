package linear

import (
	"fmt"
	"strings"
	"time"
)

// Default cache TTL for all resolution operations
const defaultCacheTTL = 5 * time.Minute

// Resolver handles intelligent resolution of human-readable identifiers to UUIDs
// It manages caching and provides smart matching with ambiguity detection
//
// Why: The MCP server should accept human-readable inputs (emails, names, CEN-123)
// instead of forcing clients to look up UUIDs. The resolver handles this translation.
type Resolver struct {
	client *Client
	cache  *resolverCache
}

// NewResolver creates a new resolver with the default cache TTL
func NewResolver(client *Client) *Resolver {
	return &Resolver{
		client: client,
		cache:  newResolverCache(defaultCacheTTL),
	}
}

// ResolveUser resolves a user identifier (email or name) to a user UUID
// Supports:
// - Email addresses: "john@company.com"
// - Display names: "John Doe"
// - First names: "John" (errors if ambiguous)
//
// Returns error with suggestions if multiple users match
func (r *Resolver) ResolveUser(nameOrEmail string) (string, error) {
	// Validate input
	if nameOrEmail == "" {
		return "", &ValidationError{
			Field:   "user",
			Message: "user identifier cannot be empty",
		}
	}

	// Check if it's an email
	if isEmail(nameOrEmail) {
		return r.resolveUserByEmail(nameOrEmail)
	}

	// Otherwise treat as a name
	return r.resolveUserByName(nameOrEmail)
}

// resolveUserByEmail resolves a user by their email address
func (r *Resolver) resolveUserByEmail(email string) (string, error) {
	// Check cache first
	if userID, found := r.cache.getUserByEmail(email); found {
		return userID, nil
	}

	// Use the existing GetUserByEmail method
	user, err := r.client.Teams.GetUserByEmail(email)
	if err != nil {
		// If not found, return helpful error
		if IsNotFoundError(err) {
			return "", &NotFoundError{
				ResourceType: "user",
				ResourceID:   email,
			}
		}
		return "", fmt.Errorf("failed to resolve user by email: %w", err)
	}

	// Cache the result
	r.cache.setUserByEmail(email, user.ID)
	r.cache.setUserByName(user.Name, user.ID)

	return user.ID, nil
}

// resolveUserByName resolves a user by their name (display name or full name)
// Performs fuzzy matching and errors if multiple matches are found
func (r *Resolver) resolveUserByName(name string) (string, error) {
	// Check cache first
	if userID, found := r.cache.getUserByName(name); found {
		return userID, nil
	}

	// Use Linear's displayName filter for efficient server-side search
	// Try exact match first
	activeOnly := true
	users, err := r.client.Teams.ListUsersWithDisplayNameFilter(name, &activeOnly, 10)
	if err != nil {
		return "", fmt.Errorf("failed to list users for name resolution: %w", err)
	}

	// Find matching users
	var matches []User
	nameLower := strings.ToLower(name)

	for _, user := range users {
		// Check exact match (case-insensitive)
		if strings.ToLower(user.Name) == nameLower ||
			strings.ToLower(user.DisplayName) == nameLower {
			matches = append(matches, user)
			continue
		}

		// Check if name is part of the full name (fuzzy match)
		if strings.Contains(strings.ToLower(user.Name), nameLower) ||
			strings.Contains(strings.ToLower(user.DisplayName), nameLower) {
			matches = append(matches, user)
		}
	}

	// Handle no matches
	if len(matches) == 0 {
		return "", &NotFoundError{
			ResourceType: "user",
			ResourceID:   name,
		}
	}

	// Handle ambiguous matches
	if len(matches) > 1 {
		// Build suggestions
		var suggestions []string
		for _, user := range matches {
			suggestions = append(suggestions, fmt.Sprintf("%s (%s)", user.Name, user.Email))
		}

		return "", &ErrorWithGuidance{
			Operation: "Resolve user",
			Reason:    fmt.Sprintf("multiple users match '%s'", name),
			Guidance: []string{
				"Use the full email address for exact matching",
				"Use the complete display name",
				"Choose from the suggestions below",
			},
			Example: fmt.Sprintf(`// Use email for exact match:
linear_update_issue("CEN-123", {assigneeId: "%s"})

// Or use full name:
linear_update_issue("CEN-123", {assigneeId: "%s"})`,
				matches[0].Email, matches[0].Name),
			OriginalErr: &ValidationError{
				Field:  "user",
				Value:  name,
				Reason: fmt.Sprintf("ambiguous, matches: %s", strings.Join(suggestions, ", ")),
			},
		}
	}

	// Single match found
	user := matches[0]

	// Cache the result
	r.cache.setUserByName(name, user.ID)
	r.cache.setUserByEmail(user.Email, user.ID)

	return user.ID, nil
}

// ResolveTeam resolves a team identifier (name or key) to a team UUID
// Supports:
// - Team keys: "ENG", "PRODUCT"
// - Team names: "Engineering", "Product Team"
//
// Returns error if team not found
func (r *Resolver) ResolveTeam(keyOrName string) (string, error) {
	// Validate input
	if keyOrName == "" {
		return "", &ValidationError{
			Field:   "team",
			Message: "team identifier cannot be empty",
		}
	}

	// Check cache by key first
	if teamID, found := r.cache.getTeamByKey(keyOrName); found {
		return teamID, nil
	}

	// Check cache by name
	if teamID, found := r.cache.getTeamByName(keyOrName); found {
		return teamID, nil
	}

	// Fetch all teams
	teams, err := r.client.Teams.GetTeams()
	if err != nil {
		return "", fmt.Errorf("failed to fetch teams for resolution: %w", err)
	}

	// Try exact match on key first (case-insensitive)
	keyUpper := strings.ToUpper(keyOrName)
	for _, team := range teams {
		if strings.ToUpper(team.Key) == keyUpper {
			// Cache by both key and name
			r.cache.setTeamByKey(team.Key, team.ID)
			r.cache.setTeamByName(team.Name, team.ID)
			return team.ID, nil
		}
	}

	// Try exact match on name (case-insensitive)
	nameLower := strings.ToLower(keyOrName)
	for _, team := range teams {
		if strings.ToLower(team.Name) == nameLower {
			// Cache by both key and name
			r.cache.setTeamByKey(team.Key, team.ID)
			r.cache.setTeamByName(team.Name, team.ID)
			return team.ID, nil
		}
	}

	// No match found
	return "", &NotFoundError{
		ResourceType: "team",
		ResourceID:   keyOrName,
	}
}

// ResolveIssue resolves an issue identifier (CEN-123) to an issue UUID
// Only accepts Linear identifiers in format TEAM-NUMBER
//
// Returns error if identifier invalid or issue not found
func (r *Resolver) ResolveIssue(identifier string) (string, error) {
	// Validate input
	if identifier == "" {
		return "", &ValidationError{
			Field:   "identifier",
			Message: "issue identifier cannot be empty",
		}
	}

	// Validate format
	if !isIssueIdentifier(identifier) {
		return "", &ValidationError{
			Field:  "identifier",
			Value:  identifier,
			Reason: "must be in format TEAM-NUMBER (e.g., CEN-123)",
		}
	}

	// Check cache
	if issueID, found := r.cache.getIssueByIdentifier(identifier); found {
		return issueID, nil
	}

	// Use GetIssue directly with the identifier
	// Linear's issue(id:) query accepts both UUIDs and identifiers (e.g., "CEN-123")
	issue, err := r.client.Issues.GetIssue(identifier)
	if err != nil {
		// Check if it's a not found error (null response or 404)
		if IsNotFoundError(err) {
			return "", &NotFoundError{
				ResourceType: "issue",
				ResourceID:   identifier,
			}
		}
		return "", fmt.Errorf("failed to get issue %s: %w", identifier, err)
	}

	// Cache and return the result
	r.cache.setIssueByIdentifier(identifier, issue.ID)
	return issue.ID, nil
}

// ResolveCycle resolves a cycle identifier (number or name) to a cycle UUID
// Supports:
// - Cycle numbers: "62" (fastest lookup)
// - Cycle names: "Cycle 67" or "Sprint Planning"
//
// Returns error with suggestions if multiple cycles match
func (r *Resolver) ResolveCycle(numberOrNameOrID string, teamID string) (string, error) {
	// Validate input
	if numberOrNameOrID == "" {
		return "", &ValidationError{
			Field:   "cycle",
			Message: "cycle identifier cannot be empty",
		}
	}

	if teamID == "" {
		return "", &ValidationError{
			Field:   "teamId",
			Message: "team ID is required for cycle resolution",
		}
	}

	// Check if it's a UUID (36 chars, contains hyphens)
	if len(numberOrNameOrID) == 36 && strings.Contains(numberOrNameOrID, "-") {
		// Assume it's already a UUID, return as-is
		return numberOrNameOrID, nil
	}

	// Try to resolve as number first (fastest path)
	cycleID, err := r.resolveCycleByNumber(numberOrNameOrID, teamID)
	if err == nil {
		return cycleID, nil
	}

	// Try to resolve as name
	return r.resolveCycleName(numberOrNameOrID, teamID)
}

// resolveCycleByNumber resolves a cycle by its number (e.g., "62")
func (r *Resolver) resolveCycleByNumber(numberStr string, teamID string) (string, error) {
	// Parse the number
	var cycleNumber int
	_, err := fmt.Sscanf(numberStr, "%d", &cycleNumber)
	if err != nil {
		// Not a valid number, let caller try other resolution methods
		return "", fmt.Errorf("not a cycle number")
	}

	// Fetch cycles for the team
	cycles, err := r.client.Cycles.ListCycles(&CycleFilter{
		TeamID: teamID,
		Limit:  100, // Get enough cycles to find the match
	})
	if err != nil {
		return "", fmt.Errorf("failed to list cycles: %w", err)
	}

	// Find matching cycle by number
	for _, cycle := range cycles.Cycles {
		if cycle.Number == cycleNumber {
			return cycle.ID, nil
		}
	}

	// Not found
	return "", fmt.Errorf("cycle number %d not found", cycleNumber)
}

// resolveCycleName resolves a cycle by its name (e.g., "Cycle 67" or "Sprint Planning")
func (r *Resolver) resolveCycleName(name string, teamID string) (string, error) {
	// Fetch cycles for the team
	cycles, err := r.client.Cycles.ListCycles(&CycleFilter{
		TeamID: teamID,
		Limit:  100,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list cycles: %w", err)
	}

	// Find matching cycles by name (case-insensitive)
	var matches []Cycle
	nameLower := strings.ToLower(name)

	for _, cycle := range cycles.Cycles {
		// Check exact match (case-insensitive)
		if strings.ToLower(cycle.Name) == nameLower {
			matches = append(matches, cycle)
			continue
		}

		// Check if name is part of cycle name (fuzzy match)
		if strings.Contains(strings.ToLower(cycle.Name), nameLower) {
			matches = append(matches, cycle)
		}
	}

	// Handle no matches
	if len(matches) == 0 {
		return "", &NotFoundError{
			ResourceType: "cycle",
			ResourceID:   name,
		}
	}

	// Handle ambiguous matches
	if len(matches) > 1 {
		// Build suggestions
		var suggestions []string
		for _, cycle := range matches {
			suggestions = append(suggestions, fmt.Sprintf("#%d: %s", cycle.Number, cycle.Name))
		}

		return "", &ErrorWithGuidance{
			Operation: "Resolve cycle",
			Reason:    fmt.Sprintf("multiple cycles match '%s'", name),
			Guidance: []string{
				"Use the cycle number for exact matching (e.g., '62' instead of 'Cycle 62')",
				"Use the complete cycle name",
				"Choose from the suggestions below",
			},
			Example: fmt.Sprintf(`// Use cycle number for exact match:
linear_search resource="issues" query={"cycleId":"%d"}

// Or use full name:
linear_search resource="issues" query={"cycleId":"%s"}`,
				matches[0].Number, matches[0].Name),
			OriginalErr: &ValidationError{
				Field:  "cycle",
				Value:  name,
				Reason: fmt.Sprintf("ambiguous, matches: %s", strings.Join(suggestions, ", ")),
			},
		}
	}

	// Single match found
	return matches[0].ID, nil
}
