package format

import (
	"strings"

	"github.com/joa23/linear-cli/internal/linear/core"
)

// TextRenderer renders Linear resources as ASCII text.
// This preserves the original token-efficient formatting behavior.
type TextRenderer struct{}

// --- Issue Rendering ---

func (r *TextRenderer) RenderIssue(issue *core.Issue, verbosity Verbosity) string {
	if issue == nil {
		return ""
	}

	switch verbosity {
	case VerbosityMinimal:
		return r.issueMinimal(issue)
	case VerbosityFull:
		return r.issueFull(issue)
	case VerbosityCompact:
		return r.issueCompact(issue)
	default:
		return r.issueCompact(issue)
	}
}

func (r *TextRenderer) RenderIssueList(issues []core.Issue, verbosity Verbosity, page *Pagination) string {
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
		b.WriteString(r.RenderIssue(&issue, verbosity))
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

func (r *TextRenderer) issueMinimal(issue *core.Issue) string {
	// Format: CEN-123 [State] Title
	state := issue.State.Name
	return fmtSprintf("%s [%s] %s", issue.Identifier, state, truncate(issue.Title, 60))
}

func (r *TextRenderer) issueCompact(issue *core.Issue) string {
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

func (r *TextRenderer) issueFull(issue *core.Issue) string {
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
			b.WriteString(fmtSprintf("  %s\n\n", truncate(cleanDescription(comment.Body), 200)))
		}
	}

	return b.String()
}

// --- Cycle Rendering ---

func (r *TextRenderer) RenderCycle(cycle *core.Cycle, verbosity Verbosity) string {
	if cycle == nil {
		return ""
	}

	switch verbosity {
	case VerbosityMinimal:
		return r.cycleMinimal(cycle)
	case VerbosityFull:
		return r.cycleFull(cycle)
	case VerbosityCompact:
		return r.cycleCompact(cycle)
	default:
		return r.cycleCompact(cycle)
	}
}

