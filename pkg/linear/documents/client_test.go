package documents

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

func TestListDocuments_Deserialization(t *testing.T) {
	graphqlResponse := `{
		"documents": {
			"nodes": [
				{
					"id": "doc-1",
					"slugId": "abc123",
					"title": "API Spec",
					"url": "https://linear.app/acme/document/api-spec-abc123",
					"createdAt": "2026-01-15T10:00:00Z",
					"updatedAt": "2026-01-16T10:00:00Z",
					"creator": {"id": "u-1", "name": "Alice", "displayName": "alice"},
					"project": {"id": "p-1", "name": "Platform"}
				},
				{
					"id": "doc-2",
					"slugId": "def456",
					"title": "Runbook",
					"url": "https://linear.app/acme/document/runbook-def456",
					"createdAt": "2026-01-15T10:00:00Z",
					"updatedAt": "2026-01-16T10:00:00Z",
					"issue": {"id": "i-1", "identifier": "ENG-42", "title": "Fix it"},
					"team": null
				}
			],
			"pageInfo": {"hasNextPage": true, "endCursor": "cursor-xyz"}
		}
	}`

	var response struct {
		Documents struct {
			Nodes    []core.Document `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"documents"`
	}

	if err := json.Unmarshal([]byte(graphqlResponse), &response); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	docs := response.Documents.Nodes
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}
	if docs[0].SlugID != "abc123" || docs[0].Title != "API Spec" {
		t.Errorf("unexpected first doc: %+v", docs[0])
	}
	if docs[0].Project == nil || docs[0].Project.Name != "Platform" {
		t.Error("expected project on first doc")
	}
	if docs[0].Creator == nil || docs[0].Creator.DisplayName != "alice" {
		t.Error("expected creator on first doc")
	}
	if docs[1].Issue == nil || docs[1].Issue.Identifier != "ENG-42" {
		t.Error("expected issue on second doc")
	}
	if docs[1].Team != nil {
		t.Error("expected nil team on second doc")
	}
	if !response.Documents.PageInfo.HasNextPage || response.Documents.PageInfo.EndCursor != "cursor-xyz" {
		t.Error("expected pagination info")
	}
}

func TestGetDocument_Deserialization(t *testing.T) {
	graphqlResponse := `{
		"document": {
			"id": "doc-1",
			"slugId": "abc123",
			"title": "API Spec",
			"content": "# API\n\nHello",
			"icon": "Rocket",
			"color": "#4EA7FC",
			"url": "https://linear.app/acme/document/api-spec-abc123",
			"createdAt": "2026-01-15T10:00:00Z",
			"updatedAt": "2026-01-16T10:00:00Z",
			"archivedAt": "2026-02-01T10:00:00Z",
			"team": {"id": "t-1", "key": "ENG", "name": "Engineering"}
		}
	}`

	var response struct {
		Document core.Document `json:"document"`
	}
	if err := json.Unmarshal([]byte(graphqlResponse), &response); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	doc := response.Document
	if doc.Content != "# API\n\nHello" {
		t.Errorf("unexpected content: %q", doc.Content)
	}
	if doc.Icon != "Rocket" || doc.Color != "#4EA7FC" {
		t.Error("expected icon and color")
	}
	if doc.ArchivedAt == nil || *doc.ArchivedAt != "2026-02-01T10:00:00Z" {
		t.Error("expected archivedAt")
	}
	if doc.Team == nil || doc.Team.Key != "ENG" {
		t.Error("expected team")
	}
}

func TestBuildFilter(t *testing.T) {
	eq := func(v string) map[string]interface{} {
		return map[string]interface{}{"id": map[string]interface{}{"eq": v}}
	}

	tests := []struct {
		name   string
		filter *ListFilter
		want   map[string]interface{}
	}{
		{name: "nil", filter: nil, want: nil},
		{name: "empty", filter: &ListFilter{}, want: nil},
		{name: "archived only is not a filter", filter: &ListFilter{IncludeArchived: true}, want: nil},
		{name: "project", filter: &ListFilter{ProjectID: "p-1"}, want: map[string]interface{}{"project": eq("p-1")}},
		{name: "issue", filter: &ListFilter{IssueID: "i-1"}, want: map[string]interface{}{"issue": eq("i-1")}},
		{name: "team", filter: &ListFilter{TeamID: "t-1"}, want: map[string]interface{}{"team": eq("t-1")}},
		{
			name:   "all with query",
			filter: &ListFilter{ProjectID: "p-1", IssueID: "i-1", TeamID: "t-1", Query: "run"},
			want: map[string]interface{}{
				"project": eq("p-1"),
				"issue":   eq("i-1"),
				"team":    eq("t-1"),
				"title":   map[string]interface{}{"containsIgnoreCase": "run"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFilter(tt.filter)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildCreateInput(t *testing.T) {
	content := ""
	got := buildCreateInput(&CreateInput{Title: "Spec", Content: &content, TeamID: "t-1"})
	want := map[string]interface{}{"title": "Spec", "content": "", "teamId": "t-1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildCreateInput() = %v, want %v", got, want)
	}

	got = buildCreateInput(&CreateInput{Title: "Spec", ProjectID: "p-1", Icon: "Rocket"})
	if _, ok := got["content"]; ok {
		t.Error("nil content must be omitted")
	}
	if got["projectId"] != "p-1" || got["icon"] != "Rocket" {
		t.Errorf("unexpected map: %v", got)
	}
}

func TestBuildUpdateInput(t *testing.T) {
	str := func(s string) *string { return &s }

	t.Run("nil pointers omitted", func(t *testing.T) {
		got := buildUpdateInput(&UpdateInput{Title: str("New")})
		want := map[string]interface{}{"title": "New"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("empty parent id becomes null", func(t *testing.T) {
		got := buildUpdateInput(&UpdateInput{ProjectID: str("p-1"), IssueID: str(""), TeamID: str("")})
		if got["projectId"] != "p-1" {
			t.Errorf("projectId = %v", got["projectId"])
		}
		if v, ok := got["issueId"]; !ok || v != nil {
			t.Errorf("issueId should be present and nil, got %v (present=%v)", v, ok)
		}
		if v, ok := got["teamId"]; !ok || v != nil {
			t.Errorf("teamId should be present and nil, got %v (present=%v)", v, ok)
		}
	})

	t.Run("empty content is kept", func(t *testing.T) {
		got := buildUpdateInput(&UpdateInput{Content: str("")})
		if v, ok := got["content"]; !ok || v != "" {
			t.Errorf("content should be empty string, got %v", v)
		}
	})

	t.Run("nil input", func(t *testing.T) {
		if got := buildUpdateInput(nil); len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})
}
