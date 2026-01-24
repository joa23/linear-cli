package service

import (
	"fmt"

	"github.com/dominikbraun/graph"
	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear/core"
)

// SearchService handles unified search operations across entities
type SearchService struct {
	client    SearchClientOperations
	formatter *format.Formatter
}

// NewSearchService creates a new SearchService
func NewSearchService(client SearchClientOperations, formatter *format.Formatter) *SearchService {
	return &SearchService{
		client:    client,
		formatter: formatter,
	}
}

// SearchOptions represents unified search parameters
type SearchOptions struct {
	EntityType string
	TextQuery  string

	// Standard filters
	TeamID     string
	StateIDs   []string
	Priority   *int
	AssigneeID string
	CycleID    string
	LabelIDs   []string

	// Dependency filters (NEW)
	BlockedBy   string // Issue ID that blocks results
	Blocks      string // Issue ID that results block
	HasBlockers bool   // Filter to issues with blockers
	HasDeps     bool   // Filter to issues with dependencies
	HasCircular bool   // Filter to issues in circular deps
	MaxDepth    int    // Max dependency depth

	// Pagination
	Limit  int
	After  string
	Format format.Format
}

// Search performs unified search across entities
func (s *SearchService) Search(opts *SearchOptions) (string, error) {
	if opts == nil {
		opts = &SearchOptions{}
	}

	switch opts.EntityType {
	case "issues", "":
		return s.searchIssues(opts)
	default:
		return "", fmt.Errorf("invalid entity type: %s", opts.EntityType)
	}
}

// searchIssues searches issues with dependency filtering
func (s *SearchService) searchIssues(opts *SearchOptions) (string, error) {
	// Set defaults
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Format == "" {
		opts.Format = format.Compact
	}

	// Build standard filters
	filters := &core.IssueSearchFilters{
		SearchTerm: opts.TextQuery,
		Limit:      opts.Limit,
		After:      opts.After,
		Format:     core.ResponseFormat(opts.Format),
	}

	// Resolve team identifier if provided
	if opts.TeamID != "" {
		teamID, err := s.client.ResolveTeamIdentifier(opts.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve team '%s': %w", opts.TeamID, err)
		}
		filters.TeamID = teamID
	}

	// Resolve assignee identifier if provided
	if opts.AssigneeID != "" {
		userID, err := s.client.ResolveUserIdentifier(opts.AssigneeID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve user '%s': %w", opts.AssigneeID, err)
		}
		filters.AssigneeID = userID
	}

	// Resolve cycle identifier if provided (requires team)
	if opts.CycleID != "" {
		if filters.TeamID == "" {
			return "", fmt.Errorf("teamId is required to resolve cycleId")
		}
		cycleID, err := s.client.ResolveCycleIdentifier(opts.CycleID, filters.TeamID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve cycle '%s': %w", opts.CycleID, err)
		}
		filters.CycleID = cycleID
	}

	// Copy remaining filters
	filters.StateIDs = opts.StateIDs
	filters.LabelIDs = opts.LabelIDs
	filters.Priority = opts.Priority

	// Execute initial search
	result, err := s.client.SearchIssues(filters)
	if err != nil {
		return "", fmt.Errorf("failed to search issues: %w", err)
	}

	// Apply dependency filters if any are set
	if opts.needsDependencyFiltering() {
		result.Issues, err = s.filterByDependencies(result.Issues, opts)
		if err != nil {
			return "", fmt.Errorf("failed to apply dependency filters: %w", err)
		}
	}

	// Format output
	pagination := &format.Pagination{
		HasNextPage: result.HasNextPage,
		EndCursor:   result.EndCursor,
	}

	return s.formatter.IssueList(result.Issues, opts.Format, pagination), nil
}

// needsDependencyFiltering checks if any dependency filters are set
func (opts *SearchOptions) needsDependencyFiltering() bool {
	return opts.BlockedBy != "" ||
		opts.Blocks != "" ||
		opts.HasBlockers ||
		opts.HasDeps ||
		opts.HasCircular ||
		opts.MaxDepth > 0
}

// filterByDependencies applies relationship-based filters
func (s *SearchService) filterByDependencies(issues []core.Issue, opts *SearchOptions) ([]core.Issue, error) {
	var filtered []core.Issue

	for _, issue := range issues {
		// Fetch full relations for this issue
		fullIssue, err := s.client.IssueClient().GetIssueWithRelations(issue.Identifier)
		if err != nil {
			// If we can't fetch relations, skip this issue
			continue
		}

		// Apply filters
		if opts.BlockedBy != "" {
			if !s.isBlockedBy(fullIssue, opts.BlockedBy) {
				continue
			}
		}

		if opts.Blocks != "" {
			if !s.doesBlock(fullIssue, opts.Blocks) {
				continue
			}
		}

		if opts.HasBlockers {
			if !s.hasAnyBlockers(fullIssue) {
				continue
			}
		}

		if opts.HasDeps {
			if !s.hasAnyDependencies(fullIssue) {
				continue
			}
		}

		if opts.HasCircular {
			if !s.hasCircularDep(fullIssue) {
				continue
			}
		}

		if opts.MaxDepth > 0 {
			depth := s.getDepChainDepth(fullIssue)
			if depth > opts.MaxDepth {
				continue
			}
		}

		filtered = append(filtered, issue)
	}

	return filtered, nil
}

