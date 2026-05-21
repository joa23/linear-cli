# Analysis Framework

## Staleness thresholds

| Last updated | Status | Action |
|---|---|---|
| > 30 days | Stale | Ping assignee, request update |
| > 60 days | Very stale | Close with comment or re-prioritize |
| No activity + no assignee | Orphaned | Assign owner or close |

## Dependency health

| Pattern | Risk | Action |
|---|---|---|
| Blocked by completed issue | Low | Unblock immediately |
| Circular dependencies | High | Flag for manual resolution |
| Long blocking chains (3+) | High | Escalate to team lead |
| Blocked + no blocker assignee | Medium | Assign blocker first |

## Priority assessment

| Situation | Action |
|---|---|
| P1/P2 but blocked | Escalate the blocker, not the issue |
| P3/P4 with no activity 30+ days | Close or downgrade |
| No priority set | Assign priority during triage |
| Customer-labeled + low priority | Review — may need escalation |

## CLI reference

Priority values: `0` = none, `1` = urgent, `2` = high, `3` = normal, `4` = low

`--format full` returns structured output suitable for analysis. `linear issues list` returns all team issues, not just assigned to you.
