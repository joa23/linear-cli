package format

import (
	"fmt"
	"strings"

	"github.com/joa23/linear-cli/internal/linear/core"
)

// Issue formats a single issue
func (f *Formatter) Issue(issue *core.Issue, fmt Format) string {
	if issue == nil {
		return ""
	}

	switch fmt {
	case Minimal:
		return f.issueMinimal(issue)
	case Compact:
		return f.issueCompact(issue)
	case Full:
		return f.issueFull(issue)
	default:
		return f.issueCompact(issue)
	}
}

// IssueList formats a list of issues with optional pagination
func (f *Formatter) IssueList(issues []core.Issue, fmt Format, page *Pagination) string {
	if len(issues) == 0 {
		return "No issues found."
	}

	var b strings.Builder

	// Header with range
	if page != nil {
		b.WriteString(formatPaginationHeader("ISSUES", page))
	} else {
		b.WriteString(fmtSprintf("ISSUES (%d)\n", len(issues)))
	}
	b.WriteString(line(40))
	b.WriteString("\n")

	// Format each issue
	for _, issue := range issues {
		b.WriteString(f.Issue(&issue, fmt))
		b.WriteString("\n")
	}

	// Pagination footer
	if page != nil && page.HasNextPage {
		b.WriteString(line(40))
		b.WriteString("\n")
		b.WriteString("More results available.\n")
		nextStart := page.Start + page.Limit
		b.WriteString(fmtSprintf("Next page: linear issues list --start %d --limit %d\n",
			nextStart, page.Limit))
	}

	return b.String()
}

// formatPaginationHeader creates consistent headers for list commands
func formatPaginationHeader(resource string, page *Pagination) string {
	start := page.Start + 1 // Convert to 1-indexed for display
	end := page.Start + page.Count

	// Show exact total when available (from viewer.assignedIssues)
	if page.TotalCount > 0 {
		return fmtSprintf("%s (%d-%d of %d)\n", resource, start, end, page.TotalCount)
	}

	// Fallback: show + when more pages exist
	if page.HasNextPage {
		return fmtSprintf("%s (%d-%d+)\n", resource, start, end)
	}
	return fmtSprintf("%s (%d-%d)\n", resource, start, end)
}

func (f *Formatter) issueMinimal(issue *core.Issue) string {
	// Format: CEN-123 [State] Title
	state := issue.State.Name
	return fmtSprintf("%s [%s] %s", issue.Identifier, state, truncate(issue.Title, 60))
}

func (f *Formatter) issueCompact(issue *core.Issue) string {
	var b strings.Builder

	// Line 1: Identifier [State] Title
	b.WriteString(fmtSprintf("%s [%s] %s\n", issue.Identifier, issue.State.Name, issue.Title))

	// Line 2: Metadata (assignee/delegate, priority, estimate, cycle)
	var meta []string

	if issue.Assignee != nil {
		meta = append(meta, "@"+issue.Assignee.Name)
	} else if issue.Delegate != nil {
		meta = append(meta, "Delegate:"+issue.Delegate.Name)
	} else {
		meta = append(meta, "Unassigned")
	}

	if pLabel := priorityLabel(issue.Priority); pLabel != "" {
		meta = append(meta, pLabel)
	}

	if issue.Estimate != nil {
		meta = append(meta, fmtSprintf("Est:%.0f", *issue.Estimate))
	}

	if issue.Cycle != nil {
		meta = append(meta, fmtSprintf("Cycle:%d", issue.Cycle.Number))
	}

	if issue.DueDate != nil {
		meta = append(meta, fmtSprintf("Due:%s", formatDate(*issue.DueDate)))
	}

	b.WriteString("  ")
	b.WriteString(strings.Join(meta, " | "))
	b.WriteString("\n")

	// Line 3: Project (if any)
	if issue.Project != nil {
		b.WriteString(fmtSprintf("  Project: %s\n", issue.Project.Name))
	}

	// Line 4: Parent/Children (if any)
	if issue.Parent != nil {
		b.WriteString(fmtSprintf("  Parent: %s\n", issue.Parent.Identifier))
	}
	if issue.Children.Nodes != nil && len(issue.Children.Nodes) > 0 {
		var childIDs []string
		for _, child := range issue.Children.Nodes {
			childIDs = append(childIDs, child.Identifier)
		}
		b.WriteString(fmtSprintf("  Sub-issues (%d): %s\n", len(childIDs), strings.Join(childIDs, ", ")))
	}

	return b.String()
}

