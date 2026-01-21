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

A **token-efficient** command-line interface for [Linear](https://linear.app).

> **Why this tool?** Human-readable identifiers ("TEST-123" not UUIDs), smart caching, and ASCII output—90% fewer tokens than alternatives.

## Installation

### Homebrew (Recommended)

```bash
brew tap joa23/linear-cli
brew install linear-cli
```

### Manual Download

**macOS:**
```bash
# Apple Silicon
curl -L https://github.com/joa23/linear-cli/releases/latest/download/linear-darwin-arm64 -o linear
chmod +x linear

# Intel
curl -L https://github.com/joa23/linear-cli/releases/latest/download/linear-darwin-amd64 -o linear
chmod +x linear
```

**Linux:**
```bash
curl -L https://github.com/joa23/linear-cli/releases/latest/download/linear-linux-amd64 -o linear
chmod +x linear
```

### Build from Source

```bash
git clone https://github.com/joa23/linear-cli.git
cd linear-cli
make build
```

## Authentication

```bash
linear auth login
```

You'll be prompted to choose an authentication mode:

**Personal Use** - Authenticate as yourself
- Your actions appear under your Linear account
- For personal task management

**Agent/App** - Authenticate as an agent
- Agent appears as a separate entity in Linear
- Requires admin approval to install
- Agent can be @mentioned and assigned issues
- For automation, bots, and MCP servers

### OAuth Setup

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

### Other Auth Commands

```bash
linear auth status   # Check login status
linear auth logout   # Remove credentials
```

## CLI Commands

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
  --labels "backend,security"

# Create sub-issue with dependencies
linear issues create "Add Google OAuth provider" \
  --team ENG \
  --parent ENG-100 \
  --blocked-by ENG-99 \
  --depends-on ENG-98,ENG-97

# Update multiple fields at once
linear issues update ENG-123 \
  --state Done \
  --assignee alice \
  --priority 1 \
  --labels "urgent,hotfix"

# Manage dependencies
linear issues update ENG-102 --blocked-by ENG-101
linear issues update ENG-103 --depends-on ENG-100,ENG-101
```

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
- `/prd` - Create agent-friendly tickets with success criteria
- `/triage` - Analyze and prioritize backlog
- `/cycle-plan` - Plan cycles using velocity data
- `/retro` - Generate sprint retrospective analysis
- `/deps` - Analyze dependency chains

## Output Format

Token-efficient ASCII:

```
ISSUE TEST-123
════════════════════════════════════════
Fix authentication bug
────────────────────────────────────────
State:     In Progress
Assignee:  Sarah Chen
Priority:  P1 (Urgent)
────────────────────────────────────────
```

Paginated results show `Next: cursor=xxx` when more pages are available.

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

## Configuration

All configuration is stored in `~/.config/linear/`:

```
~/.config/linear/
├── config.yaml    # OAuth credentials
└── token          # Access token
```

## Troubleshooting

### "cannot be opened" on macOS
Right-click → Open → Click "Open" in security dialog.

### Authentication Issues
```bash
ls -la ~/.config/linear/   # Check config exists
linear auth login          # Re-authenticate
```

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
