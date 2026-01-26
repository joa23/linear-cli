#!/bin/bash
# Vrataski Loop - continuous autonomous execution with context preservation
# Named after Rita Vrataski from Edge of Tomorrow - loops but retains memory

set -e

# Session folder - must be set to Claude's session ID
SESSION_DIR="${CLAUDE_TASKS_DIR:-$HOME/.claude/tasks/$CLAUDE_SESSION_ID}"

if [ -z "$CLAUDE_SESSION_ID" ] && [ -z "$CLAUDE_TASKS_DIR" ]; then
  echo "Error: Set CLAUDE_SESSION_ID or CLAUDE_TASKS_DIR"
  exit 1
fi

echo "Vrataski Loop starting..."
echo "Session: $SESSION_DIR"

while true; do
  # Get next To Do issue assigned to me
  ISSUE=$(linear issues list --assignee me --state "To Do" --limit 1 --output json | jq -r '.[0].identifier // empty')

  if [ -z "$ISSUE" ]; then
    echo "No issues in To Do, sleeping 60s..."
    sleep 60
    continue
  fi

  echo "Processing: $ISSUE"

  # Move to In Progress and export to Claude
  linear issues update "$ISSUE" --state "In Progress"
  linear tasks export "$ISSUE" "$SESSION_DIR"

  echo "Exported $ISSUE, waiting for completion..."

  # Wait for all tasks to complete (check task folder for pending tasks)
  while true; do
    PENDING=$(grep -l '"status": "pending"' "$SESSION_DIR"/*.json 2>/dev/null | wc -l | tr -d ' ')
    if [ "$PENDING" -eq 0 ]; then
      echo "All tasks completed"
      break
    fi
    echo "  $PENDING tasks pending..."
    sleep 10
  done

  # Inject "create PR" task, wait for completion
  echo "Injecting create-pr task..."
  cat > "$SESSION_DIR/create-pr.json" << EOF
{
  "id": "create-pr",
  "subject": "Create PR for $ISSUE",
  "description": "Create a pull request for the completed work on $ISSUE",
  "activeForm": "Creating PR",
  "status": "pending",
  "blocks": [],
  "blockedBy": []
}
EOF

  # Wait for PR task to complete
  while grep -q '"status": "pending"' "$SESSION_DIR/create-pr.json" 2>/dev/null; do
    echo "  Waiting for PR task..."
    sleep 10
  done

  # Update Linear and continue
  echo "Marking $ISSUE as Done"
  linear issues update "$ISSUE" --state "Done"

  echo "Completed $ISSUE, moving to next..."
done
