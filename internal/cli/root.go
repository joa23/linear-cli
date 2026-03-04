package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joa23/linear-cli/internal/linear"
	"github.com/joa23/linear-cli/internal/token"
	"github.com/spf13/cobra"
)

var (
	verbose   bool   // global flag for verbose output
	workspace string // global flag for workspace selection

	// Version is set via ldflags at build time
	Version = "dev"
)

// customHelpTemplate puts Flags before Examples (industry standard)
const customHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

// customUsageTemplate defines the usage format with Flags before Examples
const customUsageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// NewRootCmd creates the root command for the 'linear' CLI
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "linear",
		Short:   "Light Linear - Token-efficient Linear CLI",
		Version: Version,
		Long: `Light Linear - Token-efficient Linear CLI

A lightweight CLI for Linear. Run 'linear onboard' to get started.

Setup:
  init                         Initialize Linear for this project
  onboard                      Show setup status and quick reference
  auth login|logout|status     Manage authentication
  auth list                    List configured workspaces (offline)

Issues (alias: i):
  i list                       List your assigned issues
  i get <ID>                   Get issue details (e.g., CEN-123)
  i create <title> [flags]     Create issue
  i update <ID> [flags]        Update issue
  i comment <ID> [flags]       Add comment to issue
  i comments <ID>              List comments on issue
  i reply <ID> <COMMENT> [fl]  Reply to a comment
  i react <ID> <emoji>         Add reaction
  i dependencies <ID>          Show dependencies
  i blocked-by <ID>            Show blockers
  i blocking <ID>              Show what this blocks

  Issue flags: -t team, -d description, -s state, -p priority (0-4),
               -e estimate, -l labels, -c cycle, -P project, -a assignee,
               --parent, --blocked-by, --depends-on, --attach, --due, --title
  Comment/Reply flags: -b body, --attach <file> (inline image embed)

Projects (alias: p):
  p list [--mine]              List projects
  p get <ID>                   Get project details
  p create <name> [flags]      Create project
  p update <ID> [flags]        Update project

  Project flags: -t team, -d description, -s state, -l lead, -n name

Cycles (alias: c):
  c list [--team <KEY>]        List cycles
  c get <ID>                   Get cycle details
  c analyze                    Analyze velocity

Teams (alias: t):
  t list                       List all teams
  t get <ID>                   Get team details
  t labels <ID>                List team labels
  t states <ID>                List workflow states

Labels:
  labels list --team <KEY>     List labels
  labels create <name> [flags] Create label
  labels update <id> [flags]   Update label
  labels delete <id>           Delete label

Attachments (alias: att) — sidebar cards (GitHub PRs, Slack threads, files, URLs):
  att list <ID>                List attachment cards on issue
  att create <ID> [flags]      Create attachment card (URL or file upload)
  att update <ID> [flags]      Update attachment title/subtitle
  att delete <ID>              Delete attachment card

  Create flags: --url, --file, --title, --subtitle
  NOTE: --attach on issues/comments embeds inline images; att create makes sidebar cards

Users (alias: u):
  u list [--team <ID>]         List users
  u get <ID>                   Get user details
  u me                         Show current user

Analysis:
  deps <ID>                    Show dependency graph for issue
  deps --team <KEY>            Show all dependencies for team
  search [query] [flags]       Unified search with dependency filters

Skills:
  skills list                  List available Claude Code skills
  skills install [--all]       Install skills to .claude/skills/

Configuration:
  Run 'linear init' to set a default team. Creates .linear.yaml.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Show help if no subcommand provided
			return cmd.Help()
		},
	}

	// Apply custom help template (Flags before Examples)
	rootCmd.SetHelpTemplate(customHelpTemplate)
	rootCmd.SetUsageTemplate(customUsageTemplate)

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&workspace, "workspace", "", "Workspace to use (overrides .linear.yaml)")

	// Add subcommands - grouped logically
	rootCmd.AddCommand(
		// Setup
		newInitCmd(),
		newOnboardCmd(),

		// Authentication
		newAuthCmd(),

		// Resources
		newIssuesCmd(),
		newProjectsCmd(),
		newCyclesCmd(),
		newTeamsCmd(),
		newUsersCmd(),
		newLabelsCmd(),
		newNotificationsCmd(),
		newAttachmentsCmd(),

		// Analysis
		newDepsCmd(),
		newSearchCmd(),

		// Export
		newTasksCmd(),

		// Skills
		newSkillsCmd(),
	)

	return rootCmd
}

// Execute runs the CLI with dependency injection via PersistentPreRunE.
// Commands that handle their own auth (login, logout, status, init, onboard)
// are skipped; all others get an authenticated client injected into context.
func Execute() {
	rootCmd := NewRootCmd()

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Commands annotated with skipAuth handle their own client setup
		for c := cmd; c != nil; c = c.Parent() {
			if c.Annotations["skipAuth"] == "true" {
				return nil
			}
		}
		// Skip for root command (no parent = help display)
		if !cmd.HasParent() {
			return nil
		}

		p := resolveWorkspace()
		client, err := initializeClient(p)
		if err != nil {
			return err
		}
		deps := NewDependencies(client)
		cmd.SetContext(context.WithValue(cmd.Context(), dependenciesKey, deps))
		return nil
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// resolveWorkspace returns the workspace to use.
// Resolution order: --workspace flag > .linear.yaml workspace > single named workspace > "" (global default)
func resolveWorkspace() string {
	if workspace != "" {
		return workspace
	}
	if w := GetDefaultWorkspace(); w != "" {
		return w
	}
	// No explicit config — auto-select if exactly one named workspace exists.
	return autoSelectWorkspace()
}

// resolveWorkspaceExplicit returns the workspace from explicit sources only:
// --workspace flag or .linear.yaml. Does NOT auto-select from disk scan.
// Used by init and login so the picker always runs.
func resolveWorkspaceExplicit() string {
	if workspace != "" {
		return workspace
	}
	return GetDefaultWorkspace()
}

// autoSelectWorkspace returns the single named workspace when only one exists.
// Returns "" when zero or multiple named workspaces are present.
func autoSelectWorkspace() string {
	tokens := token.ListWorkspaceTokens()
	var named []string
	for _, t := range tokens {
		if t.Name != "" {
			named = append(named, t.Name)
		}
	}
	if len(named) == 1 {
		return named[0]
	}
	return ""
}

// initializeClient creates and configures the Linear client for the given profile.
// Loads token from the profile-specific (or default) path and returns an authenticated client.
func initializeClient(p string) (*linear.Client, error) {
	return initializeClientWithTokenPath(p, token.GetWorkspaceTokenPath(p))
}

// initializeClientWithTokenPath creates a Linear client from the given token path.
// Extracted for testability — initializeClient delegates to this with the resolved path.
//
// Resolution order: LINEAR_API_KEY env > profile token file > LINEAR_API_TOKEN (legacy)
func initializeClientWithTokenPath(p string, tokenPath string) (*linear.Client, error) {
	// LINEAR_API_KEY takes highest priority — simple bearer token override
	if envToken := os.Getenv("LINEAR_API_KEY"); envToken != "" {
		return linear.NewClient(envToken), nil
	}

	tokenStorage := token.NewStorage(tokenPath)
	exists, _ := tokenStorage.TokenExistsWithError()
	if !exists {
		if p != "" {
			return nil, fmt.Errorf("not authenticated for workspace %q. Run 'linear auth login --workspace %s' to authenticate", p, p)
		}
		// Check if multiple named workspaces exist — give a more helpful error.
		allTokens := token.ListWorkspaceTokens()
		var named []string
		for _, t := range allTokens {
			if t.Name != "" {
				named = append(named, t.Name)
			}
		}
		if len(named) > 1 {
			return nil, fmt.Errorf("multiple workspaces configured. Use --workspace <name> or run 'linear init'\nAvailable: %s", strings.Join(named, ", "))
		}
		return nil, fmt.Errorf("not authenticated. Run 'linear auth login' to authenticate")
	}

	// Use the refresh-capable provider which automatically selects between
	// static and refreshing token providers based on available credentials
	client := linear.NewClientWithTokenPath(tokenPath)
	if client == nil {
		return nil, fmt.Errorf("not authenticated. Run 'linear auth login' or set LINEAR_API_KEY")
	}

	return client, nil
}
