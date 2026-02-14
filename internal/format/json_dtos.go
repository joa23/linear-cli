package format

import "github.com/joa23/linear-cli/internal/linear/core"

// --- Issue DTOs ---

// IssueMinimalDTO contains only essential issue fields (~50 tokens)
type IssueMinimalDTO struct {
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	State      string `json:"state"`
}

// IssueCompactDTO contains key metadata (~150 tokens)
type IssueCompactDTO struct {
	Identifier string   `json:"identifier"`
	Title      string   `json:"title"`
	State      string   `json:"state"`
	Priority   *int     `json:"priority"`
	Assignee   *string  `json:"assignee"`
	Delegate   *string  `json:"delegate,omitempty"` // OAuth app delegate
	Estimate   *float64 `json:"estimate"`
	DueDate    *string  `json:"dueDate"`
	CycleNumber *int    `json:"cycleNumber"`
	ProjectName *string `json:"projectName"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
}

// IssueFullDTO contains complete issue details (~500 tokens)
type IssueFullDTO struct {
	Identifier  string               `json:"identifier"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	State       *StateDTO            `json:"state"`
	Priority    *int                 `json:"priority"`
	Assignee    *UserDTO             `json:"assignee"`
	Delegate    *UserDTO             `json:"delegate,omitempty"` // OAuth app delegate
	Creator     *UserDTO             `json:"creator"`
	Estimate    *float64             `json:"estimate"`
	DueDate     *string              `json:"dueDate"`
	Labels      []LabelDTO           `json:"labels"`
	Project     *ProjectRefDTO       `json:"project"`
	Cycle       *CycleRefDTO         `json:"cycle"`
	Parent      *IssueRefDTO         `json:"parent"`
	Children    []IssueRefDTO        `json:"children"`
	Attachments []AttachmentDTO      `json:"attachments"`
	Comments    []CommentDTO         `json:"comments"`
	CreatedAt   string               `json:"createdAt"`
	UpdatedAt   string               `json:"updatedAt"`
	URL         string               `json:"url"`
}

// --- Cycle DTOs ---

// CycleMinimalDTO contains only essential cycle fields
type CycleMinimalDTO struct {
	Number   int    `json:"number"`
	Status   string `json:"status"`
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
}

// CycleCompactDTO contains key cycle metadata
type CycleCompactDTO struct {
	Number   int     `json:"number"`
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	StartsAt string  `json:"startsAt"`
	EndsAt   string  `json:"endsAt"`
	Progress float64 `json:"progress"`
	TeamName *string `json:"teamName"`
}

// CycleFullDTO contains complete cycle details
type CycleFullDTO struct {
	Number                      int      `json:"number"`
	Name                        string   `json:"name"`
	Status                      string   `json:"status"`
	StartsAt                    string   `json:"startsAt"`
	EndsAt                      string   `json:"endsAt"`
	Progress                    float64  `json:"progress"`
	Description                 string   `json:"description"`
	Team                        *TeamDTO `json:"team"`
	ScopeHistory                []int    `json:"scopeHistory"`
	CompletedScopeHistory       []int    `json:"completedScopeHistory"`
	InProgressScopeHistory      []int    `json:"inProgressScopeHistory"`
	IssueCountHistory           []int    `json:"issueCountHistory"`
	CompletedIssueCountHistory  []int    `json:"completedIssueCountHistory"`
	CreatedAt                   string   `json:"createdAt"`
	UpdatedAt                   string   `json:"updatedAt"`
}

// --- Project DTOs ---

// ProjectDTO represents a project in JSON format
type ProjectDTO struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	State       string          `json:"state"`
	Content     string          `json:"content"`
	Issues      []IssueRefDTO   `json:"issues"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}

// --- Team DTOs ---

// TeamDTO represents a team in JSON format
type TeamDTO struct {
	ID                   string          `json:"id"`
	Key                  string          `json:"key"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	IssueEstimationType  string          `json:"issueEstimationType"`
	EstimateScale        *EstimateScale  `json:"estimateScale"`
}

// EstimateScale represents the estimation scale for a team
type EstimateScale struct {
	Values []float64 `json:"values"`
	Labels []string  `json:"labels"`
}

// --- User DTOs ---

// UserDTO represents a user in JSON format
type UserDTO struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	DisplayName string     `json:"displayName"`
	Email       string     `json:"email"`
	Active      bool       `json:"active"`
	Admin       bool       `json:"admin"`
	Teams       []TeamRef  `json:"teams"`
	CreatedAt   string     `json:"createdAt"`
}

// TeamRef is a minimal team reference
type TeamRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// --- Comment DTOs ---

// CommentDTO represents a comment in JSON format
type CommentDTO struct {
	ID        string        `json:"id"`
	Body      string        `json:"body"`
	User      *UserDTO      `json:"user"`
	Issue     *IssueRefDTO  `json:"issue"`
	Parent    *CommentRefDTO `json:"parent"`
	CreatedAt string        `json:"createdAt"`
	UpdatedAt string        `json:"updatedAt"`
}

// --- Reference DTOs (nested objects) ---

