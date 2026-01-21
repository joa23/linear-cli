package linear

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/internal/linear/testutil"
	"github.com/joa23/linear-cli/internal/token"
)

func TestLinearConnection(t *testing.T) {
	// Test with invalid token - should fail
	client := NewClient("invalid-token")
	err := client.TestConnection()

	// We expect an error with invalid token
	if err == nil {
		t.Error("Expected error with invalid token, got nil")
	}
}

func TestLinearConnectionWithValidToken(t *testing.T) {
	// Try default client first (stored token), then fallback to env var
	client := NewDefaultClient()
	if client.apiToken == "" {
		t.Skip("Skipping test: No stored token or LINEAR_API_TOKEN environment variable")
	}

	// Test with valid token - should succeed
	err := client.TestConnection()

	// We expect no error with valid token
	if err != nil {
		t.Errorf("Expected no error with valid token, got: %v", err)
	}
}

func TestGetAppUserID(t *testing.T) {
	mockResponse := `{
		"data": {
			"viewer": {
				"id": "550e8400-e29b-41d4-a716-446655440006"
			}
		}
	}`

	client := NewClient("test-token")
	client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

	userID, err := client.GetAppUserID()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedID := "550e8400-e29b-41d4-a716-446655440006"
	if userID != expectedID {
		t.Errorf("Expected user ID %s, got %s", expectedID, userID)
	}
}

func TestGetAppUserIDWithRealToken(t *testing.T) {
	client := NewDefaultClient()
	if client.apiToken == "" {
		t.Skip("Skipping test: No stored token or LINEAR_API_TOKEN environment variable")
	}

	userID, err := client.GetAppUserID()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if userID == "" {
		t.Error("Expected non-empty user ID")
	}

	t.Logf("Retrieved user ID: %s", userID)
}

func TestGetIssueWithProject(t *testing.T) {
	mockResponse := `{
		"data": {
			"issue": {
				"id": "550e8400-e29b-41d4-a716-446655440004",
				"title": "Test Issue Title",
				"description": "Test issue description",
				"project": {
					"name": "Test Project",
					"description": "Test project description with context"
				},
				"creator": {
					"id": "550e8400-e29b-41d4-a716-446655440001",
					"name": "Test Creator"
				}
			}
		}
	}`

	client := NewClient("test-token")
	client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

	issue, err := client.GetIssue("550e8400-e29b-41d4-a716-446655440004")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if issue.ID != "550e8400-e29b-41d4-a716-446655440004" {
		t.Errorf("Expected issue ID 550e8400-e29b-41d4-a716-446655440004, got %s", issue.ID)
	}

	if issue.Title != "Test Issue Title" {
		t.Errorf("Expected issue title 'Test Issue Title', got %s", issue.Title)
	}

	if issue.Description != "Test issue description" {
		t.Errorf("Expected issue description 'Test issue description', got %s", issue.Description)
	}

	if issue.Project.Name != "Test Project" {
		t.Errorf("Expected project name 'Test Project', got %s", issue.Project.Name)
	}

	if issue.Project.Description != "Test project description with context" {
		t.Errorf("Expected project description 'Test project description with context', got %s", issue.Project.Description)
	}

	if issue.Creator.ID != "550e8400-e29b-41d4-a716-446655440001" {
		t.Errorf("Expected creator ID 550e8400-e29b-41d4-a716-446655440001, got %s", issue.Creator.ID)
	}

	if issue.Creator.Name != "Test Creator" {
		t.Errorf("Expected creator name 'Test Creator', got %s", issue.Creator.Name)
	}
}

func TestGetIssueWithProjectWithRealToken(t *testing.T) {
	var apiToken string

	apiToken = os.Getenv("LINEAR_API_TOKEN")

	if apiToken == "" {
		storage := token.NewStorage(token.GetDefaultTokenPath())
		if storage.TokenExists() {
			storedToken, err := storage.LoadToken()
			if err == nil {
				apiToken = storedToken
			}
		}
	}

	if apiToken == "" {
		t.Skip("Skipping test: No stored token or LINEAR_API_TOKEN environment variable")
	}

	client := NewClient(apiToken)
	userID, err := client.GetAppUserID()
	if err != nil {
		t.Fatalf("Failed to get user ID: %v", err)
	}

	issues, err := client.ListAssignedIssues(userID)
	if err != nil {
		t.Fatalf("Failed to list assigned issues: %v", err)
	}

	if len(issues) == 0 {
		t.Skip("No assigned issues to test with")
	}

	firstIssueID := issues[0].ID

	issue, err := client.GetIssue(firstIssueID)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if issue.ID == "" {
		t.Error("Expected non-empty issue ID")
	}

	if issue.Title == "" {
		t.Error("Expected non-empty issue title")
	}

	if issue.Creator != nil && issue.Creator.ID == "" {
		t.Error("Expected non-empty creator ID when creator is present")
	}

	projectName := ""
	if issue.Project != nil {
		projectName = issue.Project.Name
	}
	creatorName := ""
	if issue.Creator != nil {
		creatorName = issue.Creator.Name
	}
	t.Logf("Retrieved issue: ID=%s, Title=%s, Project=%s, Creator=%s",
		issue.ID, issue.Title, projectName, creatorName)
}

