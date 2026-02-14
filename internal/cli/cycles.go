package cli

import (
	"errors"
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/service"
	"github.com/spf13/cobra"
)

func newCyclesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cycles",
		Aliases: []string{"c", "cycle"},
		Short:   "Manage Linear cycles",
		Long:    "List, view, and analyze Linear cycles (sprints).",
	}

	cmd.AddCommand(
		newCyclesListCmd(),
		newCyclesGetCmd(),
		newCyclesAnalyzeCmd(),
	)

	return cmd
}

func newCyclesListCmd() *cobra.Command {
	var teamID string
	var activeOnly bool
	var limit int
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cycles for a team",
		Long: `List cycles (sprints) for a team.

REQUIRED:
- Team (via 'linear init' or --team flag)

OPTIONAL:
- --active: Filter to only active cycles
- --limit: Number of cycles to return (default: 25)
- --format: Verbosity level (minimal|compact|detailed|full)
- --output: Output format (text|json)

TIP: Run 'linear init' to set default team.`,
		Example: `  # Minimal - list all cycles (requires 'linear init')
  linear cycles list

  # With explicit team
  linear cycles list --team CEN

  # Only active cycles
  linear cycles list --active

  # Specific team, active only, with limit
  linear cycles list --team CEN --active --limit 10

  # List more cycles
  linear cycles list --limit 50

  # Output as JSON
  linear cycles list --output json`,
		Annotations: map[string]string{
			"required": "team (via init or --team flag)",
			"optional": "--active, --limit, --format, --output flags",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use default team if not specified
			if teamID == "" {
				teamID = GetDefaultTeam()
			}
			if teamID == "" {
				return errors.New(ErrTeamRequired)
			}

			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			// Set default limit if not specified
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

			filters := &service.CycleFilters{
				TeamID: teamID,
				Limit:  limit,
			}
			if activeOnly {
				filters.IsActive = &activeOnly
			}

			result, err := deps.Cycles.SearchWithOutput(filters, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to list cycles: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamID, "team", "", TeamFlagDescription)
	cmd.Flags().BoolVar(&activeOnly, "active", false, "Only show active cycles")
	cmd.Flags().IntVarP(&limit, "limit", "n", 25, "Number of cycles to return (default 25)")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newCyclesGetCmd() *cobra.Command {
	var teamID string
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "get <cycle-id>",
		Short: "Get cycle details by number or UUID",
		Long: `Get details for a specific cycle.

CYCLE ID FORMATS:
- Number: 65 (requires team context from 'linear init')
- UUID: cycle-abc123-def456 (works without team)
- Name: "Sprint 2024-01" (requires team context)

REQUIRED:
- Cycle ID/number (positional argument)
- Team context (from 'linear init' or --team flag) if using numbers/names

OPTIONAL:
- --format: Verbosity level (minimal|compact|detailed|full)
- --output: Output format (text|json)

TIP: Run 'linear init' once to set default team, then use cycle numbers directly.`,
		Example: `  # Minimal - get by number (requires 'linear init')
  linear cycles get 65

  # With explicit team
  linear cycles get 65 --team CEN

  # Get by UUID (no team needed)
  linear cycles get cycle-abc123-def456-789

  # Get by name
  linear cycles get "Sprint 2024-01" --team CEN

  # Output as JSON
  linear cycles get 65 --output json`,
		Annotations: map[string]string{
			"required": "cycle-id, team (via init or --team flag for numbers/names)",
			"optional": "--team, --format, --output flags",
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cycleID := args[0]

			// Use default team if not specified
			if teamID == "" {
				teamID = GetDefaultTeam()
			}

			// Check if cycleID looks like a number and team is missing
			if teamID == "" && looksLikeCycleNumber(cycleID) {
				return fmt.Errorf("team context required for cycle number %s. "+
					"Run 'linear init' to set default team, use --team flag, or use cycle UUID instead of number", cycleID)
			}

			deps, err := getDeps(cmd)
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

			result, err := deps.Cycles.GetWithOutput(cycleID, teamID, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to get cycle: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamID, "team", "", TeamFlagDescription)
	cmd.Flags().StringVarP(&formatStr, "format", "f", "full", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newCyclesAnalyzeCmd() *cobra.Command {
	var teamID string
	var cycleCount int
	var assigneeID string
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze team velocity and cycle completion rates",
		Long: `Analyze historical cycle performance to inform planning.

Calculates:
- Average velocity (points/cycle)
- Completion rate (% of scoped work completed)
- Scope creep (% of work added mid-cycle)
- Recommended capacity for next cycles (P20/P50/P80)

REQUIRED:
- Team (via 'linear init' or --team flag)

OPTIONAL:
- --count: Number of past cycles to analyze (default: 10)
- --assignee: Filter by specific assignee
- --format: Verbosity level (minimal|compact|detailed|full)
- --output: Output format (text|json)

USE THIS BEFORE PLANNING: Always run analyze before planning cycles to understand capacity.`,
		Example: `  # Minimal - analyze last 10 cycles (requires 'linear init')
  linear cycles analyze

  # Complete - with all parameters
  linear cycles analyze --team CEN --count 15 --assignee stefan@centrum-ai.com

  # Common pattern - analyze for planning
  linear cycles analyze --team CEN --count 10

  # Analyze specific team member
  linear cycles analyze --assignee me --count 5

  # Output as JSON
  linear cycles analyze --output json`,
		Annotations: map[string]string{
			"required": "team (via init or --team flag)",
			"optional": "--count, --assignee, --format, --output flags",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use default team if not specified
			if teamID == "" {
				teamID = GetDefaultTeam()
			}
			if teamID == "" {
				return errors.New(ErrTeamRequired)
			}

			deps, err := getDeps(cmd)
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

			input := &service.AnalyzeInput{
				TeamID:                teamID,
				CycleCount:            cycleCount,
				AssigneeID:            assigneeID,
				IncludeRecommendation: true,
			}

			result, err := deps.Cycles.AnalyzeWithOutput(input, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to analyze cycles: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamID, "team", "", TeamFlagDescription)
	cmd.Flags().IntVar(&cycleCount, "count", 10, "Number of cycles to analyze")
	cmd.Flags().StringVar(&assigneeID, "assignee", "", "Filter by assignee ID")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}