// isBlockedBy checks if issue is blocked by the specified blocker
func (s *SearchService) isBlockedBy(issue *core.IssueWithRelations, blockerID string) bool {
	for _, rel := range issue.InverseRelations.Nodes {
		if rel.Type == core.RelationBlocks && rel.Issue != nil && rel.Issue.Identifier == blockerID {
			return true
		}
	}
	return false
}

// doesBlock checks if issue blocks the specified issue
func (s *SearchService) doesBlock(issue *core.IssueWithRelations, blockedID string) bool {
	for _, rel := range issue.Relations.Nodes {
		if rel.Type == core.RelationBlocks && rel.RelatedIssue != nil && rel.RelatedIssue.Identifier == blockedID {
			return true
		}
	}
	return false
}

// hasAnyBlockers checks if issue has any blocking issues
func (s *SearchService) hasAnyBlockers(issue *core.IssueWithRelations) bool {
	for _, rel := range issue.InverseRelations.Nodes {
		if rel.Type == core.RelationBlocks {
			return true
		}
	}
	return false
}

// hasAnyDependencies checks if issue depends on any other issues
func (s *SearchService) hasAnyDependencies(issue *core.IssueWithRelations) bool {
	for _, rel := range issue.Relations.Nodes {
		if rel.Type == core.RelationBlocks {
			return true
		}
	}
	return false
}

// hasCircularDep checks if issue is part of a circular dependency chain
func (s *SearchService) hasCircularDep(issue *core.IssueWithRelations) bool {
	// Build a graph from this issue's relations
	nodes := make(map[string]bool)
	var edges []depEdge

	// Add this issue
	nodes[issue.Identifier] = true

	// Add outgoing relations (what this blocks)
	for _, rel := range issue.Relations.Nodes {
		if rel.Type == core.RelationBlocks && rel.RelatedIssue != nil {
			nodes[rel.RelatedIssue.Identifier] = true
			edges = append(edges, depEdge{
				from: issue.Identifier,
				to:   rel.RelatedIssue.Identifier,
			})
		}
	}

	// Add incoming relations (what blocks this)
	for _, rel := range issue.InverseRelations.Nodes {
		if rel.Type == core.RelationBlocks && rel.Issue != nil {
			nodes[rel.Issue.Identifier] = true
			edges = append(edges, depEdge{
				from: rel.Issue.Identifier,
				to:   issue.Identifier,
			})
		}
	}

	// Detect cycles using graph library
	cycles := detectCyclesInGraph(nodes, edges)

	// Check if this issue is in any cycle
	for _, cycle := range cycles {
		for _, nodeID := range cycle {
			if nodeID == issue.Identifier {
				return true
			}
		}
	}

	return false
}

// getDepChainDepth calculates the maximum dependency chain depth
func (s *SearchService) getDepChainDepth(issue *core.IssueWithRelations) int {
	visited := make(map[string]bool)
	return s.calculateDepth(issue.Identifier, visited, 0)
}

// calculateDepth recursively calculates max depth
func (s *SearchService) calculateDepth(issueID string, visited map[string]bool, currentDepth int) int {
	if visited[issueID] {
		return currentDepth
	}
	visited[issueID] = true

	// Get issue with relations
	fullIssue, err := s.client.IssueClient().GetIssueWithRelations(issueID)
	if err != nil {
		return currentDepth
	}

	maxDepth := currentDepth
	for _, rel := range fullIssue.Relations.Nodes {
		if rel.Type == core.RelationBlocks && rel.RelatedIssue != nil {
			depth := s.calculateDepth(rel.RelatedIssue.Identifier, visited, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	return maxDepth
}

// depEdge represents a dependency edge
type depEdge struct {
	from string
	to   string
}

// detectCyclesInGraph detects cycles in a dependency graph
func detectCyclesInGraph(nodes map[string]bool, edges []depEdge) [][]string {
	// Build graph using dominikbraun/graph
	g := graph.New(graph.StringHash, graph.Directed())

	// Add vertices
	for nodeID := range nodes {
		_ = g.AddVertex(nodeID)
	}

	// Add edges
	for _, e := range edges {
		_ = g.AddEdge(e.from, e.to)
	}

	// Detect strongly connected components
	components, _ := graph.StronglyConnectedComponents(g)

	// Filter to only cycles (components with more than 1 node)
	var cycles [][]string
	for _, component := range components {
		if len(component) > 1 {
			cycles = append(cycles, component)
		} else if len(component) == 1 {
			// Check for self-loop
			nodeID := component[0]
			for _, e := range edges {
				if e.from == nodeID && e.to == nodeID {
					cycles = append(cycles, []string{nodeID})
					break
				}
			}
		}
	}

	return cycles
}
