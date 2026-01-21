package format

import (
	"strings"

	"github.com/joa23/linear-cli/internal/linear"
)

// Project formats a single project
func (f *Formatter) Project(project *linear.Project) string {
	if project == nil {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString(fmtSprintf("%s\n", project.Name))
	b.WriteString(line(50))
	b.WriteString("\n")

	// State
	b.WriteString(fmtSprintf("State: %s\n", project.State))

	// Description
	if project.Description != "" {
		b.WriteString(fmtSprintf("Description: %s\n", truncate(project.Description, 100)))
	}

	// Timestamps
	b.WriteString(fmtSprintf("Created: %s\n", formatDateTime(project.CreatedAt)))
	b.WriteString(fmtSprintf("Updated: %s\n", formatDateTime(project.UpdatedAt)))

	// Content (long description) - truncated for display
	if project.Content != "" {
		b.WriteString("\nCONTENT\n")
		b.WriteString(line(40))
		b.WriteString("\n")
		b.WriteString(truncate(cleanDescription(project.Content), 500))
		b.WriteString("\n")
	}

	// Issues if present
	issues := project.GetIssues()
	if len(issues) > 0 {
		b.WriteString(fmtSprintf("\nISSUES (%d)\n", len(issues)))
		b.WriteString(line(40))
		b.WriteString("\n")
		for _, issue := range issues {
			assignee := "Unassigned"
			if issue.Assignee != nil {
				assignee = "@" + issue.Assignee.Name
			}
			b.WriteString(fmtSprintf("  %s [%s] %s (%s)\n",
				issue.Identifier, issue.State.Name, truncate(issue.Title, 40), assignee))
		}
	}

	return b.String()
}

// ProjectList formats a list of projects with optional pagination
func (f *Formatter) ProjectList(projects []linear.Project, page *Pagination) string {
	if len(projects) == 0 {
		return "No projects found."
	}

	var b strings.Builder

	// Header
	b.WriteString(fmtSprintf("PROJECTS (%d)\n", len(projects)))
	b.WriteString(line(40))
	b.WriteString("\n")

	// Format each project
	for _, project := range projects {
		b.WriteString(f.projectCompact(&project))
		b.WriteString("\n")
	}

	// Pagination footer
	if page != nil && page.HasNextPage && page.EndCursor != "" {
		b.WriteString(line(40))
		b.WriteString("\n")
		b.WriteString(fmtSprintf("Next: cursor=%s\n", page.EndCursor))
	}

	return b.String()
}

func (f *Formatter) projectCompact(project *linear.Project) string {
	var b strings.Builder

	// Line 1: Name and state
	b.WriteString(fmtSprintf("%s [%s]\n", project.Name, project.State))

	// Line 2: Description (if any)
	if project.Description != "" {
		b.WriteString(fmtSprintf("  %s\n", truncate(project.Description, 80)))
	}

	// Line 3: Issue count (if available)
	issues := project.GetIssues()
	if len(issues) > 0 {
		b.WriteString(fmtSprintf("  Issues: %d\n", len(issues)))
	}

	return b.String()
}
