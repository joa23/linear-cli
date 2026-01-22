//go:build integration

package linear

import (
	"github.com/joa23/linear-cli/internal/token"
	"os"
	"testing"
	"time"
)

// TestReadOperationsIntegration tests all read operations in the Linear client
// This test requires valid Linear API credentials and will skip if not available
func TestReadOperationsIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}
	
	// Create client from environment token
	token := os.Getenv("LINEAR_TOKEN")
	if token == "" {
		token = os.Getenv("LINEAR_API_TOKEN")
	}
	if token == "" {
		t.Skip("No Linear API token found in environment")
	}
	
	client := NewClient(token)
	
	// Test connection first
	t.Run("TestConnection", func(t *testing.T) {
		err := client.TestConnection()
		if err != nil {
			t.Fatalf("Failed to connect to Linear API: %v", err)
		}
	})
	
	// Variables to store IDs for subsequent tests
	var (
		viewerID    string
		teamID      string
		issueID     string
		projectID   string
		commentID   string
	)
	
	// Test 1: Get authenticated user (viewer)
	t.Run("GetViewer", func(t *testing.T) {
		viewer, err := client.GetViewer()
		if err != nil {
			t.Fatalf("Failed to get viewer: %v", err)
		}
		
		if viewer.ID == "" {
			t.Error("Viewer ID is empty")
		}
		if viewer.Email == "" {
			t.Error("Viewer email is empty")
		}
		
		viewerID = viewer.ID
		t.Logf("Viewer: %s (%s)", viewer.Name, viewer.Email)
	})
	
	// Test 2: Get user ID (alternative method)
	t.Run("GetAppUserID", func(t *testing.T) {
		userID, err := client.GetAppUserID()
		if err != nil {
			t.Fatalf("Failed to get app user ID: %v", err)
		}
		
		if userID == "" {
			t.Error("User ID is empty")
		}
		
		// Should match viewer ID
		if userID != viewerID {
			t.Errorf("User ID mismatch: GetAppUserID=%s, GetViewer=%s", userID, viewerID)
		}
	})
	
	// Test 3: Get teams
	t.Run("GetTeams", func(t *testing.T) {
		teams, err := client.GetTeams()
		if err != nil {
			t.Fatalf("Failed to get teams: %v", err)
		}
		
		if len(teams) == 0 {
			t.Skip("No teams found in workspace")
		}
		
		// Use first team for subsequent tests
		teamID = teams[0].ID
		t.Logf("Found %d teams, using: %s", len(teams), teams[0].Name)
		
		// Verify team structure
		for _, team := range teams {
			if team.ID == "" {
				t.Error("Team ID is empty")
			}
			if team.Name == "" {
				t.Error("Team name is empty")
			}
			if team.Key == "" {
				t.Error("Team key is empty")
			}
		}
	})
	
	// Test 4: Get workflow states
	t.Run("GetWorkflowStates", func(t *testing.T) {
		// Test without team filter
		states, err := client.GetWorkflowStates("")
		if err != nil {
			t.Fatalf("Failed to get workflow states: %v", err)
		}
		
		if len(states) == 0 {
			t.Error("No workflow states found")
		}
		
		t.Logf("Found %d workflow states", len(states))
		
		// Verify we have different types
		typeMap := make(map[string]bool)
		for _, state := range states {
			if state.ID == "" {
				t.Error("State ID is empty")
			}
			if state.Name == "" {
				t.Error("State name is empty")
			}
			if state.Type == "" {
				t.Error("State type is empty")
			}
			typeMap[state.Type] = true
		}
		
		// Check we have various state types
		expectedTypes := []string{"backlog", "unstarted", "started", "completed", "canceled"}
		for _, expectedType := range expectedTypes {
			if !typeMap[expectedType] {
				t.Logf("Warning: Missing state type '%s'", expectedType)
			}
		}
		
		// Test with team filter if we have a team
		if teamID != "" {
			teamStates, err := client.GetWorkflowStates(teamID)
			if err != nil {
				t.Errorf("Failed to get workflow states for team: %v", err)
			} else {
				t.Logf("Found %d workflow states for team", len(teamStates))
			}
		}
	})
	
	// Test 5: List assigned issues
	t.Run("ListAssignedIssues", func(t *testing.T) {
		issues, err := client.ListAssignedIssues(viewerID)
		if err != nil {
			t.Fatalf("Failed to list assigned issues: %v", err)
		}
		
		t.Logf("Found %d assigned issues", len(issues))
		
		if len(issues) > 0 {
			// Use first issue for subsequent tests
			issueID = issues[0].ID
			
			// Verify issue structure
			for _, issue := range issues {
				if issue.ID == "" {
					t.Error("Issue ID is empty")
				}
				if issue.Identifier == "" {
					t.Error("Issue identifier is empty")
				}
				if issue.Title == "" {
					t.Error("Issue title is empty")
				}
				if issue.State.ID == "" {
					t.Error("Issue state ID is empty")
				}
				if issue.State.Name == "" {
					t.Error("Issue state name is empty")
				}
			}
		}
	})
	
	// Test 6: Get specific issue (if we found one)
	if issueID != "" {
		t.Run("GetIssue", func(t *testing.T) {
			issue, err := client.GetIssue(issueID)
			if err != nil {
				t.Fatalf("Failed to get issue: %v", err)
			}
			
			if issue.ID != issueID {
				t.Errorf("Issue ID mismatch: expected %s, got %s", issueID, issue.ID)
			}
			
			// Check metadata extraction
			if issue.Description != "" {
				t.Logf("Issue has description: %d chars", len(issue.Description))
				if issue.Metadata != nil {
					t.Logf("Issue has metadata: %v", issue.Metadata)
				}
			}
			
			// Check sub-issues
			if len(issue.Children.Nodes) > 0 {
				t.Logf("Issue has %d sub-issues", len(issue.Children.Nodes))
			}
		})
		
		t.Run("GetIssueWithProjectContext", func(t *testing.T) {
			issue, err := client.GetIssueWithProjectContext(issueID)
			if err != nil {
				t.Fatalf("Failed to get issue with project context: %v", err)
			}
			
			if issue.Project != nil {
				projectID = issue.Project.ID
				t.Logf("Issue belongs to project: %s", issue.Project.Name)
				
				// Check project metadata extraction
				if issue.Project.Metadata != nil {
					t.Logf("Project has metadata: %v", issue.Project.Metadata)
				}
			}
		})
		
		t.Run("GetIssueWithParentContext", func(t *testing.T) {
			issue, err := client.GetIssueWithParentContext(issueID)
			if err != nil {
				t.Fatalf("Failed to get issue with parent context: %v", err)
			}
			
			if issue.Parent != nil {
				t.Logf("Issue has parent: %s", issue.Parent.Identifier)
				
				// Check parent metadata extraction
				if issue.Parent.Metadata != nil {
					t.Logf("Parent has metadata: %v", issue.Parent.Metadata)
				}
			}
		})
		
		t.Run("GetSubIssues", func(t *testing.T) {
			subIssues, err := client.GetSubIssues(issueID)
			if err != nil {
				t.Fatalf("Failed to get sub-issues: %v", err)
			}
			
			t.Logf("Found %d sub-issues", len(subIssues))
			
			for _, subIssue := range subIssues {
				if subIssue.ID == "" {
					t.Error("Sub-issue ID is empty")
				}
				if subIssue.Identifier == "" {
					t.Error("Sub-issue identifier is empty")
				}
				if subIssue.State.ID == "" {
					t.Error("Sub-issue state ID is empty")
				}
			}
		})
	}
	
	// Test 7: List all projects
	t.Run("ListAllProjects", func(t *testing.T) {
		projects, err := client.ListAllProjects()
		if err != nil {
			t.Fatalf("Failed to list all projects: %v", err)
		}
		
		t.Logf("Found %d projects", len(projects))
		
		if len(projects) > 0 && projectID == "" {
			projectID = projects[0].ID
		}
		
		// Verify project structure
		for _, project := range projects {
			if project.ID == "" {
				t.Error("Project ID is empty")
			}
			if project.Name == "" {
				t.Error("Project name is empty")
			}
			
			// Check metadata extraction
			if project.Metadata != nil {
				t.Logf("Project %s has metadata: %v", project.Name, project.Metadata)
			}
		}
	})
	
	// Test 8: List user projects
	t.Run("ListUserProjects", func(t *testing.T) {
		projects, err := client.ListUserProjects(viewerID)
		if err != nil {
			t.Fatalf("Failed to list user projects: %v", err)
		}
		
		t.Logf("Found %d user projects", len(projects))
	})
	
	// Test 9: Get specific project (if we found one)
	if projectID != "" {
		t.Run("GetProject", func(t *testing.T) {
			project, err := client.GetProject(projectID)
			if err != nil {
				t.Fatalf("Failed to get project: %v", err)
			}
			
			if project.ID != projectID {
				t.Errorf("Project ID mismatch: expected %s, got %s", projectID, project.ID)
			}
			
			t.Logf("Project: %s (state: %s)", project.Name, project.State)
			
			// Check issues in project
			issues := project.GetIssues()
			if len(issues) > 0 {
				t.Logf("Project has %d issues", len(issues))
			}
			
			// Check metadata
			if project.Metadata != nil {
				t.Logf("Project has metadata: %v", project.Metadata)
			}
		})
	}
	
	// Test 10: Get notifications
	t.Run("GetNotifications", func(t *testing.T) {
		// Test with unread only
		notifications, err := client.GetNotifications(false, 10)
		if err != nil {
			t.Fatalf("Failed to get notifications: %v", err)
		}
		
		t.Logf("Found %d unread notifications", len(notifications))
		
		// Test including read
		allNotifications, err := client.GetNotifications(true, 10)
		if err != nil {
			t.Fatalf("Failed to get all notifications: %v", err)
		}
		
		t.Logf("Found %d total notifications", len(allNotifications))
		
		// Verify notification structure
		for _, notif := range allNotifications {
			if notif.ID == "" {
				t.Error("Notification ID is empty")
			}
			if notif.Type == "" {
				t.Error("Notification type is empty")
			}
			if notif.CreatedAt == "" {
				t.Error("Notification createdAt is empty")
			}
			
			// Log notification details
			if notif.Issue != nil {
				t.Logf("Notification about issue: %s", notif.Issue.Identifier)
			}
			if notif.Comment != nil && commentID == "" {
				// Try to extract comment ID for comment tests
				commentID = notif.Comment.ID
			}
		}
	})
	
	// Test 11: Get comment with replies (if we found a comment ID)
	if commentID != "" {
		t.Run("GetCommentWithReplies", func(t *testing.T) {
			commentWithReplies, err := client.GetCommentWithReplies(commentID)
			if err != nil {
				// Comment might have been deleted or we don't have access
				t.Logf("Failed to get comment with replies: %v", err)
			} else {
				t.Logf("Comment by %s: %s", commentWithReplies.Comment.User.Name, 
					truncateString(commentWithReplies.Comment.Body, 50))
				t.Logf("Comment has %d replies", len(commentWithReplies.Replies))
				
				// Verify reply structure
				for _, reply := range commentWithReplies.Replies {
					if reply.ID == "" {
						t.Error("Reply ID is empty")
					}
					if reply.User.ID == "" {
						t.Error("Reply user ID is empty")
					}
					if reply.Parent == nil || reply.Parent.ID == "" {
						t.Error("Reply parent ID is empty")
					}
				}
			}
		})
	}
	
	// Test 12: Performance test - concurrent reads
	t.Run("ConcurrentReads", func(t *testing.T) {
		start := time.Now()
		
		// Run multiple read operations concurrently
		done := make(chan bool, 4)
		
		go func() {
			_, _ = client.GetTeams()
			done <- true
		}()
		
		go func() {
			_, _ = client.GetViewer()
			done <- true
		}()
		
		go func() {
			_, _ = client.ListAllProjects()
			done <- true
		}()
		
		go func() {
			_, _ = client.GetWorkflowStates("")
			done <- true
		}()
		
		// Wait for all to complete
		for i := 0; i < 4; i++ {
			<-done
		}
		
		elapsed := time.Since(start)
		t.Logf("4 concurrent operations completed in %v", elapsed)
		
		// With connection pooling, this should be relatively fast
		if elapsed > 5*time.Second {
			t.Logf("Warning: Concurrent operations took longer than expected")
		}
	})
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}