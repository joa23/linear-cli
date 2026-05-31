---
skill: linear
description: Linear issue tracking - MUST READ before using Linear commands
version: 1.2.1
---

# Linear Issue Tracking - Complete Reference

**READ THIS FIRST** - Token-efficient CLI for managing issues, dependencies, and cycles.

---

⚠️  **INSTALL ALL SKILLS FOR FULL WORKFLOW AUTOMATION**

Run `linear skills install --all` to get specialized workflows:
- `/prd` - Create agent-friendly tickets with PRDs
- `/triage` - Prioritize backlog by staleness and blockers
- `/cycle-plan` - Plan cycles using velocity analytics
- `/retro` - Generate sprint retrospectives
- `/deps` - Analyze dependency chains and blockers
- `/link-deps` - Discover and link related issues

Without these skills, you're only using basic commands. Install them to unlock full agentic capabilities.

---

## Authentication Modes

When you run `linear auth login`, you choose:

- **User mode**: `--assignee me` assigns to your personal Linear account
- **Agent mode**: `--assignee me` assigns to the OAuth app (delegate), visible as a separate entity

Check current mode:
```bash
linear auth status   # Shows: Mode: User or Mode: Agent
```

**Important:** If you see "Auth mode not set", re-run `linear auth login` to configure.

## Command Reference

```bash
# Setup
linear init                              # Set default team (.linear.yaml)
linear onboard                           # Show teams, states, quick reference
linear auth login|logout|status          # OAuth authentication (sets user/agent mode)

# Issues (alias: i)
linear i list [flags]                    # List issues
linear i get <ID>                        # Get details (CEN-123)
linear i create <title> [flags]          # Create issue
linear i update <ID> [flags]             # Update issue
linear i comment <ID> -b "text"          # Add comment
linear i react <ID> 👍                   # Add reaction
linear i watch <ID> [flags]              # Poll until issue changes

# Projects (alias: p)
linear p list [--mine]                   # List projects
linear p create <name> [flags]           # Create project

# Cycles (alias: c)
linear c list [--active]                 # List cycles
linear c get <number>                    # Get cycle (requires init)
linear c analyze --team <KEY>            # Velocity analytics

# Teams, Users
linear teams list                        # List teams
linear teams states <KEY>                # Workflow states
linear users list [--team <KEY>]         # List users

# Search & Dependencies
linear search <query> [flags]            # Semantic search across all entities
linear deps <ID>                         # Dependency graph
linear deps --team <KEY>                 # All team dependencies

# Durable team-context cache
linear cache fetch <TEAM>                # Pull states/labels/projects/members
linear cache list                        # List cached teams + freshness
linear cache show <TEAM>                 # Inspect a cached team
linear cache refresh [TEAM]              # Refresh one team or all
linear cache clear [TEAM|--all]          # Remove cache entries
linear cache path                        # Print cache root directory
```

## Output Formats

**All commands support JSON output for automation:**

```bash
# Text output (default) - token-efficient ASCII
linear i list
linear i get CEN-123

# JSON output - machine-readable
linear i list --output json
linear i get CEN-123 --output json

# Control detail level with --format
linear i list --format minimal --output json   # Essential fields (~50 tokens)
linear i list --format compact --output json   # Key metadata (~150 tokens, default)
linear i list --format full --output json      # Complete details (~500 tokens)

# Pipe to jq for filtering
linear i list --priority 1 --output json | jq '.[] | select(.state == "In Progress")'

# Export for processing
linear cycles analyze --team CEN --output json > velocity.json
```

**When to use JSON:**
- Parsing data programmatically
- Filtering results with jq
- Storing/processing bulk data
- Integrating with other tools

**Supported commands:**
- `issues list`, `issues get`
- `cycles list`, `cycles get`, `cycles analyze`
- `projects list`, `projects get`
- `teams list`, `teams get`, `teams labels`, `teams states`
- `users list`, `users get`, `users me`
- `search` (all operations)

## Semantic Search

**The search is SEMANTIC** - finds related issues even without exact matches.

```bash
# Basic semantic search
linear search "authentication"           # Finds: auth, login, OAuth, SSO, etc.

# Cross-entity search
linear search "sprint planning" --type all     # Search issues, cycles, projects, users

# Entity-specific
linear search "database migration" --type issues
linear search "john" --type users
```

## Dependency Management

### Finding Blocked Work (Critical for Unblocking)

