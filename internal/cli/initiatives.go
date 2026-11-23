package cli

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/service"
	"github.com/spf13/cobra"
)

// initiativeRefHelp documents the accepted reference forms, shared by command help.
const initiativeRefHelp = `Initiative references accept a UUID, a slug, the full Linear initiative URL,
or the exact initiative name (case-insensitive):
  linear initiatives get 3f2a9c1e-...
  linear initiatives get q1-platform-3c7a2b1d9e0f
  linear initiatives get https://linear.app/acme/initiative/q1-platform-3c7a2b1d9e0f
  linear initiatives get "Q1 Platform"`

func newInitiativesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "initiatives",
		Aliases: []string{"initiative"},
		Short:   "Manage Linear initiatives",
		Long: `Manage Linear initiatives: owned collections of projects with a status,
health, target date, and status updates.

` + initiativeRefHelp,
	}

	cmd.AddCommand(
		newInitiativesListCmd(),
		newInitiativesGetCmd(),
		newInitiativesCreateCmd(),
		newInitiativesUpdateCmd(),
		newInitiativesArchiveCmd(),
		newInitiativesUnarchiveCmd(),
		newInitiativesDeleteCmd(),
		newInitiativesAddProjectCmd(),
		newInitiativesRemoveProjectCmd(),
		newInitiativeUpdatesCmd(),
	)

	return cmd
}