func (r *TextRenderer) RenderCycleList(cycles []core.Cycle, verbosity Verbosity, page *Pagination) string {
	if len(cycles) == 0 {
		return "No cycles found."
	}

	var b strings.Builder

	// Header
	b.WriteString(fmtSprintf("CYCLES (%d)\n", len(cycles)))
	b.WriteString(line(40))
	b.WriteString("\n")

	// Format each cycle
	for _, cycle := range cycles {
		b.WriteString(r.RenderCycle(&cycle, verbosity))
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

func (r *TextRenderer) cycleMinimal(cycle *core.Cycle) string {
	// Format: Cycle 67 [Active] (Jan 15 - Jan 28)
	status := cycleStatus(cycle)
	return fmtSprintf("Cycle %d [%s] (%s - %s)",
		cycle.Number, status, formatDate(cycle.StartsAt), formatDate(cycle.EndsAt))
}

func (r *TextRenderer) cycleCompact(cycle *core.Cycle) string {
	var b strings.Builder

	// Line 1: Cycle name/number and status
	status := cycleStatus(cycle)
	if cycle.Name != "" {
		b.WriteString(fmtSprintf("Cycle %d: %s [%s]\n", cycle.Number, cycle.Name, status))
	} else {
		b.WriteString(fmtSprintf("Cycle %d [%s]\n", cycle.Number, status))
	}

	// Line 2: Date range and progress
	b.WriteString(fmtSprintf("  %s - %s | Progress: %.0f%%\n",
		formatDate(cycle.StartsAt), formatDate(cycle.EndsAt), cycle.Progress*100))

	// Line 3: Team if available
	if cycle.Team != nil {
		b.WriteString(fmtSprintf("  Team: %s\n", cycle.Team.Name))
	}

	return b.String()
}

func (r *TextRenderer) cycleFull(cycle *core.Cycle) string {
	var b strings.Builder

	// Header
	if cycle.Name != "" {
		b.WriteString(fmtSprintf("Cycle %d: %s\n", cycle.Number, cycle.Name))
	} else {
		b.WriteString(fmtSprintf("Cycle %d\n", cycle.Number))
	}
	b.WriteString(line(60))
	b.WriteString("\n")

	// Status and dates
	b.WriteString(fmtSprintf("Status: %s\n", cycleStatus(cycle)))
	b.WriteString(fmtSprintf("Starts: %s\n", formatDate(cycle.StartsAt)))
	b.WriteString(fmtSprintf("Ends: %s\n", formatDate(cycle.EndsAt)))
	b.WriteString(fmtSprintf("Progress: %.1f%%\n", cycle.Progress*100))

	if cycle.Team != nil {
		b.WriteString(fmtSprintf("Team: %s (%s)\n", cycle.Team.Name, cycle.Team.Key))
	}

	// Description if present
	if cycle.Description != "" {
		b.WriteString("\nDESCRIPTION\n")
		b.WriteString(line(40))
		b.WriteString("\n")
		b.WriteString(cycle.Description)
		b.WriteString("\n")
	}

	// History data if available
	if len(cycle.ScopeHistory) > 0 || len(cycle.CompletedScopeHistory) > 0 {
		b.WriteString("\nMETRICS\n")
		b.WriteString(line(40))
		b.WriteString("\n")

		if len(cycle.ScopeHistory) > 0 {
			latest := cycle.ScopeHistory[len(cycle.ScopeHistory)-1]
			b.WriteString(fmtSprintf("  Current Scope: %d points\n", latest))
		}

		if len(cycle.CompletedScopeHistory) > 0 {
			latest := cycle.CompletedScopeHistory[len(cycle.CompletedScopeHistory)-1]
			b.WriteString(fmtSprintf("  Completed: %d points\n", latest))
		}

		if len(cycle.InProgressScopeHistory) > 0 {
			latest := cycle.InProgressScopeHistory[len(cycle.InProgressScopeHistory)-1]
			b.WriteString(fmtSprintf("  In Progress: %d points\n", latest))
		}

		if len(cycle.IssueCountHistory) > 0 {
			latest := cycle.IssueCountHistory[len(cycle.IssueCountHistory)-1]
			b.WriteString(fmtSprintf("  Issue Count: %d\n", latest))
		}

		if len(cycle.CompletedIssueCountHistory) > 0 {
			latest := cycle.CompletedIssueCountHistory[len(cycle.CompletedIssueCountHistory)-1]
			b.WriteString(fmtSprintf("  Completed Issues: %d\n", latest))
		}
	}

	// Timestamps
	b.WriteString(fmtSprintf("\nCreated: %s\n", formatDateTime(cycle.CreatedAt)))
	b.WriteString(fmtSprintf("Updated: %s\n", formatDateTime(cycle.UpdatedAt)))

	return b.String()
}

// --- Project Rendering ---

func (r *TextRenderer) RenderProject(project *core.Project, verbosity Verbosity) string {
	if project == nil {
		return ""
	}

	// Projects don't have varying verbosity levels in current implementation
	return r.projectFull(project)
}

func (r *TextRenderer) RenderProjectList(projects []core.Project, verbosity Verbosity, page *Pagination) string {
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
		b.WriteString(r.projectCompact(&project))
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

func (r *TextRenderer) projectFull(project *core.Project) string {
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

func (r *TextRenderer) projectCompact(project *core.Project) string {
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

// --- Team Rendering ---

func (r *TextRenderer) RenderTeam(team *core.Team, verbosity Verbosity) string {
	if team == nil {
		return ""
	}

	return r.teamFull(team)
}

func (r *TextRenderer) RenderTeamList(teams []core.Team, verbosity Verbosity) string {
	if len(teams) == 0 {
		return "No teams found."
	}

	var b strings.Builder

	// Header
	b.WriteString(fmtSprintf("TEAMS (%d)\n", len(teams)))
	b.WriteString(line(40))
	b.WriteString("\n")

	// Format each team
	for _, team := range teams {
		b.WriteString(r.teamCompact(&team))
		b.WriteString("\n")
	}

	return b.String()
}

func (r *TextRenderer) teamFull(team *core.Team) string {
	var b strings.Builder

	// Header
	b.WriteString(fmtSprintf("%s (%s)\n", team.Name, team.Key))
	b.WriteString(line(50))
	b.WriteString("\n")

	// Description
	if team.Description != "" {
		b.WriteString(fmtSprintf("Description: %s\n", team.Description))
	}

	// Estimation settings
	if team.IssueEstimationType != "" && team.IssueEstimationType != "notUsed" {
		b.WriteString(fmtSprintf("Estimation: %s\n", team.IssueEstimationType))

		scale := team.GetEstimateScale()
		if len(scale.Values) > 0 {
			var valuesStr []string
			for _, v := range scale.Values {
				valuesStr = append(valuesStr, fmtSprintf("%.0f", v))
			}
			b.WriteString(fmtSprintf("  Scale: [%s]\n", strings.Join(valuesStr, ", ")))
		}
		if len(scale.Labels) > 0 {
			b.WriteString(fmtSprintf("  Labels: [%s]\n", strings.Join(scale.Labels, ", ")))
		}
	}

	return b.String()
}

func (r *TextRenderer) teamCompact(team *core.Team) string {
	var b strings.Builder

	// Line 1: Name and key
	b.WriteString(fmtSprintf("%s (%s)\n", team.Name, team.Key))

	// Line 2: Description (if any)
	if team.Description != "" {
		b.WriteString(fmtSprintf("  %s\n", truncate(team.Description, 80)))
	}

	return b.String()
}

// --- User Rendering ---

func (r *TextRenderer) RenderUser(user *core.User, verbosity Verbosity) string {
	if user == nil {
		return ""
	}

	return r.userFull(user)
}

func (r *TextRenderer) RenderUserList(users []core.User, verbosity Verbosity) string {
	if len(users) == 0 {
		return "No users found."
	}

	var b strings.Builder

	// Header
	b.WriteString(fmtSprintf("USERS (%d)\n", len(users)))
	b.WriteString(line(40))
	b.WriteString("\n")

	// Format each user
	for _, user := range users {
		b.WriteString(r.userCompact(&user))
		b.WriteString("\n")
	}

	return b.String()
}

func (r *TextRenderer) userFull(user *core.User) string {
	var b strings.Builder

	// Header
	name := user.Name
	if user.DisplayName != "" {
		name = user.DisplayName
	}
	b.WriteString(fmtSprintf("%s\n", name))
	b.WriteString(line(50))
	b.WriteString("\n")

	// Email
	b.WriteString(fmtSprintf("Email: %s\n", user.Email))

	// Status
	status := "Active"
	if !user.Active {
		status = "Inactive"
	}
	if user.Admin {
		status += " (Admin)"
	}
	b.WriteString(fmtSprintf("Status: %s\n", status))

	// Created
	b.WriteString(fmtSprintf("Joined: %s\n", formatDate(user.CreatedAt)))

	// Teams if available
	if len(user.Teams) > 0 {
		b.WriteString("\nTEAMS\n")
		b.WriteString(line(30))
		b.WriteString("\n")
		for _, team := range user.Teams {
			b.WriteString(fmtSprintf("  %s (%s)\n", team.Name, team.Key))
		}
	}

	return b.String()
}

func (r *TextRenderer) userCompact(user *core.User) string {
	var b strings.Builder

	// Line 1: Name and email
	name := user.Name
	if user.DisplayName != "" {
		name = user.DisplayName
	}
	b.WriteString(fmtSprintf("%s <%s>\n", name, user.Email))

	// Line 2: Status and teams
	var meta []string

	if !user.Active {
		meta = append(meta, "Inactive")
	}
	if user.Admin {
		meta = append(meta, "Admin")
	}
	if len(user.Teams) > 0 {
		var teamKeys []string
		for _, team := range user.Teams {
			teamKeys = append(teamKeys, team.Key)
		}
		meta = append(meta, fmtSprintf("Teams: %s", strings.Join(teamKeys, ", ")))
	}

	if len(meta) > 0 {
		b.WriteString(fmtSprintf("  %s\n", strings.Join(meta, " | ")))
	}

	return b.String()
}

// --- Comment Rendering ---

func (r *TextRenderer) RenderComment(comment *core.Comment, verbosity Verbosity) string {
	if comment == nil {
		return ""
	}

	return r.commentFull(comment)
}

func (r *TextRenderer) RenderCommentList(comments []core.Comment, verbosity Verbosity) string {
	if len(comments) == 0 {
		return "No comments found."
	}

	var b strings.Builder

	// Header
	b.WriteString(fmtSprintf("COMMENTS (%d)\n", len(comments)))
	b.WriteString(line(40))
	b.WriteString("\n")

	// Format each comment
	for _, comment := range comments {
		b.WriteString(r.commentCompact(&comment))
		b.WriteString("\n")
	}

	return b.String()
}

func (r *TextRenderer) commentFull(comment *core.Comment) string {
	var b strings.Builder

	// Header: Author and date
	b.WriteString(fmtSprintf("@%s (%s)\n", comment.User.Name, formatDateTime(comment.CreatedAt)))
	b.WriteString(line(40))
	b.WriteString("\n")

	// Body
	b.WriteString(cleanDescription(comment.Body))
	b.WriteString("\n")

	// Issue context
	if comment.Issue.Identifier != "" {
		b.WriteString(fmtSprintf("\nOn: %s - %s\n", comment.Issue.Identifier, comment.Issue.Title))
	}

	// Reply info
	if comment.Parent != nil {
		b.WriteString(fmtSprintf("Reply to: %s\n", comment.Parent.ID))
	}

	return b.String()
}

// --- Attachment Rendering ---

func (r *TextRenderer) RenderAttachment(att *core.Attachment, verbosity Verbosity) string {
	var b strings.Builder

	sourceTag := att.SourceType
	if sourceTag == "" {
		sourceTag = "link"
	}

	b.WriteString(fmtSprintf("[%s] %s\n", sourceTag, att.Title))

	if verbosity >= VerbosityCompact {
		b.WriteString(fmtSprintf("  URL: %s\n", att.URL))
		if att.Subtitle != "" {
			b.WriteString(fmtSprintf("  Subtitle: %s\n", att.Subtitle))
		}
	}

	if verbosity >= VerbosityFull {
		b.WriteString(fmtSprintf("  ID: %s\n", att.ID))
		b.WriteString(fmtSprintf("  Created: %s\n", formatDate(att.CreatedAt)))
	}

	return b.String()
}

func (r *TextRenderer) RenderAttachmentList(atts []core.Attachment, verbosity Verbosity) string {
	if len(atts) == 0 {
		return "No attachments found.\n"
	}

	var b strings.Builder
	b.WriteString(fmtSprintf("Attachments (%d)\n", len(atts)))
	b.WriteString(line(40))
	b.WriteString("\n")

	for _, att := range atts {
		b.WriteString(r.RenderAttachment(&att, verbosity))
		b.WriteString("\n")
	}

	return b.String()
}

func (r *TextRenderer) commentCompact(comment *core.Comment) string {
	var b strings.Builder

	// Line 1: Author and date
	b.WriteString(fmtSprintf("@%s (%s):\n", comment.User.Name, formatDate(comment.CreatedAt)))

	// Line 2: Body (truncated)
	body := truncate(cleanDescription(comment.Body), 150)
	b.WriteString(fmtSprintf("  %s\n", body))

	return b.String()
}
