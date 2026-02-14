package cli

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/service"
	"github.com/spf13/cobra"
)

func newUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"u", "user"},
		Short:   "Manage Linear users",
		Long:    "List users and view user details.",
	}

	cmd.AddCommand(
		newUsersListCmd(),
		newUsersGetCmd(),
		newUsersMeCmd(),
	)

	return cmd
}

func newUsersListCmd() *cobra.Command {
	var teamID string
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		Long:  "List users in the workspace or a specific team.",
		Example: `  # List all users
  linear users list

  # List users in a team
  linear users list --team CEN

  # Output as JSON
  linear users list --output json`,
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

			filters := &service.UserFilters{
				TeamID: teamID,
				Limit:  50,
			}

			result, err := deps.Users.SearchWithOutput(filters, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to list users: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamID, "team", "", TeamFlagDescription)
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newUsersGetCmd() *cobra.Command {
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "get <user-id>",
		Short: "Get user details",
		Long:  "Display detailed information about a specific user.",
		Example: `  # Get user details
  linear users get john@example.com

  # Output as JSON
  linear users get john@example.com --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID := args[0]

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

			result, err := deps.Users.GetWithOutput(userID, verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&formatStr, "format", "f", "full", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newUsersMeCmd() *cobra.Command {
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show current user",
		Long:  "Display information about the currently authenticated user.",
		Example: `  # Show current user
  linear users me

  # Output as JSON
  linear users me --output json`,
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

			result, err := deps.Users.GetViewerWithOutput(verbosity, output)
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&formatStr, "format", "f", "full", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}