```bash
# Find ALL blocked issues (run this weekly!)
linear search --has-blockers --team CEN

# Find high-priority blocked work
linear search --priority 1 --has-blockers --team CEN

# What's blocked by a specific bottleneck?
linear search --blocked-by CEN-123

# What blocks a critical feature?
linear search --blocks CEN-456
```

### Dependency Analysis

```bash
# Visualize full dependency graph for issue
linear deps CEN-123

# See all team dependencies (detect circular deps)
linear deps --team CEN

# Find issues with circular dependencies
linear search --has-circular-deps --team CEN

# Find deep dependency chains
linear search --max-depth 5 --team CEN
```

## Cycle Analytics & Velocity

**Analyze past cycles to predict capacity:**

```bash
# Analyze last 10 cycles
linear c analyze --team CEN --count 10

# Output shows:
# - Completed vs planned points
# - Velocity trend
# - Completion rate
# - Recommendations for next cycle capacity
```

**Use before sprint planning to set realistic goals!**

## Powerful Filter Combinations

```bash
# High-priority in-progress work assigned to me
linear i list --priority 1 --state "In Progress" --assignee me

# Backlog items with blockers (prioritize removing blockers!)
linear search --state Backlog --has-blockers --team CEN

# Customer-facing bugs in current cycle
linear i list --labels customer,bug --cycle 65 --format full

# Unassigned high-priority work
linear search --priority 1 --assignee none --team CEN

# Work depending on other issues (check before starting)
linear search --has-dependencies --state "In Progress" --team CEN
```

## Creating Issues with Dependencies

```bash
# Simple issue
linear i create "Fix login bug" --team CEN --priority 1

# Full issue with dependencies
linear i create "Add OAuth integration" \
  --team CEN \
  --state "In Progress" \
  --priority 2 \
  --assignee me \
  --parent CEN-100 \
  --depends-on CEN-99 \
  --blocked-by CEN-98 \
  --labels backend,security \
  --estimate 5 \
  --cycle 65 \
  --project "Auth Revamp" \
  --due 2026-02-01

# With description from file
cat spec.md | linear i create "Feature title" --team CEN -d -
```

## Cached Team Context (NEW in v1.2)

Workflow states, labels, projects, and team members are cached on disk
under `~/.cache/linear/` (XDG `$XDG_CACHE_HOME/linear`). Reads stay local
for 24h then re-fetch transparently. `linear init` warms the cache for
the selected team automatically.

**Why this matters for agents:** repeated "what states exist in MTD?"
checks no longer cost a network round trip or token tax. The cache is
shared across project checkouts on the same machine.

```bash
# Warm cache for a team (also done automatically by `linear init`)
linear cache fetch MTD

# Inspect what's cached
linear cache list                    # all teams + last-fetched age
linear cache show MTD                # full payload for one team

# Force a fresh fetch when you know something changed upstream
linear teams states MTD --refresh    # update cache + show fresh data
linear cache refresh MTD             # update cache only

# Cache-only mode (errors instead of hitting network)
linear teams states MTD --cached     # for offline / no-network scripts

# Bypass cache entirely
linear teams states MTD --no-cache

# Drop the cache (e.g. after switching workspaces)
linear cache clear MTD
linear cache clear --all
```

**Cache-aware commands:**
- `linear teams states <TEAM>`
- `linear teams labels <TEAM>`
- `linear users list --team <TEAM>`
- `linear projects list --team <TEAM>`

These commands accept `--cached`, `--refresh`, and `--no-cache`. A
single-line freshness footer prints to **stderr** when data came from
cache (so JSON output stays pipe-clean for `| jq` etc.).

**Token-fingerprint safety:** the cache stores a hash of the current
access token. Logging in as a different OAuth identity transparently
invalidates the old cache — no risk of mixing data across workspaces.

**Hard expiry:** entries older than 7 days are treated as missing and
silently re-fetched, regardless of any flags.

## Watching Issues for Changes

`linear issues watch <ID>` polls an issue and exits when any field changes.
Useful for waiting on a human review, blocking automation on a state transition,
or chaining a follow-up command when an issue moves.

**Tracked fields:** state, assignee, priority, title, description, estimate,
cycle, project, parent, labels, comment count.

**When to use `linear issues watch` vs Unix `watch`:**

