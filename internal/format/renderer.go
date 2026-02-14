package format

import "github.com/joa23/linear-cli/internal/linear/core"

// Renderer defines the interface for formatting Linear resources.
// Different implementations can render to text, JSON, or other formats.
type Renderer interface {
	// Single entity rendering
	RenderIssue(issue *core.Issue, verbosity Verbosity) string
	RenderCycle(cycle *core.Cycle, verbosity Verbosity) string
	RenderProject(project *core.Project, verbosity Verbosity) string
	RenderTeam(team *core.Team, verbosity Verbosity) string
	RenderUser(user *core.User, verbosity Verbosity) string
	RenderComment(comment *core.Comment, verbosity Verbosity) string

	// List rendering with pagination
	RenderIssueList(issues []core.Issue, verbosity Verbosity, page *Pagination) string
	RenderCycleList(cycles []core.Cycle, verbosity Verbosity, page *Pagination) string
	RenderProjectList(projects []core.Project, verbosity Verbosity, page *Pagination) string
	RenderTeamList(teams []core.Team, verbosity Verbosity) string
	RenderUserList(users []core.User, verbosity Verbosity) string
	RenderCommentList(comments []core.Comment, verbosity Verbosity) string

	// Attachment rendering
	RenderAttachment(att *core.Attachment, verbosity Verbosity) string
	RenderAttachmentList(atts []core.Attachment, verbosity Verbosity) string
}
