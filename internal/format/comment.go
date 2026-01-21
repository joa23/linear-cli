package format

import (
	"strings"

	"github.com/joa23/linear-cli/internal/linear"
)

// Comment formats a single comment
func (f *Formatter) Comment(comment *linear.Comment) string {
	if comment == nil {
		return ""
	}

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

// CommentList formats a list of comments with optional pagination
func (f *Formatter) CommentList(comments []linear.Comment, page *Pagination) string {
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
		b.WriteString(f.commentCompact(&comment))
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

func (f *Formatter) commentCompact(comment *linear.Comment) string {
	var b strings.Builder

	// Line 1: Author and date
	b.WriteString(fmtSprintf("@%s (%s):\n", comment.User.Name, formatDate(comment.CreatedAt)))

	// Line 2: Body (truncated)
	body := truncate(cleanDescription(comment.Body), 150)
	b.WriteString(fmtSprintf("  %s\n", body))

	return b.String()
}
