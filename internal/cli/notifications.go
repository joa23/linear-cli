package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/spf13/cobra"
)

func newNotificationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "notifications",
		Aliases: []string{"notif", "n"},
		Short:   "View mentions and notifications",
		Long: `View and manage your Linear notifications.

Notifications include @mentions, issue assignments, comments on your issues,
and other activity relevant to you.`,
	}

	cmd.AddCommand(
		newNotificationsListCmd(),
		newNotificationsReadCmd(),
	)

	return cmd
}

func newNotificationsListCmd() *cobra.Command {
	var unreadOnly bool
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent notifications",
		Long: `List your recent notifications including @mentions, assignments, and comments.

By default shows both read and unread notifications. Use --unread to filter
to only unread notifications.`,
		Example: `  # List recent notifications
  linear notifications list

  # List only unread notifications
  linear notifications list --unread

  # List more notifications
  linear notifications list --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			// includeRead is the inverse of unreadOnly
			includeRead := !unreadOnly

			notifications, err := deps.Client.Notifications.GetNotifications(includeRead, limit)
			if err != nil {
				return fmt.Errorf("failed to get notifications: %w", err)
			}

			if len(notifications) == 0 {
				if unreadOnly {
					fmt.Println("No unread notifications.")
				} else {
					fmt.Println("No notifications found.")
				}
				return nil
			}

			// Filter to unread only if requested
			if unreadOnly {
				var unread []core.Notification
				for _, n := range notifications {
					if n.ReadAt == nil {
						unread = append(unread, n)
					}
				}
				notifications = unread
				if len(notifications) == 0 {
					fmt.Println("No unread notifications.")
					return nil
				}
			}

			// Format and print notifications
			output := formatNotificationList(notifications)
			fmt.Println(output)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&unreadOnly, "unread", "u", false, "Show only unread notifications")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of notifications to fetch")

	return cmd
}

func newNotificationsReadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read <notification-id>",
		Short: "Mark a notification as read",
		Long:  "Mark a specific notification as read by its ID.",
		Example: `  # Mark notification as read
  linear notifications read abc123-def456`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			notificationID := args[0]

			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			err = deps.Client.Notifications.MarkNotificationRead(notificationID)
			if err != nil {
				return fmt.Errorf("failed to mark notification as read: %w", err)
			}

			fmt.Println("Notification marked as read.")
			return nil
		},
	}

	return cmd
}

// formatNotificationList formats a list of notifications for display
func formatNotificationList(notifications []core.Notification) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Notifications (%d)\n", len(notifications)))
	b.WriteString(strings.Repeat("─", 60) + "\n")

	for _, n := range notifications {
		// Format the notification type nicely
		typeLabel := formatNotificationType(n.Type)

		// Time ago
		timeAgo := formatTimeAgo(n.CreatedAt)

		// Read status indicator
		readStatus := "●" // unread (filled circle)
		if n.ReadAt != nil {
			readStatus = "○" // read (empty circle)
		}

		// Build the notification line
		b.WriteString(fmt.Sprintf("%s %s %s\n", readStatus, typeLabel, timeAgo))

		// Add context based on notification type
		if n.Issue != nil {
			b.WriteString(fmt.Sprintf("  %s: %s\n", n.Issue.Identifier, n.Issue.Title))
		}
		if n.Comment != nil && n.Comment.Body != "" {
			// Truncate comment to first line or 80 chars
			comment := truncateComment(n.Comment.Body, 80)
			b.WriteString(fmt.Sprintf("  \"%s\"\n", comment))
		}

		// Show notification ID for marking as read
		b.WriteString(fmt.Sprintf("  ID: %s\n", n.ID))
		b.WriteString("\n")
	}

	return b.String()
}

// formatNotificationType converts API type to human-readable label
func formatNotificationType(t string) string {
	typeMap := map[string]string{
		// Mentions
		"issueMention":            "@Mention",
		"issueCommentMention":     "@Mention in comment",
		"IssueNotification":       "Issue update",

		// Assignments
		"issueAssignment":         "Assigned",
		"issueAssignedToYou":      "Assigned to you",
		"issueUnassignedFromYou":  "Unassigned from you",

		// Subscriptions
		"issueSubscribed":         "Subscribed",

		// Comments
		"issueNewComment":         "New comment",
		"issueCommentReaction":    "Reaction to comment",

		// Status changes
		"issueStatusChanged":      "Status changed",
		"issuePriorityChanged":    "Priority changed",
		"issueCreated":            "Issue created",
		"issueCompleted":          "Completed",
		"issueDue":                "Due soon",

		// Projects
		"projectUpdate":           "Project update",
		"projectMilestoneUpdate":  "Milestone update",
		"ProjectNotification":     "Project update",
	}

	if label, ok := typeMap[t]; ok {
		return label
	}
	// Fallback: return as-is
	return t
}

// formatTimeAgo formats a timestamp as relative time
func formatTimeAgo(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}

	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2")
	}
}

// truncateComment truncates a comment to maxLen characters, removing newlines
func truncateComment(comment string, maxLen int) string {
	// Replace newlines with spaces
	comment = strings.ReplaceAll(comment, "\n", " ")
	comment = strings.ReplaceAll(comment, "\r", "")

	// Trim whitespace
	comment = strings.TrimSpace(comment)

	// Truncate if needed
	if len(comment) > maxLen {
		return comment[:maxLen-3] + "..."
	}
	return comment
}