func newInitiativesListCmd() *cobra.Command {
	var (
		status          string
		owner           string
		query           string
		includeArchived bool
		limit           int
		formatStr       string
		outputType      string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List initiatives",
		Long:  `List initiatives in the workspace, most recently updated first. Filters combine with AND.`,
		Example: `  # All initiatives
  linear initiatives list

  # Active initiatives you own
  linear initiatives list --status Active --owner me

  # Search by name
  linear initiatives list --query "platform"

  # JSON output for automation
  linear initiatives list --output json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
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

			result, err := deps.Initiatives.List(&service.InitiativeListParams{
				Status:          status,
				Owner:           owner,
				Query:           query,
				IncludeArchived: includeArchived,
				Limit:           limit,
				Verbosity:       verbosity,
				OutputType:      output,
			})
			if err != nil {
				return err
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status: Planned|Active|Completed")
	cmd.Flags().StringVar(&owner, "owner", "", "Filter by owner (me, name, or email)")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Match name (case-insensitive substring)")
	cmd.Flags().BoolVar(&includeArchived, "include-archived", false, "Include archived initiatives")
	cmd.Flags().IntVarP(&limit, "limit", "n", DefaultLimit, "Maximum number of initiatives (max 250)")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newInitiativesGetCmd() *cobra.Command {
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "get <id|slug|url|name>",
		Short: "Get an initiative with its projects and recent updates",
		Long: `Show an initiative. compact: metadata; detailed: + content preview, linked projects,
sub-initiatives, recent status updates; full: untruncated content and updates.

` + initiativeRefHelp,
		Example: `  linear initiatives get "Q1 Platform"
  linear initiatives get q1-platform-3c7a2b1d9e0f --format compact
  linear initiatives get https://linear.app/acme/initiative/q1-platform-3c7a2b1d9e0f --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
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

			result, err := deps.Initiatives.Get(args[0], verbosity, output)
			if err != nil {
				return err
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&formatStr, "format", "f", "full", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newInitiativesCreateCmd() *cobra.Command {
	var (
		description string
		content     string
		contentFile string
		owner       string
		status      string
		targetDate  string
		icon        string
		color       string
		projects    []string
		formatStr   string
		outputType  string
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an initiative",
		Long: `Create an initiative. --project links existing projects (repeat the flag or pass a
comma-separated list); projects are resolved by name or UUID before anything is created.`,
		Example: `  # Minimal
  linear initiatives create "Q1 Platform"

  # Full
  linear initiatives create "Q1 Platform" \
    --description "Stabilize the platform" \
    --content-file brief.md \
    --owner me --status Active --target-date 2026-03-31 \
    --project "Auth Revamp" --project "Billing v2"

  # Content from stdin
  cat brief.md | linear initiatives create "Q1 Platform" --content -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			output, err := format.ParseOutputType(outputType)
			if err != nil {
				return err
			}
			verbosity, err := format.ParseVerbosity(formatStr)
			if err != nil {
				return err
			}

			body, contentSet, err := readContentFlags(cmd, content, contentFile)
			if err != nil {
				return err
			}

			result, err := deps.Initiatives.Create(&service.InitiativeCreateParams{
				Name:        args[0],
				Description: description,
				Content:     body,
				ContentSet:  contentSet,
				Owner:       owner,
				Status:      status,
				TargetDate:  targetDate,
				Icon:        icon,
				Color:       color,
				Projects:    projects,
				Verbosity:   verbosity,
				OutputType:  output,
			})
			if err != nil {
				return err
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Short description")
	cmd.Flags().StringVar(&content, "content", "", "Markdown content (use - to read from stdin)")
	cmd.Flags().StringVar(&contentFile, "content-file", "", "Read markdown content from a file")
	cmd.Flags().StringVar(&owner, "owner", "", "Owner (me, name, or email)")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Status: Planned|Active|Completed")
	cmd.Flags().StringVar(&targetDate, "target-date", "", "Target date YYYY-MM-DD")
	cmd.Flags().StringVar(&icon, "icon", "", "Icon name or emoji code (e.g., \"Rocket\")")
	cmd.Flags().StringVar(&color, "color", "", "Hex color (e.g., #4EA7FC)")
	cmd.Flags().StringSliceVarP(&projects, "project", "P", nil, "Link project(s) by name or UUID (repeatable or comma-separated)")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newInitiativesUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		content     string
		contentFile string
		owner       string
		status      string
		targetDate  string
		icon        string
		color       string
		formatStr   string
		outputType  string
	)

	cmd := &cobra.Command{
		Use:   "update <id|slug|url|name>",
		Short: "Update an initiative",
		Long: `Update an initiative. Only provided flags are changed.
--owner "" removes the owner; --target-date "" clears the target date; --content "" clears content.
To post a status update instead, use 'linear initiatives updates create'.`,
		Example: `  linear initiatives update "Q1 Platform" --status Completed
  linear initiatives update q1-platform-3c7a2b1d9e0f --target-date 2026-04-15 --owner alice@acme.com
  cat brief.md | linear initiatives update "Q1 Platform" --content -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			output, err := format.ParseOutputType(outputType)
			if err != nil {
				return err
			}
			verbosity, err := format.ParseVerbosity(formatStr)
			if err != nil {
				return err
			}

			body, contentSet, err := readContentFlags(cmd, content, contentFile)
			if err != nil {
				return err
			}

			params := &service.InitiativeUpdateParams{
				Verbosity:  verbosity,
				OutputType: output,
			}
			flags := cmd.Flags()
			if flags.Changed("name") {
				params.Name = &name
			}
			if flags.Changed("description") {
				params.Description = &description
			}
			if contentSet {
				params.Content = &body
			}
			if flags.Changed("owner") {
				params.Owner = &owner
			}
			if flags.Changed("status") {
				params.Status = &status
			}
			if flags.Changed("target-date") {
				params.TargetDate = &targetDate
			}
			if flags.Changed("icon") {
				params.Icon = &icon
			}
			if flags.Changed("color") {
				params.Color = &color
			}

			result, err := deps.Initiatives.Update(args[0], params)
			if err != nil {
				return err
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New short description")
	cmd.Flags().StringVar(&content, "content", "", "New markdown content (use - to read from stdin; \"\" clears)")
	cmd.Flags().StringVar(&contentFile, "content-file", "", "Read new markdown content from a file")
	cmd.Flags().StringVar(&owner, "owner", "", "New owner (me, name, or email; \"\" removes)")
	cmd.Flags().StringVarP(&status, "status", "s", "", "New status: Planned|Active|Completed")
	cmd.Flags().StringVar(&targetDate, "target-date", "", "New target date YYYY-MM-DD (\"\" clears)")
	cmd.Flags().StringVar(&icon, "icon", "", "Icon name or emoji code")
	cmd.Flags().StringVar(&color, "color", "", "Hex color")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

// newInitiativeRefActionCmd builds a single-argument command that resolves an
// initiative reference and prints the service's confirmation message.
func newInitiativeRefActionCmd(use, short, long, example string, action func(deps *Dependencies, ref string) (string, error)) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}
			result, err := action(deps, args[0])
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
}

func newInitiativesArchiveCmd() *cobra.Command {
	return newInitiativeRefActionCmd(
		"archive <id|slug|url|name>",
		"Archive an initiative",
		"Archive an initiative. Archived initiatives are hidden from lists unless --include-archived is passed.",
		`  linear initiatives archive "Q1 Platform"`,
		func(deps *Dependencies, ref string) (string, error) { return deps.Initiatives.Archive(ref) },
	)
}

func newInitiativesUnarchiveCmd() *cobra.Command {
	return newInitiativeRefActionCmd(
		"unarchive <id|slug|url|name>",
		"Restore an archived initiative",
		"Restore an archived initiative.",
		`  linear initiatives unarchive "Q1 Platform"`,
		func(deps *Dependencies, ref string) (string, error) { return deps.Initiatives.Unarchive(ref) },
	)
}

func newInitiativesDeleteCmd() *cobra.Command {
	return newInitiativeRefActionCmd(
		"delete <id|slug|url|name>",
		"Move an initiative to trash",
		"Move an initiative to trash; restore from the Linear UI. Linked projects are not deleted. Prefer 'archive' to keep history visible.",
		`  linear initiatives delete "Q1 Platform"`,
		func(deps *Dependencies, ref string) (string, error) { return deps.Initiatives.Delete(ref) },
	)
}

func newInitiativesAddProjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-project <initiative> <project>",
		Short: "Link a project to an initiative",
		Long:  "Link an existing project (name or UUID) to an initiative.",
		Example: `  linear initiatives add-project "Q1 Platform" "Auth Revamp"
  linear initiatives add-project q1-platform-3c7a2b1d9e0f 5b1c...-project-uuid`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}
			result, err := deps.Initiatives.AddProject(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
}

func newInitiativesRemoveProjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove-project <initiative> <project>",
		Short:   "Unlink a project from an initiative",
		Long:    "Unlink a project (name or UUID) from an initiative. The project itself is not changed.",
		Example: `  linear initiatives remove-project "Q1 Platform" "Auth Revamp"`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}
			result, err := deps.Initiatives.RemoveProject(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
}

func newInitiativeUpdatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "updates",
		Short: "List or post initiative status updates",
		Long:  "Initiative status updates are posts with a health (onTrack, atRisk, offTrack) and a markdown body.",
	}
	cmd.AddCommand(
		newInitiativeUpdatesListCmd(),
		newInitiativeUpdatesCreateCmd(),
	)
	return cmd
}

