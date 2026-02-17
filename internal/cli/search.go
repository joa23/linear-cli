package cli

import (
	"fmt"
	"os"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/service"
	"github.com/spf13/cobra"
)

// IssueSearchOptions holds all parameters for issue search
type IssueSearchOptions struct {
	TextQuery   string
	Team        string
	Project     string
	State       string
	Priority    int
	Assignee    string
	Cycle       string
	Labels      string
	ExcludeLabels string
	BlockedBy   string
	Blocks      string
	HasBlockers bool
	HasDeps     bool
	HasCircular bool
	DepthMax    int
	Limit       int
	Format      string
	Output      string
}

func newSearchCmd() *cobra.Command {
	var (
		entityType  string
		textQuery   string

		// Standard issue filters
		team     string
		project  string
		state    string
		priority int
		assignee string
		cycle    string
		labels   string
		excludeLabels string

		// Dependency filters (NEW)
		blockedBy     string
		blocks        string
		hasBlockers   bool
		hasDeps       bool
		hasCircular   bool
		depthMax      int

		// Output
		limit      int
		formatStr  string
		outputType string
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search across Linear entities with dependency filtering",
		Long: `Unified search across issues, cycles, projects, and users with powerful dependency filtering.

ENTITY TYPES:
- issues (default) - Search issues with full filtering
- cycles - Search cycles by name/number
- projects - Search projects by name
- users - Search users by name/email
- all - Search across all entities

STANDARD FILTERS (issues only):
- --team <KEY> - Filter by team (uses .linear.yaml default)
- --state <name> - Filter by workflow state
- --priority <0-4> - Filter by priority (0=none, 1=urgent, 4=low)
- --assignee <email> - Filter by assignee
- --cycle <number> - Filter by cycle
- --labels <names> - Filter by labels (comma-separated)

DEPENDENCY FILTERS (issues only):
These help you discover and manage dependencies across your backlog:

- --blocked-by <ID> - Find issues blocked by a specific issue
  Example: Find all work blocked by a foundation task

- --blocks <ID> - Find issues that block a specific issue
  Example: Find all blockers preventing a feature from shipping

- --has-blockers - Find ALL issues with any blockers
  Example: Discover work that's stuck waiting on other issues

- --has-dependencies - Find issues that depend on others
  Example: Find complex work with prerequisite tasks

- --has-circular-deps - Find issues in circular dependency chains
  Example: Detect dependency cycles that prevent any work from completing

- --max-depth <n> - Filter by maximum dependency chain depth
  Example: Find simple vs complex dependency trees

USE CASES:
1. Unblock work: Find all blocked issues to prioritize unblocking
   → linear search --has-blockers --state "Todo" --team ENG

2. Discover hidden dependencies: Find issues that might be related
   → linear search "authentication" --has-dependencies

3. Clean up cycles: Find and break circular dependencies
   → linear search --has-circular-deps --team ENG

4. Prioritize foundation work: See what blocks the most issues
   → linear deps --team ENG  # then check what blocks most work

TIP: Use --format full for detailed output with descriptions.`,
		Example: `  # Search all issues
  linear search "authentication"

  # Search with filters
  linear search --state "In Progress" --priority 1

  # Find issues blocked by CEN-123
  linear search --blocked-by CEN-123

  # Find all issues with blockers
  linear search --has-blockers --team CEN

  # Find issues in circular dependencies
  linear search --has-circular-deps

  # Cross-entity search
  linear search "oauth" --type all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			// Parse text query
			if len(args) > 0 {
				textQuery = args[0]
			}

			// Use default team if not specified
			if team == "" {
				team = GetDefaultTeam()
			}

			if project == "" {
				project = GetDefaultProject()
			}

			// Route to appropriate search handler
			switch entityType {
			case "issues", "":
				return searchIssues(deps, IssueSearchOptions{
					TextQuery:   textQuery,
					Team:        team,
					Project:     project,
					State:       state,
					Priority:    priority,
					Assignee:    assignee,
					Cycle:       cycle,
					Labels:      labels,
					ExcludeLabels: excludeLabels,
					BlockedBy:   blockedBy,
					Blocks:      blocks,
					HasBlockers: hasBlockers,
					HasDeps:     hasDeps,
					HasCircular: hasCircular,
					DepthMax:    depthMax,
					Limit:       limit,
					Format:      formatStr,
					Output:      outputType,
				})
			case "cycles":
				return searchCycles(deps, textQuery, team, limit, formatStr, outputType)
			case "projects":
				return searchProjects(deps, textQuery, limit, formatStr, outputType)
			case "users":
				return searchUsers(deps, textQuery, team, limit, formatStr, outputType)
			case "all":
				return searchAll(deps, textQuery, team, limit, formatStr, outputType)
			default:
				return fmt.Errorf("invalid entity type: %s (must be: issues, cycles, projects, users, all)", entityType)
			}
		},
	}

	// Entity selection
	cmd.Flags().StringVar(&entityType, "type", "issues", "Entity type: issues|cycles|projects|users|all")

	// Standard issue filters
	cmd.Flags().StringVarP(&team, "team", "t", "", TeamFlagDescription)
	cmd.Flags().StringVarP(&project, "project", "P", "", ProjectFlagDescription)
	cmd.Flags().StringVar(&state, "state", "", "Filter by state (comma-separated)")
	cmd.Flags().IntVar(&priority, "priority", 0, "Filter by priority (0=none, 1=urgent, 2=high, 3=normal, 4=low)")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Filter by assignee")
	cmd.Flags().StringVarP(&cycle, "cycle", "c", "", "Filter by cycle")
	cmd.Flags().StringVarP(&labels, "labels", "l", "", "Filter by labels (comma-separated)")
	cmd.Flags().StringVarP(&excludeLabels, "exclude-labels", "L", "", "Exclude issues with these labels (comma-separated)")

	// Dependency filters (NEW)
	cmd.Flags().StringVar(&blockedBy, "blocked-by", "", "Issues blocked by this issue ID")
	cmd.Flags().StringVar(&blocks, "blocks", "", "Issues that block this issue ID")
	cmd.Flags().BoolVar(&hasBlockers, "has-blockers", false, "Issues with any blockers")
	cmd.Flags().BoolVar(&hasDeps, "has-dependencies", false, "Issues with dependencies")
	cmd.Flags().BoolVar(&hasCircular, "has-circular-deps", false, "Issues in circular deps")
	cmd.Flags().IntVar(&depthMax, "max-depth", 0, "Max dependency chain depth")

	// Output
	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Number of results")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

// searchIssues searches issues with optional dependency filtering
func searchIssues(deps *Dependencies, opts IssueSearchOptions) error {
	// Validate limit
	var err error
	opts.Limit, err = validateAndNormalizeLimit(opts.Limit)
	if err != nil {
		return err
	}

	// Parse format flags
	verbosity, err := format.ParseVerbosity(opts.Format)
	if err != nil {
		return err
	}
	outputType, err := format.ParseOutputType(opts.Output)
	if err != nil {
		return err
	}

	// Build search filters (using existing IssueService for consistency)
	filters := &service.SearchFilters{
		TeamID:     opts.Team,
		ProjectID:  opts.Project,
		StateIDs:   nil,
		Priority:   nil,
		AssigneeID: opts.Assignee,
		CycleID:    opts.Cycle,
		LabelIDs:   nil,
		SearchTerm: opts.TextQuery,
		Limit:      opts.Limit,
	}

	// Apply optional filters
	if opts.State != "" {
		filters.StateIDs = parseCommaSeparated(opts.State)
	}
	if opts.Priority > 0 {
		filters.Priority = &opts.Priority
	}
	if opts.Labels != "" {
		filters.LabelIDs = parseCommaSeparated(opts.Labels)
	}
	if opts.ExcludeLabels != "" {
		filters.ExcludeLabelIDs = parseCommaSeparated(opts.ExcludeLabels)
	}

	// For dependency filters, use the Search service
	if opts.BlockedBy != "" || opts.Blocks != "" || opts.HasBlockers || opts.HasDeps || opts.HasCircular || opts.DepthMax > 0 {
		// Build search options for Search service
		searchOpts := &service.SearchOptions{
			EntityType:  "issues",
			TextQuery:   opts.TextQuery,
			TeamID:      opts.Team,
			ProjectID:   opts.Project,
			StateIDs:    filters.StateIDs,
			Priority:    filters.Priority,
			AssigneeID:  opts.Assignee,
			CycleID:     opts.Cycle,
			LabelIDs:    filters.LabelIDs,
			BlockedBy:   opts.BlockedBy,
			Blocks:      opts.Blocks,
			HasBlockers: opts.HasBlockers,
			HasDeps:     opts.HasDeps,
			HasCircular: opts.HasCircular,
			MaxDepth:    opts.DepthMax,
			Limit:       opts.Limit,
			Format:      format.VerbosityToFormat(verbosity),
		}

		output, err := deps.Search.Search(searchOpts)
		if err != nil {
			return fmt.Errorf("failed to search issues: %w", err)
		}
		fmt.Println(output)
		return nil
	}

	// Use IssueService with new renderer for standard searches
	output, err := deps.Issues.SearchWithOutput(filters, verbosity, outputType)
	if err != nil {
		return fmt.Errorf("failed to search issues: %w", err)
	}

	fmt.Println(output)
	return nil
}

// searchCycles searches cycles by name/number
func searchCycles(deps *Dependencies, textQuery, team string, limit int, formatStr, outputType string) error {
	// Validate limit
	limit, err := validateAndNormalizeLimit(limit)
	if err != nil {
		return err
	}

	// Parse format flags
	verbosity, err := format.ParseVerbosity(formatStr)
	if err != nil {
		return err
	}
	output, err := format.ParseOutputType(outputType)
	if err != nil {
		return err
	}

	// Build cycle filters
	filters := &service.CycleFilters{
		TeamID: team,
		Limit:  limit,
	}

	result, err := deps.Cycles.SearchWithOutput(filters, verbosity, output)
	if err != nil {
		return fmt.Errorf("failed to search cycles: %w", err)
	}

	fmt.Println(result)
	return nil
}

// searchProjects searches projects by name
func searchProjects(deps *Dependencies, textQuery string, limit int, formatStr, outputType string) error {
	// Validate limit
	limit, err := validateAndNormalizeLimit(limit)
	if err != nil {
		return err
	}

	// Parse format flags
	verbosity, err := format.ParseVerbosity(formatStr)
	if err != nil {
		return err
	}
	output, err := format.ParseOutputType(outputType)
	if err != nil {
		return err
	}

	result, err := deps.Projects.ListAllWithOutput(limit, verbosity, output)
	if err != nil {
		return fmt.Errorf("failed to search projects: %w", err)
	}

	fmt.Println(result)
	return nil
}

// searchUsers searches users by name/email
func searchUsers(deps *Dependencies, textQuery, team string, limit int, formatStr, outputType string) error {
	// Validate limit
	limit, err := validateAndNormalizeLimit(limit)
	if err != nil {
		return err
	}

	// Parse format flags
	verbosity, err := format.ParseVerbosity(formatStr)
	if err != nil {
		return err
	}
	output, err := format.ParseOutputType(outputType)
	if err != nil {
		return err
	}

	// Build user filters
	filters := &service.UserFilters{
		TeamID: team,
		Limit:  limit,
	}

	result, err := deps.Users.SearchWithOutput(filters, verbosity, output)
	if err != nil {
		return fmt.Errorf("failed to search users: %w", err)
	}

	fmt.Println(result)
	return nil
}

// searchAll searches across all entity types
func searchAll(deps *Dependencies, textQuery, team string, limit int, formatStr, outputType string) error {
	// Search each entity type and combine results
	fmt.Printf("SEARCH RESULTS: \"%s\"\n", textQuery)
	fmt.Println(generateSeparator("═", 50))

	var errs []error

	// Search issues
	fmt.Println("\nISSUES")
	fmt.Println(generateSeparator("─", 50))
	if err := searchIssues(deps, IssueSearchOptions{
		TextQuery: textQuery,
		Team:      team,
		Limit:     limit,
		Format:    formatStr,
		Output:    outputType,
	}); err != nil {
		errs = append(errs, fmt.Errorf("issues: %w", err))
		_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to search issues: %v\n", err)
	}

	// Search cycles
	fmt.Println("\nCYCLES")
	fmt.Println(generateSeparator("─", 50))
	if err := searchCycles(deps, textQuery, team, limit, formatStr, outputType); err != nil {
		errs = append(errs, fmt.Errorf("cycles: %w", err))
		_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to search cycles: %v\n", err)
	}

	// Search projects
	fmt.Println("\nPROJECTS")
	fmt.Println(generateSeparator("─", 50))
	if err := searchProjects(deps, textQuery, limit, formatStr, outputType); err != nil {
		errs = append(errs, fmt.Errorf("projects: %w", err))
		_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to search projects: %v\n", err)
	}

	// Search users
	fmt.Println("\nUSERS")
	fmt.Println(generateSeparator("─", 50))
	if err := searchUsers(deps, textQuery, team, limit, formatStr, outputType); err != nil {
		errs = append(errs, fmt.Errorf("users: %w", err))
		_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to search users: %v\n", err)
	}

	// Return error if all searches failed
	if len(errs) == 4 {
		return fmt.Errorf("all searches failed")
	}

	// Warn if some searches failed
	if len(errs) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "\nWarning: %d search(es) failed\n", len(errs))
	}

	return nil
}

// Helper function to generate separator lines
func generateSeparator(char string, length int) string {
	result := ""
	for i := 0; i < length; i++ {
		result += char
	}
	return result
}

