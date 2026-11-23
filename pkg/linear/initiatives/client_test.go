package initiatives

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// graphQLRequest is the decoded body the client sends.
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

// newTestClient starts an httptest server that answers each request with the next
// response in order and records the requests it received.
func newTestClient(t *testing.T, responses ...string) (*Client, *[]graphQLRequest) {
	t.Helper()
	var requests []graphQLRequest
	i := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req graphQLRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		requests = append(requests, req)
		if i >= len(responses) {
			t.Errorf("unexpected request #%d: %s", i+1, req.Query)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[i]))
		i++
	}))
	t.Cleanup(server.Close)
	base := core.NewTestBaseClient("test-token", server.URL, server.Client())
	return NewClient(base), &requests
}

func TestListInitiatives_Deserialization(t *testing.T) {
	graphqlResponse := `{
		"initiatives": {
			"nodes": [
				{
					"id": "ini-1",
					"slugId": "3c7a2b1d9e0f",
					"name": "Q1 Platform",
					"description": "Stabilize",
					"url": "https://linear.app/acme/initiative/q1-platform-3c7a2b1d9e0f",
					"status": "Active",
					"health": "atRisk",
					"targetDate": "2026-03-31",
					"createdAt": "2026-01-01T10:00:00Z",
					"updatedAt": "2026-01-16T10:00:00Z",
					"owner": {"id": "u-1", "name": "Alice", "displayName": "alice", "email": "a@x.io"},
					"parentInitiative": null,
					"projects": {"nodes": [{"id": "p-1"}, {"id": "p-2"}]}
				},
				{
					"id": "ini-2",
					"slugId": "aaaaaaaaaaaa",
					"name": "Later",
					"url": "https://linear.app/acme/initiative/later-aaaaaaaaaaaa",
					"status": "Planned",
					"health": null,
					"targetDate": null,
					"description": null,
					"createdAt": "2026-01-01T10:00:00Z",
					"updatedAt": "2026-01-02T10:00:00Z",
					"owner": null,
					"parentInitiative": {"id": "ini-1", "slugId": "3c7a2b1d9e0f", "name": "Q1 Platform"},
					"projects": {"nodes": []}
				}
			],
			"pageInfo": {"hasNextPage": true, "endCursor": "cur-1"}
		}
	}`

	var response struct {
		Initiatives struct {
			Nodes    []core.Initiative `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"initiatives"`
	}
	if err := json.Unmarshal([]byte(graphqlResponse), &response); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	nodes := response.Initiatives.Nodes
	if len(nodes) != 2 {
		t.Fatalf("expected 2 initiatives, got %d", len(nodes))
	}
	first := nodes[0]
	if first.Status != "Active" || first.Health != "atRisk" || first.SlugID != "3c7a2b1d9e0f" {
		t.Errorf("unexpected first initiative: %+v", first)
	}
	if first.TargetDate == nil || *first.TargetDate != "2026-03-31" {
		t.Error("expected targetDate")
	}
	if first.Owner == nil || first.Owner.DisplayName != "alice" {
		t.Error("expected owner")
	}
	if len(first.GetProjects()) != 2 {
		t.Errorf("expected 2 project ids, got %d", len(first.GetProjects()))
	}
	second := nodes[1]
	if second.Owner != nil || second.Health != "" || second.TargetDate != nil || second.Description != "" {
		t.Errorf("nulls should decode to zero values: %+v", second)
	}
	if second.ParentInitiative == nil || second.ParentInitiative.Name != "Q1 Platform" {
		t.Error("expected parent initiative")
	}
	if !response.Initiatives.PageInfo.HasNextPage || response.Initiatives.PageInfo.EndCursor != "cur-1" {
		t.Error("expected pagination info")
	}
}

func TestGetInitiative_Deserialization(t *testing.T) {
	graphqlResponse := `{
		"initiative": {
			"id": "ini-1",
			"slugId": "3c7a2b1d9e0f",
			"name": "Q1 Platform",
			"content": "# Brief",
			"url": "https://linear.app/acme/initiative/q1-platform-3c7a2b1d9e0f",
			"status": "Active",
			"health": "onTrack",
			"createdAt": "2026-01-01T10:00:00Z",
			"updatedAt": "2026-01-16T10:00:00Z",
			"archivedAt": null,
			"creator": {"id": "u-2", "name": "Bob", "displayName": "bob"},
			"subInitiatives": {"nodes": [{"id": "ini-3", "slugId": "bbbbbbbbbbbb", "name": "Auth", "status": "Planned"}]},
			"projects": {"nodes": [
				{"id": "p-1", "name": "Auth Revamp", "slugId": "111111111111", "url": "https://linear.app/acme/project/auth-111111111111",
				 "progress": 0.42, "targetDate": "2026-02-15", "status": {"name": "In Progress", "type": "started"},
				 "lead": {"id": "u-1", "name": "Alice", "displayName": "alice"}}
			]},
			"initiativeUpdates": {"nodes": [
				{"id": "up-1", "body": "All good", "health": "onTrack", "url": "https://linear.app/x", "createdAt": "2026-01-15T10:00:00Z",
				 "updatedAt": "2026-01-15T10:00:00Z", "editedAt": null, "user": {"id": "u-1", "name": "Alice", "displayName": "alice"}}
			]}
		}
	}`

	var response struct {
		Initiative core.Initiative `json:"initiative"`
	}
	if err := json.Unmarshal([]byte(graphqlResponse), &response); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	ini := response.Initiative
	if ini.Content != "# Brief" || ini.Creator == nil || ini.Creator.DisplayName != "bob" {
		t.Errorf("unexpected initiative: %+v", ini)
	}
	if subs := ini.GetSubInitiatives(); len(subs) != 1 || subs[0].Name != "Auth" {
		t.Errorf("unexpected sub-initiatives: %+v", subs)
	}
	projects := ini.GetProjects()
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	p := projects[0]
	if p.Progress != 0.42 || p.Status == nil || p.Status.Name != "In Progress" || p.Lead == nil {
		t.Errorf("unexpected project: %+v", p)
	}
	updates := ini.GetUpdates()
	if len(updates) != 1 || updates[0].Health != "onTrack" || updates[0].User == nil {
		t.Errorf("unexpected updates: %+v", updates)
	}
}

func TestNilConnectionsAreSafe(t *testing.T) {
	ini := core.Initiative{}
	if ini.GetProjects() != nil || ini.GetSubInitiatives() != nil || ini.GetUpdates() != nil {
		t.Error("nil connections should return nil slices")
	}
}

func TestBuildFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter *ListFilter
		want   map[string]interface{}
	}{
		{name: "nil", filter: nil, want: nil},
		{name: "empty", filter: &ListFilter{}, want: nil},
		{name: "archived only is not a filter", filter: &ListFilter{IncludeArchived: true}, want: nil},
		{name: "status", filter: &ListFilter{Status: "Active"}, want: map[string]interface{}{"status": map[string]interface{}{"eq": "Active"}}},
		{
			name:   "all",
			filter: &ListFilter{Status: "Planned", OwnerID: "u-1", Query: "plat"},
			want: map[string]interface{}{
				"status": map[string]interface{}{"eq": "Planned"},
				"owner":  map[string]interface{}{"id": map[string]interface{}{"eq": "u-1"}},
				"name":   map[string]interface{}{"containsIgnoreCase": "plat"},
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
	got := buildCreateInput(&CreateInput{Name: "Q1", Content: &content, OwnerID: "u-1", Status: "Active", TargetDate: "2026-03-31"})
	want := map[string]interface{}{"name": "Q1", "content": "", "ownerId": "u-1", "status": "Active", "targetDate": "2026-03-31"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildCreateInput() = %v, want %v", got, want)
	}

	got = buildCreateInput(&CreateInput{Name: "Q1", Description: "d", Icon: "Rocket", Color: "#fff"})
	if _, ok := got["content"]; ok {
		t.Error("nil content must be omitted")
	}
	if got["description"] != "d" || got["icon"] != "Rocket" || got["color"] != "#fff" {
		t.Errorf("unexpected map: %v", got)
	}
	for _, k := range []string{"ownerId", "status", "targetDate"} {
		if _, ok := got[k]; ok {
			t.Errorf("empty %s must be omitted", k)
		}
	}
}

func TestBuildUpdateInput(t *testing.T) {
	str := func(s string) *string { return &s }

	t.Run("nil pointers omitted", func(t *testing.T) {
		got := buildUpdateInput(&UpdateInput{Status: str("Completed")})
		want := map[string]interface{}{"status": "Completed"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("empty owner and target date become null", func(t *testing.T) {
		got := buildUpdateInput(&UpdateInput{OwnerID: str(""), TargetDate: str("")})
		for _, k := range []string{"ownerId", "targetDate"} {
			if v, ok := got[k]; !ok || v != nil {
				t.Errorf("%s should be present and nil, got %v (present=%v)", k, v, ok)
			}
		}
	})

	t.Run("values are kept", func(t *testing.T) {
		got := buildUpdateInput(&UpdateInput{OwnerID: str("u-1"), TargetDate: str("2026-04-01"), Content: str("")})
		if got["ownerId"] != "u-1" || got["targetDate"] != "2026-04-01" {
			t.Errorf("unexpected map: %v", got)
		}
		if v, ok := got["content"]; !ok || v != "" {
			t.Errorf("empty content should be kept, got %v", v)
		}
	})

	t.Run("nil input", func(t *testing.T) {
		if got := buildUpdateInput(nil); len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})
}

func TestBuildStatusUpdateInput(t *testing.T) {
	got := buildStatusUpdateInput(&StatusUpdateInput{InitiativeID: "ini-1", Body: "b", Health: "atRisk"})
	want := map[string]interface{}{"initiativeId": "ini-1", "body": "b", "health": "atRisk"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	got = buildStatusUpdateInput(&StatusUpdateInput{InitiativeID: "ini-1", Body: "b"})
	if _, ok := got["health"]; ok {
		t.Error("empty health must be omitted")
	}
}

func TestValidation(t *testing.T) {
	c := NewClient(nil) // validation must fail before any network call
	if _, err := c.GetInitiative(""); !core.IsValidationError(err) {
		t.Errorf("GetInitiative: expected validation error, got %v", err)
	}
	if _, err := c.CreateInitiative(&CreateInput{}); !core.IsValidationError(err) {
		t.Errorf("CreateInitiative: expected validation error, got %v", err)
	}
	if _, err := c.UpdateInitiative("ini-1", &UpdateInput{}); !core.IsValidationError(err) {
		t.Errorf("UpdateInitiative: expected validation error for empty input, got %v", err)
	}
	if _, err := c.AddProject("ini-1", ""); !core.IsValidationError(err) {
		t.Errorf("AddProject: expected validation error, got %v", err)
	}
	if err := c.RemoveProjectLink(""); !core.IsValidationError(err) {
		t.Errorf("RemoveProjectLink: expected validation error, got %v", err)
	}
	if _, err := c.CreateUpdate(&StatusUpdateInput{}); !core.IsValidationError(err) {
		t.Errorf("CreateUpdate: expected validation error, got %v", err)
	}
}

func TestListInitiatives_Request(t *testing.T) {
	c, reqs := newTestClient(t, `{"data": {"initiatives": {"nodes": [], "pageInfo": {"hasNextPage": false, "endCursor": ""}}}}`)

	if _, err := c.ListInitiatives(&ListFilter{Status: "Active", IncludeArchived: true}, 10); err != nil {
		t.Fatalf("ListInitiatives: %v", err)
	}
	req := (*reqs)[0]
	if req.Variables["first"] != float64(10) || req.Variables["includeArchived"] != true {
		t.Errorf("unexpected variables: %v", req.Variables)
	}
	if _, ok := req.Variables["filter"]; !ok {
		t.Error("expected filter variable")
	}
}

func TestListInitiatives_NoFilterOmitsVariables(t *testing.T) {
	c, reqs := newTestClient(t, `{"data": {"initiatives": {"nodes": [], "pageInfo": {"hasNextPage": false}}}}`)
	if _, err := c.ListInitiatives(nil, 0); err != nil {
		t.Fatalf("ListInitiatives: %v", err)
	}
	vars := (*reqs)[0].Variables
	if vars["first"] != float64(50) {
		t.Errorf("default limit should be 50, got %v", vars["first"])
	}
	if _, ok := vars["filter"]; ok {
		t.Error("filter must be omitted when empty")
	}
	if _, ok := vars["includeArchived"]; ok {
		t.Error("includeArchived must be omitted when false")
	}
}

func TestGetInitiativeBySlug_NotFound(t *testing.T) {
	c, _ := newTestClient(t, `{"data": {"initiatives": {"nodes": []}}}`)
	_, err := c.GetInitiativeBySlug("3c7a2b1d9e0f")
	if !core.IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %v", err)
	}
}

func TestGetInitiative_GraphQLError(t *testing.T) {
	c, _ := newTestClient(t, `{"errors": [{"message": "Entity not found: Initiative"}]}`)
	_, err := c.GetInitiative("ini-1")
	if err == nil || !strings.Contains(err.Error(), "Entity not found") {
		t.Errorf("expected entity not found error, got %v", err)
	}
}

func TestFindProjectLink(t *testing.T) {
	links := `{"data": {"project": {"initiativeToProjects": {"nodes": [
		{"id": "link-a", "initiative": {"id": "ini-other"}},
		{"id": "link-b", "initiative": {"id": "ini-1"}}
	]}}}}`

	t.Run("found", func(t *testing.T) {
		c, reqs := newTestClient(t, links)
		id, err := c.FindProjectLink("ini-1", "p-1")
		if err != nil {
			t.Fatalf("FindProjectLink: %v", err)
		}
		if id != "link-b" {
			t.Errorf("got %q, want link-b", id)
		}
		if (*reqs)[0].Variables["projectId"] != "p-1" {
			t.Errorf("unexpected variables: %v", (*reqs)[0].Variables)
		}
	})

	t.Run("not linked", func(t *testing.T) {
		c, _ := newTestClient(t, links)
		_, err := c.FindProjectLink("ini-missing", "p-1")
		if !core.IsNotFoundError(err) {
			t.Errorf("expected NotFoundError, got %v", err)
		}
	})
}

func TestAddProjectAndRemoveLink(t *testing.T) {
	c, reqs := newTestClient(t,
		`{"data": {"initiativeToProjectCreate": {"success": true, "initiativeToProject": {"id": "link-1", "initiative": {"id": "ini-1", "name": "Q1"}, "project": {"id": "p-1", "name": "Auth"}}}}}`,
		`{"data": {"initiativeToProjectDelete": {"success": true}}}`,
	)

	link, err := c.AddProject("ini-1", "p-1")
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if link.ID != "link-1" || link.Project == nil || link.Project.Name != "Auth" {
		t.Errorf("unexpected link: %+v", link)
	}
	input, _ := (*reqs)[0].Variables["input"].(map[string]interface{})
	if input["initiativeId"] != "ini-1" || input["projectId"] != "p-1" {
		t.Errorf("unexpected input: %v", input)
	}

	if err := c.RemoveProjectLink("link-1"); err != nil {
		t.Fatalf("RemoveProjectLink: %v", err)
	}
	if (*reqs)[1].Variables["id"] != "link-1" {
		t.Errorf("unexpected delete variables: %v", (*reqs)[1].Variables)
	}
}

func TestMutationSuccessFalse(t *testing.T) {
	c, _ := newTestClient(t,
		`{"data": {"initiativeArchive": {"success": false}}}`,
		`{"data": {"initiativeDelete": {"success": false}}}`,
	)
	if err := c.ArchiveInitiative("ini-1"); err == nil {
		t.Error("expected error when archive returns success=false")
	}
	if err := c.DeleteInitiative("ini-1"); err == nil {
		t.Error("expected error when delete returns success=false")
	}
}

func TestCreateUpdate_Request(t *testing.T) {
	c, reqs := newTestClient(t, `{"data": {"initiativeUpdateCreate": {"success": true, "initiativeUpdate": {
		"id": "up-1", "body": "Shipped", "health": "onTrack", "createdAt": "2026-01-15T10:00:00Z",
		"user": {"id": "u-1", "name": "Alice"}, "initiative": {"id": "ini-1", "name": "Q1"}}}}}`)

	up, err := c.CreateUpdate(&StatusUpdateInput{InitiativeID: "ini-1", Body: "Shipped", Health: "onTrack"})
	if err != nil {
		t.Fatalf("CreateUpdate: %v", err)
	}
	if up.ID != "up-1" || up.Initiative == nil || up.Initiative.Name != "Q1" {
		t.Errorf("unexpected update: %+v", up)
	}
	input, _ := (*reqs)[0].Variables["input"].(map[string]interface{})
	if input["health"] != "onTrack" || input["body"] != "Shipped" {
		t.Errorf("unexpected input: %v", input)
	}
}

func TestListUpdates_Request(t *testing.T) {
	c, reqs := newTestClient(t, `{"data": {"initiative": {"initiativeUpdates": {"nodes": [
		{"id": "up-2", "body": "b2", "health": "atRisk", "createdAt": "2026-01-20T10:00:00Z"},
		{"id": "up-1", "body": "b1", "health": "onTrack", "createdAt": "2026-01-10T10:00:00Z"}
	]}}}}`)

	updates, err := c.ListUpdates("ini-1", 0)
	if err != nil {
		t.Fatalf("ListUpdates: %v", err)
	}
	if len(updates) != 2 || updates[0].ID != "up-2" {
		t.Errorf("unexpected updates: %+v", updates)
	}
	if (*reqs)[0].Variables["first"] != float64(10) {
		t.Errorf("default limit should be 10, got %v", (*reqs)[0].Variables["first"])
	}
}