func TestUpdateIssueState(t *testing.T) {
	t.Run("valid state transition", func(t *testing.T) {
		mockResponse := `{
			"data": {
				"issueUpdate": {
					"success": true,
					"issue": {
						"id": "550e8400-e29b-41d4-a716-446655440004",
						"state": {
							"name": "In Progress"
						}
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		err := client.UpdateIssueState("550e8400-e29b-41d4-a716-446655440004", "in-progress-state-id")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("invalid state transition", func(t *testing.T) {
		mockResponse := testutil.NewGraphQLErrorResponse("Invalid state transition", "INVALID_STATE_TRANSITION")

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		err := client.UpdateIssueState("550e8400-e29b-41d4-a716-446655440004", "invalid-state-id")

		if err == nil {
			t.Error("Expected error for invalid state transition, got nil")
		}

		if !strings.Contains(err.Error(), "Invalid state transition") {
			t.Errorf("Expected error to contain 'Invalid state transition', got: %v", err)
		}
	})
}

func TestUpdateIssueStateWithRealToken(t *testing.T) {
	t.Skip("Skipping real token test - requires knowledge of valid state IDs for workspace")
}

func TestAssignIssue(t *testing.T) {
	t.Run("assign to creator", func(t *testing.T) {
		mockResponse := `{
			"data": {
				"issueUpdate": {
					"success": true,
					"issue": {
						"id": "550e8400-e29b-41d4-a716-446655440004",
						"assignee": {
							"id": "550e8400-e29b-41d4-a716-446655440001",
							"name": "Test Creator"
						}
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		err := client.AssignIssue("550e8400-e29b-41d4-a716-446655440004", "550e8400-e29b-41d4-a716-446655440001")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("assign to self", func(t *testing.T) {
		mockResponse := `{
			"data": {
				"issueUpdate": {
					"success": true,
					"issue": {
						"id": "550e8400-e29b-41d4-a716-446655440004",
						"assignee": {
							"id": "550e8400-e29b-41d4-a716-446655440002",
							"name": "Linear Agent"
						}
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		err := client.AssignIssue("550e8400-e29b-41d4-a716-446655440004", "550e8400-e29b-41d4-a716-446655440002")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("assign with error", func(t *testing.T) {
		mockResponse := testutil.NewGraphQLErrorResponse("User not found", "USER_NOT_FOUND")

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		err := client.AssignIssue("550e8400-e29b-41d4-a716-446655440004", "invalid-user-id")

		if err == nil {
			t.Error("Expected error for invalid assignment, got nil")
		}

		if !strings.Contains(err.Error(), "User not found") {
			t.Errorf("Expected error to contain 'User not found', got: %v", err)
		}
	})
}

func TestAssignIssueWithRealToken(t *testing.T) {
	t.Skip("Skipping real token test - would modify real issue assignments")
}

func TestCreateComment(t *testing.T) {
	t.Run("create comment with markdown", func(t *testing.T) {
		mockResponse := `{
			"data": {
				"commentCreate": {
					"success": true,
					"comment": {
						"id": "550e8400-e29b-41d4-a716-446655440007",
						"body": "## Analysis\n\nHere's my **proposal**:\n- Item 1\n- Item 2",
						"createdAt": "2024-01-01T00:00:00Z"
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		commentBody := "## Analysis\n\nHere's my **proposal**:\n- Item 1\n- Item 2"
		commentID, err := client.CreateComment("550e8400-e29b-41d4-a716-446655440004", commentBody)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if commentID == "" {
			t.Error("Expected non-empty comment ID")
		}
	})

	t.Run("create empty comment", func(t *testing.T) {
		mockResponse := testutil.NewGraphQLErrorResponse("Comment body cannot be empty", "VALIDATION_ERROR")

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		_, err := client.CreateComment("550e8400-e29b-41d4-a716-446655440004", "")

		if err == nil {
			t.Error("Expected error for empty comment, got nil")
		}

		if !strings.Contains(err.Error(), "body cannot be empty") {
			t.Errorf("Expected error to contain 'body cannot be empty', got: %v", err)
		}
	})

	t.Run("create simple comment", func(t *testing.T) {
		mockResponse := `{
			"data": {
				"commentCreate": {
					"success": true,
					"comment": {
						"id": "550e8400-e29b-41d4-a716-446655440008",
						"body": "This is a simple comment",
						"createdAt": "2024-01-01T00:00:00Z"
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		commentID, err := client.CreateComment("550e8400-e29b-41d4-a716-446655440004", "This is a simple comment")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		expectedID := "550e8400-e29b-41d4-a716-446655440008"
		if commentID != expectedID {
			t.Errorf("Expected comment ID %s, got %s", expectedID, commentID)
		}
	})
}

func TestCreateCommentWithRealToken(t *testing.T) {
	t.Skip("Skipping real token test - would create real comments on issues")
}

func TestAddReaction(t *testing.T) {
	t.Run("add eyes emoji", func(t *testing.T) {
		mockResponse := `{
			"data": {
				"reactionCreate": {
					"success": true,
					"reaction": {
						"id": "550e8400-e29b-41d4-a716-446655440009",
						"emoji": "👀"
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		err := client.AddReaction("550e8400-e29b-41d4-a716-446655440004", "👀")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("add checkmark emoji", func(t *testing.T) {
		mockResponse := `{
			"data": {
				"reactionCreate": {
					"success": true,
					"reaction": {
						"id": "550e8400-e29b-41d4-a716-446655440010",
						"emoji": "✅"
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		err := client.AddReaction("550e8400-e29b-41d4-a716-446655440011", "✅")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("add reaction with error", func(t *testing.T) {
		mockResponse := testutil.NewGraphQLErrorResponse("Invalid emoji", "INVALID_EMOJI")

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		err := client.AddReaction("550e8400-e29b-41d4-a716-446655440004", "invalid-emoji")

		if err == nil {
			t.Error("Expected error for invalid emoji, got nil")
		}

		if !strings.Contains(err.Error(), "must be a single valid emoji") && !strings.Contains(err.Error(), "Invalid emoji") {
			t.Errorf("Expected error to contain 'must be a single valid emoji' or 'Invalid emoji', got: %v", err)
		}
	})
}

func TestAddReactionWithRealToken(t *testing.T) {
	t.Skip("Skipping real token test - would add reactions to real issues/comments")
}

func TestListAssignedIssues(t *testing.T) {
	t.Run("zero issues", func(t *testing.T) {
		mockResponse := `{
			"data": {
				"issues": {
					"nodes": []
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		issues, err := client.ListAssignedIssues("550e8400-e29b-41d4-a716-446655440006")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(issues) != 0 {
			t.Errorf("Expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("multiple issues", func(t *testing.T) {
		mockResponse := `{
			"data": {
				"issues": {
					"nodes": [
						{
							"id": "550e8400-e29b-41d4-a716-446655440014",
							"title": "First Issue",
							"state": {
								"name": "Todo"
							}
						},
						{
							"id": "550e8400-e29b-41d4-a716-446655440015",
							"title": "Second Issue",
							"state": {
								"name": "In Progress"
							}
						}
					]
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		issues, err := client.ListAssignedIssues("550e8400-e29b-41d4-a716-446655440006")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(issues) != 2 {
			t.Errorf("Expected 2 issues, got %d", len(issues))
		}

		if issues[0].ID != "550e8400-e29b-41d4-a716-446655440014" {
			t.Errorf("Expected first issue ID '550e8400-e29b-41d4-a716-446655440014', got '%s'", issues[0].ID)
		}

		if issues[0].Title != "First Issue" {
			t.Errorf("Expected first issue title 'First Issue', got '%s'", issues[0].Title)
		}

		if issues[1].ID != "550e8400-e29b-41d4-a716-446655440015" {
			t.Errorf("Expected second issue ID '550e8400-e29b-41d4-a716-446655440015', got '%s'", issues[1].ID)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		mockResponse := `{
			"data": {
				"issues": {
					"nodes": [
						{
							"id": "550e8400-e29b-41d4-a716-446655440014",
							"title": "First Issue",
							"state": {
								"name": "Todo"
							}
						}
					],
					"pageInfo": {
						"hasNextPage": true,
						"endCursor": "cursor123"
					}
				}
			}
		}`

		client := NewClient("test-token")
		client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

		issues, err := client.ListAssignedIssues("550e8400-e29b-41d4-a716-446655440006")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}
	})
}

func TestListAssignedIssuesWithRealToken(t *testing.T) {
	client := NewDefaultClient()
	if client.apiToken == "" {
		t.Skip("Skipping test: No stored token or LINEAR_API_TOKEN environment variable")
	}

	userID, err := client.GetAppUserID()
	if err != nil {
		t.Fatalf("Failed to get user ID: %v", err)
	}

	issues, err := client.ListAssignedIssues(userID)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	t.Logf("Found %d assigned issues", len(issues))
	for i, issue := range issues {
		t.Logf("Issue %d: ID=%s, Title=%s, State=%s", i+1, issue.ID, issue.Title, issue.State.Name)
	}
}

func TestClientTokenLoading(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "token")

	testToken := "stored-token-123"
	err := os.WriteFile(tokenPath, []byte(testToken), 0600)
	if err != nil {
		t.Fatalf("Failed to write test token: %v", err)
	}

	client := NewClientWithTokenPath(tokenPath)

	if client.apiToken != testToken {
		t.Errorf("Expected token %s, got %s", testToken, client.apiToken)
	}
}

func TestClientTokenFallback(t *testing.T) {
	testToken := "env-token-456"

	old := os.Getenv("LINEAR_API_TOKEN")
	os.Setenv("LINEAR_API_TOKEN", testToken)
	defer os.Setenv("LINEAR_API_TOKEN", old)

	nonExistentPath := "/this/path/does/not/exist"
	client := NewClientWithTokenPath(nonExistentPath)

	if client.apiToken != testToken {
		t.Errorf("Expected token %s, got %s", testToken, client.apiToken)
	}
}

func TestCreateIssue(t *testing.T) {
	mockResponse := `{
		"data": {
			"issueCreate": {
				"success": true,
				"issue": {
					"id": "550e8400-e29b-41d4-a716-446655440004",
					"title": "Test Issue",
					"description": "Test Description",
					"state": {
						"name": "Todo"
					}
				}
			}
		}
	}`

	client := NewClient("test-token")
	client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

	issue, err := client.CreateIssue("Test Issue", "Test Description", "550e8400-e29b-41d4-a716-446655440003")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedID := "550e8400-e29b-41d4-a716-446655440004"
	if issue == nil || issue.ID != expectedID {
		t.Errorf("Expected issue ID %s, got %v", expectedID, issue)
	}
}

func TestCreateIssueWithError(t *testing.T) {
	mockResponse := testutil.NewGraphQLErrorResponse("Team not found", "TEAM_NOT_FOUND")

	client := NewClient("test-token")
	client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

	_, err := client.CreateIssue("Test Issue", "Test Description", "invalid-team")

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Team not found") {
		t.Errorf("Expected error to contain 'Team not found', got: %v", err)
	}
}

func TestGetTeams(t *testing.T) {
	mockResponse := `{
		"data": {
			"teams": {
				"nodes": [
					{
						"id": "550e8400-e29b-41d4-a716-446655440012",
						"name": "Engineering",
						"key": "ENG"
					},
					{
						"id": "550e8400-e29b-41d4-a716-446655440013",
						"name": "Product",
						"key": "PROD"
					}
				]
			}
		}
	}`

	client := NewClient("test-token")
	client.base.httpClient = &http.Client{Transport: testutil.NewSuccessTransport(mockResponse)}

	teams, err := client.GetTeams()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(teams) != 2 {
		t.Errorf("Expected 2 teams, got %d", len(teams))
	}

	if teams[0].ID != "550e8400-e29b-41d4-a716-446655440012" {
		t.Errorf("Expected first team ID '550e8400-e29b-41d4-a716-446655440012', got '%s'", teams[0].ID)
	}
	if teams[0].Name != "Engineering" {
		t.Errorf("Expected first team name 'Engineering', got '%s'", teams[0].Name)
	}
	if teams[0].Key != "ENG" {
		t.Errorf("Expected first team key 'ENG', got '%s'", teams[0].Key)
	}
}

func TestGetTeamsWithRealToken(t *testing.T) {
	client := NewDefaultClient()
	if client.apiToken == "" {
		t.Skip("Skipping test: No stored token or LINEAR_API_TOKEN environment variable")
	}

	teams, err := client.GetTeams()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	t.Logf("Found %d teams", len(teams))
	for i, team := range teams {
		t.Logf("Team %d: ID=%s, Name=%s, Key=%s", i+1, team.ID, team.Name, team.Key)
	}
}