func (f *Formatter) issueFull(issue *core.Issue) string {
	var b strings.Builder

	// Header
	b.WriteString(fmtSprintf("%s: %s\n", issue.Identifier, issue.Title))
	b.WriteString(line(60))
	b.WriteString("\n")

	// Status section
	b.WriteString(fmtSprintf("Status: %s\n", issue.State.Name))

	if issue.Assignee != nil {
		b.WriteString(fmtSprintf("Assignee: %s <%s>\n", issue.Assignee.Name, issue.Assignee.Email))
	} else if issue.Delegate != nil {
		b.WriteString(fmtSprintf("Delegate: %s <%s>\n", issue.Delegate.Name, issue.Delegate.Email))
	} else {
		b.WriteString("Assignee: Unassigned\n")
	}

	if pLabel := priorityLabel(issue.Priority); pLabel != "" {
		b.WriteString(fmtSprintf("Priority: %s\n", pLabel))
	}

	if issue.Estimate != nil {
		b.WriteString(fmtSprintf("Estimate: %.0f points\n", *issue.Estimate))
	}

	if issue.DueDate != nil {
		b.WriteString(fmtSprintf("Due Date: %s\n", formatDate(*issue.DueDate)))
	}

	if issue.Project != nil {
		b.WriteString(fmtSprintf("Project: %s\n", issue.Project.Name))
	}

	if issue.Cycle != nil {
		b.WriteString(fmtSprintf("Cycle: %s (#%d)\n", issue.Cycle.Name, issue.Cycle.Number))
	}

	// Labels
	if issue.Labels != nil && len(issue.Labels.Nodes) > 0 {
		var labelNames []string
		for _, label := range issue.Labels.Nodes {
			labelNames = append(labelNames, label.Name)
		}
		b.WriteString(fmtSprintf("Labels: %s\n", strings.Join(labelNames, ", ")))
	}

	// Timestamps
	b.WriteString(fmtSprintf("Created: %s\n", formatDateTime(issue.CreatedAt)))
	b.WriteString(fmtSprintf("Updated: %s\n", formatDateTime(issue.UpdatedAt)))
	b.WriteString(fmtSprintf("URL: %s\n", issue.URL))

	// Description
	if issue.Description != "" {
		b.WriteString("\nDESCRIPTION\n")
		b.WriteString(line(60))
		b.WriteString("\n")
		b.WriteString(cleanDescription(issue.Description))
		b.WriteString("\n")
	}

	// Parent/Children
	if issue.Parent != nil {
		b.WriteString(fmtSprintf("\nParent: %s - %s\n", issue.Parent.Identifier, issue.Parent.Title))
	}

	if issue.Children.Nodes != nil && len(issue.Children.Nodes) > 0 {
		b.WriteString(fmtSprintf("\nSUB-ISSUES (%d)\n", len(issue.Children.Nodes)))
		b.WriteString(line(40))
		b.WriteString("\n")
		for _, child := range issue.Children.Nodes {
			b.WriteString(fmtSprintf("  %s [%s] %s\n", child.Identifier, child.State.Name, child.Title))
		}
	}

	// Attachments
	if issue.Attachments != nil && len(issue.Attachments.Nodes) > 0 {
		b.WriteString(fmtSprintf("\nATTACHMENTS (%d)\n", len(issue.Attachments.Nodes)))
		b.WriteString(line(40))
		b.WriteString("\n")
		for _, att := range issue.Attachments.Nodes {
			b.WriteString(fmtSprintf("  [%s] %s\n", att.SourceType, att.Title))
			b.WriteString(fmtSprintf("    URL: %s\n", att.URL))
		}
	}

	// Comments
	if issue.Comments != nil && len(issue.Comments.Nodes) > 0 {
		b.WriteString(fmtSprintf("\nCOMMENTS (%d)\n", len(issue.Comments.Nodes)))
		b.WriteString(line(40))
		b.WriteString("\n")
		for _, comment := range issue.Comments.Nodes {
			b.WriteString(fmtSprintf("@%s (%s):\n", comment.User.Name, formatDate(comment.CreatedAt)))
			body := cleanDescription(comment.Body)
			for _, line := range strings.Split(body, "\n") {
				b.WriteString(fmtSprintf("  %s\n", line))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// fmtSprintf is an alias for fmt.Sprintf to avoid conflict with format.Format
func fmtSprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}
