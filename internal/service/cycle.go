package service

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear/cycles"
	"github.com/joa23/linear-cli/internal/linear/core"
)

// CycleService handles cycle-related operations
type CycleService struct {
	client    CycleClientOperations
	formatter *format.Formatter
}

// NewCycleService creates a new CycleService
func NewCycleService(client CycleClientOperations, formatter *format.Formatter) *CycleService {
	return &CycleService{
		client:    client,
		formatter: formatter,
	}
}

// CycleFilters represents filters for searching cycles
type CycleFilters struct {
	TeamID   string
	IsActive *bool
	IsFuture *bool
	IsPast   *bool
	Limit    int
	After    string
	Format   format.Format
}

// Get retrieves a single cycle by ID, number, or name
func (s *CycleService) Get(cycleIDOrNumber string, teamID string, outputFormat format.Format) (string, error) {
	// Resolve cycle identifier (number/name/UUID) to UUID
	// teamID is optional - if empty, cycleIDOrNumber must be a UUID
	resolvedID := cycleIDOrNumber
	if teamID != "" {
		var err error
		resolvedID, err = s.client.ResolveCycleIdentifier(cycleIDOrNumber, teamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve cycle '%s': %w", cycleIDOrNumber, err)
		}
	}

	cycle, err := s.client.CycleClient().GetCycle(resolvedID)
	if err != nil {
		return "", fmt.Errorf("failed to get cycle %s: %w", cycleIDOrNumber, err)
	}

	return s.formatter.Cycle(cycle, outputFormat), nil
}

// Search searches for cycles with the given filters
func (s *CycleService) Search(filters *CycleFilters) (string, error) {
	if filters == nil {
		filters = &CycleFilters{}
	}

	// Set defaults
	if filters.Limit <= 0 {
		filters.Limit = 10
	}
	if filters.Format == "" {
		filters.Format = format.Compact
	}

	// Build Linear API filter
	linearFilters := &core.CycleFilter{
		Limit:    filters.Limit,
		After:    filters.After,
		IsActive: filters.IsActive,
		IsFuture: filters.IsFuture,
		IsPast:   filters.IsPast,
		Format:   core.ResponseFormat(filters.Format),
	}

	// Resolve team identifier if provided
	if filters.TeamID != "" {
		teamID, err := s.client.ResolveTeamIdentifier(filters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve team '%s': %w", filters.TeamID, err)
		}
		linearFilters.TeamID = teamID
	}

	// Execute search
	result, err := s.client.ListCycles(linearFilters)
	if err != nil {
		return "", fmt.Errorf("failed to search cycles: %w", err)
	}

	// Format output
	pagination := &format.Pagination{
		HasNextPage: result.HasNextPage,
		EndCursor:   result.EndCursor,
	}

	return s.formatter.CycleList(result.Cycles, filters.Format, pagination), nil
}

// CreateCycleInput represents input for creating a cycle
type CreateCycleInput struct {
	TeamID      string
	Name        string
	Description string
	StartsAt    string
	EndsAt      string
}

// Create creates a new cycle
func (s *CycleService) Create(input *CreateCycleInput) (string, error) {
	if input.TeamID == "" {
		return "", fmt.Errorf("teamId is required")
	}
	if input.StartsAt == "" {
		return "", fmt.Errorf("startsAt is required")
	}
	if input.EndsAt == "" {
		return "", fmt.Errorf("endsAt is required")
	}

	// Resolve team identifier
	teamID, err := s.client.ResolveTeamIdentifier(input.TeamID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", input.TeamID, err)
	}

	linearInput := &core.CreateCycleInput{
		TeamID:      teamID,
		Name:        input.Name,
		Description: input.Description,
		StartsAt:    input.StartsAt,
		EndsAt:      input.EndsAt,
	}

	cycle, err := s.client.CreateCycle(linearInput)
	if err != nil {
		return "", fmt.Errorf("failed to create cycle: %w", err)
	}

	return s.formatter.Cycle(cycle, format.Full), nil
}

// AnalyzeInput represents input for cycle analysis
type AnalyzeInput struct {
	TeamID                string
	CycleCount            int
	AssigneeID            string
	IncludeRecommendation bool
}

// Analyze performs cycle analytics for capacity planning
func (s *CycleService) Analyze(input *AnalyzeInput) (string, error) {
	if input.TeamID == "" {
		return "", fmt.Errorf("teamId is required")
	}

	// Set defaults
	if input.CycleCount <= 0 || input.CycleCount > 100 {
		input.CycleCount = 10
	}

	// Resolve team identifier
	teamID, err := s.client.ResolveTeamIdentifier(input.TeamID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", input.TeamID, err)
	}

	// Resolve assignee if provided
	var assigneeID string
	if input.AssigneeID != "" {
		assigneeID, err = s.client.ResolveUserIdentifier(input.AssigneeID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve user '%s': %w", input.AssigneeID, err)
		}
	}

	// Get past cycles
	isPast := true
	result, err := s.client.ListCycles(&core.CycleFilter{
		TeamID: teamID,
		IsPast: &isPast,
		Limit:  input.CycleCount,
		Format: core.FormatFull,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list cycles: %w", err)
	}

	if result == nil || len(result.Cycles) == 0 {
		return "No completed cycles found for analysis.", nil
	}

	// Get full cycle data and filter issues by user if needed
	var fullCycles []*core.Cycle
	userIssuesMap := make(map[string][]core.Issue)

	for _, cycle := range result.Cycles {
		fullCycle, err := s.client.CycleClient().GetCycle(cycle.ID)
		if err != nil {
			continue
		}
		fullCycles = append(fullCycles, fullCycle)

		if assigneeID != "" {
			issues, err := s.client.CycleClient().GetCycleIssues(cycle.ID, 100)
			if err == nil {
				for _, issue := range issues {
					if issue.Assignee != nil && issue.Assignee.ID == assigneeID {
						userIssuesMap[cycle.ID] = append(userIssuesMap[cycle.ID], issue)
					}
				}
			}
		}
	}

	if len(fullCycles) == 0 {
		return "No cycle data available for analysis.", nil
	}

	// Calculate metrics
	var analysis *cycles.CycleAnalysis
	if assigneeID != "" {
		analysis = cycles.AnalyzeMultipleCycles(fullCycles, userIssuesMap)
	} else {
		analysis = cycles.AnalyzeMultipleCycles(fullCycles, nil)
	}

	// Get team name
	teamName := input.TeamID
	if len(fullCycles) > 0 && fullCycles[0].Team != nil {
		teamName = fullCycles[0].Team.Name
	}

	// Get assignee name
	var assigneeName string
	if assigneeID != "" && len(userIssuesMap) > 0 {
		for _, issues := range userIssuesMap {
			if len(issues) > 0 && issues[0].Assignee != nil {
				assigneeName = issues[0].Assignee.Name
				break
			}
		}
	}

	return s.formatter.CycleAnalysis(analysis, teamName, assigneeName, input.IncludeRecommendation), nil
}
