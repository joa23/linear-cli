package cli

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/service"
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
	return &cobra.Command{
		Use:   "list",
		Short: "List all teams",
		Long:  "List all teams in your workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getTeamService()
			if err != nil {
				return err
			}

			output, err := svc.ListAll()
			if err != nil {
				return fmt.Errorf("failed to list teams: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}
}

func newTeamsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <team-id>",
		Short: "Get team details",
		Long:  "Display detailed information about a specific team.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]

			svc, err := getTeamService()
			if err != nil {
				return err
			}

			output, err := svc.Get(teamID)
			if err != nil {
				return fmt.Errorf("failed to get team: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}
}

func newTeamsLabelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "labels <team-id>",
		Short: "List team labels",
		Long:  "List all labels available for a team.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]

			svc, err := getTeamService()
			if err != nil {
				return err
			}

			output, err := svc.GetLabels(teamID)
			if err != nil {
				return fmt.Errorf("failed to get labels: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}
}

func newTeamsStatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "states <team-id>",
		Short: "List workflow states",
		Long:  "List all workflow states for a team.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamID := args[0]

			svc, err := getTeamService()
			if err != nil {
				return err
			}

			output, err := svc.GetWorkflowStates(teamID)
			if err != nil {
				return fmt.Errorf("failed to get workflow states: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}
}

func getTeamService() (*service.TeamService, error) {
	client, err := getLinearClient()
	if err != nil {
		return nil, err
	}
	return service.New(client).Teams, nil
}
