---
name: triage
description: >
  Triage and prioritize Linear backlog issues using the linear CLI.
  Analyzes staleness, blockers, dependency health, and priority mismatches.
  Use when the user asks about backlog grooming, sprint planning, issue
  prioritization, managing stale tickets, or cleaning up Linear issues.
---

# Triage Skill - Backlog Analysis

## Process

### 1. Setup

```bash
linear init  # if .linear.yaml doesn't exist
```

### 2. Fetch the backlog

```bash
linear issues list --state Backlog --format full --limit 100
```

**Verify:** confirm issues are returned before proceeding. If empty, check team context with `linear init`.

### 3. Analyze dependencies

```bash
linear deps --team ENG
```

### 4. Filter by priority and labels

```bash
# Urgent (P1)
linear issues list --state Backlog --priority 1 --format full

# High priority (P2)
linear issues list --state Backlog --priority 2 --format full

# Customer issues
linear issues list --labels customer --format full

# Bugs in backlog
linear issues list --state Backlog --labels bug --format full

# Combined filters
linear issues list --state Backlog --priority 1 --labels customer --format full
```

### 5. Identify issues

Flag issues matching these patterns:

| Pattern | Criteria | Action |
|---|---|---|
| Stale | No updates 30+ days | Ping assignee or close |
| Very stale | No updates 60+ days | Close with comment |
| Blocked | Dependencies unresolved | Escalate blocker |
| Orphaned | No assignee + no activity | Assign or close |
| Priority mismatch | P1/P2 but blocked | Escalate the blocker first |

**Verify:** cross-check flagged issues against dependency graph from step 3. Blocked issues with completed blockers should be unblocked immediately:

```bash
linear issues update ENG-123 --priority 2
linear issues comment ENG-123 --body "Triaged: Needs unblocking before sprint"
```

### 6. Generate report

Use this output format:

```
BACKLOG TRIAGE: Team ENG
════════════════════════════════════════

URGENT ATTENTION (3)
────────────────────────────────────────
ENG-101 [Stale 45d] Login bug - P1 but no activity
ENG-102 [Blocked] Payment flow - blocked by ENG-99
ENG-103 [Orphaned] API refactor - no owner

RECOMMENDED ACTIONS
────────────────────────────────────────
1. Unblock ENG-102: Complete ENG-99 or remove dependency
2. Assign ENG-103: Needs owner or close if abandoned
3. Update ENG-101: Stale P1 needs attention

HEALTH SUMMARY
────────────────────────────────────────
Total issues: 45
Blocked: 8 (17%)
Stale: 12 (26%)
Healthy: 25 (55%)
```

**Verify:** confirm every recommended action references a real issue ID from the fetched data.

## References

- [Analysis framework](references/analysis-framework.md) — staleness thresholds, dependency health rules, priority assessment criteria
