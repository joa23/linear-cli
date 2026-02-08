package cli

import (
	"errors"
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/spf13/cobra"
)

func newLabelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "labels",
		Aliases: []string{"label"},
		Short:   "Manage Linear labels",
		Long:    "List, create, update, and delete Linear labels.",
	}

	cmd.AddCommand(
		newLabelsListCmd(),
		newLabelsCreateCmd(),
		newLabelsUpdateCmd(),
		newLabelsDeleteCmd(),
	)

	return cmd
}

func newLabelsListCmd() *cobra.Command {
	var teamID, formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List labels for a team",
		Long:  "List all labels available for a team.",
		Example: `  # List labels (uses .linear.yaml default team)
  linear labels list

  # List labels for specific team
  linear labels list --team TEC

  # Output as JSON
  linear labels list --team TEC --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			if teamID == "" {
				teamID = GetDefaultTeam()
			}
			if teamID == "" {
				return errors.New(ErrTeamRequired)
			}

			verbosity, err := format.ParseVerbosity(formatStr)
			if err != nil {
				return err
			}
			output, err := format.ParseOutputType(outputType)
			if err != nil {
				return err
			}

			result, err := deps.Labels.List(teamID, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to list labels: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&teamID, "team", "t", "", TeamFlagDescription)
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newLabelsCreateCmd() *cobra.Command {
	var (
		teamID      string
		color       string
		description string
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new label",
		Long:  "Create a new label for a team.",
		Example: `  # Create a label
  linear labels create "needs-review" --team TEC

  # Create with color and description
  linear labels create "needs-review" --team TEC --color "#ff0000" --description "PR needs review"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			if teamID == "" {
				teamID = GetDefaultTeam()
			}
			if teamID == "" {
				return errors.New(ErrTeamRequired)
			}

			input := &core.CreateLabelInput{
				Name:        name,
				TeamID:      teamID,
				Color:       color,
				Description: description,
			}

			result, err := deps.Labels.Create(input)
			if err != nil {
				return fmt.Errorf("failed to create label: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&teamID, "team", "t", "", TeamFlagDescription)
	cmd.Flags().StringVar(&color, "color", "", "Label color as hex (e.g., #ff0000)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Label description")

	return cmd
}

func newLabelsUpdateCmd() *cobra.Command {
	var (
		name        string
		color       string
		description string
	)

	cmd := &cobra.Command{
		Use:   "update <label-id>",
		Short: "Update an existing label",
		Long:  "Update an existing label. Only provided flags are changed.",
		Example: `  # Update label name
  linear labels update <id> --name "needs-code-review"

  # Update label color
  linear labels update <id> --color "#ff6600"

  # Update multiple fields
  linear labels update <id> --name "urgent" --color "#ff0000" --description "Urgent issues"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			labelID := args[0]
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			hasFlags := name != "" || color != "" || description != ""
			if !hasFlags {
				return fmt.Errorf("no updates specified. Use flags like --name, --color, --description")
			}

			input := &core.UpdateLabelInput{}
			if name != "" {
				input.Name = &name
			}
			if color != "" {
				input.Color = &color
			}
			if description != "" {
				input.Description = &description
			}

			result, err := deps.Labels.Update(labelID, input)
			if err != nil {
				return fmt.Errorf("failed to update label: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Update label name")
	cmd.Flags().StringVar(&color, "color", "", "Update label color as hex (e.g., #ff0000)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Update label description")

	return cmd
}

func newLabelsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <label-id>",
		Short: "Delete a label",
		Long:  "Delete a label by its ID.",
		Example: `  # Delete a label
  linear labels delete <id>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			labelID := args[0]
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			result, err := deps.Labels.Delete(labelID)
			if err != nil {
				return fmt.Errorf("failed to delete label: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	return cmd
}
