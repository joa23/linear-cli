# Contributing to Linear CLI

Thank you for your interest in contributing!

## Development Setup

```bash
# Clone the repository
git clone https://github.com/joa23/linear-cli.git
cd linear-cli

# Build
make build

# Run tests
make test
```

## Project Structure

```
cmd/
  linear/              # CLI entry point
internal/
  cli/                 # CLI commands (Cobra)
  config/              # Configuration management
  format/              # ASCII formatting
  linear/              # Linear GraphQL client
  oauth/               # OAuth2 flow
  service/             # Business logic
  skills/              # Claude Code skills
  token/               # Token storage
```

## Making Changes

1. Create your branch from `main`
2. Write tests for new functionality
3. Run `make test` to ensure all tests pass
4. Run `go vet ./...` to check for issues
5. Update documentation if changing behavior

## Code Style

- Follow standard Go conventions
- Keep functions focused and small
- Add comments for exported functions
- Use meaningful variable names

## Commit Messages

Use conventional commit format:

```
feat: add new feature
fix: resolve bug in X
docs: update README
refactor: improve code structure
test: add tests for X
```

## Pull Requests

1. Update CHANGELOG.md with your changes
2. Ensure all tests pass
3. Provide a clear description of the changes
4. Link any related issues

## Testing

- Unit tests are preferred over integration tests
- Mock external dependencies
- Run `make test` before submitting
