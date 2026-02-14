package attachments

import (
	"encoding/json"
	"testing"

	"github.com/joa23/linear-cli/internal/linear/core"
)

func TestListAttachments_Deserialization(t *testing.T) {
	graphqlResponse := `{
		"issue": {
			"attachments": {
				"nodes": [
					{
						"id": "att-1",
						"url": "https://github.com/org/repo/pull/42",
						"title": "PR #42: Fix auth",
						"subtitle": "Merged",
						"sourceType": "github",
						"createdAt": "2026-01-15T10:00:00Z",
						"updatedAt": "2026-01-15T10:00:00Z"
					},
					{
						"id": "att-2",
						"url": "https://slack.com/archives/C123/p456",
						"title": "Slack thread",
						"subtitle": "",
						"sourceType": "slack",
						"createdAt": "2026-01-16T10:00:00Z",
						"updatedAt": "2026-01-16T10:00:00Z"
					}
				]
			}
		}
	}`

	var response struct {
		Issue struct {
			Attachments struct {
				Nodes []core.Attachment `json:"nodes"`
			} `json:"attachments"`
		} `json:"issue"`
	}

	err := json.Unmarshal([]byte(graphqlResponse), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	atts := response.Issue.Attachments.Nodes
	if len(atts) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(atts))
	}

	if atts[0].ID != "att-1" {
		t.Errorf("att[0].ID = %q, want %q", atts[0].ID, "att-1")
	}
	if atts[0].Title != "PR #42: Fix auth" {
		t.Errorf("att[0].Title = %q, want %q", atts[0].Title, "PR #42: Fix auth")
	}
	if atts[0].SourceType != "github" {
		t.Errorf("att[0].SourceType = %q, want %q", atts[0].SourceType, "github")
	}
	if atts[1].SourceType != "slack" {
		t.Errorf("att[1].SourceType = %q, want %q", atts[1].SourceType, "slack")
	}
}
