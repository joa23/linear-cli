package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	outputFormat string // global flag for output format (table, json, markdown)
	verbose      bool   // global flag for verbose output

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

Issues (alias: i):
  i list                       List your assigned issues
  i get <ID>                   Get issue details (e.g., CEN-123)
  i create <title> [flags]     Create issue
  i update <ID> [flags]        Update issue
  i comment <ID> [flags]       Add comment to issue
  i comments <ID>              List comments on issue
  i reply <ID> <COMMENT> [fl]  Reply to a comment
  i react <ID> <emoji>         Add reaction (👍 ❤️ 🎉 etc)
  i dependencies <ID>          Show dependencies
  i blocked-by <ID>            Show blockers
  i blocking <ID>              Show what this blocks

  Issue flags: -t team, -d description, -s state, -p priority (0-4),
               -e estimate, -l labels, -c cycle, -P project, -a assignee
  Comment/Reply flags: -b body, --attach <file>

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

Users (alias: u):
  u list [--team <ID>]         List users
  u get <ID>                   Get user details
  u me                         Show current user

Analysis:
  deps <ID>                    Show dependency graph for issue
  deps --team <KEY>            Show all dependencies for team

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
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "table", "Output format (table, json, markdown)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")

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

		// Analysis
		newDepsCmd(),

		// Skills
		newSkillsCmd(),
	)

	return rootCmd
}

// Execute runs the CLI
func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
