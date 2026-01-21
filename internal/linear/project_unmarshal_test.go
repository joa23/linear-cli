package linear_test

import (
	"encoding/json"
	"testing"

	"github.com/joa23/linear-cli/internal/linear"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectUnmarshalWithIssues(t *testing.T) {
	// Mock GraphQL response with projects containing issues in nodes structure
	mockResponse := `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": [
						{
							"id": "issue-1",
							"title": "Test Issue",
							"project": {
								"id": "project-1",
								"name": "Test Project",
								"description": "Test project description",
								"state": "started",
								"issues": {
									"nodes": [
										{
											"id": "issue-1",
											"title": "Test Issue"
										},
										{
											"id": "issue-2",
											"title": "Another Issue"
										}
									]
								}
							}
						}
					]
				}
			}
		}
	}`

	// Test 1: Current structure successfully unmarshals nested issues
	t.Run("current structure handles nested issues", func(t *testing.T) {
		var response struct {
			Data struct {
				Viewer struct {
					AssignedIssues struct {
						Nodes []struct {
							ID      string `json:"id"`
							Title   string `json:"title"`
							Project linear.Project `json:"project"`
						} `json:"nodes"`
					} `json:"assignedIssues"`
				} `json:"viewer"`
			} `json:"data"`
		}

		err := json.Unmarshal([]byte(mockResponse), &response)
		require.NoError(t, err, "Should unmarshal with nested project issues")

		// Verify the data was unmarshaled correctly
		assert.Len(t, response.Data.Viewer.AssignedIssues.Nodes, 1)
		project := response.Data.Viewer.AssignedIssues.Nodes[0].Project
		assert.Equal(t, "project-1", project.ID)
		assert.Equal(t, "Test Project", project.Name)

		issues := project.GetIssues()
		assert.Len(t, issues, 2)
		assert.Equal(t, "issue-1", issues[0].ID)
		assert.Equal(t, "issue-2", issues[1].ID)
	})

	// Test 2: Direct unmarshal into Project with nested issues structure
	t.Run("unmarshal project with nested issues structure", func(t *testing.T) {
		projectJSON := `{
			"id": "project-1",
			"name": "Test Project",
			"description": "Test project description",
			"state": "started",
			"issues": {
				"nodes": [
					{
						"id": "issue-1",
						"title": "Test Issue"
					},
					{
						"id": "issue-2",
						"title": "Another Issue"
					}
				]
			}
		}`

		var project linear.Project
		err := json.Unmarshal([]byte(projectJSON), &project)
		require.NoError(t, err, "Should unmarshal project with nested issues")

		assert.Equal(t, "project-1", project.ID)
		assert.Equal(t, "Test Project", project.Name)

		issues := project.GetIssues()
		assert.Len(t, issues, 2)
		assert.Equal(t, "issue-1", issues[0].ID)
		assert.Equal(t, "issue-2", issues[1].ID)
	})
}

// Test the fixed Project structure
func TestProjectUnmarshalWithFixedStructure(t *testing.T) {

	mockResponse := `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": [
						{
							"id": "issue-1",
							"title": "Test Issue",
							"project": {
								"id": "project-1",
								"name": "Test Project",
								"description": "Test project description",
								"state": "started",
								"issues": {
									"nodes": [
										{
											"id": "issue-1",
											"title": "Test Issue"
										},
										{
											"id": "issue-2",
											"title": "Another Issue"
										}
									]
								}
							}
						}
					]
				}
			}
		}
	}`

	t.Run("fixed structure handles nested issues correctly", func(t *testing.T) {
		var response struct {
			Data struct {
				Viewer struct {
					AssignedIssues struct {
						Nodes []struct {
							ID      string         `json:"id"`
							Title   string         `json:"title"`
							Project linear.Project `json:"project"`
						} `json:"nodes"`
					} `json:"assignedIssues"`
				} `json:"viewer"`
			} `json:"data"`
		}

		err := json.Unmarshal([]byte(mockResponse), &response)
		require.NoError(t, err, "Fixed structure should unmarshal without error")

		// Verify the data was unmarshaled correctly
		assert.Len(t, response.Data.Viewer.AssignedIssues.Nodes, 1)
		project := response.Data.Viewer.AssignedIssues.Nodes[0].Project
		assert.Equal(t, "project-1", project.ID)
		assert.Equal(t, "Test Project", project.Name)
		
		issues := project.GetIssues()
		assert.Len(t, issues, 2)
		assert.Equal(t, "issue-1", issues[0].ID)
		assert.Equal(t, "issue-2", issues[1].ID)
	})

	t.Run("fixed structure handles projects without issues", func(t *testing.T) {
		projectJSON := `{
			"id": "project-2",
			"name": "Project Without Issues",
			"description": "No issues here",
			"state": "started"
		}`

		var project linear.Project
		err := json.Unmarshal([]byte(projectJSON), &project)
		require.NoError(t, err, "Should unmarshal project without issues field")

		assert.Equal(t, "project-2", project.ID)
		assert.Equal(t, "Project Without Issues", project.Name)
		assert.Empty(t, project.GetIssues())
	})
}