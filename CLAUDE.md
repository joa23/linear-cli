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
