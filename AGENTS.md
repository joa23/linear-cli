## Linear

This project uses **Linear** for issue tracking.
Default team: **TEST**

### Creating Issues

```bash
# Create a simple issue
linear issues create "Fix login bug" --team TEST --priority high

# Create with full details and dependencies
linear issues create "Add OAuth integration" \
  --team TEST \
  --description "Integrate Google and GitHub OAuth providers" \
  --parent TEST-100 \
  --depends-on TEST-99 \
  --labels "backend,security" \
  --estimate 5

# List and view issues
linear issues list
linear issues get TEST-123
```

### Fetching private Linear images

`uploads.linear.app` URLs in issue descriptions require authentication.
Do **NOT** use `WebFetch` or `curl` — they will 401.

Instead, use the CLI:

```bash
linear attachments download "https://uploads.linear.app/..."
# → /tmp/linear-img-<hash>.png
```

Then `Read` that path to view the image.

### Claude Code Skills

Available workflow skills (install with `linear skills install --all`):
- `/prd` - Create agent-friendly tickets with PRDs and sub-issues
- `/triage` - Analyze and prioritize backlog
- `/cycle-plan` - Plan cycles using velocity analytics
- `/retro` - Generate sprint retrospectives
- `/deps` - Analyze dependency chains

Run `linear skills list` for details.
