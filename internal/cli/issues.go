package cli

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
	"github.com/joa23/linear-cli/internal/service"
	"github.com/joa23/linear-cli/internal/token"
	"github.com/spf13/cobra"
)

func newIssuesCmd() *cobra.Command {
	issuesCmd := &cobra.Command{
		Use:     "issues",
		Aliases: []string{"i", "issue"},
		Short:   "Manage Linear issues",
		Long:    "Create, list, and view Linear issues.",
	}

	issuesCmd.AddCommand(
		newIssuesListCmd(),
		newIssuesGetCmd(),
		newIssuesCreateCmd(),
		newIssuesUpdateCmd(),
		newIssuesCommentCmd(),
		newIssuesCommentsCmd(),
		newIssuesReplyCmd(),
		newIssuesReactCmd(),
		newIssuesDependenciesCmd(),
		newIssuesBlockedByCmd(),
		newIssuesBlockingCmd(),
	)

	return issuesCmd
}

func newIssuesListCmd() *cobra.Command {
	var (
		teamID    string
		state     string
		priority  int
		assignee  string
		cycle     string
		labels    string
		limit     int
		formatStr string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all issues (not just assigned)",
		Long: `List Linear issues with filtering, sorting, and pagination.

IMPORTANT BEHAVIORS:
- Returns ALL issues by default (not just assigned to you)
- Requires team context from 'linear init' or --team flag
- Use filters to narrow results
- Returns 10 issues by default, use --limit to change

TIP: Use --format full for detailed output, --format minimal for concise output.`,
		Example: `  # Minimal - list first 10 issues (requires 'linear init')
  linear issues list

  # Complete - using ALL available parameters
  linear issues list \
    --team CEN \
    --state "In Progress" \
    --priority 1 \
    --assignee johannes.zillmann@centrum-ai.com \
    --cycle 65 \
    --labels "customer,bug" \
    --limit 50 \
    --format full

  # Common pattern - high priority customer issues
  linear issues list \
    --labels customer \
    --priority 1 \
    --limit 20

  # Get issues in specific cycle
  linear issues list --cycle 65 --format full

  # Filter by assignee
  linear issues list --assignee me

  # Filter by state
  linear issues list --state Backlog --limit 100`,
		Annotations: map[string]string{
			"required": "team (via init or --team flag)",
			"optional": "all filter/pagination flags",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use default team if not specified
			if teamID == "" {
				teamID = GetDefaultTeam()
			}
			if teamID == "" {
				return errors.New(ErrTeamRequired)
			}

			// Validate limit
			limit, err := validateAndNormalizeLimit(limit)
			if err != nil {
				return err
			}

			svc, err := getIssueService()
			if err != nil {
				return err
			}

			// Build search filters
			filters := &service.SearchFilters{
				TeamID: teamID,
				Limit:  limit,
			}

			// Apply optional filters
			if state != "" {
				filters.StateIDs = []string{state}
			}
			if cmd.Flags().Changed("priority") {
				filters.Priority = &priority
			}
			if assignee != "" {
				filters.AssigneeID = assignee
			}
			if cycle != "" {
				filters.CycleID = cycle
			}
			if labels != "" {
				filters.LabelIDs = parseCommaSeparated(labels)
			}

			// Set format
			outputFormat := format.Compact
			if formatStr == "full" {
				outputFormat = format.Full
			} else if formatStr == "minimal" {
				outputFormat = format.Minimal
			}
			filters.Format = outputFormat

			output, err := svc.Search(filters)
			if err != nil {
				return fmt.Errorf("failed to list issues: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}

	cmd.Flags().StringVarP(&teamID, "team", "t", "", TeamFlagDescription)
	cmd.Flags().StringVar(&state, "state", "", "Filter by workflow state (e.g., 'In Progress', 'Backlog')")
	cmd.Flags().IntVar(&priority, "priority", 0, "Filter by priority (0=none, 1=urgent, 2=high, 3=normal, 4=low)")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Filter by assignee (email or 'me')")
	cmd.Flags().StringVarP(&cycle, "cycle", "c", "", "Filter by cycle (number, 'current', or 'next')")
	cmd.Flags().StringVarP(&labels, "labels", "l", "", "Filter by labels (comma-separated)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Number of items (max 250)")
	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Output format: minimal|compact|full")

	return cmd
}

func newIssuesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <issue-id>",
		Short: "Get issue details",
		Long:  "Display detailed information about a specific issue (e.g., 'linear issues get CEN-123').",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]

			svc, err := getIssueService()
			if err != nil {
				return err
			}

			output, err := svc.Get(issueID, format.Full)
			if err != nil {
				return fmt.Errorf("failed to get issue: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}
}

func newIssuesCreateCmd() *cobra.Command {
	var (
		team        string
		description string
		state       string
		priority    int
		estimate    float64
		labels      string
		cycle       string
		project     string
		assignee    string
		dueDate     string
		parent      string
		dependsOn   string
		blockedBy   string
		attachFiles []string
	)

	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new issue",
		Long: `Create a new issue in Linear.

REQUIRED:
- Title (positional argument)
- Team context (from 'linear init' or --team flag)

OPTIONAL: All other flags (assignee, priority, labels, etc.)

TIP: Run 'linear init' first to set default team.`,
		Example: `  # Minimal - create with just title (requires 'linear init')
  linear issues create "Fix login bug"

  # Complete - using ALL available parameters
  linear issues create "Implement OAuth" \
    --team CEN \
    --project "Auth Revamp" \
    --parent CEN-100 \
    --state "In Progress" \
    --priority 1 \
    --assignee stefan@centrum-ai.com \
    --estimate 5 \
    --cycle 65 \
    --labels "backend,security" \
    --blocked-by CEN-99 \
    --depends-on CEN-98,CEN-97 \
    --due 2026-02-15 \
    --attach /tmp/diagram.png \
    --description "Full OAuth implementation with Google provider"

  # Common pattern - bug fix with assignee
  linear issues create "Fix null pointer" \
    --team CEN \
    --priority 0 \
    --assignee me \
    --labels bug

  # With screenshot attachment
  linear issues create "UI Bug" --team CEN --attach /tmp/screenshot.png

  # With multiple attachments
  linear issues create "Bug report" --team CEN --attach img1.png --attach img2.png

  # Pipe description from file (use - for stdin)
  cat .claude/plans/feature-plan.md | linear issues create "Implementation" --team CEN -d -

  # Pipe PRD into ticket
  cat prd.md | linear issues create "Feature: OAuth" --team CEN --description -`,
		Annotations: map[string]string{
			"required": "title, team (via init or --team flag)",
			"optional": "all other flags",
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]

			// Get team from flag or config
			if team == "" {
				team = GetDefaultTeam()
			}
			if team == "" {
				return errors.New(ErrTeamRequired)
			}

			// Get description from flag or stdin
			desc, err := getDescriptionFromFlagOrStdin(description)
			if err != nil {
				return fmt.Errorf("failed to read description: %w", err)
			}

			// Upload attachments and append to description
			if len(attachFiles) > 0 {
				client, err := getLinearClient()
				if err != nil {
					return err
				}
				desc, err = uploadAndAppendAttachments(client, desc, attachFiles)
				if err != nil {
					return err
				}
			}

			// Build create input
			input := &service.CreateIssueInput{
				Title:       title,
				TeamID:      team,
				Description: desc,
			}

			// Set optional fields
			if state != "" {
				input.StateID = state
			}
			if cmd.Flags().Changed("priority") {
				input.Priority = &priority
			}
			if estimate > 0 {
				input.Estimate = &estimate
			}
			if labels != "" {
				input.LabelIDs = parseCommaSeparated(labels)
			}
			if cycle != "" {
				input.CycleID = cycle
			}
			if project != "" {
				input.ProjectID = project
			}
			if assignee != "" {
				input.AssigneeID = assignee
			}
			if dueDate != "" {
				input.DueDate = dueDate
			}
			if parent != "" {
				input.ParentID = parent
			}
			if dependsOn != "" {
				input.DependsOn = parseCommaSeparated(dependsOn)
			}
			if blockedBy != "" {
				input.BlockedBy = parseCommaSeparated(blockedBy)
			}

			svc, err := getIssueService()
			if err != nil {
				return err
			}

			output, err := svc.Create(input)
			if err != nil {
				return fmt.Errorf("failed to create issue: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}

	// Add flags (with short versions for common flags)
	cmd.Flags().StringVarP(&team, "team", "t", "", TeamFlagDescription)
	cmd.Flags().StringVarP(&description, "description", "d", "", "Issue description (or pipe to stdin)")
	cmd.Flags().StringVarP(&state, "state", "s", "", "Workflow state name (e.g., 'In Progress', 'Backlog')")
	cmd.Flags().IntVarP(&priority, "priority", "p", 0, "Priority 0-4 (0=none, 1=urgent, 4=low)")
	cmd.Flags().Float64VarP(&estimate, "estimate", "e", 0, "Story points estimate")
	cmd.Flags().StringVarP(&labels, "labels", "l", "", "Comma-separated label names/IDs")
	cmd.Flags().StringVarP(&cycle, "cycle", "c", "", "Cycle number or name (e.g., 'current', 'next')")
	cmd.Flags().StringVarP(&project, "project", "P", "", "Project name or ID")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Assignee name or email (use 'me' for yourself)")
	cmd.Flags().StringVar(&dueDate, "due", "", "Due date YYYY-MM-DD")
	cmd.Flags().StringVar(&parent, "parent", "", "Parent issue ID (for sub-issues)")
	cmd.Flags().StringVar(&dependsOn, "depends-on", "", "Comma-separated issue IDs this depends on")
	cmd.Flags().StringVar(&blockedBy, "blocked-by", "", "Comma-separated issue IDs blocking this")
	cmd.Flags().StringArrayVar(&attachFiles, "attach", nil, "File(s) to attach (can be used multiple times)")

	return cmd
}

func newIssuesUpdateCmd() *cobra.Command {
	var (
		team        string
		title       string
		description string
		state       string
		priority    string
		estimate    string
		labels      string
		cycle       string
		project     string
		assignee    string
		dueDate     string
		parent      string
		dependsOn   string
		blockedBy   string
		attachFiles []string
	)

	cmd := &cobra.Command{
		Use:   "update <issue-id>",
		Short: "Update an existing issue",
		Long:  `Update an existing issue. Only provided flags are changed.`,
		Example: `  # Update state and priority
  linear issues update CEN-123 --state Done --priority 0

  # Add attachment to existing issue
  linear issues update CEN-123 --attach /tmp/screenshot.png

  # Assign to yourself
  linear issues update CEN-123 --assignee me

  # Update description from file (use - for stdin)
  cat updated-spec.md | linear issues update CEN-123 -d -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]

			// Get team from flag or config (for cycle resolution)
			if team == "" {
				team = GetDefaultTeam()
			}
			// Note: team can still be "" if no .linear.yaml, will fallback to issue identifier

			// Check if any updates provided (stdin counts as description update)
			hasStdin := hasStdinPipe()
			hasFlags := title != "" || description != "" || state != "" ||
				priority != "" || estimate != "" || labels != "" ||
				cycle != "" || project != "" || assignee != "" ||
				dueDate != "" || parent != "" || dependsOn != "" || blockedBy != "" ||
				len(attachFiles) > 0

			if !hasFlags && !hasStdin {
				return fmt.Errorf("no updates specified. Use flags like --state, --priority, etc")
			}

			// Get description from flag or stdin
			desc, err := getDescriptionFromFlagOrStdin(description)
			if err != nil {
				return fmt.Errorf("failed to read description: %w", err)
			}

			// Upload attachments and append to description
			if len(attachFiles) > 0 {
				client, err := getLinearClient()
				if err != nil {
					return err
				}
				desc, err = uploadAndAppendAttachments(client, desc, attachFiles)
				if err != nil {
					return err
				}
			}

			// Build update input
			input := &service.UpdateIssueInput{}

			if title != "" {
				input.Title = &title
			}
			if desc != "" {
				input.Description = &desc
			}
			if state != "" {
				input.StateID = &state
			}
			if priority != "" {
				p, err := strconv.Atoi(priority)
				if err != nil {
					return fmt.Errorf("invalid priority: %w", err)
				}
				input.Priority = &p
			}
			if estimate != "" {
				e, err := strconv.ParseFloat(estimate, 64)
				if err != nil {
					return fmt.Errorf("invalid estimate: %w", err)
				}
				input.Estimate = &e
			}
			if labels != "" {
				input.LabelIDs = parseCommaSeparated(labels)
			}
			if cycle != "" {
				input.CycleID = &cycle
			}
			if project != "" {
				input.ProjectID = &project
			}
			if assignee != "" {
				input.AssigneeID = &assignee
			}
			if dueDate != "" {
				input.DueDate = &dueDate
			}
			if parent != "" {
				input.ParentID = &parent
			}
			if dependsOn != "" {
				input.DependsOn = parseCommaSeparated(dependsOn)
			}
			if blockedBy != "" {
				input.BlockedBy = parseCommaSeparated(blockedBy)
			}
			if team != "" {
				input.TeamID = &team
			}

			svc, err := getIssueService()
			if err != nil {
				return err
			}

			output, err := svc.Update(issueID, input)
			if err != nil {
				return fmt.Errorf("failed to update issue: %w", err)
			}

			fmt.Println(output)
			return nil
		},
	}

	// Add flags (with short versions for common flags)
	cmd.Flags().StringVarP(&title, "title", "T", "", "Update issue title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Update description (or pipe to stdin)")
	cmd.Flags().StringVarP(&state, "state", "s", "", "Update workflow state name (e.g., 'In Progress', 'Backlog')")
	cmd.Flags().StringVarP(&priority, "priority", "p", "", "Update priority 0-4 (0=none, 1=urgent, 4=low)")
	cmd.Flags().StringVarP(&estimate, "estimate", "e", "", "Update story points estimate")
	cmd.Flags().StringVarP(&labels, "labels", "l", "", "Update labels (comma-separated)")
	cmd.Flags().StringVarP(&cycle, "cycle", "c", "", "Update cycle number or name")
	cmd.Flags().StringVarP(&project, "project", "P", "", "Update project name or ID")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Update assignee name or email (use 'me' for yourself)")
	cmd.Flags().StringVar(&dueDate, "due", "", "Update due date YYYY-MM-DD")
	cmd.Flags().StringVar(&parent, "parent", "", "Update parent issue")
	cmd.Flags().StringVar(&dependsOn, "depends-on", "", "Update dependencies (comma-separated issue IDs)")
	cmd.Flags().StringVar(&blockedBy, "blocked-by", "", "Update blocked-by (comma-separated issue IDs)")
	cmd.Flags().StringArrayVar(&attachFiles, "attach", nil, "File(s) to attach (can be used multiple times)")
	cmd.Flags().StringVarP(&team, "team", "t", "", TeamFlagDescription)

	return cmd
}

func newIssuesCommentCmd() *cobra.Command {
	var (
		body        string
		attachFiles []string
	)

	cmd := &cobra.Command{
		Use:   "comment <issue-id>",
		Short: "Add a comment to an issue",
		Long:  `Add a comment to an issue. Comment body can be provided via --body flag or piped from stdin.`,
		Example: `  # Add a simple comment
  linear issues comment CEN-123 --body "This is a comment"

  # Comment with screenshot attachment
  linear issues comment CEN-123 --body "Bug screenshot:" --attach /tmp/screenshot.png

  # Pipe content from file
  cat notes.md | linear issues comment CEN-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]

			// Get Linear client
			client, err := getLinearClient()
			if err != nil {
				return err
			}

			// Get body from flag or stdin
			commentBody, err := getDescriptionFromFlagOrStdin(body)
			if err != nil {
				return fmt.Errorf("failed to read comment body: %w", err)
			}

			// Upload attachments and append to body
			if len(attachFiles) > 0 {
				commentBody, err = uploadAndAppendAttachments(client, commentBody, attachFiles)
				if err != nil {
					return err
				}
			}

			if commentBody == "" {
				return fmt.Errorf("comment body is required. Use --body flag or pipe content to stdin")
			}

			// Get the issue first to get its ID (comments need issue ID, not identifier)

			issue, err := client.GetIssue(issueID)
			if err != nil {
				return fmt.Errorf("failed to get issue: %w", err)
			}

			// Create the comment
			comment, err := client.Comments.CreateComment(issue.ID, commentBody)
			if err != nil {
				return fmt.Errorf("failed to create comment: %w", err)
			}

			fmt.Printf("Comment added to %s\n", issue.Identifier)
			fmt.Printf("  ID: %s\n", comment.ID)
			fmt.Printf("  By: %s\n", comment.User.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body (or pipe to stdin)")
	cmd.Flags().StringArrayVar(&attachFiles, "attach", nil, "File(s) to attach (can be used multiple times)")

	return cmd
}

func newIssuesCommentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "comments <issue-id>",
		Short: "List comments on an issue",
		Long:  "Display all comments on a specific issue.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]

			client, err := getLinearClient()
			if err != nil {
				return err
			}

			// Resolve issue identifier to get the UUID
			issue, err := client.GetIssue(issueID)
			if err != nil {
				return fmt.Errorf("failed to get issue: %w", err)
			}

			comments, err := client.Comments.GetIssueComments(issue.ID)
			if err != nil {
				return fmt.Errorf("failed to get comments: %w", err)
			}

			if len(comments) == 0 {
				fmt.Printf("No comments on %s\n", issue.Identifier)
				return nil
			}

			fmt.Printf("COMMENTS ON %s (%d)\n", issue.Identifier, len(comments))
			fmt.Println("────────────────────────────────────────")
			for _, c := range comments {
				prefix := ""
				if c.Parent != nil {
					prefix = "  ↳ "
				}
				fmt.Printf("%s%s (%s):\n", prefix, c.User.Name, c.CreatedAt[:10])
				// Truncate long comments
				body := c.Body
				if len(body) > 200 {
					body = body[:200] + "..."
				}
				fmt.Printf("%s  %s\n\n", prefix, body)
			}
			return nil
		},
	}
}

func newIssuesReplyCmd() *cobra.Command {
	var (
		body        string
		attachFiles []string
	)

	cmd := &cobra.Command{
		Use:   "reply <issue-id> <comment-id>",
		Short: "Reply to a comment",
		Long:  `Reply to an existing comment on an issue. Reply body can be provided via --body flag or piped from stdin.`,
		Example: `  # Reply to a comment
  linear issues reply CEN-123 abc-comment-id --body "Thanks for the feedback!"

  # Reply with attachment
  linear issues reply CEN-123 abc-comment-id --body "Here's the fix:" --attach /tmp/screenshot.png

  # Pipe content from file
  cat response.md | linear issues reply CEN-123 abc-comment-id`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			commentID := args[1]

			// Get body from flag or stdin
			replyBody, err := getDescriptionFromFlagOrStdin(body)
			if err != nil {
				return fmt.Errorf("failed to read reply body: %w", err)
			}

			// Upload attachments and append to body
			if len(attachFiles) > 0 {
				client, err := getLinearClient()
				if err != nil {
					return err
				}
				replyBody, err = uploadAndAppendAttachments(client, replyBody, attachFiles)
				if err != nil {
					return err
				}
			}

			if replyBody == "" {
				return fmt.Errorf("reply body is required. Use --body flag or pipe content to stdin")
			}

			svc, err := getIssueService()
			if err != nil {
				return err
			}

			comment, err := svc.ReplyToComment(issueID, commentID, replyBody)
			if err != nil {
				return fmt.Errorf("failed to create reply: %w", err)
			}

			fmt.Printf("Reply added to comment on %s\n", issueID)
			fmt.Printf("  ID: %s\n", comment.ID)
			fmt.Printf("  By: %s\n", comment.User.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Reply body (or pipe to stdin)")
	cmd.Flags().StringArrayVar(&attachFiles, "attach", nil, "File(s) to attach (can be used multiple times)")

	return cmd
}

func newIssuesReactCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "react <issue-or-comment-id> <emoji>",
		Short: "Add a reaction to an issue or comment",
		Long:  `Add an emoji reaction to an issue or comment.`,
		Example: `  # React to an issue
  linear issues react CEN-123 👍

  # React to a comment
  linear issues react abc-comment-id 🎉

  # Common reactions: 👍 👎 ❤️ 🎉 😄 🚀`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetID := args[0]
			emoji := args[1]

			svc, err := getIssueService()
			if err != nil {
				return err
			}

			// If targetID looks like an issue identifier (e.g., CEN-123), resolve it first
			if len(targetID) > 0 && targetID[0] >= 'A' && targetID[0] <= 'Z' {
				resolvedID, err := svc.GetIssueID(targetID)
				if err != nil {
					return fmt.Errorf("failed to resolve issue: %w", err)
				}
				targetID = resolvedID
			}

			err = svc.AddReaction(targetID, emoji)
			if err != nil {
				return fmt.Errorf("failed to add reaction: %w", err)
			}

			fmt.Printf("Added %s reaction\n", emoji)
			return nil
		},
	}
}

func newIssuesDependenciesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dependencies <issue-id>",
		Short: "List issue dependencies (what it depends on)",
		Long:  "Show compressed list of issues this ticket depends on. Uses metadata or URL references.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			client, err := getLinearClient()
			if err != nil {
				return err
			}

			issue, err := client.GetIssue(issueID)
			if err != nil {
				return fmt.Errorf("failed to get issue: %w", err)
			}

			// Check metadata for dependency info
			deps := []string{}
			if metadata, ok := issue.Metadata["dependencies"].([]interface{}); ok {
				for _, dep := range metadata {
					if depStr, ok := dep.(string); ok {
						deps = append(deps, depStr)
					}
				}
			}

			if len(deps) == 0 {
				fmt.Println("none")
				return nil
			}

			fmt.Println(fmt.Sprintf("%v", deps))
			return nil
		},
	}
}

func newIssuesBlockedByCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "blocked-by <issue-id>",
		Short: "List issues blocking this one",
		Long:  "Show compressed list of issues that are blocking this ticket.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			client, err := getLinearClient()
			if err != nil {
				return err
			}

			issue, err := client.GetIssue(issueID)
			if err != nil {
				return fmt.Errorf("failed to get issue: %w", err)
			}

			// Check if blocked in metadata or description
			blocked := []string{}
			if blockList, ok := issue.Metadata["blocked_by"].([]interface{}); ok {
				for _, blocker := range blockList {
					if blockerStr, ok := blocker.(string); ok {
						blocked = append(blocked, blockerStr)
					}
				}
			}

			// Check description for "Blocked by:" mentions
			if len(blocked) == 0 && issue.Description != "" {
				// Simple extraction - in practice would be more sophisticated
				fmt.Println("check description or Linear UI for blocking issues")
				return nil
			}

			if len(blocked) == 0 {
				fmt.Println("none")
				return nil
			}

			fmt.Println(fmt.Sprintf("%v", blocked))
			return nil
		},
	}
}

func newIssuesBlockingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "blocking <issue-id>",
		Short: "List issues blocked by this one",
		Long:  "Show compressed list of issues that are blocked by this ticket.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			client, err := getLinearClient()
			if err != nil {
				return err
			}

			issue, err := client.GetIssue(issueID)
			if err != nil {
				return fmt.Errorf("failed to get issue: %w", err)
			}

			// Check metadata for blocked issues
			blocking := []string{}
			if blockList, ok := issue.Metadata["blocking"].([]interface{}); ok {
				for _, blocked := range blockList {
					if blockedStr, ok := blocked.(string); ok {
						blocking = append(blocking, blockedStr)
					}
				}
			}

			if len(blocking) == 0 {
				fmt.Println("none")
				return nil
			}

			fmt.Println(fmt.Sprintf("%v", blocking))
			return nil
		},
	}
}

// getLinearClient retrieves an authenticated Linear client
func getLinearClient() (*linear.Client, error) {
	tokenStorage := token.NewStorage(token.GetDefaultTokenPath())
	exists, _ := tokenStorage.TokenExistsWithError()
	if !exists {
		return nil, fmt.Errorf("not authenticated. Run 'linear auth login' to authenticate")
	}

	tokenData, err := tokenStorage.LoadTokenData()
	if err != nil {
		return nil, fmt.Errorf("failed to load token: %w", err)
	}

	// Use the access token from the structured data
	return linear.NewClient(tokenData.AccessToken), nil
}

// getIssueService retrieves the issue service with authenticated client
func getIssueService() (*service.IssueService, error) {
	client, err := getLinearClient()
	if err != nil {
		return nil, err
	}
	return service.New(client).Issues, nil
}
