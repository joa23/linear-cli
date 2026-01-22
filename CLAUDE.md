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

**ALWAYS run this first** before any other Linear commands:
```bash
linear init  # Select default team - required for cycle operations
```

This creates `.linear.yaml` with your default team. Without this, many commands will fail with "team is required".

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

### Skills Usage

After running `linear init`, use these skills:
- `/prd` - Create agent-friendly tickets (searches for existing work first)
- `/triage` - Prioritize backlog issues (uses search to find blocked work)
- `/cycle-plan` - Analyze and plan cycles
- `/retro` - Generate sprint retrospectives
- `/deps` - Visualize dependency graphs
- `/link-deps` - **NEW** Discover and link missing dependencies across backlog
- `/release` - **NEW** Pre-release checklist ensuring CHANGELOG is updated

## CLI Commands

### Dependency Graph
```bash
linear deps ENG-100          # Show deps for issue
linear deps --team ENG       # Show all deps for team
```

### Unified Search with Dependency Filtering
```bash
# Basic search across all issues
linear search "authentication"

# Search with standard filters
linear search --state "In Progress" --priority 1 --team CEN

# Find issues blocked by a specific issue
linear search --blocked-by CEN-123

# Find issues that block a specific issue
linear search --blocks CEN-100

# Find all issues with any blockers
linear search --has-blockers --team CEN

# Find all issues with dependencies
linear search --has-dependencies --team CEN

# Find issues in circular dependency chains
linear search --has-circular-deps --team CEN

# Filter by max dependency depth
linear search --max-depth 3 --team CEN

# Combine text search with dependency filters
linear search "auth" --has-blockers --priority 1

# Cross-entity search (issues, cycles, projects, users)
linear search "oauth" --type all

# Search specific entity types
linear search "cycle 65" --type cycles
linear search "auth project" --type projects
linear search "john" --type users
```

**Dependency Filter Options:**
- `--blocked-by <ID>` - Find issues blocked by specified issue
- `--blocks <ID>` - Find issues that block specified issue
- `--has-blockers` - Find issues with any blocking issues
- `--has-dependencies` - Find issues that depend on others
- `--has-circular-deps` - Find issues in circular dependency chains
- `--max-depth <n>` - Filter by maximum dependency chain depth

**Entity Types:**
- `issues` (default) - Search issues with full filtering
- `cycles` - Search cycles by name/number
- `projects` - Search projects by name
- `users` - Search users by name/email
- `all` - Search across all entity types

### Skills Management
```bash
linear skills list           # List available skills
linear skills install --all  # Install all skills
linear skills install prd    # Install specific skill
```

Available skills: `/prd`, `/triage`, `/cycle-plan`, `/retro`, `/deps`

## Key Design Decisions

- **ASCII output** - Token-efficient, no JSON overhead
- **Human-readable IDs** - "TEST-123" not UUIDs
- **Service layer** - Validation and formatting abstraction

## Testing

```bash
go test -v ./internal/linear -run TestCreateIssue
go test -cover ./...
```

## Session Completion

1. `make test` must pass
2. Commit with clear messages
3. Push to remote - work is NOT complete until pushed