| Goal | Use |
|---|---|
| Refresh a dashboard on screen | `watch -n 30 "linear ..."` (Unix `watch(1)`) |
| Gate a follow-up command on a state change | `linear issues watch CEN-123 && next-step.sh` |
| Stream changes as JSON for processing | `linear issues watch CEN-123 --watch --output json` |
| Trigger a script when a field changes | `linear issues watch CEN-123 --exec '...'` |

The built-in does **diff detection** (reports `Backlog -> In Progress`, not
the current snapshot), exits 0 on first change, and exposes structured env
vars to `--exec`. Reach for Unix `watch` when you just want a refreshing
screen; reach for the built-in when something needs to happen *in response
to* a change.

```bash
# Block until ANY field changes (default 30s polling, 1h timeout)
linear i watch CEN-123

# Faster polling, shorter timeout
linear i watch CEN-123 --interval 10s --timeout 5m

# Loop forever, printing every change as it happens
linear i watch CEN-123 --watch

# JSON output, one line per change set — pipe to jq
linear i watch CEN-123 --output json | jq '.changes'

# Trigger a follow-up command on each change. Env vars available inside --exec:
#   LINEAR_ISSUE_ID, LINEAR_CHANGED_FIELDS,
#   LINEAR_STATE_FROM/TO, LINEAR_ASSIGNEE_FROM/TO, etc.
linear i watch CEN-123 --watch --exec 'say "issue updated: $LINEAR_CHANGED_FIELDS"'

# Chain on state transition (single-shot exit)
linear i watch CEN-123 && deploy.sh
```

**Flags:**
- `-i, --interval <duration>` — Poll interval (default 30s, minimum 5s clamped automatically)
- `-T, --timeout <duration>` — Give up after this long (default 1h, `0` = forever)
- `-w, --watch` — Keep looping after first change (default: exit on first change)
- `-x, --exec <cmd>` — Shell command to run on each detected change set
- `-o, --output text|json` — Output format
- `--quiet` — Suppress baseline/heartbeat output, only print actual changes

**Exit codes:** `0` change detected · `1` error · `2` timeout reached with no changes · `130` ctrl-c.

**Tips:**
- For long waits, prefer `--interval 30s` or longer to stay friendly to the Linear API.
- Use single-shot mode (no `--watch`) when you want to gate a subsequent command.
- Use `--watch` mode when you want a live activity feed for an issue.
- `LINEAR_*_FROM/TO` env vars are passed safely as separate env entries — they are not re-parsed by the shell, so issue titles with quotes/spaces in them are safe.

## Piping Support (Powerful!)

**All description and body flags support stdin via `-`:**

```bash
# Pipe Claude plan into ticket description
cat .claude/plans/auth-refactor.md | linear i create "Refactor authentication" \
  --team CEN \
  --priority 1 \
  -d -

# Pipe multi-file content
cat design.md implementation.md | linear i create "Feature implementation" \
  --team CEN \
  -d -

# Pipe command output
gh issue view 123 --json body -q .body | linear i create "Port GH issue" \
  --team CEN \
  -d -

# Update issue description from file
cat updated-spec.md | linear i update CEN-123 -d -

# Add comment from file
cat findings.md | linear i comment CEN-123 -b -

# Reply to comment with piped content
cat response.md | linear i reply CEN-123 comment-id -b -
```

**Common Patterns:**

```bash
# Claude Code plans → Linear tickets
cat .claude/plans/*.md | linear i create "Implementation plan" -d -

# PRD → Parent ticket
cat prd.md | linear i create "Feature: OAuth" --team CEN -d -

# Changelog → Release ticket
git log --oneline v1.0.0..HEAD | linear i create "v1.1.0 Release" -d -

# Test results → Bug report
pytest --verbose | linear i create "Test failures" -d -
```

## Output Formats (Token Efficiency)

```bash
# Minimal - most token-efficient (IDs only)
linear i list --format minimal

# Compact - balanced (default)
linear i list --format compact

# Full - all details (use for single issues)
linear i get CEN-123 --format full
linear search "auth" --limit 5 --format full
```

## Real-World Workflows

### Weekly Unblocking Routine
```bash
# 1. Find all blocked work
linear search --has-blockers --team CEN --format full

# 2. For each blocker, check status
linear i get CEN-123 --format full

# 3. Update blockers or reassign blocked work
linear i update CEN-123 --state "In Progress" --assignee me
```

