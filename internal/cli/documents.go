package cli

import (
	"fmt"
	"os"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/service"
	"github.com/spf13/cobra"
)

func newDocumentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "documents",
		Aliases: []string{"docs", "doc", "document"},
		Short:   "Manage Linear documents",
		Long: `Manage Linear documents attached to projects, issues, or teams.

Documents hold long-form markdown (specs, PRDs, runbooks). Every document has exactly
one parent: a project, an issue, or a team.

Document references accept a UUID, a slug, or the full Linear document URL:
  linear documents get 3f2a9c1e-...
  linear documents get my-spec-abc123def456
  linear documents get https://linear.app/acme/document/my-spec-abc123def456`,
	}

	cmd.AddCommand(
		newDocumentsListCmd(),
		newDocumentsGetCmd(),
		newDocumentsCreateCmd(),
		newDocumentsUpdateCmd(),
		newDocumentsDeleteCmd(),
	)

	return cmd
}

func newDocumentsListCmd() *cobra.Command {
	var (
		project         string
		issue           string
		team            string
		query           string
		includeArchived bool
		limit           int
		formatStr       string
		outputType      string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List documents",
		Long: `List documents, newest updated first. Without filters, lists the whole workspace.
Filters combine with AND. Document bodies are not included; use 'documents get' for content.`,
		Example: `  # All documents in the workspace
  linear documents list

  # Documents on a project (use --team to scope name resolution)
  linear documents list --project "Mobile App" --team ENG

  # Documents on an issue or a team
  linear documents list --issue ENG-123
  linear documents list --team ENG

  # Search titles
  linear documents list --query "runbook"

  # JSON output for automation
  linear documents list --output json`,
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

			result, err := deps.Documents.List(&service.DocumentListParams{
				Project:         project,
				Issue:           issue,
				Team:            team,
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

	cmd.Flags().StringVarP(&project, "project", "P", "", "Filter by project name or UUID")
	cmd.Flags().StringVarP(&issue, "issue", "i", "", "Filter by issue identifier (e.g., ENG-123) or UUID")
	cmd.Flags().StringVarP(&team, "team", "t", "", "Filter by team key, name, or UUID")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Match title (case-insensitive substring)")
	cmd.Flags().BoolVar(&includeArchived, "include-archived", false, "Include archived documents")
	cmd.Flags().IntVarP(&limit, "limit", "n", DefaultLimit, "Maximum number of documents (max 250)")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newDocumentsGetCmd() *cobra.Command {
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "get <id|slug|url>",
		Short: "Get a document with its content",
		Example: `  # By slug (the trailing token of the Linear URL)
  linear documents get my-spec-abc123def456

  # By URL
  linear documents get https://linear.app/acme/document/my-spec-abc123def456

  # Metadata only
  linear documents get my-spec-abc123def456 --format compact

  # JSON (content in .content)
  linear documents get my-spec-abc123def456 --output json`,
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

			result, err := deps.Documents.Get(args[0], verbosity, output)
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

func newDocumentsCreateCmd() *cobra.Command {
	var (
		title       string
		content     string
		contentFile string
		project     string
		issue       string
		team        string
		icon        string
		color       string
		formatStr   string
		outputType  string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a document",
		Long: `Create a document under exactly one parent: --project, --issue, or --team.

If no parent flag is given, the default project from .linear.yaml is used, then the
default team. Pass --team together with --project to scope project name resolution.`,
		Example: `  # On the default project (from .linear.yaml)
  linear documents create --title "API Spec" --content "# API\n..."

  # On a project, content from a file
  linear documents create --title "Runbook" --project "Platform" --content-file runbook.md

  # On an issue, content from stdin
  cat notes.md | linear documents create --title "Notes" --issue ENG-123 --content -

  # On a team
  linear documents create --title "Team Charter" --team ENG`,
		Args: cobra.NoArgs,
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

			// Fall back to .linear.yaml defaults when no parent is given
			if project == "" && issue == "" && team == "" {
				project = GetDefaultProject()
				if project == "" {
					team = GetDefaultTeam()
				}
			}
			if project == "" && issue == "" && team == "" {
				return fmt.Errorf("a parent is required: --project, --issue, or --team (or run 'linear init')")
			}

			result, err := deps.Documents.Create(&service.DocumentCreateParams{
				Title:      title,
				Content:    body,
				ContentSet: contentSet,
				Project:    project,
				Issue:      issue,
				Team:       team,
				Icon:       icon,
				Color:      color,
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

	cmd.Flags().StringVar(&title, "title", "", "Document title (required)")
	cmd.Flags().StringVar(&content, "content", "", "Markdown content (use - to read from stdin)")
	cmd.Flags().StringVar(&contentFile, "content-file", "", "Read markdown content from a file")
	cmd.Flags().StringVarP(&project, "project", "P", "", ProjectFlagDescription)
	cmd.Flags().StringVarP(&issue, "issue", "i", "", "Parent issue identifier (e.g., ENG-123) or UUID")
	cmd.Flags().StringVarP(&team, "team", "t", "", "Parent team key (or scope for --project name lookup)")
	cmd.Flags().StringVar(&icon, "icon", "", "Icon name or emoji code (e.g., \"Rocket\" or \":eagle:\")")
	cmd.Flags().StringVar(&color, "color", "", "Hex color (e.g., #4EA7FC)")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")
	cmd.MarkFlagRequired("title")

	return cmd
}

func newDocumentsUpdateCmd() *cobra.Command {
	var (
		title       string
		content     string
		contentFile string
		project     string
		issue       string
		team        string
		icon        string
		color       string
		formatStr   string
		outputType  string
	)

	cmd := &cobra.Command{
		Use:   "update <id|slug|url>",
		Short: "Update a document",
		Long: `Update a document's title, content, icon, color, or parent.
Passing --project, --issue, or --team moves the document to that parent.
--content "" clears the content.`,
		Example: `  # Rename
  linear documents update my-spec-abc123def456 --title "API Spec v2"

  # Replace content from a file
  linear documents update my-spec-abc123def456 --content-file spec.md

  # Replace content from stdin
  cat spec.md | linear documents update my-spec-abc123def456 --content -

  # Move to another project
  linear documents update my-spec-abc123def456 --project "Platform" --team ENG`,
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

			params := &service.DocumentUpdateParams{
				Verbosity:  verbosity,
				OutputType: output,
			}
			flags := cmd.Flags()
			if flags.Changed("title") {
				params.Title = &title
			}
			if contentSet {
				params.Content = &body
			}
			if flags.Changed("icon") {
				params.Icon = &icon
			}
			if flags.Changed("color") {
				params.Color = &color
			}
			if flags.Changed("project") {
				params.Project = &project
			}
			if flags.Changed("issue") {
				params.Issue = &issue
			}
			if flags.Changed("team") {
				params.Team = &team
			}

			result, err := deps.Documents.Update(args[0], params)
			if err != nil {
				return err
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&content, "content", "", "New markdown content (use - to read from stdin; \"\" clears)")
	cmd.Flags().StringVar(&contentFile, "content-file", "", "Read new markdown content from a file")
	cmd.Flags().StringVarP(&project, "project", "P", "", "Move to project (name or UUID)")
	cmd.Flags().StringVarP(&issue, "issue", "i", "", "Move to issue (e.g., ENG-123 or UUID)")
	cmd.Flags().StringVarP(&team, "team", "t", "", "Move to team key (or scope for --project name lookup)")
	cmd.Flags().StringVar(&icon, "icon", "", "Icon name or emoji code")
	cmd.Flags().StringVar(&color, "color", "", "Hex color")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|detailed|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newDocumentsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <id|slug|url>",
		Short:   "Move a document to trash",
		Long:    "Move a document to trash. Linear does not hard-delete documents; restore from the Linear UI.",
		Example: `  linear documents delete my-spec-abc123def456`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			result, err := deps.Documents.Delete(args[0])
			if err != nil {
				return err
			}

			fmt.Println(result)
			return nil
		},
	}

	return cmd
}

// readContentFlags resolves --content / --content-file into a body.
// "-" reads stdin. set reports whether the caller supplied content at all
// (so an explicit --content "" is distinguishable from no flag).
func readContentFlags(cmd *cobra.Command, content, contentFile string) (body string, set bool, err error) {
	flags := cmd.Flags()
	contentSet := flags.Changed("content")
	fileSet := flags.Changed("content-file")

	if contentSet && fileSet {
		return "", false, fmt.Errorf("--content and --content-file are mutually exclusive")
	}
	if fileSet {
		data, err := os.ReadFile(contentFile)
		if err != nil {
			return "", false, fmt.Errorf("failed to read --content-file: %w", err)
		}
		return string(data), true, nil
	}
	if contentSet {
		body, err := getDescriptionFromFlagOrStdin(content)
		if err != nil {
			return "", false, err
		}
		return body, true, nil
	}
	return "", false, nil
}
