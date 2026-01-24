package cli

import (
	"errors"
	"fmt"

	"github.com/joa23/linear-cli/internal/service"
	"github.com/spf13/cobra"
)

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"p", "project"},
		Short:   "Manage Linear projects",
		Long:    "List, view, create, and update Linear projects.",
	}

	cmd.AddCommand(
		newProjectsListCmd(),
		newProjectsGetCmd(),
		newProjectsCreateCmd(),
		newProjectsUpdateCmd(),
	)

	return cmd
}

func newProjectsListCmd() *cobra.Command {
	var mine bool
	var teamID string
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects for a team",
		Long:  "List projects for a specific team.",
		Example: `  # List projects for team (uses .linear.yaml)
  linear projects list

  # List projects for specific team
  linear projects list --team CEN

  # List projects you're involved in
  linear projects list --mine

  # List with custom limit
  linear projects list --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := getProjectService()
			if err != nil {
				return err
			}

			// Set default limit
			limit, err := validateAndNormalizeLimit(limit)
			if err != nil {
				return err
			}

			var output string
			if mine {
				// --mine overrides team requirement
				output, err = svc.ListUserProjects(limit)
			} else {
				// Get team from flag or config
				if teamID == "" {
					teamID = GetDefaultTeam()
				}
				if teamID == "" {
					return errors.New(ErrTeamRequired)
				}

				output, err = svc.ListByTeam(teamID, limit)
			}
			if err != nil {
				return fmt.Errorf("failed to list projects: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}

	cmd.Flags().BoolVar(&mine, "mine", false, "Only show projects you're involved in (ignores team)")
	cmd.Flags().StringVarP(&teamID, "team", "t", "", TeamFlagDescription)
	cmd.Flags().IntVarP(&limit, "limit", "n", 25, "Number of projects to return")

	return cmd
}

func newProjectsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <project-id>",
		Short: "Get project details",
		Long:  "Display detailed information about a specific project.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			svc, err := getProjectService()
			if err != nil {
				return err
			}

			output, err := svc.Get(projectID)
			if err != nil {
				return fmt.Errorf("failed to get project: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}
}

func newProjectsCreateCmd() *cobra.Command {
	var (
		team        string
		description string
		state       string
		lead        string
		startDate   string
		endDate     string
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new project",
		Long:  `Create a new project. States: planned, started, paused, completed, canceled.`,
		Example: `  # Create a simple project
  linear projects create "Q1 Release" --team CEN

  # Create with description from stdin
  cat project-spec.md | linear projects create "Q1 Release" --team CEN

  # Create with all options
  linear projects create "Q1 Release" --team CEN --state started --lead Stefan`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get team from flag or config
			if team == "" {
				team = GetDefaultTeam()
			}
			if team == "" {
				return errors.New(ErrTeamRequired)
			}

			// Get description from flag or stdin
			desc, err := getDescriptionFromFlagOrStdin(description)
			if err != nil {
				return fmt.Errorf("failed to read description: %w", err)
			}

			// Build create input
			input := &service.CreateProjectInput{
				Name:        name,
				TeamID:      team,
				Description: desc,
			}

			// Set optional fields
			if state != "" {
				input.State = state
			}
			if lead != "" {
				// Resolve lead to user ID
				userSvc, err := getUserService()
				if err != nil {
					return err
				}
				leadID, err := userSvc.ResolveByName(lead)
				if err != nil {
					return fmt.Errorf("failed to resolve lead '%s': %w", lead, err)
				}
				input.LeadID = leadID
			}
			if startDate != "" {
				input.StartDate = startDate
			}
			if endDate != "" {
				input.EndDate = endDate
			}

			svc, err := getProjectService()
			if err != nil {
				return err
			}

			output, err := svc.Create(input)
			if err != nil {
				return fmt.Errorf("failed to create project: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}

	// Add flags (with short versions for common flags)
	cmd.Flags().StringVarP(&team, "team", "t", "", TeamFlagDescription)
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description (or pipe to stdin)")
	cmd.Flags().StringVarP(&state, "state", "s", "", "Project state: planned, started, paused, completed, canceled")
	cmd.Flags().StringVarP(&lead, "lead", "l", "", "Project lead name (use 'me' for yourself)")
	cmd.Flags().StringVar(&startDate, "start-date", "", "Start date YYYY-MM-DD")
	cmd.Flags().StringVar(&endDate, "end-date", "", "Target end date YYYY-MM-DD")

	return cmd
}

func newProjectsUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		state       string
		lead        string
		startDate   string
		endDate     string
	)

	cmd := &cobra.Command{
		Use:   "update <project-id>",
		Short: "Update an existing project",
		Long:  `Update an existing project. Only provided flags are changed.`,
		Example: `  # Update project state
  linear projects update PROJ-123 --state completed

  # Update project lead
  linear projects update PROJ-123 --lead john@example.com

  # Update description from stdin
  cat updated-spec.md | linear projects update PROJ-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			// Check if any updates provided (stdin counts as description update)
			hasStdin := hasStdinPipe()
			hasFlags := name != "" || description != "" || state != "" ||
				lead != "" || startDate != "" || endDate != ""

			if !hasFlags && !hasStdin {
				return fmt.Errorf("no updates specified. Use flags like --state, --lead, etc")
			}

			// Get description from flag or stdin
			desc, err := getDescriptionFromFlagOrStdin(description)
			if err != nil {
				return fmt.Errorf("failed to read description: %w", err)
			}

			// Build update input
			input := &service.UpdateProjectInput{}

			if name != "" {
				input.Name = &name
			}
			if desc != "" {
				input.Description = &desc
			}
			if state != "" {
				input.State = &state
			}
			if lead != "" {
				// Resolve lead to user ID
				userSvc, err := getUserService()
				if err != nil {
					return err
				}
				leadID, err := userSvc.ResolveByName(lead)
				if err != nil {
					return fmt.Errorf("failed to resolve lead '%s': %w", lead, err)
				}
				input.LeadID = &leadID
			}
			if startDate != "" {
				input.StartDate = &startDate
			}
			if endDate != "" {
				input.EndDate = &endDate
			}

			svc, err := getProjectService()
			if err != nil {
				return err
			}

			output, err := svc.Update(projectID, input)
			if err != nil {
				return fmt.Errorf("failed to update project: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}

	// Add flags (with short versions for common flags)
	cmd.Flags().StringVarP(&name, "name", "n", "", "Update project name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Update description (or pipe to stdin)")
	cmd.Flags().StringVarP(&state, "state", "s", "", "Update state: planned, started, paused, completed, canceled")
	cmd.Flags().StringVarP(&lead, "lead", "l", "", "Update project lead (use 'me' for yourself)")
	cmd.Flags().StringVar(&startDate, "start-date", "", "Update start date YYYY-MM-DD")
	cmd.Flags().StringVar(&endDate, "end-date", "", "Update target end date YYYY-MM-DD")

	return cmd
}

func getProjectService() (*service.ProjectService, error) {
	client, err := getLinearClient()
	if err != nil {
		return nil, err
	}
	return service.New(client).Projects, nil
}
