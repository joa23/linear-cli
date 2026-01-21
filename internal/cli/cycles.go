package cli

import (
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

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cycles",
		Long:  "List cycles for a team. Uses default team from .linear.yaml if not specified.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use default team if not specified
			if teamID == "" {
				teamID = GetDefaultTeam()
			}
			if teamID == "" {
				return fmt.Errorf("--team is required (or run 'linear init' to set a default)")
			}

			svc, err := getCycleService()
			if err != nil {
				return err
			}

			filters := &service.CycleFilters{
				TeamID: teamID,
				Limit:  25,
				Format: format.Compact,
			}
			if activeOnly {
				filters.IsActive = &activeOnly
			}

			output, err := svc.Search(filters)
			if err != nil {
				return fmt.Errorf("failed to list cycles: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamID, "team", "", "Team ID or key (uses .linear.yaml default)")
	cmd.Flags().BoolVar(&activeOnly, "active", false, "Only show active cycles")

	return cmd
}

func newCyclesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <cycle-id>",
		Short: "Get cycle details",
		Long:  "Display detailed information about a specific cycle.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cycleID := args[0]

			svc, err := getCycleService()
			if err != nil {
				return err
			}

			output, err := svc.Get(cycleID, format.Full)
			if err != nil {
				return fmt.Errorf("failed to get cycle: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}
}

func newCyclesAnalyzeCmd() *cobra.Command {
	var teamID string
	var cycleCount int
	var assigneeID string

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze cycle velocity",
		Long:  "Analyze historical cycles to understand team velocity and get scope recommendations.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use default team if not specified
			if teamID == "" {
				teamID = GetDefaultTeam()
			}
			if teamID == "" {
				return fmt.Errorf("--team is required (or run 'linear init' to set a default)")
			}

			svc, err := getCycleService()
			if err != nil {
				return err
			}

			input := &service.AnalyzeInput{
				TeamID:                teamID,
				CycleCount:            cycleCount,
				AssigneeID:            assigneeID,
				IncludeRecommendation: true,
			}

			output, err := svc.Analyze(input)
			if err != nil {
				return fmt.Errorf("failed to analyze cycles: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamID, "team", "", "Team ID or key (uses .linear.yaml default)")
	cmd.Flags().IntVar(&cycleCount, "count", 10, "Number of cycles to analyze")
	cmd.Flags().StringVar(&assigneeID, "assignee", "", "Filter by assignee ID")

	return cmd
}

func getCycleService() (*service.CycleService, error) {
	client, err := getLinearClient()
	if err != nil {
		return nil, err
	}
	return service.New(client).Cycles, nil
}
