package cli

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/spf13/cobra"
)

func newTeamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "teams",
		Aliases: []string{"t", "team"},
		Short:   "Manage Linear teams",
		Long:    "List teams and view team details, labels, and workflow states.",
	}

	cmd.AddCommand(
		newTeamsListCmd(),
		newTeamsGetCmd(),
		newTeamsLabelsCmd(),
		newTeamsStatesCmd(),
	)

	return cmd
}

func newTeamsListCmd() *cobra.Command {
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all teams",
		Long:  "List all teams in your workspace.",
		Example: `  # List all teams
  linear teams list

  # Output as JSON
  linear teams list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			result, err := deps.Teams.ListAllWithOutput(verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to list teams: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newTeamsGetCmd() *cobra.Command {
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "get <team-id>",
		Short: "Get team details",
		Long:  "Display detailed information about a specific team.",
		Example: `  # Get team details
  linear teams get CEN

  # Output as JSON
  linear teams get CEN --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]

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

			result, err := deps.Teams.GetWithOutput(teamID, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to get team: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&formatStr, "format", "f", "full", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newTeamsLabelsCmd() *cobra.Command {
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "labels <team-id>",
		Short: "List team labels",
		Long:  "List all labels available for a team.",
		Example: `  # List team labels
  linear teams labels CEN

  # Output as JSON
  linear teams labels CEN --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]

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

			result, err := deps.Teams.GetLabelsWithOutput(teamID, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to get labels: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newTeamsStatesCmd() *cobra.Command {
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "states <team-id>",
		Short: "List workflow states",
		Long:  "List all workflow states for a team.",
		Example: `  # List team workflow states
  linear teams states CEN

  # Output as JSON
  linear teams states CEN --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]

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

			result, err := deps.Teams.GetWorkflowStatesWithOutput(teamID, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to get workflow states: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}
