package format

import (
	"strings"

	"github.com/joa23/linear-cli/internal/linear"
)

// Team formats a single team
func (f *Formatter) Team(team *linear.Team) string {
	if team == nil {
		return ""
	}

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

// TeamList formats a list of teams with optional pagination
func (f *Formatter) TeamList(teams []linear.Team, page *Pagination) string {
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
		b.WriteString(f.teamCompact(&team))
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

func (f *Formatter) teamCompact(team *linear.Team) string {
	var b strings.Builder

	// Line 1: Name and key
	b.WriteString(fmtSprintf("%s (%s)\n", team.Name, team.Key))

	// Line 2: Description (if any)
	if team.Description != "" {
		b.WriteString(fmtSprintf("  %s\n", truncate(team.Description, 80)))
	}

	return b.String()
}
