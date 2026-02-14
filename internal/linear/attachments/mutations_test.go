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

func TestCreateAttachment_ResponseDeserialization(t *testing.T) {
	graphqlResponse := `{
		"attachmentCreate": {
			"success": true,
			"attachment": {
				"id": "att-new",
				"url": "https://github.com/org/repo/pull/42",
				"title": "PR #42",
				"subtitle": "Fix auth bug",
				"sourceType": "github",
				"createdAt": "2026-02-14T10:00:00Z",
				"updatedAt": "2026-02-14T10:00:00Z"
			}
		}
	}`

	var response struct {
		AttachmentCreate struct {
			Success    bool            `json:"success"`
			Attachment core.Attachment `json:"attachment"`
		} `json:"attachmentCreate"`
	}

	err := json.Unmarshal([]byte(graphqlResponse), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !response.AttachmentCreate.Success {
		t.Error("expected success=true")
	}
	att := response.AttachmentCreate.Attachment
	if att.ID != "att-new" {
		t.Errorf("ID = %q, want %q", att.ID, "att-new")
	}
	if att.Title != "PR #42" {
		t.Errorf("Title = %q, want %q", att.Title, "PR #42")
	}
	if att.Subtitle != "Fix auth bug" {
		t.Errorf("Subtitle = %q, want %q", att.Subtitle, "Fix auth bug")
	}
}

func TestDeleteAttachment_ResponseDeserialization(t *testing.T) {
	graphqlResponse := `{
		"attachmentDelete": {
			"success": true
		}
	}`

	var response struct {
		AttachmentDelete struct {
			Success bool `json:"success"`
		} `json:"attachmentDelete"`
	}

	err := json.Unmarshal([]byte(graphqlResponse), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !response.AttachmentDelete.Success {
		t.Error("expected success=true")
	}
}
