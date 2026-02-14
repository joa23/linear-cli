# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

**New `--format detailed` Verbosity Level:**
- Added `detailed` format between `compact` and `full`: `minimal → compact → detailed → full`
- `detailed` shows truncated comments + a hint to use `linear issues comments <id>` for full text
- `full` now shows truly untruncated comment bodies (adopting PR #24's semantic)
- `issues get` default changed from `full` to `detailed` to preserve existing behavior

**Comment Display:**
- `linear issues comments <id>` now shows full comment bodies instead of truncating to 200 characters
- Added `--last N` flag to `issues comments` to show only the N most recent comments

### Fixed

**Native Issue Relations (GitHub #6):**
- `--blocked-by` and `--depends-on` flags now use native Linear relations (`issueRelationCreate`) instead of metadata storage
- Fixed "no fields to update" error when using `--blocked-by` or `--depends-on` without other flags on `issues update`
- Fixed silent no-op when `--blocked-by` or `--depends-on` were combined with other flags (e.g. `--labels`)
- Relations created with these flags are now visible in Linear's UI immediately

**OAuth Token Refresh (GitHub #7):**
- Fixed OAuth token expiring daily and requiring re-authentication
- Root cause: `initializeClient()` used a static token provider (`NewClientWithAuthMode`) instead of the existing refresh-capable provider (`NewClientWithTokenPath`)
- Secondary fix: `NewClientWithTokenPath` now preserves `authMode` on the client struct, maintaining user/agent distinction
- Tokens are now refreshed automatically (proactive before expiry, reactive on 401)
- Legacy tokens (no refresh token) and `LINEAR_API_TOKEN` env var continue to work unchanged

## [1.4.9] - 2026-02-10

### Added

**`--project` Flag for Issues List, Search, and Deps (GitHub #12, PR #13):**
- Added `--project` / `-P` flag to `issues list`, `search`, and `deps` commands
- Accepts project name (case-insensitive) or UUID
- Server-side GraphQL filtering for `list` and `search`; client-side filtering for `deps`
- Invalid project name shows helpful error with available projects
- Team-scoped resolution: when `--team` is provided, only that team's projects are searched

## [1.4.8] - 2026-02-09

### Fixed

**Update Command Hangs in Piped/Agent Contexts:**
- Fixed `linear issues update` (and `projects update`) hanging indefinitely when stdin is a pipe
- Root cause: `getDescriptionFromFlagOrStdin` auto-detected pipes via `os.Stdin.Stat()` and called `readStdin()` even when no description was requested, blocking forever on EOF
- Affected all non-terminal contexts: Claude Code, scripts, cron jobs, CI pipelines
- Fix: Stdin is now only read when explicitly requested via `-d -` flag (standard Unix convention)
- `hasStdinPipe()` removed — pipe auto-detection was the source of the bug

## [1.4.7] - 2026-02-08

### Fixed

**Project Name Resolution in Create/Update (GitHub #4):**
- Fixed `-P` / `--project` flag failing with "GraphQL Argument Validation Error" on `issues create` and `issues update`
- Issue: `linear issues create "Title" -P "My Project"` and `linear issues update CEN-123 -P "My Project"` passed the project name as a string to the GraphQL API, which expects a UUID
- Root cause: `Create` and `Update` methods passed the project value directly without resolving names to UUIDs (unlike state, labels, and cycles which already resolved correctly)
- Fix: Added `ResolveProject` resolver that converts project names to UUIDs (case-insensitive, with fuzzy matching fallback)
- UUIDs are passed through unchanged for backwards compatibility
- Example: `linear issues create "Fix bug" -P "SCWM Experiments"` now works correctly

## [1.4.6] - 2026-01-28

### Fixed

**State/Label Resolution in Search (GitHub #3):**
- Fixed `--state` and `--labels` filters failing with "GraphQL error: Argument Validation Error" on search commands
- Issue: `linear issues list --state "In Progress"` and `linear search --labels "bug"` passed raw names to the API, which expects UUIDs
- Root cause: `Search`, `SearchWithOutput`, and `searchIssues` copied state/label filters directly without resolving names to IDs (unlike `create`/`update` which already resolved correctly)
- Fix: Added name-to-UUID resolution for state and label filters in all three search methods
- `--team` flag is now required when filtering by `--state` or `--labels` (states and labels are team-scoped)
- Example: `linear issues list --state "In Progress" --team CEN` now works correctly

## [1.4.5] - 2026-01-27

### Fixed

**OAuth App "me" Resolution:**
- Fixed `--assignee me` resolving to human account owner instead of OAuth application
- Root cause: Email suffix detection was unreliable; Linear's viewer query returns inconsistent data
- Fix: Store auth mode ("user" or "agent") at login time in token file
- Agent mode: `--assignee me` uses `delegateId` (OAuth app visible in Linear)
- User mode: `--assignee me` uses `assigneeId` (personal account)
- Run `linear auth login` and select mode to activate fix

### Changed

- `linear auth status` now displays current auth mode
- `linear auth login` help text explains mode differences
- Updated README, CLAUDE.md, and `/linear` skill with auth mode documentation

## [1.4.4] - 2026-01-27

### Fixed

**OAuth Delegate Display:**
- Fixed issue where `--assignee me` for OAuth apps appeared to succeed but showed "Unassigned"
- Root cause: Linear has separate `delegate` field for OAuth apps, distinct from `assignee`
- Fix: Added `delegate` field to GraphQL queries and display logic
- Text output now shows "Delegate: AppName <email>" when delegate is set
- JSON output includes full `delegate` object with id, name, email
- When both assignee and delegate exist, assignee takes display priority

### Added

**Notifications Command:**
- New `linear notifications list` - View recent @mentions and notifications
- New `linear notifications read <id>` - Mark notification as read
- Supports `--unread` flag to filter to unread only
- Supports `--limit` flag to control number of results
- Shows notification type, related issue/project, and timestamps

### Changed

- Updated GraphQL queries: `GetIssue`, `UpdateIssue` now include `delegate` field
- Updated formatters: text and JSON renderers handle delegate display
- Added `Delegate` field to `Issue` struct in core types

## [1.4.3] - 2026-01-27

### Fixed

**OAuth Application Assignee Support:**
- Fixed `--assignee me` silently failing when authenticated as an OAuth application
- Issue: Linear uses `delegateId` for apps but `assigneeId` for humans; we only used `assigneeId`
- Fix: Auto-detect user type by email suffix (`@oauthapp.linear.app`) and use appropriate field
- Detection happens at resolution time via new `ResolvedUser` struct with `IsApplication` flag
- Applies to: `linear issues create --assignee`, `linear issues update --assignee`
- Example: `linear issues update CEN-123 --assignee me` now works for OAuth apps
- Added comprehensive test coverage for delegate vs assignee routing

### Added

**Test Coverage:**
- `resolver_test.go` - Tests for OAuth application email detection (8 test cases)
- `client_test.go` - Tests for `delegateId` in GraphQL input building (10 test cases)
- `issue_delegate_test.go` - Tests for service layer routing logic

## [1.4.2] - 2026-01-27

### Fixed

**"me" Assignee Resolution:**
- Fixed `--assignee me` not resolving to the current authenticated user
- Issue: Resolver didn't handle the special "me" value documented in help
- Fix: Added "me" handling that calls `GetViewer()` to resolve to authenticated user's ID
- Also added UUID passthrough so already-resolved IDs aren't re-resolved as names
- Example: `linear issues list --assignee me` now correctly filters your issues

## [1.4.1] - 2026-01-26

### Changed

**Priority Flag Now Accepts String Values:**
- `--priority` flag now accepts both integers (0-4) and names (none/urgent/high/normal/low)
- Example: `--priority high` instead of `--priority 2`
- Works for `create`, `update`, and `list` commands
- Backwards compatible: numeric values still work

## [1.4.0] - 2026-01-26

### Fixed

**Label Resolution for Issue Create/Update:**
- Fixed `--labels` flag failing with "Argument Validation Error" on `linear issues create` and `linear issues update`
- Issue: Labels were passed as names (e.g., "Security") but API expects UUIDs
- Root cause: Unlike states, cycles, and assignees, labels were never resolved from names to IDs
- Fix: Added `ResolveLabelIdentifier` method with full resolver infrastructure:
  - Case-insensitive label name matching
  - Team-scoped resolution (labels are team-specific)
  - Caching for performance (5-minute TTL)
  - Helpful error messages listing available labels when not found
- Example: `linear issues create "Bug fix" --team CEN --labels "Security"` now works
- Files changed: `resolver.go`, `resolver_cache.go`, `client.go`, `issue.go`, `client_interfaces.go`

## [1.3.0] - 2026-01-25

### Added

**Claude Code Task Export:**
- Added `linear tasks export` command to convert Linear issues into Claude Code task format
- Exports complete dependency tree (children + blocking relationships) recursively
- Detects and prevents circular dependencies with clear error messages
- Bottom-up hierarchy: children block parent (matches Claude task semantics)
- Files named using Linear identifiers (e.g., `CEN-123.json`) for idempotency
- Dry-run mode (`--dry-run`) for preview without writing files
- Smart verb conjugation for `activeForm` field (e.g., "Fix bug" → "Fixing bug")
- Export to any folder or directly to Claude Code session directories
- Example: `linear tasks export CEN-123 ~/.claude/tasks/<session-uuid>/`
- Perfect for integrating Linear planning with Claude Code execution

**JSON Output Support:**
- Added `--output json` flag to all list and get commands for machine-readable output
- JSON output available for: issues, cycles, projects, teams, users, search operations
- Three verbosity levels for both text and JSON:
  - `--format minimal` - Essential fields only (~50 tokens)
  - `--format compact` - Key metadata (~150 tokens, default)
  - `--format full` - Complete details with all relationships (~500 tokens)
- Examples: `linear issues list --output json`, `linear cycles analyze --output json --format full`
- Pipe-friendly: `linear issues list --output json | jq '.[] | select(.priority == 1)'`
- Perfect for automation, scripting, and integration with other tools

**Documentation:**
- Added "JSON Automation Examples" section to README with 10+ practical recipes
- Examples include: filtering with jq, counting by state, exporting to CSV, bulk processing, weekly reports
- Added comprehensive Table of Contents to README for easier navigation
- Reorganized README with clear section dividers (589 lines → well-structured)
- Added comprehensive package documentation for all internal packages

### Changed

**Code Quality:**
- Restructured helpers into specific packages (identifiers, validation, pagination)
- Implemented dependency injection pattern with service interfaces
- All services now use sub-clients directly instead of wrapper methods
- Comprehensive test coverage for task export feature (40+ test cases)
- Fixed all golangci-lint warnings (11 issues resolved)

### Fixed

**Projects List Team Filtering:**
- Implemented team filtering for `linear projects list --team` command
- Issue: Command showed warning "Team filtering for projects is not yet implemented"
- Fix: Added `ListByTeam` method that queries projects via team relationship
- Removed unpredictable "smart routing" behavior - now requires team (from flag or `.linear.yaml`)
- Behavior: flag → `.linear.yaml` → error (consistent with issues/cycles commands)
- Exception: `--mine` flag bypasses team requirement to show user's projects
- Example: `linear projects list --team CEN` shows only CEN team projects

## [1.2.3] - 2026-01-23

### Fixed

**Cycle Resolution Bug:**
- Fixed `linear issues update` failing when setting cycle with `--cycle` flag
- Issue: Command required explicit `--team` flag even when `.linear.yaml` config existed
- Root cause: Update command didn't read team context from config or extract from issue identifier
- Fix: Added proper team resolution hierarchy to match `create` command behavior:
  1. Explicit `--team` flag (allows moving issues to different teams)
  2. Default team from `.linear.yaml` config file
  3. Automatic extraction from issue identifier (e.g., CEN-4322 → CEN team)
- Example: `linear issues update CEN-4322 --cycle 66` now works without `--team` flag
- Added `--team` flag to update command for explicit team override

## [1.2.2] - 2026-01-22

### Added

**New Skill for LLMs:**
- `/linear` - Comprehensive CLI reference optimized for LLM in-context learning
  - Token-compact format designed for frequent loading by Claude Code
  - Covers all features: semantic search, dependency tracking, cycle analytics
  - Example-driven with powerful filter combinations and real-world workflows
  - Highlights semantic search capabilities and dependency management
  - Output format guidance (minimal/compact/full) for token efficiency
  - Prominent skills installation prompt with ⚠️ warning
  - Replaces internal `/release` skill (moved to `.claude/skills/` for maintainer use)

### Changed

**Skill System:**
- Removed `/release` from user-installable skills (maintainer-only)
- Made `/linear` skill description more forceful: "MUST READ before using Linear commands"
- Added installation urgency prompt to encourage `linear skills install --all`

### Fixed

**Version Display:**
- Fixed goreleaser ldflags path: binaries now show correct version instead of "dev"
- Changed from `main.version` to `github.com/joa23/linear-cli/internal/cli.Version`

## [1.2.1] - 2026-01-22

### Fixed

**CRITICAL: Authentication Bug**
- Fixed token loading mismatch that caused all commands to fail after authentication
- Issue: `linear auth login` saved tokens as JSON, but all other commands tried to load them as plain strings
- Result: Every command after initial auth returned "unauthorized - token may be expired or invalid"
- Fix: Changed all commands to use `LoadTokenData()` instead of `LoadToken()`
- Affected commands: `linear init`, `linear auth status`, `linear issues`, `linear onboard`, and all others

**Duplicate Content Prevention**
- Fixed `linear init` creating duplicate Linear sections in CLAUDE.md and AGENTS.md when run multiple times
- Changed from checking for "## Linear" header to checking for unique content marker
- Now properly detects if Linear section already exists before appending

### Added

**New Skill:**
- `/release` - Pre-release checklist ensuring CHANGELOG, versions, tests, and docs are updated
  - 9-step systematic validation process
  - Catches missing CHANGELOG entries before release
  - Verifies version number consistency
  - Ensures all tests pass
  - Includes rollback procedure

**Documentation:**
- Updated CHANGELOG with complete v1.2.0 release notes (was missing in v1.2.0)
- README: Moved Homebrew installation to Quickstart for better discoverability

## [1.2.0] - 2026-01-22

### Added

**Unified Search Command:**
- `linear search` - Powerful search across all Linear entities (issues, cycles, projects, users)
- Text search: `linear search "authentication"`
- Entity type filtering: `--type issues|cycles|projects|users|all`
- Cross-entity search: `linear search "oauth" --type all`

**Dependency Filtering:**
- `--blocked-by <ID>` - Find issues blocked by a specific issue
- `--blocks <ID>` - Find issues that block a specific issue
- `--has-blockers` - Find all issues with any blockers
- `--has-dependencies` - Find issues with dependencies
- `--has-circular-deps` - Detect circular dependency chains
- `--max-depth <n>` - Filter by dependency chain depth

**New Skill:**
- `/link-deps` - Comprehensive skill for discovering and linking dependencies across backlog
  - Systematic discovery process with 4 discovery patterns
  - Example workflow for 100+ issue backlogs
  - Complete command reference for discovery and linking
  - Best practices and anti-patterns

**Enhanced Documentation:**
- README: Added search section with 8+ examples
- CLAUDE.md: Search best practices and workflow patterns
- All skills updated to reference search commands
- Enhanced `--help` output with use cases and detailed filter descriptions

### Changed

**Skills Updated:**
- `/prd` - Now searches for existing work before creating tickets
- `/triage` - Uses search to find blocked work
- `/deps` - Added discovery commands section
- All skills include cross-references to search functionality

**Search Integration:**
- Comprehensive help text with 4 use cases
- Discovery patterns documented across all skills
- Weekly unblocking routine recommended

### Fixed
- Token sanitization for OAuth tokens (added Bearer prefix validation)
- Invalid Authorization header issues
- Missing flags on various commands

## [1.0.1] - 2026-01-21

### Added
- `--version` flag to display version information
- Version is auto-generated from git tags during build
- Comprehensive OAuth setup documentation for agent mode
- Specific callback URL guidance (`http://localhost:3000/oauth-callback` for agents)

### Fixed
- **Critical**: GraphQL pagination query error (removed invalid `totalCount` field from IssueConnection)
- Users can now run `linear issues list` without HTTP 400 errors

### Changed
- Removed "MCP server" references from all user-facing text
- Updated authentication prompts to say "automation, bots" instead of "MCP server"
- Changed onboard command title from "Light Linear MCP" to "Linear CLI"
- Updated README with correct Homebrew installation instructions
- Updated manual download URLs to use tar.gz archives

## [1.0.0] - 2026-01-20

### Initial Release

A token-efficient CLI for Linear.

#### Features

**CLI Commands:**
- `linear auth` - OAuth2 authentication (login, logout, status)
- `linear issues` - Full issue management (list, get, create, update, comment, reply, react)
- `linear projects` - Project management (list, get, create, update)
- `linear cycles` - Cycle management and velocity analytics
- `linear teams` - Team info, labels, workflow states
- `linear users` - User listing and lookup
- `linear deps` - Dependency graph visualization
- `linear skills` - Claude Code skill installation

**Key Capabilities:**
- Human-readable identifiers (TEST-123, not UUIDs)
- Token-efficient ASCII output format
- OAuth2 authentication with secure local storage
- Cycle velocity analytics with recommendations
- Comment threading and emoji reactions
- Issue dependency tracking

#### Platform Support

- macOS (Apple Silicon and Intel)
- Linux (64-bit)
- Windows (64-bit)

[1.4.6]: https://github.com/joa23/linear-cli/compare/v1.4.5...v1.4.6
[1.4.5]: https://github.com/joa23/linear-cli/compare/v1.4.4...v1.4.5
[1.4.4]: https://github.com/joa23/linear-cli/compare/v1.4.3...v1.4.4
[1.4.3]: https://github.com/joa23/linear-cli/compare/v1.4.2...v1.4.3
[1.4.2]: https://github.com/joa23/linear-cli/compare/v1.4.1...v1.4.2
[1.4.1]: https://github.com/joa23/linear-cli/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/joa23/linear-cli/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/joa23/linear-cli/compare/v1.2.3...v1.3.0
[1.2.3]: https://github.com/joa23/linear-cli/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/joa23/linear-cli/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/joa23/linear-cli/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/joa23/linear-cli/compare/v1.0.1...v1.2.0
[1.0.1]: https://github.com/joa23/linear-cli/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/joa23/linear-cli/releases/tag/v1.0.0
