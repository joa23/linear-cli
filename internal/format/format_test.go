package format

import (
	"strings"
	"testing"

	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/joa23/linear-cli/internal/linear/cycles"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Format
		wantErr bool
	}{
		{"empty returns compact", "", Compact, false},
		{"minimal", "minimal", Minimal, false},
		{"compact", "compact", Compact, false},
		{"detailed", "detailed", Detailed, false},
		{"full", "full", Full, false},
		{"invalid", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatter_Issue(t *testing.T) {
	f := New()

	priority := 2
	estimate := 3.0
	dueDate := "2025-01-20"

	issue := &core.Issue{
		ID:         "uuid-123",
		Identifier: "CEN-123",
		Title:      "Add login functionality",
		State: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{Name: "In Progress"},
		Assignee: &core.User{Name: "Stefan", Email: "stefan@test.com"},
		Priority: &priority,
		Estimate: &estimate,
		DueDate:  &dueDate,
		Project:  &core.Project{Name: "Auth System"},
		Cycle:    &core.CycleReference{Number: 67, Name: "Cycle 67"},
	}

	t.Run("minimal format", func(t *testing.T) {
		result := f.Issue(issue, Minimal)
		if !strings.Contains(result, "CEN-123") {
			t.Error("minimal format should contain identifier")
		}
		if !strings.Contains(result, "In Progress") {
			t.Error("minimal format should contain state")
		}
		if !strings.Contains(result, "Add login") {
			t.Error("minimal format should contain title")
		}
	})

	t.Run("compact format", func(t *testing.T) {
		result := f.Issue(issue, Compact)
		if !strings.Contains(result, "CEN-123") {
			t.Error("compact format should contain identifier")
		}
		if !strings.Contains(result, "@Stefan") {
			t.Error("compact format should contain assignee")
		}
		if !strings.Contains(result, "P2:High") {
			t.Error("compact format should contain priority")
		}
		if !strings.Contains(result, "Est:3") {
			t.Error("compact format should contain estimate")
		}
		if !strings.Contains(result, "Cycle:67") {
			t.Error("compact format should contain cycle")
		}
		if !strings.Contains(result, "Project: Auth System") {
			t.Error("compact format should contain project")
		}
	})

	t.Run("full format", func(t *testing.T) {
		result := f.Issue(issue, Full)
		if !strings.Contains(result, "CEN-123: Add login functionality") {
			t.Error("full format should contain identifier and title")
		}
		if !strings.Contains(result, "Status: In Progress") {
			t.Error("full format should contain status")
		}
		if !strings.Contains(result, "stefan@test.com") {
			t.Error("full format should contain email")
		}
	})

	t.Run("nil issue", func(t *testing.T) {
		result := f.Issue(nil, Full)
		if result != "" {
			t.Error("nil issue should return empty string")
		}
	})
}

func TestFormatter_IssueList(t *testing.T) {
	f := New()

	issues := []core.Issue{
		{
			Identifier: "CEN-1",
			Title:      "First issue",
			State:      struct {ID string `json:"id"`; Name string `json:"name"`}{Name: "Todo"},
		},
		{
			Identifier: "CEN-2",
			Title:      "Second issue",
			State:      struct {ID string `json:"id"`; Name string `json:"name"`}{Name: "Done"},
		},
	}

	t.Run("list with issues", func(t *testing.T) {
		result := f.IssueList(issues, Minimal, nil)
		if !strings.Contains(result, "ISSUES (2)") {
			t.Error("should contain count header")
		}
		if !strings.Contains(result, "CEN-1") {
			t.Error("should contain first issue")
		}
		if !strings.Contains(result, "CEN-2") {
			t.Error("should contain second issue")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		result := f.IssueList([]core.Issue{}, Minimal, nil)
		if result != "No issues found." {
			t.Errorf("expected 'No issues found.', got '%s'", result)
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		page := &Pagination{
			Start:       10,
			Limit:       10,
			Count:       2,
			HasNextPage: true,
		}
		result := f.IssueList(issues, Minimal, page)
		if !strings.Contains(result, "--start 20 --limit 10") {
			t.Error("should contain offset-based pagination")
		}
		if !strings.Contains(result, "More results available") {
			t.Error("should contain pagination message")
		}
		if !strings.Contains(result, "ISSUES (11-12+)") {
			t.Error("should contain pagination range with +")
		}
	})
}

func TestFormatter_Cycle(t *testing.T) {
	f := New()

	cycle := &core.Cycle{
		ID:       "uuid-cycle",
		Name:     "Sprint 67",
		Number:   67,
		StartsAt: "2025-01-15T00:00:00Z",
		EndsAt:   "2025-01-28T00:00:00Z",
		Progress: 0.45,
		IsActive: true,
		Team:     &core.Team{Name: "Engineering", Key: "ENG"},
	}

	t.Run("minimal format", func(t *testing.T) {
		result := f.Cycle(cycle, Minimal)
		if !strings.Contains(result, "Cycle 67") {
			t.Error("minimal format should contain cycle number")
		}
		if !strings.Contains(result, "Active") {
			t.Error("minimal format should contain status")
		}
	})

	t.Run("compact format", func(t *testing.T) {
		result := f.Cycle(cycle, Compact)
		if !strings.Contains(result, "Sprint 67") {
			t.Error("compact format should contain name")
		}
		if !strings.Contains(result, "45%") {
			t.Error("compact format should contain progress")
		}
		if !strings.Contains(result, "Team: Engineering") {
			t.Error("compact format should contain team")
		}
	})

	t.Run("full format", func(t *testing.T) {
		result := f.Cycle(cycle, Full)
		if !strings.Contains(result, "Cycle 67: Sprint 67") {
			t.Error("full format should contain number and name")
		}
		if !strings.Contains(result, "Progress: 45.0%") {
			t.Error("full format should contain progress")
		}
	})
}

func TestFormatter_Project(t *testing.T) {
	f := New()

	project := &core.Project{
		Name:        "Auth System",
		Description: "Authentication and authorization",
		State:       "started",
		CreatedAt:   "2025-01-01T00:00:00Z",
		UpdatedAt:   "2025-01-15T00:00:00Z",
	}

	t.Run("project format", func(t *testing.T) {
		result := f.Project(project)
		if !strings.Contains(result, "Auth System") {
			t.Error("should contain project name")
		}
		if !strings.Contains(result, "started") {
			t.Error("should contain state")
		}
		if !strings.Contains(result, "Authentication") {
			t.Error("should contain description")
		}
	})

	t.Run("nil project", func(t *testing.T) {
		result := f.Project(nil)
		if result != "" {
			t.Error("nil project should return empty string")
		}
	})
}

func TestFormatter_Team(t *testing.T) {
	f := New()

	team := &core.Team{
		Name:                "Engineering",
		Key:                 "ENG",
		Description:         "Core engineering team",
		IssueEstimationType: "fibonacci",
	}

	t.Run("team format", func(t *testing.T) {
		result := f.Team(team)
		if !strings.Contains(result, "Engineering (ENG)") {
			t.Error("should contain team name and key")
		}
		if !strings.Contains(result, "Core engineering team") {
			t.Error("should contain description")
		}
		if !strings.Contains(result, "fibonacci") {
			t.Error("should contain estimation type")
		}
	})
}

func TestFormatter_User(t *testing.T) {
	f := New()

	user := &core.User{
		Name:        "Stefan",
		DisplayName: "Stefan M",
		Email:       "stefan@test.com",
		Active:      true,
		Admin:       true,
		CreatedAt:   "2024-01-01T00:00:00Z",
		Teams: []core.Team{
			{Name: "Engineering", Key: "ENG"},
		},
	}

	t.Run("user format", func(t *testing.T) {
		result := f.User(user)
		if !strings.Contains(result, "Stefan M") {
			t.Error("should prefer display name")
		}
		if !strings.Contains(result, "stefan@test.com") {
			t.Error("should contain email")
		}
		if !strings.Contains(result, "Admin") {
			t.Error("should show admin status")
		}
		if !strings.Contains(result, "Engineering") {
			t.Error("should contain team")
		}
	})
}

func TestFormatter_Comment(t *testing.T) {
	f := New()

	comment := &core.Comment{
		ID:        "comment-1",
		Body:      "This is a comment body",
		CreatedAt: "2025-01-15T10:30:00Z",
		User:      core.User{Name: "Stefan"},
		Issue: core.CommentIssue{
			Identifier: "CEN-123",
			Title:      "Test issue",
		},
	}

	t.Run("comment format", func(t *testing.T) {
		result := f.Comment(comment)
		if !strings.Contains(result, "@Stefan") {
			t.Error("should contain author")
		}
		if !strings.Contains(result, "This is a comment body") {
			t.Error("should contain body")
		}
		if !strings.Contains(result, "CEN-123") {
			t.Error("should contain issue reference")
		}
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("line", func(t *testing.T) {
		result := line(5)
		if result != "─────" {
			t.Errorf("expected 5 dashes, got '%s'", result)
		}
	})

	t.Run("formatDate", func(t *testing.T) {
		result := formatDate("2025-01-15T10:30:00Z")
		if result != "2025-01-15" {
			t.Errorf("expected '2025-01-15', got '%s'", result)
		}
	})

	t.Run("formatDateTime", func(t *testing.T) {
		result := formatDateTime("2025-01-15T10:30:00Z")
		if result != "2025-01-15 10:30" {
			t.Errorf("expected '2025-01-15 10:30', got '%s'", result)
		}
	})

	t.Run("truncate", func(t *testing.T) {
		result := truncate("Hello World", 5)
		if result != "He..." {
			t.Errorf("expected 'He...', got '%s'", result)
		}

		result = truncate("Hi", 10)
		if result != "Hi" {
			t.Errorf("expected 'Hi', got '%s'", result)
		}
	})

	t.Run("priorityLabel", func(t *testing.T) {
		tests := []struct {
			priority *int
			want     string
		}{
			{nil, ""},
			{intPtr(0), ""},
			{intPtr(1), "P1:Urgent"},
			{intPtr(2), "P2:High"},
			{intPtr(3), "P3:Medium"},
			{intPtr(4), "P4:Low"},
		}

		for _, tt := range tests {
			got := priorityLabel(tt.priority)
			if got != tt.want {
				t.Errorf("priorityLabel(%v) = %v, want %v", tt.priority, got, tt.want)
			}
		}
	})

	t.Run("cleanDescription", func(t *testing.T) {
		input := "## Header\n\nSome content\n\n---\n\n```code```\n\nMore text"
		result := cleanDescription(input)
		if strings.Contains(result, "##") {
			t.Error("should remove markdown headers")
		}
		if strings.Contains(result, "---") {
			t.Error("should remove horizontal rules")
		}
		if !strings.Contains(result, "Some content") {
			t.Error("should keep content")
		}
	})
}

func TestFormatter_CycleAnalysis(t *testing.T) {
	f := New()

	analysis := &cycles.CycleAnalysis{
		CycleCount:          10,
		AvgVelocity:         25.5,
		AvgCompletionRate:   85.0,
		AvgScopeCreepPercent: 12.0,
		AvgThroughput:       8.5,
		StdDevVelocity:      5.2,
		MedianVelocity:      24.0,
		P80Velocity:         22.0,
		P20Velocity:         28.0,
	}

	t.Run("team analysis", func(t *testing.T) {
		result := f.CycleAnalysis(analysis, "Engineering", "", true)
		if !strings.Contains(result, "Engineering (Team)") {
			t.Error("should contain team name")
		}
		if !strings.Contains(result, "Cycles Analyzed: 10") {
			t.Error("should contain cycle count")
		}
		if !strings.Contains(result, "Avg Velocity: 25.5") {
			t.Error("should contain velocity")
		}
		if !strings.Contains(result, "RECOMMENDATION") {
			t.Error("should contain recommendation section")
		}
	})

	t.Run("per-user analysis", func(t *testing.T) {
		result := f.CycleAnalysis(analysis, "Engineering", "Stefan", true)
		if !strings.Contains(result, "Stefan (Engineering)") {
			t.Error("should contain user and team name")
		}
	})

	t.Run("no recommendation", func(t *testing.T) {
		result := f.CycleAnalysis(analysis, "Engineering", "", false)
		if strings.Contains(result, "RECOMMENDATION") {
			t.Error("should not contain recommendation section")
		}
	})
}

func TestFormatter_CycleList(t *testing.T) {
	f := New()

	cycles := []core.Cycle{
		{
			Number:   67,
			Name:     "Sprint 67",
			StartsAt: "2025-01-15T00:00:00Z",
			EndsAt:   "2025-01-28T00:00:00Z",
			IsActive: true,
		},
		{
			Number:   68,
			Name:     "Sprint 68",
			StartsAt: "2025-01-29T00:00:00Z",
			EndsAt:   "2025-02-11T00:00:00Z",
			IsFuture: true,
		},
	}

	t.Run("list with cycles", func(t *testing.T) {
		result := f.CycleList(cycles, Compact, nil)
		if !strings.Contains(result, "CYCLES (2)") {
			t.Error("should contain count header")
		}
		if !strings.Contains(result, "Cycle 67") {
			t.Error("should contain first cycle")
		}
		if !strings.Contains(result, "Cycle 68") {
			t.Error("should contain second cycle")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		result := f.CycleList([]core.Cycle{}, Compact, nil)
		if result != "No cycles found." {
			t.Errorf("expected 'No cycles found.', got '%s'", result)
		}
	})
}

func TestFormatter_ProjectList(t *testing.T) {
	f := New()

	projects := []core.Project{
		{Name: "Project A", State: "started"},
		{Name: "Project B", State: "completed"},
	}

	t.Run("list with projects", func(t *testing.T) {
		result := f.ProjectList(projects, nil)
		if !strings.Contains(result, "PROJECTS (2)") {
			t.Error("should contain count header")
		}
		if !strings.Contains(result, "Project A") {
			t.Error("should contain first project")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		result := f.ProjectList([]core.Project{}, nil)
		if result != "No projects found." {
			t.Errorf("expected 'No projects found.', got '%s'", result)
		}
	})
}

func TestFormatter_TeamList(t *testing.T) {
	f := New()

	teams := []core.Team{
		{Name: "Engineering", Key: "ENG"},
		{Name: "Product", Key: "PROD"},
	}

	t.Run("list with teams", func(t *testing.T) {
		result := f.TeamList(teams, nil)
		if !strings.Contains(result, "TEAMS (2)") {
			t.Error("should contain count header")
		}
		if !strings.Contains(result, "Engineering (ENG)") {
			t.Error("should contain first team")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		result := f.TeamList([]core.Team{}, nil)
		if result != "No teams found." {
			t.Errorf("expected 'No teams found.', got '%s'", result)
		}
	})
}

func TestFormatter_UserList(t *testing.T) {
	f := New()

	users := []core.User{
		{Name: "Stefan", Email: "stefan@test.com", Active: true},
		{Name: "Maria", Email: "maria@test.com", Active: true, Admin: true},
	}

	t.Run("list with users", func(t *testing.T) {
		result := f.UserList(users, nil)
		if !strings.Contains(result, "USERS (2)") {
			t.Error("should contain count header")
		}
		if !strings.Contains(result, "Stefan") {
			t.Error("should contain first user")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		result := f.UserList([]core.User{}, nil)
		if result != "No users found." {
			t.Errorf("expected 'No users found.', got '%s'", result)
		}
	})
}

func TestFormatter_CommentList(t *testing.T) {
	f := New()

	comments := []core.Comment{
		{
			ID:        "c1",
			Body:      "First comment",
			CreatedAt: "2025-01-15T10:00:00Z",
			User:      core.User{Name: "Stefan"},
		},
	}

	t.Run("list with comments", func(t *testing.T) {
		result := f.CommentList(comments, nil)
		if !strings.Contains(result, "COMMENTS (1)") {
			t.Error("should contain count header")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		result := f.CommentList([]core.Comment{}, nil)
		if result != "No comments found." {
			t.Errorf("expected 'No comments found.', got '%s'", result)
		}
	})
}

func TestFormatter_Issue_Full_WithAllFields(t *testing.T) {
	f := New()

	priority := 1
	estimate := 5.0
	dueDate := "2025-01-30"

	issue := &core.Issue{
		ID:          "uuid-123",
		Identifier:  "CEN-999",
		Title:       "Complete Feature",
		Description: "## Overview\n\nThis is a detailed description.",
		URL:         "https://linear.app/test/issue/CEN-999",
		CreatedAt:   "2025-01-10T10:00:00Z",
		UpdatedAt:   "2025-01-15T15:30:00Z",
		State:       struct{ID string `json:"id"`; Name string `json:"name"`}{Name: "In Progress"},
		Assignee:    &core.User{Name: "Stefan", Email: "stefan@test.com"},
		Priority:    &priority,
		Estimate:    &estimate,
		DueDate:     &dueDate,
		Project:     &core.Project{Name: "Main Project"},
		Cycle:       &core.CycleReference{Number: 67, Name: "Sprint 67"},
		Labels:      &core.LabelConnection{Nodes: []core.Label{{Name: "urgent"}, {Name: "frontend"}}},
		Parent:      &core.ParentIssue{Identifier: "CEN-100", Title: "Epic"},
		Children:    core.ChildrenNodes{Nodes: []core.SubIssue{{Identifier: "CEN-1000", Title: "Subtask", State: struct{ID string `json:"id"`; Name string `json:"name"`}{Name: "Todo"}}}},
	}

	result := f.Issue(issue, Full)

	// Check all major sections are present
	if !strings.Contains(result, "CEN-999: Complete Feature") {
		t.Error("should contain header")
	}
	if !strings.Contains(result, "Status: In Progress") {
		t.Error("should contain status")
	}
	if !strings.Contains(result, "P1:Urgent") {
		t.Error("should contain priority")
	}
	if !strings.Contains(result, "5 points") {
		t.Error("should contain estimate")
	}
	if !strings.Contains(result, "2025-01-30") {
		t.Error("should contain due date")
	}
	if !strings.Contains(result, "DESCRIPTION") {
		t.Error("should contain description section")
	}
	if !strings.Contains(result, "Parent: CEN-100") {
		t.Error("should contain parent")
	}
	if !strings.Contains(result, "SUB-ISSUES") {
		t.Error("should contain sub-issues section")
	}
	if !strings.Contains(result, "urgent, frontend") {
		t.Error("should contain labels")
	}
}

func TestFormatter_Issue_Detailed_CommentsHint(t *testing.T) {
	f := New()

	longBody := strings.Repeat("x", 300)
	issue := &core.Issue{
		ID:         "uuid-hint",
		Identifier: "CEN-456",
		Title:      "Issue with comments",
		State:      struct{ ID string `json:"id"`; Name string `json:"name"` }{Name: "Todo"},
		CreatedAt:  "2025-01-10T10:00:00Z",
		UpdatedAt:  "2025-01-15T15:30:00Z",
		Comments: &core.CommentConnection{
			Nodes: []core.Comment{
				{
					ID:        "c1",
					Body:      longBody,
					CreatedAt: "2025-01-15T10:00:00Z",
					User:      core.User{Name: "Alice"},
				},
			},
		},
	}

	t.Run("old formatter detailed includes hint", func(t *testing.T) {
		result := f.Issue(issue, Detailed)
		if !strings.Contains(result, "COMMENTS (1)") {
			t.Error("should contain comment count")
		}
		if !strings.Contains(result, "linear issues comments CEN-456") {
			t.Error("should contain hint with issue identifier")
		}
	})

	t.Run("old formatter detailed truncates comment body", func(t *testing.T) {
		result := f.Issue(issue, Detailed)
		if strings.Contains(result, longBody) {
			t.Error("detailed format should truncate long comment bodies")
		}
		if !strings.Contains(result, "...") {
			t.Error("truncated comment should end with ellipsis")
		}
	})

	t.Run("old formatter full does NOT truncate", func(t *testing.T) {
		result := f.Issue(issue, Full)
		if !strings.Contains(result, longBody) {
			t.Error("full format should show untruncated comment bodies")
		}
		if strings.Contains(result, "linear issues comments") {
			t.Error("full format should not contain hint")
		}
	})

	t.Run("new renderer detailed includes hint", func(t *testing.T) {
		result := f.RenderIssue(issue, VerbosityDetailed, OutputText)
		if !strings.Contains(result, "linear issues comments CEN-456") {
			t.Error("text renderer detailed should contain hint")
		}
	})

	t.Run("new renderer detailed truncates comment body", func(t *testing.T) {
		result := f.RenderIssue(issue, VerbosityDetailed, OutputText)
		if strings.Contains(result, longBody) {
			t.Error("text renderer detailed should truncate long comment bodies")
		}
	})

	t.Run("new renderer full does NOT truncate", func(t *testing.T) {
		result := f.RenderIssue(issue, VerbosityFull, OutputText)
		if !strings.Contains(result, longBody) {
			t.Error("text renderer full should show untruncated comment bodies")
		}
		if strings.Contains(result, "linear issues comments") {
			t.Error("text renderer full should not contain hint")
		}
	})

	t.Run("JSON detailed truncates comment body", func(t *testing.T) {
		result := f.RenderIssue(issue, VerbosityDetailed, OutputJSON)
		if strings.Contains(result, longBody) {
			t.Error("JSON detailed should truncate comment bodies")
		}
		if !strings.Contains(result, "...") {
			t.Error("JSON detailed truncated comment should end with ellipsis")
		}
	})

	t.Run("JSON full does NOT truncate comment body", func(t *testing.T) {
		result := f.RenderIssue(issue, VerbosityFull, OutputJSON)
		if !strings.Contains(result, longBody) {
			t.Error("JSON full should show untruncated comment bodies")
		}
	})
}

func TestFormatter_Cycle_Full(t *testing.T) {
	f := New()

	cycle := &core.Cycle{
		ID:          "cycle-uuid",
		Name:        "Sprint 67",
		Number:      67,
		Description: "Main sprint for Q1",
		StartsAt:    "2025-01-15T00:00:00Z",
		EndsAt:      "2025-01-28T00:00:00Z",
		CreatedAt:   "2025-01-01T00:00:00Z",
		UpdatedAt:   "2025-01-14T00:00:00Z",
		Progress:    0.75,
		IsActive:    true,
		Team:        &core.Team{Name: "Engineering", Key: "ENG"},
		ScopeHistory:          []int{50, 52, 55},
		CompletedScopeHistory: []int{10, 25, 40},
		IssueCountHistory:     []int{20, 22, 24},
	}

	result := f.Cycle(cycle, Full)

	if !strings.Contains(result, "Cycle 67: Sprint 67") {
		t.Error("should contain header")
	}
	if !strings.Contains(result, "Status: Active") {
		t.Error("should contain status")
	}
	if !strings.Contains(result, "Progress: 75.0%") {
		t.Error("should contain progress")
	}
	if !strings.Contains(result, "Team: Engineering") {
		t.Error("should contain team")
	}
	if !strings.Contains(result, "Main sprint for Q1") {
		t.Error("should contain description")
	}
	if !strings.Contains(result, "METRICS") {
		t.Error("should contain metrics section")
	}
}

func TestFormatter_NilInputs(t *testing.T) {
	f := New()

	t.Run("nil cycle", func(t *testing.T) {
		result := f.Cycle(nil, Full)
		if result != "" {
			t.Error("nil cycle should return empty string")
		}
	})

	t.Run("nil team", func(t *testing.T) {
		result := f.Team(nil)
		if result != "" {
			t.Error("nil team should return empty string")
		}
	})

	t.Run("nil user", func(t *testing.T) {
		result := f.User(nil)
		if result != "" {
			t.Error("nil user should return empty string")
		}
	})

	t.Run("nil comment", func(t *testing.T) {
		result := f.Comment(nil)
		if result != "" {
			t.Error("nil comment should return empty string")
		}
	})

	t.Run("nil cycle analysis", func(t *testing.T) {
		result := f.CycleAnalysis(nil, "Team", "", false)
		if result != "No cycle analysis data available." {
			t.Error("nil analysis should return message")
		}
	})
}

// Helper function
func intPtr(i int) *int {
	return &i
}

// --- Tests for new renderer architecture ---

func TestParseVerbosity(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Verbosity
		wantErr bool
	}{
		{"empty returns compact", "", VerbosityCompact, false},
		{"minimal", "minimal", VerbosityMinimal, false},
		{"compact", "compact", VerbosityCompact, false},
		{"full", "full", VerbosityFull, false},
		{"min alias", "min", VerbosityMinimal, false},
		{"default alias", "default", VerbosityCompact, false},
		{"detailed", "detailed", VerbosityDetailed, false},
		{"invalid", "invalid", VerbosityCompact, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVerbosity(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVerbosity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseVerbosity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseOutputType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    OutputType
		wantErr bool
	}{
		{"empty returns text", "", OutputText, false},
		{"text", "text", OutputText, false},
		{"json", "json", OutputJSON, false},
		{"ascii alias", "ascii", OutputText, false},
		{"txt alias", "txt", OutputText, false},
		{"invalid", "xml", OutputText, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOutputType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOutputType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseOutputType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerbosityConversion(t *testing.T) {
	t.Run("FormatToVerbosity", func(t *testing.T) {
		if FormatToVerbosity(Minimal) != VerbosityMinimal {
			t.Error("Minimal should convert to VerbosityMinimal")
		}
		if FormatToVerbosity(Compact) != VerbosityCompact {
			t.Error("Compact should convert to VerbosityCompact")
		}
		if FormatToVerbosity(Detailed) != VerbosityDetailed {
			t.Error("Detailed should convert to VerbosityDetailed")
		}
		if FormatToVerbosity(Full) != VerbosityFull {
			t.Error("Full should convert to VerbosityFull")
		}
	})

	t.Run("VerbosityToFormat", func(t *testing.T) {
		if VerbosityToFormat(VerbosityMinimal) != Minimal {
			t.Error("VerbosityMinimal should convert to Minimal")
		}
		if VerbosityToFormat(VerbosityCompact) != Compact {
			t.Error("VerbosityCompact should convert to Compact")
		}
		if VerbosityToFormat(VerbosityDetailed) != Detailed {
			t.Error("VerbosityDetailed should convert to Detailed")
		}
		if VerbosityToFormat(VerbosityFull) != Full {
			t.Error("VerbosityFull should convert to Full")
		}
	})
}

func TestRenderer_JSONOutput(t *testing.T) {
	f := New()

	priority := 2
	estimate := 3.0
	dueDate := "2025-01-20"

	issue := &core.Issue{
		ID:         "uuid-123",
		Identifier: "CEN-123",
		Title:      "Add login functionality",
		State: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{Name: "In Progress"},
		Assignee:  &core.User{Name: "Stefan", Email: "stefan@test.com"},
		Priority:  &priority,
		Estimate:  &estimate,
		DueDate:   &dueDate,
		Project:   &core.Project{Name: "Auth System"},
		Cycle:     &core.CycleReference{Number: 67, Name: "Cycle 67"},
		CreatedAt: "2025-01-10T10:00:00Z",
		UpdatedAt: "2025-01-15T15:30:00Z",
	}

	t.Run("JSON minimal", func(t *testing.T) {
		result := f.RenderIssue(issue, VerbosityMinimal, OutputJSON)
		if !strings.Contains(result, `"identifier": "CEN-123"`) {
			t.Error("JSON should contain identifier")
		}
		if !strings.Contains(result, `"state": "In Progress"`) {
			t.Error("JSON should contain state")
		}
		if !strings.Contains(result, `"title": "Add login functionality"`) {
			t.Error("JSON should contain title")
		}
	})

	t.Run("JSON compact", func(t *testing.T) {
		result := f.RenderIssue(issue, VerbosityCompact, OutputJSON)
		if !strings.Contains(result, `"identifier": "CEN-123"`) {
			t.Error("JSON should contain identifier")
		}
		if !strings.Contains(result, `"assignee": "Stefan"`) {
			t.Error("JSON should contain assignee")
		}
		if !strings.Contains(result, `"priority": 2`) {
			t.Error("JSON should contain priority")
		}
		if !strings.Contains(result, `"estimate": 3`) {
			t.Error("JSON should contain estimate")
		}
		if !strings.Contains(result, `"cycleNumber": 67`) {
			t.Error("JSON should contain cycle number")
		}
	})

	t.Run("JSON full", func(t *testing.T) {
		result := f.RenderIssue(issue, VerbosityFull, OutputJSON)
		if !strings.Contains(result, `"identifier": "CEN-123"`) {
			t.Error("JSON should contain identifier")
		}
		if !strings.Contains(result, `"email": "stefan@test.com"`) {
			t.Error("JSON should contain assignee email")
		}
		if !strings.Contains(result, `"url"`) {
			t.Error("JSON should contain URL field")
		}
	})
}

func TestRenderer_JSONList(t *testing.T) {
	f := New()

	issues := []core.Issue{
		{
			Identifier: "CEN-1",
			Title:      "First issue",
			State:      struct {ID string `json:"id"`; Name string `json:"name"`}{Name: "Todo"},
			CreatedAt:  "2025-01-10T10:00:00Z",
			UpdatedAt:  "2025-01-15T15:30:00Z",
		},
		{
			Identifier: "CEN-2",
			Title:      "Second issue",
			State:      struct {ID string `json:"id"`; Name string `json:"name"`}{Name: "Done"},
			CreatedAt:  "2025-01-11T10:00:00Z",
			UpdatedAt:  "2025-01-16T15:30:00Z",
		},
	}

	t.Run("JSON list output", func(t *testing.T) {
		result := f.RenderIssueList(issues, VerbosityMinimal, OutputJSON, nil)

		// Should be a JSON array
		if !strings.HasPrefix(result, "[") || !strings.HasSuffix(strings.TrimSpace(result), "]") {
			t.Error("JSON list should be an array")
		}
		if !strings.Contains(result, "CEN-1") {
			t.Error("should contain first issue")
		}
		if !strings.Contains(result, "CEN-2") {
			t.Error("should contain second issue")
		}
	})

	t.Run("empty JSON list", func(t *testing.T) {
		result := f.RenderIssueList([]core.Issue{}, VerbosityMinimal, OutputJSON, nil)
		if result != "[]" {
			t.Errorf("expected empty array '[]', got '%s'", result)
		}
	})
}

func TestRenderer_TextOutput(t *testing.T) {
	f := New()

	priority := 2
	issue := &core.Issue{
		Identifier: "CEN-123",
		Title:      "Test issue",
		State:      struct {ID string `json:"id"`; Name string `json:"name"`}{Name: "Todo"},
		Priority:   &priority,
	}

	t.Run("Text minimal", func(t *testing.T) {
		result := f.RenderIssue(issue, VerbosityMinimal, OutputText)
		if !strings.Contains(result, "CEN-123") {
			t.Error("Text should contain identifier")
		}
		if !strings.Contains(result, "[Todo]") {
			t.Error("Text should contain state in brackets")
		}
	})

	t.Run("Text compact", func(t *testing.T) {
		result := f.RenderIssue(issue, VerbosityCompact, OutputText)
		if !strings.Contains(result, "CEN-123") {
			t.Error("Text should contain identifier")
		}
		if !strings.Contains(result, "P2:High") {
			t.Error("Text should contain priority label")
		}
	})
}

func TestJSONRenderer_ErrorHandling(t *testing.T) {
	renderer := &JSONRenderer{}

	t.Run("nil issue", func(t *testing.T) {
		result := renderer.RenderIssue(nil, VerbosityCompact)
		if !strings.Contains(result, `"error"`) {
			t.Error("should return error JSON for nil issue")
		}
	})

	t.Run("nil cycle", func(t *testing.T) {
		result := renderer.RenderCycle(nil, VerbosityCompact)
		if !strings.Contains(result, `"error"`) {
			t.Error("should return error JSON for nil cycle")
		}
	})
}

func TestRendererFactory(t *testing.T) {
	factory := NewRendererFactory()

	t.Run("GetRenderer text", func(t *testing.T) {
		renderer := factory.GetRenderer(OutputText)
		if _, ok := renderer.(*TextRenderer); !ok {
			t.Error("should return TextRenderer for OutputText")
		}
	})

	t.Run("GetRenderer JSON", func(t *testing.T) {
		renderer := factory.GetRenderer(OutputJSON)
		if _, ok := renderer.(*JSONRenderer); !ok {
			t.Error("should return JSONRenderer for OutputJSON")
		}
	})

	t.Run("GetRenderer invalid defaults to text", func(t *testing.T) {
		renderer := factory.GetRenderer("invalid")
		if _, ok := renderer.(*TextRenderer); !ok {
			t.Error("should default to TextRenderer for invalid type")
		}
	})
}
