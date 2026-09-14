package issues

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/testutil"
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

// TestLabelParent_Deserialization verifies that Parent is populated when the
// GraphQL response includes a parent field on a label, and is nil when absent.
func TestLabelParent_Deserialization(t *testing.T) {
	// Simulated GraphQL response: one label with a parent, one without.
	graphqlResponse := `{
		"issue": {
			"id": "issue-uuid",
			"identifier": "TEC-200",
			"title": "Issue with labels",
			"description": "",
			"state": {"id": "state-1", "name": "Todo"},
			"createdAt": "2025-01-01T00:00:00Z",
			"updatedAt": "2025-01-01T00:00:00Z",
			"url": "https://linear.app/test/issue/TEC-200",
			"labels": {
				"nodes": [
					{
						"id": "label-child-1",
						"name": "Bug",
						"color": "#ff0000",
						"description": "A bug",
						"parent": {
							"id": "label-parent-1",
							"name": "Type"
						}
					},
					{
						"id": "label-orphan-1",
						"name": "Urgent",
						"color": "#ff9900",
						"description": "Urgent issue"
					}
				]
			}
		}
	}`

	var response struct {
		Issue core.Issue `json:"issue"`
	}

	if err := json.Unmarshal([]byte(graphqlResponse), &response); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if response.Issue.Labels == nil {
		t.Fatal("Labels is nil")
	}
	if len(response.Issue.Labels.Nodes) != 2 {
		t.Fatalf("expected 2 label nodes, got %d", len(response.Issue.Labels.Nodes))
	}

	// First label has a parent
	first := response.Issue.Labels.Nodes[0]
	if first.Parent == nil {
		t.Fatal("Labels.Nodes[0].Parent is nil, want non-nil")
	}
	if first.Parent.ID != "label-parent-1" {
		t.Errorf("Labels.Nodes[0].Parent.ID = %q, want %q", first.Parent.ID, "label-parent-1")
	}
	if first.Parent.Name != "Type" {
		t.Errorf("Labels.Nodes[0].Parent.Name = %q, want %q", first.Parent.Name, "Type")
	}

	// Second label has no parent
	second := response.Issue.Labels.Nodes[1]
	if second.Parent != nil {
		t.Errorf("Labels.Nodes[1].Parent = %+v, want nil", second.Parent)
	}
}

// newCapturingTestClient wires an issues.Client to a testutil.CapturingTransport,
// so tests can inspect the outgoing GraphQL request while feeding back a canned
// response body.
func newCapturingTestClient(responseBody interface{}) (*Client, *testutil.CapturingTransport) {
	transport := testutil.NewCapturingTransport(http.StatusOK, responseBody)
	base := core.NewBaseClient("test-token")
	base.SetHTTPClient(&http.Client{Transport: transport})
	return NewClient(base), transport
}

// assertQueryRequestsFields fails the test if any of the given field names is
// not present as a token in the outgoing GraphQL selection set. This is a
// lightweight substring check (not a GraphQL parser), used as regression
// protection against a field silently going missing from a query's selection
// set — the exact bug class TL-563 fixed for SearchIssuesEnhanced,
// ListAssignedIssues, and ListAllIssues.
func assertQueryRequestsFields(t *testing.T, query string, fields []string) {
	t.Helper()
	for _, field := range fields {
		if !strings.Contains(query, field) {
			t.Errorf("query does not request field %q:\n%s", field, query)
		}
	}
}

// requiredIssueFields is the set of fields TL-563 added to SearchIssuesEnhanced,
// ListAssignedIssues, and ListAllIssues, matching what GetIssue already requested.
var requiredIssueFields = []string{"dueDate", "estimate", "delegate"}