func newInitiativeUpdatesListCmd() *cobra.Command {
	var (
		limit      int
		formatStr  string
		outputType string
	)

	cmd := &cobra.Command{
		Use:   "list <initiative>",
		Short: "List status updates for an initiative (newest first)",
		Example: `  linear initiatives updates list "Q1 Platform"
  linear initiatives updates list "Q1 Platform" --limit 3 --format full
  linear initiatives updates list q1-platform-3c7a2b1d9e0f --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
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

			result, err := deps.Initiatives.ListUpdates(args[0], limit, verbosity, output)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Maximum number of updates (max 250)")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newInitiativeUpdatesCreateCmd() *cobra.Command {
	var (
		body       string
		bodyFile   string
		health     string
		formatStr  string
		outputType string
	)

	cmd := &cobra.Command{
		Use:   "create <initiative>",
		Short: "Post a status update on an initiative",
		Example: `  linear initiatives updates create "Q1 Platform" --health onTrack --body "Auth shipped; billing on schedule."
  cat update.md | linear initiatives updates create "Q1 Platform" --health atRisk --body -
  linear initiatives updates create q1-platform-3c7a2b1d9e0f --body-file update.md --health offTrack`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
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

			text, _, err := readTextFlags(cmd, "body", "body-file", body, bodyFile)
			if err != nil {
				return err
			}

			result, err := deps.Initiatives.CreateUpdate(&service.InitiativeStatusUpdateParams{
				Initiative: args[0],
				Body:       text,
				Health:     health,
				Verbosity:  verbosity,
				OutputType: output,
			})
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&body, "body", "", "Markdown body (use - to read from stdin)")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read markdown body from a file")
	cmd.Flags().StringVar(&health, "health", "", "Health: onTrack|atRisk|offTrack")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "full", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}
