package linear

import (
	"os"
	"testing"
)

// TestLinearAttachmentAPI tests actual Linear API behavior for attachments
// This test verifies what attachment metadata Linear provides and how URLs work
func TestLinearAttachmentAPI(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Integration tests are skipped")
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

	// First, let's check if we can find any issues with attachments
	t.Run("FindIssuesWithAttachments", func(t *testing.T) {
		// Test query to find issues with attachments
		query := `
		query FindIssuesWithAttachments {
		  issues(first: 10) {
		    nodes {
		      id
		      identifier
		      title
		      attachments {
		        totalCount
		        nodes {
		          id
		          url
		          title
		          subtitle
		          createdAt
		          updatedAt
		          metadata
		          source
		          sourceType
		          groupBySource
		          creator {
		            id
		            name
		            email
		          }
		          externalUserCreator {
		            id
		            name
		            displayName
		            email
		          }
		        }
		      }
		    }
		  }
		}`

		var response struct {
			Data struct {
				Issues struct {
					Nodes []struct {
						ID         string `json:"id"`
						Identifier string `json:"identifier"`
						Title      string `json:"title"`
						Attachments struct {
							Nodes      []struct {
								ID                  string                 `json:"id"`
								URL                 string                 `json:"url"`
								Title               string                 `json:"title"`
								Subtitle            string                 `json:"subtitle"`
								CreatedAt           string                 `json:"createdAt"`
								UpdatedAt           string                 `json:"updatedAt"`
								Metadata            map[string]interface{} `json:"metadata"`
								Source              map[string]interface{} `json:"source"`
								SourceType          string                 `json:"sourceType"`
								GroupBySource       bool                   `json:"groupBySource"`
								Creator             *User                  `json:"creator"`
								ExternalUserCreator *struct {
									ID          string `json:"id"`
									Name        string `json:"name"`
									DisplayName string `json:"displayName"`
									Email       string `json:"email"`
								} `json:"externalUserCreator"`
							} `json:"nodes"`
						} `json:"attachments"`
					} `json:"nodes"`
				} `json:"issues"`
			} `json:"data"`
		}

		err := client.base.executeRequest(query, nil, &response)
		if err != nil {
			t.Fatalf("Failed to query issues with attachments: %v", err)
		}

		t.Logf("Found %d issues", len(response.Data.Issues.Nodes))

		// Look for issues with attachments
		var issueWithAttachments *struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			Title      string `json:"title"`
			Attachments struct {
				Nodes      []struct {
					ID                  string                 `json:"id"`
					URL                 string                 `json:"url"`
					Title               string                 `json:"title"`
					Subtitle            string                 `json:"subtitle"`
					CreatedAt           string                 `json:"createdAt"`
					UpdatedAt           string                 `json:"updatedAt"`
					Metadata            map[string]interface{} `json:"metadata"`
					Source              map[string]interface{} `json:"source"`
					SourceType          string                 `json:"sourceType"`
					GroupBySource       bool                   `json:"groupBySource"`
					Creator             *User                  `json:"creator"`
					ExternalUserCreator *struct {
						ID          string `json:"id"`
						Name        string `json:"name"`
						DisplayName string `json:"displayName"`
						Email       string `json:"email"`
					} `json:"externalUserCreator"`
				} `json:"nodes"`
			} `json:"attachments"`
		}

		for _, issue := range response.Data.Issues.Nodes {
			if len(issue.Attachments.Nodes) > 0 {
				issueWithAttachments = &issue
				break
			}
		}

		if issueWithAttachments != nil {
			t.Logf("Found issue with attachments: %s (%s)", issueWithAttachments.Identifier, issueWithAttachments.Title)
			t.Logf("Attachment count: %d", len(issueWithAttachments.Attachments.Nodes))

			for i, attachment := range issueWithAttachments.Attachments.Nodes {
				t.Logf("Attachment %d:", i+1)
				t.Logf("  ID: %s", attachment.ID)
				t.Logf("  URL: %s", attachment.URL)
				t.Logf("  Title: %s", attachment.Title)
				t.Logf("  Subtitle: %s", attachment.Subtitle)
				t.Logf("  SourceType: %s", attachment.SourceType)
				t.Logf("  Metadata: %+v", attachment.Metadata)
				t.Logf("  Source: %+v", attachment.Source)
				if attachment.Creator != nil {
					t.Logf("  Creator: %s (%s)", attachment.Creator.Name, attachment.Creator.Email)
				}
			}
		} else {
			t.Log("No issues with attachments found in first 10 issues")
		}
	})
}

// TestAttachmentMetadataFields tests what fields Linear provides for attachments
func TestAttachmentMetadataFields(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Integration tests are skipped")
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

	// Query for all possible attachment fields to see what's actually available
	t.Run("QueryAllAttachmentFields", func(t *testing.T) {
		query := `
		query TestAttachmentFields {
		  issues(first: 1) {
		    nodes {
		      attachments {
		        nodes {
		          id
		          url
		          title
		          subtitle
		          createdAt
		          updatedAt
		          archivedAt
		          metadata
		          source
		          sourceType
		          groupBySource
		          creator {
		            id
		            name
		            displayName
		            email
		          }
		          externalUserCreator {
		            id
		            name
		            displayName
		            email
		          }
		        }
		      }
		    }
		  }
		}`

		var response struct {
			Data struct {
				Issues struct {
					Nodes []struct {
						Attachments struct {
							Nodes []map[string]interface{} `json:"nodes"`
						} `json:"attachments"`
					} `json:"nodes"`
				} `json:"issues"`
			} `json:"data"`
		}

		err := client.base.executeRequest(query, nil, &response)
		if err != nil {
			t.Fatalf("Failed to query attachment fields: %v", err)
		}

		t.Logf("GraphQL query executed successfully")
		if len(response.Data.Issues.Nodes) > 0 && len(response.Data.Issues.Nodes[0].Attachments.Nodes) > 0 {
			attachment := response.Data.Issues.Nodes[0].Attachments.Nodes[0]
			t.Logf("Available attachment fields:")
			for key, value := range attachment {
				t.Logf("  %s: %+v", key, value)
			}
		} else {
			t.Log("No attachments found to analyze fields")
		}
	})
}