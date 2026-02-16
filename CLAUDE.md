# CLAUDE.md

Linear CLI - A token-efficient CLI for Linear, written in Go.

## Commands

```bash
make build    # Build binary (bin/linear)
make test     # Run unit tests
make clean    # Clean build artifacts
```

## Project Structure

```
cmd/linear/          # CLI entry point
internal/cli/        # CLI commands (Cobra)
internal/format/     # ASCII formatters for token-efficient output
internal/linear/     # Linear GraphQL client
internal/service/    # Service layer
internal/skills/     # Embedded Claude Code skills
internal/oauth/      # OAuth2 flow
internal/token/      # Secure token storage
```

## For AI Agents (Claude Code)

### Essential Setup

**Step 1: Authenticate and set mode**
```bash
linear auth login    # Choose "Agent" mode for bots/automation
linear auth status   # Verify: should show "Mode: Agent"
```

**Step 2: Set default team**
```bash
linear init          # Select default team - creates .linear.yaml
```

### Authentication Modes

- **User mode**: `--assignee me` assigns to the human's Linear account
- **Agent mode**: `--assignee me` assigns to the OAuth app (delegate), visible as a separate entity in Linear

If `linear auth status` shows "Auth mode not set", re-run `linear auth login`.

### Output Formats

**IMPORTANT**: All commands now support JSON output for automation and scripting.

**Text Output (default)** - Token-efficient ASCII format:
```bash
linear issues list
linear issues get CEN-123
```

**JSON Output** - Machine-readable for parsing:
```bash
linear issues list --output json
linear issues get CEN-123 --output json
```

**Verbosity Levels** (`--format` flag):
- `minimal` - Essential fields only (~50 tokens)
- `compact` - Key metadata (~150 tokens, default)
- `full` - Complete details (~500 tokens)

**When to use JSON:**
- Parsing data programmatically
- Filtering with jq or similar tools
- Storing results for later processing
- Integrating with other tools

**Examples for AI Agents:**
```bash
# Get all high-priority issues as JSON for processing
linear issues list --priority 1 --output json

# Export cycle analysis for reporting
linear cycles analyze --team CEN --output json

# Parse specific fields with jq
linear issues list --output json | jq '.[] | {id: .identifier, title: .title, priority: .priority}'

# Get minimal JSON for quick checks
linear issues get CEN-123 --format minimal --output json
```

**All commands with --output support:**
- `issues list`, `issues get`
- `cycles list`, `cycles get`, `cycles analyze`
- `projects list`, `projects get`
- `teams list`, `teams get`, `teams labels`, `teams states`
- `users list`, `users get`, `users me`
- `search` (all search operations)
- `attachments list`, `attachments create`, `attachments update`

### Common Patterns

#### Listing Issues
```bash
# List all issues (NOT just assigned - this is the default behavior)
linear issues list --format full

# Filter by state
linear issues list --team CEN --state "In Progress" --format full

# Filter by priority (1=urgent, 2=high, 3=normal, 4=low)
linear issues list --priority 1 --format full

# Get issues in specific cycle
linear issues list --cycle 65 --format full

# Filter by assignee
linear issues list --assignee me --format full

# Combine filters
linear issues list --state Backlog --labels customer --priority 1 --format full
```

#### Cycle Operations
```bash
# CRITICAL: Analyze velocity BEFORE planning cycles
linear cycles analyze --team CEN --count 10

# Get cycle details (by number - requires 'linear init')
linear cycles get 65

# List all cycles
linear cycles list --team CEN --format full

# List only active cycles
linear cycles list --active
```

#### Creating Issues
```bash
# Minimal - requires 'linear init' first
linear issues create "Fix authentication bug"

# Complete with all parameters
linear issues create "Implement feature" \
  --team CEN \
  --state "In Progress" \
  --priority 1 \
  --assignee me \
  --estimate 5 \
  --cycle 65 \
  --labels "backend,security"
```

#### Working with Team Context
- Cycle numbers (65, 66) require team context from `linear init`
- Issue identifiers (CEN-123) work without team context
- UUIDs work without team context

### Error Handling

| Error | Solution |
|-------|----------|
| `--team is required` | Run `linear init` or add `--team CEN` flag |
| `Entity not found: Cycle` | Verify team context: run `linear init` |
| `Unknown type "IssueOrderBy"` | Update CLI to latest version |

### Search & Dependency Management

The `linear search` command is powerful for discovery and dependency management:

#### Finding Blocked Work
```bash
# Find ALL issues with blockers (weekly routine to unblock work)
linear search --has-blockers --team CEN

# Find high-priority blocked work
linear search --priority 1 --has-blockers --team CEN

# Find what's blocked by a specific bottleneck
linear search --blocked-by CEN-123
```

