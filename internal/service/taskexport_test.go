package service

import (
	"strings"
	"testing"

	"github.com/joa23/linear-cli/internal/linear/core"
)

func TestTaskExportService_buildActiveForm(t *testing.T) {
	service := &TaskExportService{}

	tests := []struct {
		input    string
		expected string
	}{
		{"Fix authentication bug", "Fixing authentication bug"},
		{"Add OAuth support", "Adding OAuth support"},
		{"Implement feature", "Implementing feature"},
		{"Update documentation", "Updating documentation"},
		{"Remove deprecated code", "Removing deprecated code"},
		{"Refactor service layer", "Refactoring service layer"},
		{"Test API endpoints", "Testing API endpoints"},
		{"Create new component", "Creating new component"},
		{"Delete old files", "Deleting old files"},
		{"Improve performance", "Improving performance"},
		{"Optimize database queries", "Optimizing database queries"},
		{"Debug connection issue", "Debugging connection issue"},
		{"Migrate to new API", "Migrating to new API"},
		{"Upgrade dependencies", "Upgrading dependencies"},
		{"Install new package", "Installing new package"},
		{"Configure CI/CD", "Configuring CI/CD"},
		{"Setup development environment", "Setting up development environment"},
		{"Write unit tests", "Writing unit tests"},
		{"Read configuration file", "Reading configuration file"},
		{"Parse JSON response", "Parsing JSON response"},
		{"Validate user input", "Validating user input"},
		{"Verify deployment", "Verifying deployment"},
		{"Check code coverage", "Checking code coverage"},
		{"Review pull request", "Reviewing pull request"},
		{"Investigate error logs", "Investigating error logs"},
		{"Analyze performance metrics", "Analyzing performance metrics"},
		{"Design new architecture", "Designing new architecture"},
		{"Plan sprint work", "Planning sprint work"},
		{"Research solutions", "Researching solutions"},
		{"Document API", "Documenting API"},
		{"Clean up codebase", "Cleaning up codebase"},
		{"Rebuild Docker image", "Rebuilding Docker image"},
		{"Deploy to production", "Deploying to production"},
		{"Release version 2.0", "Releasing version 2.0"},
		{"Merge feature branch", "Merging feature branch"},
		{"Rebase on main", "Rebasing on main"},
		{"Revert breaking changes", "Reverting breaking changes"},
		{"Restore from backup", "Restoring from backup"},
		{"Backup database", "Backing up database"},
		{"Archive old data", "Archiving old data"},
		{"", "Working on task"},
		{"Already Working", "Already Working"}, // No matching verb - capitalize first word
		{"Running tests", "Running tests"},      // Already in -ing form
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.buildActiveForm(tt.input)
			if result != tt.expected {
				t.Errorf("buildActiveForm(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTaskExportService_buildTaskDescription(t *testing.T) {
	service := &TaskExportService{}

	t.Run("complete issue with all fields", func(t *testing.T) {
		priority := 1
		estimate := 5.0
		dueDate := "2024-01-15"

		issue := &core.Issue{
			Identifier:  "TEST-123",
			Title:       "Test Issue",
			Description: "This is a test description\nwith multiple lines",
			State: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "state-1", Name: "In Progress"},
			Priority: &priority,
			Assignee: &core.User{Name: "John Doe"},
			Estimate: &estimate,
			DueDate:  &dueDate,
			URL:      "https://linear.app/test/issue/TEST-123",
		}

		desc := service.buildTaskDescription(issue)

		// Verify key components are present
		requiredParts := []string{
			"**Linear Issue:** TEST-123",
			"This is a test description",
			"**State:** In Progress",
			"**Priority:** 1",
			"**Assignee:** John Doe",
			"**Estimate:** 5",
			"**Due:** 2024-01-15",
			"**URL:** https://linear.app/test/issue/TEST-123",
		}

		for _, part := range requiredParts {
			if !strings.Contains(desc, part) {
				t.Errorf("Description missing required part: %s\nFull description:\n%s", part, desc)
			}
		}

		// Verify separator is present
		if !strings.Contains(desc, "---") {
			t.Error("Description should contain separator line")
		}
	})

	t.Run("minimal issue with only required fields", func(t *testing.T) {
		issue := &core.Issue{
			Identifier:  "TEST-456",
			Title:       "Minimal Issue",
			Description: "",
			State: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "state-1", Name: "Todo"},
			URL: "https://linear.app/test/issue/TEST-456",
		}

		desc := service.buildTaskDescription(issue)

		// Should have identifier and state at minimum
		if !strings.Contains(desc, "**Linear Issue:** TEST-456") {
			t.Error("Description should contain Linear issue identifier")
		}
		if !strings.Contains(desc, "**State:** Todo") {
			t.Error("Description should contain state")
		}

		// Should NOT have optional fields
		if strings.Contains(desc, "**Priority:**") {
			t.Error("Description should not contain priority when nil")
		}
		if strings.Contains(desc, "**Assignee:**") {
			t.Error("Description should not contain assignee when nil")
		}
		if strings.Contains(desc, "**Estimate:**") {
			t.Error("Description should not contain estimate when nil")
		}
		if strings.Contains(desc, "**Due:**") {
			t.Error("Description should not contain due date when nil")
		}
	})

	t.Run("issue with only some optional fields", func(t *testing.T) {
		priority := 2

		issue := &core.Issue{
			Identifier:  "TEST-789",
			Title:       "Partial Issue",
			Description: "Some description",
			State: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "state-1", Name: "Done"},
			Priority: &priority,
			// No assignee, estimate, or due date
			URL: "https://linear.app/test/issue/TEST-789",
		}

		desc := service.buildTaskDescription(issue)

		// Should have priority
		if !strings.Contains(desc, "**Priority:** 2") {
			t.Error("Description should contain priority")
		}

		// Should NOT have other optional fields
		if strings.Contains(desc, "**Assignee:**") {
			t.Error("Description should not contain assignee when nil")
		}
		if strings.Contains(desc, "**Estimate:**") {
			t.Error("Description should not contain estimate when nil")
		}
		if strings.Contains(desc, "**Due:**") {
			t.Error("Description should not contain due date when nil")
		}
	})
}

