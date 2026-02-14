package format

import (
	"encoding/json"
	"fmt"

	"github.com/joa23/linear-cli/internal/linear/core"
)

// JSONRenderer renders Linear resources as JSON.
// This provides machine-readable output for scripting and automation.
type JSONRenderer struct{}

// --- Issue Rendering ---

func (r *JSONRenderer) RenderIssue(issue *core.Issue, verbosity Verbosity) string {
	if issue == nil {
		return r.renderError("Issue is nil")
	}

	var dto interface{}

	switch verbosity {
	case VerbosityMinimal:
		dto = IssueToMinimalDTO(issue)
	case VerbosityFull:
		dto = IssueToFullDTO(issue)
	case VerbosityCompact:
		dto = IssueToCompactDTO(issue)
	default:
		dto = IssueToCompactDTO(issue)
	}

	return r.marshal(dto)
}

func (r *JSONRenderer) RenderIssueList(issues []core.Issue, verbosity Verbosity, page *Pagination) string {
	if len(issues) == 0 {
		return "[]"
	}

	dtos := make([]interface{}, len(issues))
	for i, issue := range issues {
		switch verbosity {
		case VerbosityMinimal:
			dtos[i] = IssueToMinimalDTO(&issue)
		case VerbosityFull:
			dtos[i] = IssueToFullDTO(&issue)
		case VerbosityCompact:
			dtos[i] = IssueToCompactDTO(&issue)
		default:
			dtos[i] = IssueToCompactDTO(&issue)
		}
	}

	return r.marshal(dtos)
}

// --- Cycle Rendering ---

func (r *JSONRenderer) RenderCycle(cycle *core.Cycle, verbosity Verbosity) string {
	if cycle == nil {
		return r.renderError("Cycle is nil")
	}

	var dto interface{}

	switch verbosity {
	case VerbosityMinimal:
		dto = CycleToMinimalDTO(cycle)
	case VerbosityFull:
		dto = CycleToFullDTO(cycle)
	case VerbosityCompact:
		dto = CycleToCompactDTO(cycle)
	default:
		dto = CycleToCompactDTO(cycle)
	}

	return r.marshal(dto)
}

func (r *JSONRenderer) RenderCycleList(cycles []core.Cycle, verbosity Verbosity, page *Pagination) string {
	if len(cycles) == 0 {
		return "[]"
	}

	dtos := make([]interface{}, len(cycles))
	for i, cycle := range cycles {
		switch verbosity {
		case VerbosityMinimal:
			dtos[i] = CycleToMinimalDTO(&cycle)
		case VerbosityFull:
			dtos[i] = CycleToFullDTO(&cycle)
		case VerbosityCompact:
			dtos[i] = CycleToCompactDTO(&cycle)
		default:
			dtos[i] = CycleToCompactDTO(&cycle)
		}
	}

	return r.marshal(dtos)
}

// --- Project Rendering ---

func (r *JSONRenderer) RenderProject(project *core.Project, verbosity Verbosity) string {
	if project == nil {
		return r.renderError("Project is nil")
	}

	dto := ProjectToDTO(project)
	return r.marshal(dto)
}

func (r *JSONRenderer) RenderProjectList(projects []core.Project, verbosity Verbosity, page *Pagination) string {
	if len(projects) == 0 {
		return "[]"
	}

	dtos := make([]ProjectDTO, len(projects))
	for i, project := range projects {
		dtos[i] = ProjectToDTO(&project)
	}

	return r.marshal(dtos)
}

// --- Team Rendering ---

func (r *JSONRenderer) RenderTeam(team *core.Team, verbosity Verbosity) string {
	if team == nil {
		return r.renderError("Team is nil")
	}

	dto := TeamToDTO(team)
	return r.marshal(dto)
}

func (r *JSONRenderer) RenderTeamList(teams []core.Team, verbosity Verbosity) string {
	if len(teams) == 0 {
		return "[]"
	}

	dtos := make([]TeamDTO, len(teams))
	for i, team := range teams {
		dtos[i] = TeamToDTO(&team)
	}

	return r.marshal(dtos)
}

// --- User Rendering ---

func (r *JSONRenderer) RenderUser(user *core.User, verbosity Verbosity) string {
	if user == nil {
		return r.renderError("User is nil")
	}

	dto := UserToDTO(user)
	return r.marshal(dto)
}

func (r *JSONRenderer) RenderUserList(users []core.User, verbosity Verbosity) string {
	if len(users) == 0 {
		return "[]"
	}

	dtos := make([]UserDTO, len(users))
	for i, user := range users {
		dtos[i] = UserToDTO(&user)
	}

	return r.marshal(dtos)
}

// --- Comment Rendering ---

func (r *JSONRenderer) RenderComment(comment *core.Comment, verbosity Verbosity) string {
	if comment == nil {
		return r.renderError("Comment is nil")
	}

	dto := CommentToDTO(comment)
	return r.marshal(dto)
}

func (r *JSONRenderer) RenderCommentList(comments []core.Comment, verbosity Verbosity) string {
	if len(comments) == 0 {
		return "[]"
	}

	dtos := make([]CommentDTO, len(comments))
	for i, comment := range comments {
		dtos[i] = CommentToDTO(&comment)
	}

	return r.marshal(dtos)
}

// --- Attachment Rendering ---

func (r *JSONRenderer) RenderAttachment(att *core.Attachment, verbosity Verbosity) string {
	dto := AttachmentToDTO(att)
	data, _ := json.MarshalIndent(dto, "", "  ")
	return string(data)
}

func (r *JSONRenderer) RenderAttachmentList(atts []core.Attachment, verbosity Verbosity) string {
	dtos := make([]AttachmentDTO, len(atts))
	for i, att := range atts {
		dtos[i] = AttachmentToDTO(&att)
	}
	data, _ := json.MarshalIndent(dtos, "", "  ")
	return string(data)
}

// --- Helper methods ---

// marshal converts an object to pretty-printed JSON
func (r *JSONRenderer) marshal(v interface{}) string {
	jsonBytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return r.renderError(fmt.Sprintf("failed to marshal JSON: %s", err))
	}
	return string(jsonBytes)
}

// renderError wraps an error message in JSON format
func (r *JSONRenderer) renderError(message string) string {
	errorObj := map[string]string{
		"error": message,
	}
	jsonBytes, err := json.MarshalIndent(errorObj, "", "  ")
	if err != nil {
		// Fallback if even error marshaling fails
		return fmt.Sprintf(`{"error": "%s"}`, message)
	}
	return string(jsonBytes)
}
