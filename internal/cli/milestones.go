package cli

import (
	"fmt"
	"time"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/service"
	"github.com/spf13/cobra"
)

func newMilestonesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "milestones",
		Aliases: []string{"milestone", "m"},
		Short:   "Manage Linear project milestones",
		Long:    "List, view, create, update, and delete Linear project milestones.",
	}

	cmd.AddCommand(
		newMilestonesListCmd(),
		newMilestonesGetCmd(),
		newMilestonesCreateCmd(),
		newMilestonesUpdateCmd(),
		newMilestonesDeleteCmd(),
	)

	return cmd
}

func newMilestonesListCmd() *cobra.Command {
	var project, teamID, formatStr, outputType string
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List milestones for a project",
		Example: `  # List milestones for the default project
  linear milestones list

  # List milestones for a specific project
  linear milestones list --project "Q3 Launch"

  # Output as JSON
  linear milestones list --project "Q3 Launch" --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			if project == "" {
				project = GetDefaultProject()
			}
			if teamID == "" {
				teamID = GetDefaultTeam()
			}

			limit, err := validateAndNormalizeLimit(limit)
			if err != nil {
				return err
			}
			verbosity, err := format.ParseVerbosity(formatStr)
			if err != nil {
				return err
			}
			output, err := format.ParseOutputType(outputType)
			if err != nil {
				return err
			}

			result, err := deps.Milestones.List(&service.MilestoneListInput{
				ProjectID: project,
				TeamID:    teamID,
				Limit:     limit,
			}, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to list milestones: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&project, "project", "P", "", ProjectFlagDescription)
	cmd.Flags().StringVarP(&teamID, "team", "t", "", TeamFlagDescription)
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Number of milestones to return")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newMilestonesGetCmd() *cobra.Command {
	var project, teamID, formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "get <milestone-id-or-name>",
		Short: "Get milestone details",
		Example: `  # Get by UUID
  linear milestones get <id>

  # Get by name within a project
  linear milestones get "Beta" --project "Q3 Launch"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}
			if project == "" {
				project = GetDefaultProject()
			}
			if teamID == "" {
				teamID = GetDefaultTeam()
			}

			verbosity, err := format.ParseVerbosity(formatStr)
			if err != nil {
				return err
			}
			output, err := format.ParseOutputType(outputType)
			if err != nil {
				return err
			}

			result, err := deps.Milestones.Get(args[0], project, teamID, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to get milestone: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&project, "project", "P", "", ProjectFlagDescription)
	cmd.Flags().StringVarP(&teamID, "team", "t", "", TeamFlagDescription)
	cmd.Flags().StringVarP(&formatStr, "format", "f", "full", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newMilestonesCreateCmd() *cobra.Command {
	var project, teamID, description, targetDate, formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a milestone",
		Example: `  # Create milestone in default project
  linear milestones create "Beta"

  # Create with target date and description
  linear milestones create "Launch" --project "Q3 Launch" --target-date 2026-08-01 --description "Public launch"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}
			if project == "" {
				project = GetDefaultProject()
			}
			if teamID == "" {
				teamID = GetDefaultTeam()
			}

			desc, err := getDescriptionFromFlagOrStdin(description)
			if err != nil {
				return fmt.Errorf("failed to read description: %w", err)
			}
			if err := validateDateOnly(targetDate, "--target-date"); err != nil {
				return err
			}

			verbosity, err := format.ParseVerbosity(formatStr)
			if err != nil {
				return err
			}
			output, err := format.ParseOutputType(outputType)
			if err != nil {
				return err
			}

			result, err := deps.Milestones.Create(&service.CreateMilestoneInput{
				Name:        args[0],
				Description: desc,
				ProjectID:   project,
				TeamID:      teamID,
				TargetDate:  targetDate,
			}, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to create milestone: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&project, "project", "P", "", ProjectFlagDescription)
	cmd.Flags().StringVarP(&teamID, "team", "t", "", TeamFlagDescription)
	cmd.Flags().StringVarP(&description, "description", "d", "", "Milestone description (or - for stdin)")
	cmd.Flags().StringVar(&targetDate, "target-date", "", "Target date YYYY-MM-DD")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "full", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newMilestonesUpdateCmd() *cobra.Command {
	var name, project, teamID, description, targetDate, formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "update <milestone-id-or-name>",
		Short: "Update a milestone",
		Example: `  # Rename milestone
  linear milestones update "Beta" --project "Q3 Launch" --name "Private beta"

  # Update target date
  linear milestones update <id> --target-date 2026-08-15`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}
			if project == "" {
				project = GetDefaultProject()
			}
			if teamID == "" {
				teamID = GetDefaultTeam()
			}

			hasFlags := name != "" || description != "" || targetDate != ""
			if !hasFlags {
				return fmt.Errorf("no updates specified. Use flags like --name, --description, --target-date")
			}

			desc, err := getDescriptionFromFlagOrStdin(description)
			if err != nil {
				return fmt.Errorf("failed to read description: %w", err)
			}
			if err := validateDateOnly(targetDate, "--target-date"); err != nil {
				return err
			}

			verbosity, err := format.ParseVerbosity(formatStr)
			if err != nil {
				return err
			}
			output, err := format.ParseOutputType(outputType)
			if err != nil {
				return err
			}

			input := &service.UpdateMilestoneInput{TeamID: teamID, LookupProjectID: project}
			if name != "" {
				input.Name = &name
			}
			if desc != "" {
				input.Description = &desc
			}
			if targetDate != "" {
				input.TargetDate = &targetDate
			}

			result, err := deps.Milestones.Update(args[0], input, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to update milestone: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Update milestone name")
	cmd.Flags().StringVarP(&project, "project", "P", "", ProjectFlagDescription)
	cmd.Flags().StringVarP(&teamID, "team", "t", "", TeamFlagDescription)
	cmd.Flags().StringVarP(&description, "description", "d", "", "Update description (or - for stdin)")
	cmd.Flags().StringVar(&targetDate, "target-date", "", "Update target date YYYY-MM-DD")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "full", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newMilestonesDeleteCmd() *cobra.Command {
	var project, teamID string

	cmd := &cobra.Command{
		Use:   "delete <milestone-id-or-name>",
		Short: "Delete a milestone",
		Example: `  # Delete by UUID
  linear milestones delete <id>

  # Delete by name within a project
  linear milestones delete "Beta" --project "Q3 Launch"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}
			if project == "" {
				project = GetDefaultProject()
			}
			if teamID == "" {
				teamID = GetDefaultTeam()
			}

			result, err := deps.Milestones.Delete(args[0], project, teamID)
			if err != nil {
				return fmt.Errorf("failed to delete milestone: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&project, "project", "P", "", ProjectFlagDescription)
	cmd.Flags().StringVarP(&teamID, "team", "t", "", TeamFlagDescription)

	return cmd
}

func validateDateOnly(value string, flagName string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("invalid %s %q: expected YYYY-MM-DD", flagName, value)
	}
	return nil
}
