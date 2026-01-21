package cli

import (
	"fmt"

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

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		Long:  "List users in the workspace or a specific team.",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getUserService()
			if err != nil {
				return err
			}

			filters := &service.UserFilters{
				TeamID: teamID,
				Limit:  50,
			}

			output, err := svc.Search(filters)
			if err != nil {
				return fmt.Errorf("failed to list users: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamID, "team", "", "Filter by team ID")

	return cmd
}

func newUsersGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <user-id>",
		Short: "Get user details",
		Long:  "Display detailed information about a specific user.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID := args[0]

			svc, err := getUserService()
			if err != nil {
				return err
			}

			output, err := svc.Get(userID)
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}
}

func newUsersMeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show current user",
		Long:  "Display information about the currently authenticated user.",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getUserService()
			if err != nil {
				return err
			}

			output, err := svc.GetViewer()
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}
}

func getUserService() (*service.UserService, error) {
	client, err := getLinearClient()
	if err != nil {
		return nil, err
	}
	return service.New(client).Users, nil
}