// StateDTO represents workflow state
type StateDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LabelDTO represents an issue label
type LabelDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ProjectRefDTO is a minimal project reference
type ProjectRefDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CycleRefDTO is a minimal cycle reference
type CycleRefDTO struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Name   string `json:"name"`
}

// IssueRefDTO is a minimal issue reference
type IssueRefDTO struct {
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	State      string `json:"state"`
}

// AttachmentDTO represents an attachment
type AttachmentDTO struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	Subtitle   string `json:"subtitle,omitempty"`
	SourceType string `json:"sourceType"`
	CreatedAt  string `json:"createdAt"`
}

// CommentRefDTO is a minimal comment reference
type CommentRefDTO struct {
	ID string `json:"id"`
}

// --- Conversion helpers ---

// IssueToMinimalDTO converts an issue to minimal DTO
func IssueToMinimalDTO(issue *core.Issue) IssueMinimalDTO {
	return IssueMinimalDTO{
		Identifier: issue.Identifier,
		Title:      issue.Title,
		State:      issue.State.Name,
	}
}

// IssueToCompactDTO converts an issue to compact DTO
func IssueToCompactDTO(issue *core.Issue) IssueCompactDTO {
	dto := IssueCompactDTO{
		Identifier: issue.Identifier,
		Title:      issue.Title,
		State:      issue.State.Name,
		Priority:   issue.Priority,
		Estimate:   issue.Estimate,
		DueDate:    issue.DueDate,
		CreatedAt:  issue.CreatedAt,
		UpdatedAt:  issue.UpdatedAt,
	}

	if issue.Assignee != nil {
		name := issue.Assignee.Name
		dto.Assignee = &name
	}

	if issue.Delegate != nil {
		name := issue.Delegate.Name
		dto.Delegate = &name
	}

	if issue.Cycle != nil {
		dto.CycleNumber = &issue.Cycle.Number
	}

	if issue.Project != nil {
		name := issue.Project.Name
		dto.ProjectName = &name
	}

	return dto
}

// IssueToFullDTO converts an issue to full DTO
func IssueToFullDTO(issue *core.Issue) IssueFullDTO {
	dto := IssueFullDTO{
		Identifier:  issue.Identifier,
		Title:       issue.Title,
		Description: issue.Description,
		State: &StateDTO{
			ID:   issue.State.ID,
			Name: issue.State.Name,
		},
		Priority:  issue.Priority,
		Estimate:  issue.Estimate,
		DueDate:   issue.DueDate,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		URL:       issue.URL,
	}

	if issue.Assignee != nil {
		dto.Assignee = &UserDTO{
			ID:    issue.Assignee.ID,
			Name:  issue.Assignee.Name,
			Email: issue.Assignee.Email,
		}
	}

	if issue.Delegate != nil {
		dto.Delegate = &UserDTO{
			ID:    issue.Delegate.ID,
			Name:  issue.Delegate.Name,
			Email: issue.Delegate.Email,
		}
	}

	if issue.Creator != nil {
		dto.Creator = &UserDTO{
			ID:    issue.Creator.ID,
			Name:  issue.Creator.Name,
			Email: issue.Creator.Email,
		}
	}

	// Labels
	if issue.Labels != nil && len(issue.Labels.Nodes) > 0 {
		dto.Labels = make([]LabelDTO, len(issue.Labels.Nodes))
		for i, label := range issue.Labels.Nodes {
			dto.Labels[i] = LabelDTO{
				ID:   label.ID,
				Name: label.Name,
			}
		}
	}

	// Project
	if issue.Project != nil {
		dto.Project = &ProjectRefDTO{
			ID:   issue.Project.ID,
			Name: issue.Project.Name,
		}
	}

	// Cycle
	if issue.Cycle != nil {
		dto.Cycle = &CycleRefDTO{
			ID:     issue.Cycle.ID,
			Number: issue.Cycle.Number,
			Name:   issue.Cycle.Name,
		}
	}

	// Parent
	if issue.Parent != nil {
		dto.Parent = &IssueRefDTO{
			Identifier: issue.Parent.Identifier,
			Title:      issue.Parent.Title,
			State:      issue.Parent.State.Name,
		}
	}

	// Children
	if issue.Children.Nodes != nil && len(issue.Children.Nodes) > 0 {
		dto.Children = make([]IssueRefDTO, len(issue.Children.Nodes))
		for i, child := range issue.Children.Nodes {
			dto.Children[i] = IssueRefDTO{
				Identifier: child.Identifier,
				Title:      child.Title,
				State:      child.State.Name,
			}
		}
	}

	// Attachments
	if issue.Attachments != nil && len(issue.Attachments.Nodes) > 0 {
		dto.Attachments = make([]AttachmentDTO, len(issue.Attachments.Nodes))
		for i, att := range issue.Attachments.Nodes {
			dto.Attachments[i] = AttachmentToDTO(&att)
		}
	}

	// Comments
	if issue.Comments != nil && len(issue.Comments.Nodes) > 0 {
		dto.Comments = make([]CommentDTO, len(issue.Comments.Nodes))
		for i, comment := range issue.Comments.Nodes {
			dto.Comments[i] = CommentDTO{
				ID:   comment.ID,
				Body: comment.Body,
				User: &UserDTO{
					ID:   comment.User.ID,
					Name: comment.User.Name,
				},
				CreatedAt: comment.CreatedAt,
			}
		}
	}

	return dto
}