func TestTaskExportService_buildTaskDescription_EmptyDueDate(t *testing.T) {
	service := &TaskExportService{}

	emptyDueDate := ""
	issue := &core.Issue{
		Identifier:  "TEST-999",
		Title:       "Issue with empty due date",
		Description: "Test",
		State: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{ID: "state-1", Name: "Todo"},
		DueDate: &emptyDueDate, // Empty string pointer
		URL:     "https://linear.app/test/issue/TEST-999",
	}

	desc := service.buildTaskDescription(issue)

	// Should NOT include due date when it's an empty string
	if strings.Contains(desc, "**Due:**") {
		t.Error("Description should not contain due date when it's an empty string")
	}
}

func TestTaskExportService_convertToTasks_BottomUpHierarchy(t *testing.T) {
	service := &TaskExportService{}

	// Create a simple graph: parent with 2 children
	nodes := map[string]*issueNode{
		"PARENT-1": {
			issue: &core.Issue{
				Identifier: "PARENT-1",
				Title:      "Parent task",
				State: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{ID: "state-1", Name: "Todo"},
			},
			children:     []string{"CHILD-1", "CHILD-2"},
			dependencies: []string{},
		},
		"CHILD-1": {
			issue: &core.Issue{
				Identifier: "CHILD-1",
				Title:      "Child task 1",
				State: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{ID: "state-1", Name: "Todo"},
			},
			children:     []string{},
			dependencies: []string{},
		},
		"CHILD-2": {
			issue: &core.Issue{
				Identifier: "CHILD-2",
				Title:      "Child task 2",
				State: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{ID: "state-1", Name: "Todo"},
			},
			children:     []string{},
			dependencies: []string{},
		},
	}

	tasks := service.convertToTasks(nodes)

	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// Find parent task
	var parentTask *struct {
		ID        string
		BlockedBy []string
	}
	for i := range tasks {
		if tasks[i].ID == "PARENT-1" {
			parentTask = &struct {
				ID        string
				BlockedBy []string
			}{
				ID:        tasks[i].ID,
				BlockedBy: tasks[i].BlockedBy,
			}
			break
		}
	}

	if parentTask == nil {
		t.Fatal("Parent task not found")
	}

	// Verify parent is blocked by both children (bottom-up hierarchy)
	if len(parentTask.BlockedBy) != 2 {
		t.Errorf("Expected parent to be blocked by 2 children, got %d", len(parentTask.BlockedBy))
	}

	hasChild1 := false
	hasChild2 := false
	for _, blocker := range parentTask.BlockedBy {
		if blocker == "CHILD-1" {
			hasChild1 = true
		}
		if blocker == "CHILD-2" {
			hasChild2 = true
		}
	}

	if !hasChild1 || !hasChild2 {
		t.Errorf("Expected parent to be blocked by CHILD-1 and CHILD-2, got %v", parentTask.BlockedBy)
	}
}
