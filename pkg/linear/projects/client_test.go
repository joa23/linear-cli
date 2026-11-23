package projects

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

func TestBuildInitiativeProjectFilter(t *testing.T) {
	someID := func(id string) map[string]interface{} {
		return map[string]interface{}{"some": map[string]interface{}{"id": map[string]interface{}{"eq": id}}}
	}

	got := buildInitiativeProjectFilter("ini-1", "")
	want := map[string]interface{}{"initiatives": someID("ini-1")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("without team: got %v, want %v", got, want)
	}

	got = buildInitiativeProjectFilter("ini-1", "team-1")
	want = map[string]interface{}{"initiatives": someID("ini-1"), "accessibleTeams": someID("team-1")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("with team: got %v, want %v", got, want)
	}
}

func TestListByInitiative(t *testing.T) {
	var gotVars map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Variables map[string]interface{} `json:"variables"`
		}
		_ = json.Unmarshal(body, &req)
		gotVars = req.Variables
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"projects": {"nodes": [{"id": "p-1", "name": "Auth Revamp", "description": "", "state": "started"}]}}}`))
	}))
	defer server.Close()

	pc := NewClient(core.NewTestBaseClient("test-token", server.URL, server.Client()))

	if _, err := pc.ListByInitiative("", "", 10); !core.IsValidationError(err) {
		t.Errorf("expected validation error for empty initiative, got %v", err)
	}

	projects, err := pc.ListByInitiative("ini-1", "", 0)
	if err != nil {
		t.Fatalf("ListByInitiative: %v", err)
	}
	if len(projects) != 1 || projects[0].Name != "Auth Revamp" {
		t.Errorf("unexpected projects: %+v", projects)
	}
	if gotVars["first"] != float64(50) {
		t.Errorf("default limit should be 50, got %v", gotVars["first"])
	}
	if _, ok := gotVars["filter"].(map[string]interface{})["initiatives"]; !ok {
		t.Errorf("filter should include initiatives: %v", gotVars["filter"])
	}
}

func TestProjectInitiatives_Deserialization(t *testing.T) {
	var p core.Project
	err := json.Unmarshal([]byte(`{"id": "p-1", "name": "Auth", "initiatives": {"nodes": [{"id": "ini-1", "slugId": "3c7a2b1d9e0f", "name": "Q1 Platform", "status": "Active"}]}}`), &p)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if inis := p.GetInitiatives(); len(inis) != 1 || inis[0].Name != "Q1 Platform" {
		t.Errorf("unexpected initiatives: %+v", inis)
	}
	if (&core.Project{}).GetInitiatives() != nil {
		t.Error("nil connection should return nil")
	}
}
