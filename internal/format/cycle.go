package format

import (
	"strings"

	"github.com/joa23/linear-cli/internal/linear"
)

// Cycle formats a single cycle
func (f *Formatter) Cycle(cycle *linear.Cycle, fmt Format) string {
	if cycle == nil {
		return ""
	}

	switch fmt {
	case Minimal:
		return f.cycleMinimal(cycle)
	case Compact:
		return f.cycleCompact(cycle)
	case Full:
		return f.cycleFull(cycle)
	default:
		return f.cycleCompact(cycle)
	}
}

// CycleList formats a list of cycles with optional pagination
func (f *Formatter) CycleList(cycles []linear.Cycle, fmt Format, page *Pagination) string {
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
		b.WriteString(f.Cycle(&cycle, fmt))
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

func (f *Formatter) cycleMinimal(cycle *linear.Cycle) string {
	// Format: Cycle 67 [Active] (Jan 15 - Jan 28)
	status := cycleStatus(cycle)
	return fmtSprintf("Cycle %d [%s] (%s - %s)",
		cycle.Number, status, formatDate(cycle.StartsAt), formatDate(cycle.EndsAt))
}

func (f *Formatter) cycleCompact(cycle *linear.Cycle) string {
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

func (f *Formatter) cycleFull(cycle *linear.Cycle) string {
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

// cycleStatus returns a human-readable status for a cycle
func cycleStatus(cycle *linear.Cycle) string {
	switch {
	case cycle.IsActive:
		return "Active"
	case cycle.IsFuture:
		return "Future"
	case cycle.IsPast:
		return "Past"
	case cycle.IsNext:
		return "Next"
	case cycle.IsPrevious:
		return "Previous"
	default:
		return "Unknown"
	}
}

// CycleAnalysis formats cycle analytics data
func (f *Formatter) CycleAnalysis(analysis *linear.CycleAnalysis, teamName, assigneeName string, showRecommendation bool) string {
	if analysis == nil {
		return "No cycle analysis data available."
	}

	var b strings.Builder

	// Header
	if assigneeName != "" {
		b.WriteString(fmtSprintf("CYCLE ANALYSIS: %s (%s)\n", assigneeName, teamName))
	} else {
		b.WriteString(fmtSprintf("CYCLE ANALYSIS: %s (Team)\n", teamName))
	}
	b.WriteString(line(50))
	b.WriteString("\n\n")

	// Summary section
	b.WriteString(fmtSprintf("Cycles Analyzed: %d\n", analysis.CycleCount))
	b.WriteString(fmtSprintf("Avg Velocity: %.1f points/cycle\n", analysis.AvgVelocity))
	b.WriteString(fmtSprintf("Completion Rate: %.1f%%\n", analysis.AvgCompletionRate))
	b.WriteString(fmtSprintf("Avg Scope Creep: %.1f%%\n", analysis.AvgScopeCreepPercent))
	b.WriteString(fmtSprintf("Throughput: %.1f issues/cycle\n", analysis.AvgThroughput))

	// Variability metrics
	b.WriteString("\nVARIABILITY\n")
	b.WriteString(line(30))
	b.WriteString("\n")
	b.WriteString(fmtSprintf("  Velocity StdDev: %.1f\n", analysis.StdDevVelocity))
	b.WriteString(fmtSprintf("  Median Velocity: %.1f\n", analysis.MedianVelocity))

	// Recommendation section
	if showRecommendation {
		b.WriteString("\nRECOMMENDATION\n")
		b.WriteString(line(30))
		b.WriteString("\n")
		b.WriteString(fmtSprintf("  Conservative: %.0f points (P80)\n", analysis.P80Velocity))
		b.WriteString(fmtSprintf("  Target: %.0f points (Median)\n", analysis.MedianVelocity))
		b.WriteString(fmtSprintf("  Optimistic: %.0f points (P20)\n", analysis.P20Velocity))
	}

	return b.String()
}
