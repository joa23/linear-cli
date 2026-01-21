# Changelog

All notable changes to this project will be documented in this file.

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
