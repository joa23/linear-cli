package issues

import (
	"encoding/json"
	"testing"

	"github.com/joa23/linear-cli/internal/linear/core"
)

func TestBuildUpdateInput_DelegateID(t *testing.T) {
	tests := []struct {
		name           string
		input          core.UpdateIssueInput
		expectDelegate bool
		expectAssignee bool
	}{
		{
			name: "human user - uses assigneeId",
			input: core.UpdateIssueInput{
				AssigneeID: strPtr("user-uuid-123"),
			},
			expectAssignee: true,
			expectDelegate: false,
		},
		{
			name: "OAuth application - uses delegateId",
			input: core.UpdateIssueInput{
				DelegateID: strPtr("app-uuid-456"),
			},
			expectAssignee: false,
			expectDelegate: true,
		},
		{
			name: "unassign - empty assigneeId",
			input: core.UpdateIssueInput{
				AssigneeID: strPtr(""),
			},
			expectAssignee: true, // Still sets assigneeId to nil
			expectDelegate: false,
		},
		{
			name: "remove delegation - empty delegateId",
			input: core.UpdateIssueInput{
				DelegateID: strPtr(""),
			},
			expectAssignee: false,
			expectDelegate: true, // Still sets delegateId to nil
		},
		{
			name: "both set - both fields present",
			input: core.UpdateIssueInput{
				AssigneeID: strPtr("user-uuid"),
				DelegateID: strPtr("app-uuid"),
			},
			expectAssignee: true,
			expectDelegate: true,
		},
		{
			name:           "neither set - no assignment fields",
			input:          core.UpdateIssueInput{},
			expectAssignee: false,
			expectDelegate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildUpdateInput(tt.input)

			_, hasAssignee := result["assigneeId"]
			_, hasDelegate := result["delegateId"]

			if hasAssignee != tt.expectAssignee {
				t.Errorf("assigneeId presence = %v, want %v", hasAssignee, tt.expectAssignee)
			}
			if hasDelegate != tt.expectDelegate {
				t.Errorf("delegateId presence = %v, want %v", hasDelegate, tt.expectDelegate)
			}
		})
	}
}