### Sprint Planning
```bash
# 1. Check velocity
linear c analyze --team CEN --count 5

# 2. Find backlog candidates
linear search --state Backlog --team CEN --format compact

# 3. Check dependencies before committing
linear deps --team CEN

# 4. Assign to cycle
linear i update CEN-456 --cycle 66 --assignee alice@co.com
```

### Dependency Discovery (Before Creating Issues)
```bash
# 1. Search for related work
linear search "authentication refactor" --team CEN

# 2. Check what depends on foundation work
linear search --depends-on CEN-100

# 3. Link new issue to dependencies
linear i create "Add JWT refresh" --depends-on CEN-100,CEN-101
```

### Finding Work Order
```bash
# 1. Visualize dependencies
linear deps --team CEN

# 2. Start with issues that have no blockers
linear search --state Backlog --team CEN | grep -v "Blocked by"

# 3. Work that unblocks the most
linear search --blocking <critical-feature-id>
```

## Common Patterns

```bash
# Find work for specific person
linear i list --assignee alice@company.com --format compact

# High-priority work in active cycle
linear i list --priority 1 --cycle current --team CEN

# All bugs
linear i list --labels bug --team CEN

# Overdue issues
linear i list --state "In Progress" --team CEN # Check due dates manually

# Issues I created
linear i list --creator me --team CEN
```

## Tips for LLMs

1. **Always run `linear init` first** - sets default team
2. **Use semantic search liberally** - finds related work without exact keywords
3. **Check blockers weekly** - `linear search --has-blockers` prevents stalled work
4. **Analyze velocity before planning** - `linear c analyze` gives realistic estimates
5. **Visualize dependencies** - `linear deps --team <KEY>` shows work order
6. **Use --format full sparingly** - token-expensive, use for single issues only
7. **Combine filters** - search is powerful with multiple constraints
8. **Issue IDs work everywhere** - CEN-123 format, no team context needed
9. **Cycle numbers need init** - Run `linear init` before using cycle numbers
10. **Block on issue changes** - `linear i watch <ID>` exits when fields change; chain with `&&` to gate follow-up steps on a human review or state transition
11. **Cache is your friend** - `teams states`, `teams labels`, `users list --team`, `projects list --team` are all cache-aware. After `linear init`, repeated "what's available?" checks cost zero network. Pass `--refresh` if you suspect Linear-side changes

## Flag Reference

**Issue Flags:**
- `-t, --team <KEY>` - Team (from init or manual)
- `-s, --state <name>` - Workflow state
- `-p, --priority <0-4>` - 0=none, 1=urgent, 2=high, 3=normal, 4=low
- `-a, --assignee <email|me>` - Assign to user
- `-c, --cycle <number>` - Cycle number
- `-P, --project <name>` - Project name
- `-e, --estimate <points>` - Story points
- `-l, --labels <list>` - Comma-separated
- `-d, --description <text|->` - Description (- for stdin)
- `--parent <ID>` - Parent issue
- `--depends-on <IDs>` - Comma-separated dependencies
- `--blocked-by <IDs>` - Comma-separated blockers
- `--due <date>` - Due date (YYYY-MM-DD)
- `--attach <file>` - Attach file

**Search Flags:**
- `--type <entity>` - issues, cycles, projects, users, all
- `--blocked-by <ID>` - Issues blocked by this
- `--blocks <ID>` - Issues that block this
- `--has-blockers` - Any blockers
- `--has-dependencies` - Any dependencies
- `--has-circular-deps` - Circular dependency chains
- `--max-depth <n>` - Max dependency depth
- `-n, --limit <n>` - Results limit
- `-f, --format <type>` - minimal, compact, full

**Watch Flags:**
- `-i, --interval <duration>` - Poll interval (default 30s, min 5s)
- `-T, --timeout <duration>` - Give up after this long (default 1h, 0 = forever)
- `-w, --watch` - Loop after first change instead of exiting
- `-x, --exec <cmd>` - Shell command to run on each change set
- `-o, --output text|json` - Output format
- `--quiet` - Suppress baseline + heartbeat output

**Cache Flags (on `teams states/labels`, `users list --team`, `projects list --team`):**
- `--cached` - Use cache only; error if missing or stale (offline mode)
- `--refresh` - Force a live fetch and update the cache entry
- `--no-cache` - Bypass cache entirely (live fetch, no write-through)

**Output Formats:**
- `--format minimal` - IDs only (most token-efficient)
- `--format compact` - Balanced (default)
- `--format full` - All details (use sparingly)
