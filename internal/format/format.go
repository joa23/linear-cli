// Package format provides ASCII and JSON formatting for Linear resources.
// This is a token-efficient alternative to Markdown templates.
package format

import (
	"fmt"
	"strings"
	"time"

	"github.com/joa23/linear-cli/internal/linear/core"
)

// Format specifies the level of detail in formatted output
type Format string

const (
	// Minimal returns only essential fields (~50 tokens per issue)
	Minimal Format = "minimal"
	// Compact returns commonly needed fields (~150 tokens per issue)
	Compact Format = "compact"
	// Full returns all fields (~500 tokens per issue)
	Full Format = "full"
)

// ParseFormat parses a string into a Format with validation
func ParseFormat(s string) (Format, error) {
	if s == "" {
		return Compact, nil // Default to compact for balanced output
	}
	format := Format(s)
	switch format {
	case Minimal, Compact, Full:
		return format, nil
	default:
		return "", fmt.Errorf("invalid format '%s': must be 'minimal', 'compact', or 'full'", s)
	}
}

// Pagination holds pagination metadata for list responses
type Pagination struct {
	Start       int  // Starting position (0-indexed)
	Limit       int  // Items per page
	Count       int  // Items in this page
	TotalCount  int  // Total items
	HasNextPage bool // More results exist
	// Deprecated: Use offset-based pagination instead
	EndCursor   string // Cursor for cursor-based pagination
}

// Formatter formats Linear resources as ASCII text or JSON
type Formatter struct {
	factory *RendererFactory
}

// New creates a new Formatter
func New() *Formatter {
	return &Formatter{
		factory: NewRendererFactory(),
	}
}

// --- New unified rendering methods ---

// RenderIssue renders a single issue with the specified verbosity and output type
func (f *Formatter) RenderIssue(issue *core.Issue, verbosity Verbosity, outputType OutputType) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderIssue(issue, verbosity)
}

// RenderIssueList renders a list of issues with the specified verbosity and output type
func (f *Formatter) RenderIssueList(issues []core.Issue, verbosity Verbosity, outputType OutputType, page *Pagination) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderIssueList(issues, verbosity, page)
}

// RenderCycle renders a single cycle with the specified verbosity and output type
func (f *Formatter) RenderCycle(cycle *core.Cycle, verbosity Verbosity, outputType OutputType) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderCycle(cycle, verbosity)
}

// RenderCycleList renders a list of cycles with the specified verbosity and output type
func (f *Formatter) RenderCycleList(cycles []core.Cycle, verbosity Verbosity, outputType OutputType, page *Pagination) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderCycleList(cycles, verbosity, page)
}

// RenderProject renders a single project with the specified verbosity and output type
func (f *Formatter) RenderProject(project *core.Project, verbosity Verbosity, outputType OutputType) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderProject(project, verbosity)
}

// RenderProjectList renders a list of projects with the specified verbosity and output type
func (f *Formatter) RenderProjectList(projects []core.Project, verbosity Verbosity, outputType OutputType, page *Pagination) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderProjectList(projects, verbosity, page)
}

// RenderTeam renders a single team with the specified verbosity and output type
func (f *Formatter) RenderTeam(team *core.Team, verbosity Verbosity, outputType OutputType) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderTeam(team, verbosity)
}

// RenderTeamList renders a list of teams with the specified verbosity and output type
func (f *Formatter) RenderTeamList(teams []core.Team, verbosity Verbosity, outputType OutputType) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderTeamList(teams, verbosity)
}

// RenderUser renders a single user with the specified verbosity and output type
func (f *Formatter) RenderUser(user *core.User, verbosity Verbosity, outputType OutputType) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderUser(user, verbosity)
}

// RenderUserList renders a list of users with the specified verbosity and output type
func (f *Formatter) RenderUserList(users []core.User, verbosity Verbosity, outputType OutputType) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderUserList(users, verbosity)
}

// RenderComment renders a single comment with the specified verbosity and output type
func (f *Formatter) RenderComment(comment *core.Comment, verbosity Verbosity, outputType OutputType) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderComment(comment, verbosity)
}

// RenderCommentList renders a list of comments with the specified verbosity and output type
func (f *Formatter) RenderCommentList(comments []core.Comment, verbosity Verbosity, outputType OutputType) string {
	renderer := f.factory.GetRenderer(outputType)
	return renderer.RenderCommentList(comments, verbosity)
}

// --- Utility functions ---

// line creates a horizontal separator line
func line(width int) string {
	return strings.Repeat("─", width)
}

// formatDate formats an ISO date string to a short format
func formatDate(isoDate string) string {
	if isoDate == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, isoDate)
	if err != nil {
		// Try parsing as date-only
		t, err = time.Parse("2006-01-02", isoDate)
		if err != nil {
			return isoDate
		}
	}
	return t.Format("2006-01-02")
}

// formatDateTime formats an ISO date string to include time
func formatDateTime(isoDate string) string {
	if isoDate == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, isoDate)
	if err != nil {
		return isoDate
	}
	return t.Format("2006-01-02 15:04")
}

// priorityLabel converts a priority number to a label
func priorityLabel(priority *int) string {
	if priority == nil {
		return ""
	}
	switch *priority {
	case 0:
		return ""
	case 1:
		return "P1:Urgent"
	case 2:
		return "P2:High"
	case 3:
		return "P3:Medium"
	case 4:
		return "P4:Low"
	default:
		return fmt.Sprintf("P%d", *priority)
	}
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// CleanDescription removes markdown formatting and normalizes whitespace.
// Exported for use by CLI commands that render comment bodies directly.
func CleanDescription(desc string) string {
	return cleanDescription(desc)
}

// cleanDescription removes markdown formatting and normalizes whitespace
func cleanDescription(desc string) string {
	if desc == "" {
		return ""
	}
	// Remove markdown headers
	lines := strings.Split(desc, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and markdown artifacts
		if line == "" || line == "---" || strings.HasPrefix(line, "```") {
			continue
		}
		// Remove markdown header prefixes
		for strings.HasPrefix(line, "#") {
			line = strings.TrimPrefix(line, "#")
			line = strings.TrimSpace(line)
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}
