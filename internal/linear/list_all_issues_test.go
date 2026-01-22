package linear

import (
	"github.com/joa23/linear-cli/internal/token"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAllIssues(t *testing.T) {
	t.Run("list issues with no filters", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			query := req["query"].(string)
			assert.Contains(t, query, "query ListAllIssues")
			assert.Contains(t, query, "$first: Int!")
			assert.Contains(t, query, "$after: String")
			assert.Contains(t, query, "$filter: IssueFilter")
			assert.Contains(t, query, "$orderBy: IssueOrderBy")
			
			// Check that the query includes all necessary fields
			assert.Contains(t, query, "nodes {")
			assert.Contains(t, query, "id")
			assert.Contains(t, query, "title")
			assert.Contains(t, query, "description")
			assert.Contains(t, query, "state {")
			assert.Contains(t, query, "assignee {")
			assert.Contains(t, query, "labels {")
			assert.Contains(t, query, "project {")
			assert.Contains(t, query, "team {")
			assert.Contains(t, query, "pageInfo {")

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":          "issue-1",
								"title":       "First Issue",
								"description": "Description with metadata\n\n<details><summary>🤖 Metadata</summary>\n\n```json\n{\"category\": \"bug\"}\n```\n</details>",
								"priority":    3,
								"createdAt":   "2024-01-01T10:00:00Z",
								"updatedAt":   "2024-01-02T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-1",
									"name": "In Progress",
									"type": "started",
								},
								"assignee": map[string]interface{}{
									"id":    "user-1",
									"name":  "John Doe",
									"email": "john@example.com",
								},
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{
										{
											"id":    "label-1",
											"name":  "bug",
											"color": "#ff0000",
										},
									},
								},
								"project": map[string]interface{}{
									"id":   "project-1",
									"name": "Project Alpha",
								},
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
							{
								"id":          "issue-2",
								"title":       "Second Issue",
								"description": "Simple description",
								"priority":    1,
								"createdAt":   "2024-01-03T10:00:00Z",
								"updatedAt":   "2024-01-04T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-2",
									"name": "Todo",
									"type": "unstarted",
								},
								"assignee": nil,
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{},
								},
								"project": nil,
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
						},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "cursor-end",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First: 50,
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Len(t, result.Issues, 2)
		assert.False(t, result.HasNextPage)
		assert.Equal(t, "cursor-end", result.EndCursor)
		
		// Check first issue
		issue1 := result.Issues[0]
		assert.Equal(t, "issue-1", issue1.ID)
		assert.Equal(t, "First Issue", issue1.Title)
		assert.NotNil(t, issue1.Metadata)
		assert.Equal(t, "bug", (*issue1.Metadata)["category"])
		assert.Equal(t, "In Progress", issue1.State.Name)
		assert.NotNil(t, issue1.Assignee)
		assert.Equal(t, "John Doe", issue1.Assignee.Name)
		assert.Len(t, issue1.Labels, 1)
		assert.Equal(t, "bug", issue1.Labels[0].Name)
		assert.NotNil(t, issue1.Project)
		assert.Equal(t, "Project Alpha", issue1.Project.Name)
		
		// Check second issue
		issue2 := result.Issues[1]
		assert.Equal(t, "issue-2", issue2.ID)
		assert.Nil(t, issue2.Metadata)
		assert.Nil(t, issue2.Assignee)
		assert.Empty(t, issue2.Labels)
		assert.Nil(t, issue2.Project)
	})

	t.Run("list issues with state filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			variables := req["variables"].(map[string]interface{})
			filter := variables["filter"].(map[string]interface{})
			
			// Check that state filter is properly set
			assert.Equal(t, []interface{}{"state-1", "state-2"}, filter["state"].(map[string]interface{})["id"].(map[string]interface{})["in"])

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":          "issue-1",
								"title":       "Filtered Issue",
								"description": "",
								"priority":    2,
								"createdAt":   "2024-01-01T10:00:00Z",
								"updatedAt":   "2024-01-02T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-1",
									"name": "In Progress",
									"type": "started",
								},
								"assignee": nil,
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{},
								},
								"project": nil,
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
						},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "cursor-end",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First:    50,
			StateIDs: []string{"state-1", "state-2"},
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Len(t, result.Issues, 1)
		assert.Equal(t, "In Progress", result.Issues[0].State.Name)
	})

	t.Run("list issues with assignee filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			variables := req["variables"].(map[string]interface{})
			filter := variables["filter"].(map[string]interface{})
			
			// Check that assignee filter is properly set
			assert.Equal(t, "user-1", filter["assignee"].(map[string]interface{})["id"].(map[string]interface{})["eq"])

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":          "issue-1",
								"title":       "Assigned Issue",
								"description": "",
								"priority":    2,
								"createdAt":   "2024-01-01T10:00:00Z",
								"updatedAt":   "2024-01-02T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-1",
									"name": "In Progress",
									"type": "started",
								},
								"assignee": map[string]interface{}{
									"id":    "user-1",
									"name":  "John Doe",
									"email": "john@example.com",
								},
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{},
								},
								"project": nil,
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
						},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "cursor-end",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First:      50,
			AssigneeID: "user-1",
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Len(t, result.Issues, 1)
		assert.Equal(t, "user-1", result.Issues[0].Assignee.ID)
	})

	t.Run("list issues with label filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			variables := req["variables"].(map[string]interface{})
			filter := variables["filter"].(map[string]interface{})
			
			// Check that label filter is properly set
			labels := filter["labels"].(map[string]interface{})["id"].(map[string]interface{})["in"]
			assert.Equal(t, []interface{}{"label-1", "label-2"}, labels)

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":          "issue-1",
								"title":       "Labeled Issue",
								"description": "",
								"priority":    2,
								"createdAt":   "2024-01-01T10:00:00Z",
								"updatedAt":   "2024-01-02T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-1",
									"name": "In Progress",
									"type": "started",
								},
								"assignee": nil,
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{
										{
											"id":    "label-1",
											"name":  "bug",
											"color": "#ff0000",
										},
									},
								},
								"project": nil,
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
						},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "cursor-end",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First:    50,
			LabelIDs: []string{"label-1", "label-2"},
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Len(t, result.Issues, 1)
		assert.Len(t, result.Issues[0].Labels, 1)
		assert.Equal(t, "bug", result.Issues[0].Labels[0].Name)
	})

	t.Run("list issues with project filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			variables := req["variables"].(map[string]interface{})
			filter := variables["filter"].(map[string]interface{})
			
			// Check that project filter is properly set
			assert.Equal(t, "project-1", filter["project"].(map[string]interface{})["id"].(map[string]interface{})["eq"])

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":          "issue-1",
								"title":       "Project Issue",
								"description": "",
								"priority":    2,
								"createdAt":   "2024-01-01T10:00:00Z",
								"updatedAt":   "2024-01-02T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-1",
									"name": "In Progress",
									"type": "started",
								},
								"assignee": nil,
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{},
								},
								"project": map[string]interface{}{
									"id":   "project-1",
									"name": "Project Alpha",
								},
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
						},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "cursor-end",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First:     50,
			ProjectID: "project-1",
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Len(t, result.Issues, 1)
		assert.NotNil(t, result.Issues[0].Project)
		assert.Equal(t, "project-1", result.Issues[0].Project.ID)
	})

	t.Run("list issues with team filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			variables := req["variables"].(map[string]interface{})
			filter := variables["filter"].(map[string]interface{})
			
			// Check that team filter is properly set
			assert.Equal(t, "team-1", filter["team"].(map[string]interface{})["id"].(map[string]interface{})["eq"])

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":          "issue-1",
								"title":       "Team Issue",
								"description": "",
								"priority":    2,
								"createdAt":   "2024-01-01T10:00:00Z",
								"updatedAt":   "2024-01-02T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-1",
									"name": "In Progress",
									"type": "started",
								},
								"assignee": nil,
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{},
								},
								"project": nil,
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
						},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "cursor-end",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First:  50,
			TeamID: "team-1",
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Len(t, result.Issues, 1)
		assert.Equal(t, "team-1", result.Issues[0].Team.ID)
	})

	t.Run("list issues with sorting", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			variables := req["variables"].(map[string]interface{})
			orderBy := variables["orderBy"].(map[string]interface{})
			
			// Check that sorting is properly set
			assert.Equal(t, "PriorityDesc", orderBy["field"])
			assert.Equal(t, "DESCENDING", orderBy["direction"])

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":          "issue-1",
								"title":       "High Priority",
								"description": "",
								"priority":    1,
								"createdAt":   "2024-01-01T10:00:00Z",
								"updatedAt":   "2024-01-02T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-1",
									"name": "In Progress",
									"type": "started",
								},
								"assignee": nil,
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{},
								},
								"project": nil,
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
							{
								"id":          "issue-2",
								"title":       "Low Priority",
								"description": "",
								"priority":    4,
								"createdAt":   "2024-01-01T10:00:00Z",
								"updatedAt":   "2024-01-02T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-1",
									"name": "In Progress",
									"type": "started",
								},
								"assignee": nil,
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{},
								},
								"project": nil,
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
						},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "cursor-end",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First:     50,
			OrderBy:   "priority",
			Direction: "desc",
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Len(t, result.Issues, 2)
		// Linear priority 1 is highest, 4 is lowest
		assert.Equal(t, 1, result.Issues[0].Priority)
		assert.Equal(t, 4, result.Issues[1].Priority)
	})

	t.Run("list issues with pagination", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			variables := req["variables"].(map[string]interface{})
			
			// Check pagination parameters
			assert.Equal(t, float64(10), variables["first"])
			assert.Equal(t, "cursor-123", variables["after"])

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":          "issue-10",
								"title":       "Page 2 Issue",
								"description": "",
								"priority":    2,
								"createdAt":   "2024-01-01T10:00:00Z",
								"updatedAt":   "2024-01-02T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-1",
									"name": "In Progress",
									"type": "started",
								},
								"assignee": nil,
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{},
								},
								"project": nil,
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
						},
						"pageInfo": map[string]interface{}{
							"hasNextPage": true,
							"endCursor":   "cursor-456",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First: 10,
			After: "cursor-123",
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Len(t, result.Issues, 1)
		assert.True(t, result.HasNextPage)
		assert.Equal(t, "cursor-456", result.EndCursor)
	})

	t.Run("list issues with multiple filters", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			variables := req["variables"].(map[string]interface{})
			filter := variables["filter"].(map[string]interface{})
			
			// Check all filters are properly combined
			assert.Equal(t, []interface{}{"state-1"}, filter["state"].(map[string]interface{})["id"].(map[string]interface{})["in"])
			assert.Equal(t, "user-1", filter["assignee"].(map[string]interface{})["id"].(map[string]interface{})["eq"])
			assert.Equal(t, []interface{}{"label-1"}, filter["labels"].(map[string]interface{})["id"].(map[string]interface{})["in"])
			assert.Equal(t, "project-1", filter["project"].(map[string]interface{})["id"].(map[string]interface{})["eq"])
			assert.Equal(t, "team-1", filter["team"].(map[string]interface{})["id"].(map[string]interface{})["eq"])

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"id":          "issue-1",
								"title":       "Fully Filtered Issue",
								"description": "",
								"priority":    2,
								"createdAt":   "2024-01-01T10:00:00Z",
								"updatedAt":   "2024-01-02T10:00:00Z",
								"state": map[string]interface{}{
									"id":   "state-1",
									"name": "In Progress",
									"type": "started",
								},
								"assignee": map[string]interface{}{
									"id":    "user-1",
									"name":  "John Doe",
									"email": "john@example.com",
								},
								"labels": map[string]interface{}{
									"nodes": []map[string]interface{}{
										{
											"id":    "label-1",
											"name":  "bug",
											"color": "#ff0000",
										},
									},
								},
								"project": map[string]interface{}{
									"id":   "project-1",
									"name": "Project Alpha",
								},
								"team": map[string]interface{}{
									"id":   "team-1",
									"name": "Engineering",
									"key":  "ENG",
								},
							},
						},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "cursor-end",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First:      50,
			StateIDs:   []string{"state-1"},
			AssigneeID: "user-1",
			LabelIDs:   []string{"label-1"},
			ProjectID:  "project-1",
			TeamID:     "team-1",
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Len(t, result.Issues, 1)
		issue := result.Issues[0]
		assert.Equal(t, "state-1", issue.State.ID)
		assert.Equal(t, "user-1", issue.Assignee.ID)
		assert.Equal(t, "label-1", issue.Labels[0].ID)
		assert.Equal(t, "project-1", issue.Project.ID)
		assert.Equal(t, "team-1", issue.Team.ID)
	})

	t.Run("handle empty results", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"issues": map[string]interface{}{
						"nodes": []map[string]interface{}{},
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First: 50,
		}

		result, err := client.ListAllIssues(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Empty(t, result.Issues)
		assert.False(t, result.HasNextPage)
		assert.Equal(t, "", result.EndCursor)
	})

	t.Run("handle GraphQL errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"message": "Invalid filter parameter",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := &Client{
			base: &BaseClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				tokenProvider: token.NewStaticProvider("test-token"),
			},
			Issues: &IssueClient{
				base: &BaseClient{
					httpClient: server.Client(),
					baseURL:    server.URL,
					tokenProvider: token.NewStaticProvider("test-token"),
				},
			},
		}

		filter := &IssueFilter{
			First: 50,
		}

		_, err := client.ListAllIssues(filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid filter parameter")
	})
}