// CycleToMinimalDTO converts a cycle to minimal DTO
func CycleToMinimalDTO(cycle *core.Cycle) CycleMinimalDTO {
	return CycleMinimalDTO{
		Number:   cycle.Number,
		Status:   cycleStatus(cycle),
		StartsAt: cycle.StartsAt,
		EndsAt:   cycle.EndsAt,
	}
}

// CycleToCompactDTO converts a cycle to compact DTO
func CycleToCompactDTO(cycle *core.Cycle) CycleCompactDTO {
	dto := CycleCompactDTO{
		Number:   cycle.Number,
		Name:     cycle.Name,
		Status:   cycleStatus(cycle),
		StartsAt: cycle.StartsAt,
		EndsAt:   cycle.EndsAt,
		Progress: cycle.Progress,
	}

	if cycle.Team != nil {
		name := cycle.Team.Name
		dto.TeamName = &name
	}

	return dto
}

// CycleToFullDTO converts a cycle to full DTO
func CycleToFullDTO(cycle *core.Cycle) CycleFullDTO {
	dto := CycleFullDTO{
		Number:                     cycle.Number,
		Name:                       cycle.Name,
		Status:                     cycleStatus(cycle),
		StartsAt:                   cycle.StartsAt,
		EndsAt:                     cycle.EndsAt,
		Progress:                   cycle.Progress,
		Description:                cycle.Description,
		ScopeHistory:               cycle.ScopeHistory,
		CompletedScopeHistory:      cycle.CompletedScopeHistory,
		InProgressScopeHistory:     cycle.InProgressScopeHistory,
		IssueCountHistory:          cycle.IssueCountHistory,
		CompletedIssueCountHistory: cycle.CompletedIssueCountHistory,
		CreatedAt:                  cycle.CreatedAt,
		UpdatedAt:                  cycle.UpdatedAt,
	}

	if cycle.Team != nil {
		dto.Team = &TeamDTO{
			ID:   cycle.Team.ID,
			Key:  cycle.Team.Key,
			Name: cycle.Team.Name,
		}
	}

	return dto
}

// ProjectToDTO converts a project to DTO
func ProjectToDTO(project *core.Project) ProjectDTO {
	dto := ProjectDTO{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		State:       project.State,
		Content:     project.Content,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}

	// Convert issues
	issues := project.GetIssues()
	if len(issues) > 0 {
		dto.Issues = make([]IssueRefDTO, len(issues))
		for i, issue := range issues {
			dto.Issues[i] = IssueRefDTO{
				Identifier: issue.Identifier,
				Title:      issue.Title,
				State:      issue.State.Name,
			}
		}
	}

	return dto
}

// TeamToDTO converts a team to DTO
func TeamToDTO(team *core.Team) TeamDTO {
	dto := TeamDTO{
		ID:                  team.ID,
		Key:                 team.Key,
		Name:                team.Name,
		Description:         team.Description,
		IssueEstimationType: team.IssueEstimationType,
	}

	// Convert estimate scale
	scale := team.GetEstimateScale()
	if len(scale.Values) > 0 || len(scale.Labels) > 0 {
		dto.EstimateScale = &EstimateScale{
			Values: scale.Values,
			Labels: scale.Labels,
		}
	}

	return dto
}

// UserToDTO converts a user to DTO
func UserToDTO(user *core.User) UserDTO {
	dto := UserDTO{
		ID:          user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		Active:      user.Active,
		Admin:       user.Admin,
		CreatedAt:   user.CreatedAt,
	}

	// Convert teams
	if len(user.Teams) > 0 {
		dto.Teams = make([]TeamRef, len(user.Teams))
		for i, team := range user.Teams {
			dto.Teams[i] = TeamRef{
				Key:  team.Key,
				Name: team.Name,
			}
		}
	}

	return dto
}

// AttachmentToDTO converts an attachment to DTO
func AttachmentToDTO(att *core.Attachment) AttachmentDTO {
	return AttachmentDTO{
		ID:         att.ID,
		Title:      att.Title,
		URL:        att.URL,
		Subtitle:   att.Subtitle,
		SourceType: att.SourceType,
		CreatedAt:  att.CreatedAt,
	}
}

// CommentToDTO converts a comment to DTO
func CommentToDTO(comment *core.Comment) CommentDTO {
	dto := CommentDTO{
		ID:        comment.ID,
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}

	if comment.User.ID != "" {
		dto.User = &UserDTO{
			ID:   comment.User.ID,
			Name: comment.User.Name,
		}
	}

	if comment.Issue.Identifier != "" {
		dto.Issue = &IssueRefDTO{
			Identifier: comment.Issue.Identifier,
			Title:      comment.Issue.Title,
		}
	}

	if comment.Parent != nil {
		dto.Parent = &CommentRefDTO{
			ID: comment.Parent.ID,
		}
	}

	return dto
}
