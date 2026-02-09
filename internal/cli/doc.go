/*
Package cli implements the command-line interface for the Linear CLI.

# Architecture

The CLI is built using the Cobra framework and follows a modular command structure.

The root command is defined in root.go, which serves as the entry point and
coordinates all subcommands. Each major entity (issues, cycles, projects, etc.)
has its own file:

  - root.go      - Root command and global flags (entry point)
  - auth.go      - Authentication commands (login, logout, status)
  - issues.go    - Issue management (list, get, create, update, comment, link)
  - cycles.go    - Cycle operations (list, get, analyze)
  - projects.go  - Project management (list, get, create, update)
  - search.go    - Unified search across entities with dependency filtering
  - deps.go      - Dependency graph visualization
  - skills.go    - Claude Code skill management
  - init.go      - Team configuration
  - onboard.go   - Setup status and quick start guide

# Common Patterns

## Team Resolution

Commands that require team context follow a consistent resolution order:
  1. Explicit --team flag
  2. Default team from .linear.yaml (set via 'linear init')
  3. Error if neither is provided

Use [GetDefaultTeam] to retrieve the configured default team.

## Limit Validation

All list commands support --limit flags with consistent validation:
  - Default: 25 results
  - Maximum: 250 results (Linear API limit)
  - Values <= 0 default to 25
  - Values > 250 return an error

Use [validateAndNormalizeLimit] for consistent validation.

## Error Handling

Standardized error messages are defined in errors.go:
  - [ErrTeamRequired] - For missing team context
  - Use errors.New() for constant messages (not fmt.Errorf)

## Flag Descriptions

Reusable flag descriptions are centralized in flags.go:
  - [TeamFlagDescription] - Standard team flag help text

## Helpers

Common utilities in helpers.go:
  - [readStdin]                       - Read from stdin
  - [parseCommaSeparated]             - Parse comma-separated values
  - [getDescriptionFromFlagOrStdin]   - Get text from flag or stdin (use "-" for stdin)
  - [uploadAndAppendAttachments]      - Upload files and generate markdown
  - [validateAndNormalizeLimit]       - Validate --limit flags
  - [looksLikeCycleNumber]            - Detect numeric cycle IDs

# Dependency Management

The search command supports powerful dependency filtering:
  - --blocked-by ID    - Find issues blocked by a specific issue
  - --blocks ID        - Find issues blocking a specific issue
  - --has-blockers     - Find all blocked issues
  - --has-dependencies - Find issues with prerequisites
  - --has-circular-deps - Detect circular dependency chains
  - --max-depth N      - Filter by dependency chain depth

The deps command visualizes dependency graphs using Mermaid syntax.

# Service Layer

Commands interact with Linear via the service layer (internal/service):
  - IssueService  - Issue operations
  - CycleService  - Cycle analytics and retrieval
  - ProjectService - Project management
  - SearchService - Unified search with dependency filters
  - DepsService   - Dependency graph analysis
  - UserService   - User resolution (name/email → ID)

Services are obtained via helper functions:
  - [getLinearClient]
  - [getIssueService]
  - [getCycleService]
  - [getProjectService]
  - [getSearchService]
  - [getDepsService]
  - [getUserService]

# State Management

The CLI is stateless between invocations. Each command:
  1. Creates a fresh linear.Client from stored OAuth tokens
  2. Constructs service layer instances as needed
  3. Executes the operation
  4. Exits

State persisted between runs:
  - OAuth tokens (via token.Storage in ~/.linear/)
  - Team defaults (via ConfigManager in .linear.yaml)

# Configuration

Team defaults are stored in .linear.yaml via ConfigManager.
OAuth tokens are stored securely via token.Storage.

# Testing

Each command file has corresponding tests in *_test.go.
See test files for examples of command validation and flag parsing.
*/
package cli