// TestSearchIssuesEnhanced_RequestsAndDecodesDueDateEstimateDelegate is a
// regression test for TL-563: SearchIssuesEnhanced's query silently omitted
// dueDate, estimate, and delegate, so "issues list"/"search" never returned
// them even though GetIssue's identical selection worked fine.
func TestSearchIssuesEnhanced_RequestsAndDecodesDueDateEstimateDelegate(t *testing.T) {
	responseBody := testutil.NewGraphQLDataResponse(testutil.IssuesData{
		Issues: testutil.IssuesNodes{
			Nodes: []interface{}{
				// Populated: has a due date, estimate, and delegate.
				map[string]interface{}{
					"id":         "issue-1",
					"identifier": "TL-1",
					"title":      "Issue with due date, estimate, and delegate",
					"state":      map[string]interface{}{"id": "state-1", "name": "Todo"},
					"priority":   2,
					"estimate":   3.5,
					"dueDate":    "2026-09-01",
					"delegate":   map[string]interface{}{"id": "delegate-1", "name": "Bot", "email": "bot@example.com"},
					"createdAt":  "2026-01-01T00:00:00Z",
					"updatedAt":  "2026-01-02T00:00:00Z",
					"url":        "https://linear.app/test/issue/TL-1",
				},
				// Unpopulated: none of the three fields set.
				map[string]interface{}{
					"id":         "issue-2",
					"identifier": "TL-2",
					"title":      "Issue without due date, estimate, or delegate",
					"state":      map[string]interface{}{"id": "state-1", "name": "Todo"},
					"labels":     map[string]interface{}{"nodes": []interface{}{}},
					"createdAt":  "2026-01-01T00:00:00Z",
					"updatedAt":  "2026-01-02T00:00:00Z",
					"url":        "https://linear.app/test/issue/TL-2",
				},
			},
			PageInfo: &testutil.PageInfo{HasNextPage: false},
		},
	})

	client, transport := newCapturingTestClient(responseBody)

	result, err := client.SearchIssuesEnhanced(&core.IssueSearchFilters{TeamID: "team-1"})
	if err != nil {
		t.Fatalf("SearchIssuesEnhanced returned error: %v", err)
	}

	query, err := transport.CapturedQuery()
	if err != nil {
		t.Fatalf("failed to extract captured query: %v", err)
	}
	assertQueryRequestsFields(t, query, requiredIssueFields)

	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}

	populated := result.Issues[0]
	if populated.Estimate == nil || *populated.Estimate != 3.5 {
		t.Errorf("Issues[0].Estimate = %v, want 3.5", populated.Estimate)
	}
	if populated.DueDate == nil || *populated.DueDate != "2026-09-01" {
		t.Errorf("Issues[0].DueDate = %v, want \"2026-09-01\"", populated.DueDate)
	}
	if populated.Delegate == nil || populated.Delegate.ID != "delegate-1" {
		t.Errorf("Issues[0].Delegate = %v, want ID \"delegate-1\"", populated.Delegate)
	}

	unpopulated := result.Issues[1]
	if unpopulated.Estimate != nil {
		t.Errorf("Issues[1].Estimate = %v, want nil", unpopulated.Estimate)
	}
	if unpopulated.DueDate != nil {
		t.Errorf("Issues[1].DueDate = %v, want nil", unpopulated.DueDate)
	}
	if unpopulated.Delegate != nil {
		t.Errorf("Issues[1].Delegate = %v, want nil", unpopulated.Delegate)
	}
}

// TestListAssignedIssues_RequestsAndDecodesDueDateEstimateDelegate is a
// regression test for TL-563: ListAssignedIssues's query, like
// SearchIssuesEnhanced's, silently omitted dueDate, estimate, and delegate.
func TestListAssignedIssues_RequestsAndDecodesDueDateEstimateDelegate(t *testing.T) {
	responseBody := testutil.NewGraphQLDataResponse(testutil.IssuesData{
		Issues: testutil.IssuesNodes{
			Nodes: []interface{}{
				map[string]interface{}{
					"id":         "issue-1",
					"identifier": "TL-1",
					"title":      "Issue with due date, estimate, and delegate",
					"state":      map[string]interface{}{"id": "state-1", "name": "Todo"},
					"estimate":   2.0,
					"dueDate":    "2026-09-01",
					"delegate":   map[string]interface{}{"id": "delegate-1", "name": "Bot", "email": "bot@example.com"},
					"createdAt":  "2026-01-01T00:00:00Z",
					"updatedAt":  "2026-01-02T00:00:00Z",
					"url":        "https://linear.app/test/issue/TL-1",
				},
				map[string]interface{}{
					"id":         "issue-2",
					"identifier": "TL-2",
					"title":      "Issue without due date, estimate, or delegate",
					"state":      map[string]interface{}{"id": "state-1", "name": "Todo"},
					"createdAt":  "2026-01-01T00:00:00Z",
					"updatedAt":  "2026-01-02T00:00:00Z",
					"url":        "https://linear.app/test/issue/TL-2",
				},
			},
		},
	})

	client, transport := newCapturingTestClient(responseBody)

	issues, err := client.ListAssignedIssues(10)
	if err != nil {
		t.Fatalf("ListAssignedIssues returned error: %v", err)
	}

	query, err := transport.CapturedQuery()
	if err != nil {
		t.Fatalf("failed to extract captured query: %v", err)
	}
	assertQueryRequestsFields(t, query, requiredIssueFields)

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	populated := issues[0]
	if populated.Estimate == nil || *populated.Estimate != 2.0 {
		t.Errorf("issues[0].Estimate = %v, want 2.0", populated.Estimate)
	}
	if populated.DueDate == nil || *populated.DueDate != "2026-09-01" {
		t.Errorf("issues[0].DueDate = %v, want \"2026-09-01\"", populated.DueDate)
	}
	if populated.Delegate == nil || populated.Delegate.ID != "delegate-1" {
		t.Errorf("issues[0].Delegate = %v, want ID \"delegate-1\"", populated.Delegate)
	}

	unpopulated := issues[1]
	if unpopulated.Estimate != nil {
		t.Errorf("issues[1].Estimate = %v, want nil", unpopulated.Estimate)
	}
	if unpopulated.DueDate != nil {
		t.Errorf("issues[1].DueDate = %v, want nil", unpopulated.DueDate)
	}
	if unpopulated.Delegate != nil {
		t.Errorf("issues[1].Delegate = %v, want nil", unpopulated.Delegate)
	}
}

