# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.1] - 2026-01-22

### Fixed

**CRITICAL: Authentication Bug**
- Fixed token loading mismatch that caused all commands to fail after authentication
- Issue: `linear auth login` saved tokens as JSON, but all other commands tried to load them as plain strings
- Result: Every command after initial auth returned "unauthorized - token may be expired or invalid"
- Fix: Changed all commands to use `LoadTokenData()` instead of `LoadToken()`
- Affected commands: `linear init`, `linear auth status`, `linear issues`, `linear onboard`, and all others

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

[1.2.1]: https://github.com/joa23/linear-cli/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/joa23/linear-cli/compare/v1.0.1...v1.2.0
[1.0.1]: https://github.com/joa23/linear-cli/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/joa23/linear-cli/releases/tag/v1.0.0
