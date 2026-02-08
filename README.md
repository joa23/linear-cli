# Linear CLI

## Why Linear CLI?

**The economics of code have changed.** A decade ago, a line of production code cost [$15-50 to write](https://wiki.c2.com/?CostOfLineOfCode). Today, AI coding assistants generate code for pennies. This explosion in code generation creates a new bottleneck: **coordination**.

Agentic coding works brilliantly for individual developers. But scaling it to teams—where multiple agents, human developers, and product managers collaborate—remains unsolved.

Linear CLI bridges this gap:

- **For agentic coding**: Agents read and update Linear issues as they work, maintaining shared state across your entire team
- **For agentic product management**: AI assistants can search, triage, and analyze your backlog alongside human PMs
- **For hybrid teams**: Human and AI contributors work from the same source of truth

Linear becomes the coordination layer—tracking what's planned, what's in progress, and what's done—while agents and humans focus on building.

---

## Quickstart

Install with Homebrew:

```bash
brew tap joa23/linear-cli
brew install linear-cli
```

Then configure in 3 steps:

```bash
# 1. Authenticate
linear auth login

# 2. Initialize your project (select default team)
linear init

# 3. Install Claude Code skills (optional but recommended)
linear skills install --all
```

You're ready! Try:
```bash
linear issues list
linear issues create "My first issue" --team YOUR-TEAM
```

### Updating

Already installed? Update to the latest version:

```bash
brew upgrade linear-cli
```

If you encounter issues after updating, try re-authenticating:
```bash
linear auth logout
linear auth login
```

**Other installation methods:** See [Installation](#installation) below for manual download options.

**Authentication setup:** See [Authentication](#authentication) for OAuth configuration details.

---

## Table of Contents

- [Why Linear CLI?](#why-linear-cli)
- [Quickstart](#quickstart)
  - [Updating](#updating)
- [Installation](#installation)
- [Authentication](#authentication)
- [CLI Commands](#cli-commands)
  - [Output Formats & JSON](#output-formats)
  - [JSON Automation Examples](#json-automation-examples)
  - [Claude Code Task Export](#claude-code-task-export)
  - [Issues](#issues)
  - [Search](#search)
  - [Dependencies](#dependencies)
  - [Projects](#projects)
  - [Cycles](#cycles)
  - [Teams](#teams)
  - [Labels](#labels)
  - [Users](#users)
  - [Claude Code Skills](#claude-code-skills)
- [Cycle Analytics](#cycle-analytics)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [License](#license)

---

> **Token-efficient CLI for [Linear](https://linear.app).** Human-readable identifiers ("TEST-123" not UUIDs), smart caching, and ASCII output—90% fewer tokens than alternatives.

---

## Installation

### Homebrew (Recommended)

```bash
brew tap joa23/linear-cli https://github.com/joa23/linear-cli
brew install linear-cli
```

### Manual Download

**macOS:**
```bash
# Apple Silicon
curl -L https://github.com/joa23/linear-cli/releases/latest/download/linear-darwin-arm64.tar.gz | tar -xz
chmod +x linear

# Intel
curl -L https://github.com/joa23/linear-cli/releases/latest/download/linear-darwin-amd64.tar.gz | tar -xz
chmod +x linear
```

**Linux:**
```bash
curl -L https://github.com/joa23/linear-cli/releases/latest/download/linear-linux-amd64.tar.gz | tar -xz
chmod +x linear
```

### Build from Source

```bash
git clone https://github.com/joa23/linear-cli.git
cd linear-cli
make build
```

---

## Authentication

```bash
linear auth login
```

You'll be prompted to choose an authentication mode:

**Personal Use** - Authenticate as yourself
- Your actions appear under your Linear account
- `--assignee me` assigns to your personal account
- For personal task management

**Agent/App** - Authenticate as an agent
- Agent appears as a separate entity in Linear
- `--assignee me` assigns to the agent (uses delegate field)
- Requires admin approval to install
- Agent can be @mentioned and assigned issues
- For automation, bots, and integrations

### OAuth Setup

#### Personal Mode Setup

During login, you'll be prompted for:

1. **OAuth callback port** (default: 37412)
   - Press Enter to use default, or specify a custom port
   - The wizard shows you the callback URL to configure

2. **Create an OAuth app in Linear:**
   - Go to Linear → Settings → API → OAuth Applications → New
   - Set callback URL: `http://localhost:YOUR_PORT/oauth-callback`
   - Copy the Client ID and Secret

3. **Enter credentials** when prompted

Credentials (including port) are saved to `~/.config/linear/config.yaml`.

#### Agent Mode Setup

For agent/automation use (bots, CI/CD, integrations):

1. **Create an OAuth app in Linear:**
   - Go to Linear → Settings → API → OAuth Applications → New
   - Name: `linear-cli-agent` (or your preferred name)
   - **Callback URL: `http://localhost:3000/oauth-callback`**
   - Actor: Application (not User)
   - Click "Create"

2. **Save your credentials:**
   - **Client ID**: Copy and save this (you'll need it for login)
   - **Client Secret**: Copy and save this immediately (shown only once)
   - Store these securely - you'll need them to authenticate

3. **Authenticate:**
   ```bash
   linear auth login
   # Select "Agent/App" mode
   # Enter port: 3000
   # Enter Client ID and Secret when prompted
   ```

4. **Admin approval required:**
   - After authentication, a Linear admin must approve the app installation
   - Go to Linear → Settings → Installed Applications
   - Approve the pending installation

Once approved, the agent can be @mentioned and assigned issues like any team member.

### Other Auth Commands

```bash
linear auth status   # Check login status and auth mode
linear auth logout   # Remove credentials
```

The status command shows your current auth mode:
```
✅ Logged in to Linear
User: Your Name (email@example.com)
Mode: Agent (--assignee me uses delegate)
```

---

## CLI Commands

### Output Formats

All list and get commands support two output formats:

**Text Output (default)**: Token-efficient ASCII format optimized for LLMs
```bash
linear issues get ENG-123
linear issues list --team ENG
```

**JSON Output**: Machine-readable format for scripting and automation
```bash
linear issues get ENG-123 --output json
linear issues list --team ENG --output json
```

**Verbosity Levels**: Control detail level with `--format`
- `minimal` - Essential fields only (~50 tokens/issue)
- `compact` - Key metadata (~150 tokens/issue, default)
- `full` - Complete details (~500 tokens/issue)

**Examples:**
```bash
# Minimal JSON for quick parsing
linear issues list --format minimal --output json

# Full detail as text (default)
linear issues get ENG-123 --format full

# Pipe JSON to jq for filtering
linear issues list --output json | jq '.[] | select(.priority == 1)'

# Export all high-priority issues to file
linear issues list --priority 1 --output json > high-priority.json
```

All commands support both flags:
- **Issues**: `list`, `get`
- **Cycles**: `list`, `get`, `analyze`
- **Projects**: `list`, `get`
- **Teams**: `list`, `get`, `labels`, `states`
- **Labels**: `list`
- **Users**: `list`, `get`, `me`
- **Search**: all search operations

### JSON Automation Examples

Powerful recipes for scripting and automation:

```bash
# Get all urgent issues and extract just identifiers
linear issues list --priority 1 --output json | jq -r '.[].identifier'

# Find all issues in a project
linear issues list --team ENG --output json | \
  jq '.[] | select(.projectName == "Q1 Release")'

# Count issues by state
linear issues list --team ENG --output json | \
  jq 'group_by(.state) | map({state: .[0].state, count: length})'

# Get all unassigned high-priority issues
linear issues list --priority 2 --output json | \
  jq '.[] | select(.assignee == null) | {identifier, title, state}'

# Export cycle velocity data to CSV
linear cycles analyze --team ENG --output json | \
  jq -r '.cycles[] | [.number, .completedPoints, .plannedPoints] | @csv'

# Find all issues created in the last 7 days
linear issues list --team ENG --output json | \
  jq --arg date "$(date -u -v-7d +%Y-%m-%d)" \
  '.[] | select(.createdAt >= $date) | {identifier, title, createdAt}'

# Bulk update: Get all backlog items for processing
BACKLOG_IDS=$(linear search --state Backlog --team ENG --output json | jq -r '.[].identifier')
for id in $BACKLOG_IDS; do
  echo "Processing $id..."
  # Add your automation here
done

# Generate weekly status report
cat << 'EOF' > weekly-report.sh
#!/bin/bash
TEAM="ENG"
echo "=== Weekly Status Report ==="
echo ""
echo "High Priority In Progress:"
linear issues list --team $TEAM --priority 1 --state "In Progress" --output json | \
  jq -r '.[] | "  - \(.identifier): \(.title) (@\(.assignee // "unassigned"))"'
echo ""
echo "Blocked Issues:"
linear search --has-blockers --team $TEAM --output json | \
  jq -r '.[] | "  - \(.identifier): \(.title)"'
EOF
chmod +x weekly-report.sh
```

**Pro tip**: Combine with watch for live monitoring:
```bash
# Monitor high-priority issues every 30 seconds
watch -n 30 "linear issues list --priority 1 --output json | jq '.[] | {id: .identifier, title: .title, state: .state}'"
```

### Claude Code Task Export

Export Linear issues to Claude Code task format for seamless integration between planning and execution:

```bash
# Export single issue and its dependencies
linear tasks export CEN-123 ./my-tasks/

# Export directly to Claude Code session
linear tasks export CEN-123 ~/.claude/tasks/a5721284-f64e-4705-8983-b7d6c4e032aa/

# Preview without writing files
linear tasks export CEN-123 ./my-tasks/ --dry-run
```

**Features:**
- **Recursive export**: Automatically includes children and blocking dependencies
- **Circular dependency detection**: Errors if cycles detected (e.g., A→B→A)
- **Bottom-up hierarchy**: Children block parent (matches Claude task semantics)
- **Idempotent**: Files named `{identifier}.json` (e.g., `CEN-123.json`) for safe re-export
- **Smart formatting**: Converts titles to present continuous (e.g., "Fix bug" → "Fixing bug")

**Output format** (matches Claude Code schema):
```json
{
  "id": "CEN-123",
  "subject": "Implement OAuth authentication",
  "description": "**Linear Issue:** CEN-123\n\n[Full description]...\n\n---\n**State:** In Progress\n**Priority:** 1\n**Assignee:** Stefan",
  "activeForm": "Implementing OAuth authentication",
  "status": "pending",
  "blocks": [],
  "blockedBy": ["CEN-124", "CEN-125"]
}
```

**Workflow:**
1. Plan work in Linear (issues, dependencies, estimates)
2. Export to Claude Code: `linear tasks export EPIC-100 ~/.claude/tasks/<session>/`
3. Claude Code loads tasks and executes in dependency order
4. Update Linear as work progresses

**For Claude Code Agents:**

Claude can export tasks directly to its own session by discovering its session ID from the transcript path:

```bash
# Claude discovers its session ID from hook input or transcript path
# Example: ~/.claude/projects/.../e162ccc7-f5a5-4328-b173-20ab7a0d13c5.jsonl
# Session ID: e162ccc7-f5a5-4328-b173-20ab7a0d13c5

# Export tasks to current session
linear tasks export CEN-123 ~/.claude/tasks/e162ccc7-f5a5-4328-b173-20ab7a0d13c5/

# Tasks immediately appear in Claude's task system
# Use TaskList, TaskGet, TaskUpdate to manage exported work
```

**Autonomous Workflows (Ralph Loop Integration):**

The Ralph Loop pattern enables continuous autonomous execution. Combine Linear task export with Claude Code stop hooks for fully autonomous development cycles:

```json
{
  "hooks": {
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": "linear tasks export $(linear issues list --assignee me --state 'In Progress' --format minimal --output json | jq -r '.[0].identifier' 2>/dev/null) ~/.claude/tasks/$CLAUDE_SESSION_ID/ 2>/dev/null || true"
      }]
    }]
  }
}
```

This creates a self-sustaining loop:
1. Claude completes current tasks
2. Stop hook triggers, exports next Linear issue to task queue
3. Claude picks up the new task and continues working
4. Repeat indefinitely until backlog is clear

Memory persists in git commits and Linear updates, not LLM context. Each iteration starts fresh with clean context but full awareness of completed work through file system state.

**Rita Vrataski Loop (Context-Preserving Alternative):**

Traditional Ralph Loops restart the CLI each iteration, losing session context. Task Injection keeps Claude in the same session indefinitely - the CLI exports issues directly into Claude's task folder, and Claude picks them up immediately without restart.

```bash
#!/bin/bash
# Vrataski Loop - continuous autonomous execution with context preservation
SESSION=~/.claude/tasks/$CLAUDE_SESSION_ID

while true; do
  # Get next To Do issue assigned to me
  ISSUE=$(linear issues list --assignee me --state "To Do" --limit 1 --output json | jq -r '.[0].identifier')
  [ -z "$ISSUE" ] && { sleep 60; continue; }

  # Move to In Progress and export to Claude
  linear issues update $ISSUE --state "In Progress"
  linear tasks export $ISSUE $SESSION

  # Wait for all tasks to complete (check task folder for pending tasks)
  while true; do
    PENDING=$(grep -l '"status": "pending"' $SESSION/*.json 2>/dev/null | wc -l)
    [ "$PENDING" -eq 0 ] && break
    sleep 10
  done

  # All tasks done - inject "create PR" task
  echo '{"id":"create-pr","subject":"Create PR for '$ISSUE'","status":"pending"}' > $SESSION/create-pr.json

  # Wait for PR task to complete
  while grep -q '"status": "pending"' $SESSION/create-pr.json 2>/dev/null; do
    sleep 10
  done

  # Update Linear and continue to next issue
  linear issues update $ISSUE --state "Done"
done
```

This pattern enables:
- **Context preservation**: Claude stays in the same session across the entire backlog
- **Prefix/postfix tasks**: Inject "create worktree" before work, "create PR" after
- **Agent farming**: Multiple scripts can write to the task folder, coordinating work for one or more Claude sessions
- **Linear as state machine**: External coordination layer - scripts inject, Claude executes

### Issues

```bash
# Basic operations
linear issues list                           # List your assigned issues
linear issues get ENG-123                    # Get issue details

# Pagination - offset-based for easy navigation
linear issues list                           # First 10 issues (default)
linear issues list --start 10 --limit 10     # Items 11-20
linear issues list --start 20 --limit 5      # Items 21-25

# Sorting
linear issues list --sort priority           # Sort by priority (highest first)
linear issues list --sort created            # Sort by creation date
linear issues list --sort updated --direction asc  # Oldest updated first

# Combine pagination and sorting
linear issues list --start 20 --limit 10 --sort created --direction asc

# Create with full context
linear issues create "Implement OAuth2 login" \
  --team ENG \
  --project "Q1 Auth Revamp" \
  --state "In Progress" \
  --priority 2 \
  --assignee me \
  --estimate 5 \
  --cycle current \
  --labels "backend,security" \
  --due 2026-03-31

# Create sub-issue with dependencies
linear issues create "Add Google OAuth provider" \
  --team ENG \
  --parent ENG-100 \
  --blocked-by ENG-99 \
  --depends-on ENG-98,ENG-97

# Update multiple fields at once
linear issues update ENG-123 \
  --title "Updated: OAuth login implementation" \
  --state Done \
  --assignee alice \
  --priority 1 \
  --labels "urgent,hotfix" \
  --due 2026-02-15

# Manage dependencies
linear issues update ENG-102 --blocked-by ENG-101
linear issues update ENG-103 --depends-on ENG-100,ENG-101

# File attachments
linear issues create "UI Bug" --team ENG --attach /tmp/screenshot.png
linear issues create "Bug report" --team ENG --attach img1.png --attach img2.png
linear issues update ENG-123 --attach /tmp/additional-context.png
linear issues comment ENG-123 --body "Here's the screenshot:" --attach /tmp/bug.png

# Piping content (powerful!)
cat .claude/plans/feature-plan.md | linear issues create "Implementation plan" --team ENG -d -
cat prd.md | linear issues create "Feature: OAuth" --team ENG --description -
cat bug-report.txt | linear issues comment ENG-123 --body -
cat response.md | linear issues reply ENG-123 comment-id --body -
```

### Search

Powerful unified search with dependency filtering:

```bash
# Basic text search
linear search "authentication"

# Search with standard filters
linear search --state "In Progress" --priority 1 --team ENG

# Find issues blocked by a specific issue
linear search --blocked-by ENG-123

# Find all issues with blockers (useful for unblocking work)
linear search --has-blockers --team ENG

# Find issues in circular dependencies
linear search --has-circular-deps --team ENG

# Combine text search with dependency filters
linear search "auth" --has-blockers --priority 1

# Cross-entity search (issues, cycles, projects, users)
linear search "oauth" --type all
```

**Dependency Filters:**
- `--blocked-by <ID>` - Find issues blocked by a specific issue
- `--blocks <ID>` - Find issues that block a specific issue
- `--has-blockers` - Find issues with any blockers
- `--has-dependencies` - Find issues with dependencies
- `--has-circular-deps` - Find circular dependency chains
- `--max-depth <n>` - Filter by dependency chain depth

**Entity Types:**
- `issues` (default) - Full issue filtering
- `cycles` - Search cycles
- `projects` - Search projects
- `users` - Search team members
- `all` - Cross-entity search

### Dependencies

Visualize and manage issue dependencies:

```bash
# ASCII dependency tree for single issue
linear deps ENG-100

# All dependencies for a team
linear deps --team ENG
```

Output:
```
DEPENDENCY GRAPH: ENG-100
══════════════════════════════════════════════════
ENG-100 User Authentication Epic
├─ ENG-101 [In Progress] Login flow
│  ├─ ENG-103 [Todo] OAuth integration
│  │     → blocks: ENG-105, ENG-106
│  └─ ENG-104 [Todo] Session management
│        → blocks: ENG-107
├─ ENG-102 [Blocked] Logout flow
│     ← blocked by: ENG-101
└─ ENG-105 [Blocked] Token refresh
      ← blocked by: ENG-103
──────────────────────────────────────────────────
6 issues, 5 dependencies, 0 cycles

⚠ Circular dependency detected:
  ENG-201 → ENG-202 → ENG-201
```

### Comments & Reactions

```bash
linear issues comment ENG-123 --body "Fixed!"
linear issues comment ENG-123 --body "Screenshot attached:" --attach /tmp/fix.png
linear issues comments ENG-123               # List comments
linear issues reply ENG-123 <id> --body "Thanks!"
linear issues react ENG-123 👍               # Add reaction
```

### Projects

```bash
linear projects list
linear projects list --mine                  # Your projects only
linear projects get PROJECT-ID
linear projects create "Q1 Release" --team ENG
linear projects update PROJECT-ID --state completed
```

### Cycles

```bash
linear cycles list --team ENG
linear cycles list --team ENG --active
linear cycles get CYCLE-ID
linear cycles analyze --team ENG --count 10  # Velocity analytics
```

### Teams

```bash
linear teams list
linear teams get ENG
linear teams labels ENG                      # Team labels
linear teams states ENG                      # Workflow states
```

### Labels

```bash
linear labels list --team ENG               # List all labels (with IDs)
linear labels list --team ENG --output json  # JSON output

# Create, update, delete (requires user auth, not agent/app)
linear labels create "needs-review" --team ENG --color "#ff0000" --description "PR needs review"
linear labels update LABEL-UUID --name "needs-code-review" --color "#ff6600"
linear labels delete LABEL-UUID
```

> **Note:** Label mutations (create/update/delete) require user authentication. OAuth app actors cannot manage labels due to Linear workspace permissions. Use `linear auth login` as a user.

### Users

```bash
linear users list
linear users list --team ENG
linear users get USER-ID
linear users me                              # Current user
```

### Claude Code Skills

Install productivity skills for agentic workflows:

```bash
linear skills list              # Show available skills
linear skills install --all     # Install all skills
linear skills install prd       # Install specific skill
```

Available skills:
- `/linear` - Complete CLI reference for LLMs (semantic search, dependencies, cycle analytics)
- `/prd` - Create agent-friendly tickets with success criteria
- `/triage` - Analyze and prioritize backlog
- `/cycle-plan` - Plan cycles using velocity data
- `/retro` - Generate sprint retrospective analysis
- `/deps` - Analyze dependency chains
- `/link-deps` - Discover and link related issues as dependencies

---

## Cycle Analytics

Analyze team performance:

```bash
linear cycles analyze --team ENG --count 10
```

Output includes:
- **Velocity**: Average points completed per cycle
- **Completion Rate**: Percentage of scoped work completed
- **Scope Creep**: Work added mid-cycle
- **Recommendations**: Data-driven scope suggestions

---

## Configuration

All configuration is stored in `~/.config/linear/`:

```
~/.config/linear/
├── config.yaml    # OAuth credentials
└── token          # Access token
```

---

## Troubleshooting

### "cannot be opened" on macOS
Right-click → Open → Click "Open" in security dialog.

### Re-authenticating Agent Mode (Already Installed App)

If you've previously installed the app in agent mode and need to re-authenticate, you'll see "This app is already installed" with no authorize button. To fix this:

1. **Go to Linear → Settings → Installed Applications**
2. **Find your OAuth app** (match the Client ID you're using)
3. **Click "Manage"** → then click **"Revoke Access"**
4. **Run `linear auth login`** again and select agent mode
5. Click "Install" when the authorization screen appears

This generates a new authorization token while keeping the app installed.

### Authentication Issues
```bash
ls -la ~/.config/linear/   # Check config exists
linear auth login          # Re-authenticate
```

---

## Development

```bash
make build    # Build bin/linear
make test     # Run tests
make clean    # Clean build artifacts
```

### Project Structure

```
cmd/
  linear/              # CLI entry point
internal/
  cli/                 # CLI commands (Cobra)
  config/              # Configuration
  format/              # ASCII formatting
  linear/              # Linear GraphQL client
  oauth/               # OAuth2 flow
  service/             # Business logic
  skills/              # Claude Code skills
  token/               # Token storage
```

## License

MIT License - see [LICENSE](LICENSE) file.