// TestListAllIssues_RequestsAndDecodesDueDateEstimateDelegate is a regression
// test for TL-563: ListAllIssues's query, like SearchIssuesEnhanced's and
// ListAssignedIssues's, silently omitted dueDate, estimate, and delegate.
// Unlike those two, ListAllIssues decodes into a local anonymous struct and
// maps into core.IssueWithDetails, so this also exercises that mapping.
func TestListAllIssues_RequestsAndDecodesDueDateEstimateDelegate(t *testing.T) {
	responseBody := testutil.NewGraphQLDataResponse(testutil.IssuesData{
		Issues: testutil.IssuesNodes{
			Nodes: []interface{}{
				map[string]interface{}{
					"id":         "issue-1",
					"identifier": "TL-1",
					"title":      "Issue with due date, estimate, and delegate",
					"priority":   2,
					"estimate":   1.5,
					"dueDate":    "2026-09-01",
					"createdAt":  "2026-01-01T00:00:00Z",
					"updatedAt":  "2026-01-02T00:00:00Z",
					"state":      map[string]interface{}{"id": "state-1", "name": "Todo", "type": "unstarted"},
					"delegate":   map[string]interface{}{"id": "delegate-1", "name": "Bot", "email": "bot@example.com"},
					"team":       map[string]interface{}{"id": "team-1", "name": "Team", "key": "TL"},
				},
				map[string]interface{}{
					"id":         "issue-2",
					"identifier": "TL-2",
					"title":      "Issue without due date, estimate, or delegate",
					"priority":   1,
					"createdAt":  "2026-01-01T00:00:00Z",
					"updatedAt":  "2026-01-02T00:00:00Z",
					"state":      map[string]interface{}{"id": "state-1", "name": "Todo", "type": "unstarted"},
					"labels":     map[string]interface{}{"nodes": []interface{}{}},
					"team":       map[string]interface{}{"id": "team-1", "name": "Team", "key": "TL"},
				},
			},
			PageInfo: &testutil.PageInfo{HasNextPage: false},
		},
	})

	client, transport := newCapturingTestClient(responseBody)

	result, err := client.ListAllIssues(&core.IssueFilter{First: 10})
	if err != nil {
		t.Fatalf("ListAllIssues returned error: %v", err)
	}

	query, err := transport.CapturedQuery()
	if err != nil {
		t.Fatalf("failed to extract captured query: %v", err)
	}
	assertQueryRequestsFields(t, query, requiredIssueFields)

	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}

	populated := result.Issues[0]
	if populated.Estimate == nil || *populated.Estimate != 1.5 {
		t.Errorf("Issues[0].Estimate = %v, want 1.5", populated.Estimate)
	}
	if populated.DueDate == nil || *populated.DueDate != "2026-09-01" {
		t.Errorf("Issues[0].DueDate = %v, want \"2026-09-01\"", populated.DueDate)
	}
	if populated.Delegate == nil || populated.Delegate.ID != "delegate-1" {
		t.Errorf("Issues[0].Delegate = %v, want ID \"delegate-1\"", populated.Delegate)
	}

	unpopulated := result.Issues[1]
	if unpopulated.Estimate != nil {
		t.Errorf("Issues[1].Estimate = %v, want nil", unpopulated.Estimate)
	}
	if unpopulated.DueDate != nil {
		t.Errorf("Issues[1].DueDate = %v, want nil", unpopulated.DueDate)
	}
	if unpopulated.Delegate != nil {
		t.Errorf("Issues[1].Delegate = %v, want nil", unpopulated.Delegate)
	}
}