#### Discovering Dependencies
```bash
# Find related work by keyword
linear search "authentication" --team CEN

# Check if work has dependencies
linear search "OAuth" --has-dependencies --team CEN

# Find complex dependency chains
linear search --max-depth 5 --team CEN

# Detect circular dependencies (always fix these!)
linear search --has-circular-deps --team CEN
```

#### Best Practices
1. **Weekly unblocking routine**: Run `linear search --has-blockers` to find stuck work
2. **Before creating issues**: Search first to avoid duplicates
3. **After creating issues**: Use `/link-deps` skill to establish dependencies
4. **Sprint planning**: Check `linear deps --team CEN` for work order
5. **Priority alignment**: Ensure foundation work is prioritized over features it blocks

#### Attachments (sidebar cards)

Attachment objects are structured sidebar cards (GitHub PRs, Slack threads, uploaded files, URLs).
This is different from `--attach` on `issues create/update/comment/reply`, which embeds files as inline markdown images.

```bash
# List attachment cards on an issue
linear attachments list TEC-123

# Attach a URL (sidebar card, auto-detects source type)
linear attachments create TEC-123 --url "https://github.com/org/repo/pull/42" --title "PR #42"

# Upload a file as attachment card (title defaults to filename)
linear attachments create TEC-123 --file /tmp/screenshot.png

# Upload with custom title
linear attachments create TEC-123 --file /tmp/screenshot.png --title "Bug screenshot"

# Update attachment metadata
linear attachments update <uuid> --title "Updated title"

# Delete attachment
linear attachments delete <uuid>

# Embed file as inline image in description (NOT a sidebar card)
linear issues create "Bug" --attach /tmp/screenshot.png
```

### Skills Usage

After running `linear init`, use these skills:
- `/linear` - **NEW** Complete CLI reference (MUST READ before using Linear commands)
- `/prd` - Create agent-friendly tickets (searches for existing work first)
- `/triage` - Prioritize backlog issues (uses search to find blocked work)
- `/cycle-plan` - Analyze and plan cycles
- `/retro` - Generate sprint retrospectives
- `/deps` - Visualize dependency graphs
- `/link-deps` - Discover and link missing dependencies across backlog

## CLI Commands

### Dependency Graph
```bash
linear deps ENG-100          # Show deps for issue
linear deps --team ENG       # Show all deps for team
```

### Skills Management
```bash
linear skills list           # List available skills
linear skills install --all  # Install all skills
linear skills install prd    # Install specific skill
```

Available skills: `/linear`, `/prd`, `/triage`, `/cycle-plan`, `/retro`, `/deps`, `/link-deps`

## Key Design Decisions

- **ASCII output** - Token-efficient, no JSON overhead
- **JSON output** - Machine-readable for automation via `--output json`
- **Human-readable IDs** - "TEST-123" not UUIDs
- **Service layer** - Validation and formatting abstraction
- **Verbosity levels** - Control detail with `--format minimal|compact|full`

## Testing

Run the full test suite before committing:
```bash
make test                    # Run unit tests
./test-release.sh           # Run comprehensive release test against TEST team
```

## Releasing

**IMPORTANT**: Do NOT run `goreleaser` locally. Let GitHub Actions handle the build and upload.

### Release Workflow

1. **Update CHANGELOG.md** with the new version and changes
2. **Commit** the changelog: `git commit -m "chore: Update CHANGELOG for vX.Y.Z"`
3. **Create and push tag**:
   ```bash
   git tag vX.Y.Z && git push && git push --tags
   ```
4. **Wait for GitHub Actions** to complete the release workflow (~2 min)
   - Monitor at: `gh run watch` or check GitHub Actions tab
   - The workflow runs goreleaser and uploads all artifacts
5. **Get checksums** from the release:
   ```bash
   gh release download vX.Y.Z --pattern checksums.txt --output -
   ```
6. **Update Homebrew formula** (`Formula/linear-cli.rb`):
   - Update `version "X.Y.Z"`
   - Update all SHA256 checksums for each platform
7. **Commit and push formula**:
   ```bash
   git add Formula/linear-cli.rb
   git commit -m "chore: Update formula to vX.Y.Z"
   git push
   ```
8. **Install locally** to verify:
   ```bash
   cd /opt/homebrew/Library/Taps/joa23/homebrew-linear-cli && git pull && cd -
   brew upgrade linear-cli
   linear --version
   ```

### Why Not Local goreleaser?

Running `goreleaser release` locally and then having GitHub Actions also run it causes duplicate upload errors. The CI workflow is the source of truth for release artifacts.

## Session Completion

1. `make test` must pass
2. Run `./test-release.sh` for manual verification (optional but recommended for releases)
3. Commit with clear messages
4. Push to remote - work is NOT complete until pushed