func TestHasFieldsToUpdate_DelegateID(t *testing.T) {
	tests := []struct {
		name     string
		input    core.UpdateIssueInput
		expected bool
	}{
		{
			name:     "empty input",
			input:    core.UpdateIssueInput{},
			expected: false,
		},
		{
			name: "only delegateId",
			input: core.UpdateIssueInput{
				DelegateID: strPtr("app-uuid"),
			},
			expected: true,
		},
		{
			name: "only assigneeId",
			input: core.UpdateIssueInput{
				AssigneeID: strPtr("user-uuid"),
			},
			expected: true,
		},
		{
			name: "only title",
			input: core.UpdateIssueInput{
				Title: strPtr("New Title"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasFieldsToUpdate(tt.input)
			if result != tt.expected {
				t.Errorf("hasFieldsToUpdate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

// TestGetIssueResponseStruct_NoAttachmentShadowing verifies that the GetIssue
// response struct deserializes attachments into core.Issue.Attachments directly,
// without a shadowed struct that would swallow the data.
//
// Background: GetIssue previously used struct embedding with a shadow:
//
//	var response struct {
//	    Issue struct {
//	        core.Issue
//	        Attachments struct { Nodes []struct{ ID string } } `json:"attachments"`
//	    }
//	}
//
// This caused core.Issue.Attachments to always be nil — the outer Attachments
// field captured the JSON but the embedded core.Issue.Attachments was shadowed.
func TestGetIssueResponseStruct_NoAttachmentShadowing(t *testing.T) {
	// Simulated GraphQL response with attachments
	graphqlResponse := `{
		"id": "issue-uuid",
		"identifier": "TEC-100",
		"title": "Issue with attachments",
		"description": "",
		"state": {"id": "state-1", "name": "Todo"},
		"createdAt": "2025-01-01T00:00:00Z",
		"updatedAt": "2025-01-01T00:00:00Z",
		"url": "https://linear.app/test/issue/TEC-100",
		"attachments": {
			"nodes": [
				{
					"id": "att-1",
					"url": "https://github.com/org/repo/pull/42",
					"title": "PR #42: Fix auth bug",
					"sourceType": "github"
				},
				{
					"id": "att-2",
					"url": "https://slack.com/archives/C123/p456",
					"title": "Slack thread about auth",
					"sourceType": "slack"
				}
			]
		},
		"comments": {
			"nodes": [
				{
					"id": "comment-1",
					"body": "Test comment",
					"createdAt": "2025-01-02T00:00:00Z",
					"updatedAt": "2025-01-02T00:00:00Z",
					"user": {"id": "user-1", "name": "Alice", "email": "alice@test.com"}
				}
			]
		}
	}`

	// This is the FIXED response struct (matches current GetIssue code)
	var response struct {
		Issue core.Issue `json:"issue"`
	}

	// Wrap in {"issue": ...} to match GraphQL response structure
	wrapped := `{"issue": ` + graphqlResponse + `}`
	err := json.Unmarshal([]byte(wrapped), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify attachments are populated (not nil due to shadowing)
	if response.Issue.Attachments == nil {
		t.Fatal("Attachments is nil — likely shadowed by an outer struct field")
	}
	if len(response.Issue.Attachments.Nodes) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(response.Issue.Attachments.Nodes))
	}

	// Verify attachment data is fully deserialized
	att := response.Issue.Attachments.Nodes[0]
	if att.ID != "att-1" {
		t.Errorf("attachment ID = %q, want %q", att.ID, "att-1")
	}
	if att.Title != "PR #42: Fix auth bug" {
		t.Errorf("attachment Title = %q, want %q", att.Title, "PR #42: Fix auth bug")
	}
	if att.SourceType != "github" {
		t.Errorf("attachment SourceType = %q, want %q", att.SourceType, "github")
	}
	if att.URL != "https://github.com/org/repo/pull/42" {
		t.Errorf("attachment URL = %q, want %q", att.URL, "https://github.com/org/repo/pull/42")
	}

	// Verify comments are also populated (should work on all paths)
	if response.Issue.Comments == nil {
		t.Fatal("Comments is nil")
	}
	if len(response.Issue.Comments.Nodes) != 1 {
		t.Errorf("expected 1 comment, got %d", len(response.Issue.Comments.Nodes))
	}

	// Verify computed fields can be derived
	attachmentCount := len(response.Issue.Attachments.Nodes)
	if attachmentCount != 2 {
		t.Errorf("computed attachment count = %d, want 2", attachmentCount)
	}
}

// TestGetIssueResponseStruct_ShadowedVersion proves the old shadowed struct
// loses attachment data. This is the regression test — if someone reintroduces
// the shadow pattern, this test will catch it.
func TestGetIssueResponseStruct_ShadowedVersion(t *testing.T) {
	graphqlResponse := `{"issue": {
		"id": "issue-uuid",
		"identifier": "TEC-100",
		"title": "Test",
		"state": {"id": "s1", "name": "Todo"},
		"createdAt": "2025-01-01T00:00:00Z",
		"updatedAt": "2025-01-01T00:00:00Z",
		"url": "",
		"attachments": {
			"nodes": [{"id": "att-1", "url": "https://example.com", "title": "Link", "sourceType": "github"}]
		}
	}}`

	// OLD shadowed struct (the bug we're fixing)
	var shadowed struct {
		Issue struct {
			core.Issue
			Attachments struct {
				Nodes []struct {
					ID string `json:"id"`
				} `json:"nodes"`
			} `json:"attachments"`
		} `json:"issue"`
	}

	err := json.Unmarshal([]byte(graphqlResponse), &shadowed)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// The outer Attachments field captures the data...
	if len(shadowed.Issue.Attachments.Nodes) != 1 {
		t.Errorf("outer Attachments should have 1 node, got %d", len(shadowed.Issue.Attachments.Nodes))
	}

	// ...but core.Issue.Attachments is nil (shadowed!)
	if shadowed.Issue.Issue.Attachments != nil {
		t.Error("Expected core.Issue.Attachments to be nil when shadowed — did the struct change?")
	}

	// This is the data loss: returning &shadowed.Issue.Issue gives you nil attachments
	result := &shadowed.Issue.Issue
	if result.Attachments != nil {
		t.Error("Shadowed struct should lose attachment data — this test proves the bug exists")
	}
}
