package issues

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// newCreateTestClient wires a Client to an httptest server so tests can inspect
// the outgoing GraphQL mutation and control the response.
//
// Why this exists: the service-layer create tests mock the whole issue client and
// hand back a pre-built core.Issue, so they never see the mutation string at all.
// Deleting a field from the issueCreate selection set would leave every one of
// those tests green while the CLI silently reported `null` again — which is the
// exact defect TL-572 fixed. Only a test that captures the real request body can
// fail on that regression.
func newCreateTestClient(t *testing.T, handler func(w http.ResponseWriter, body []byte)) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		handler(w, body)
	}))
	t.Cleanup(server.Close)

	return NewClient(core.NewTestBaseClient("test-token", server.URL, server.Client()))
}

// graphQLQuery pulls the "query" field out of a captured request body.
func graphQLQuery(t *testing.T, body []byte) string {
	t.Helper()

	var payload struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	return payload.Query
}

// minimalCreateResponse is a valid issueCreate payload with no optional fields set.
const minimalCreateResponse = `{
	"data": {
		"issueCreate": {
			"success": true,
			"issue": {
				"id": "issue-uuid",
				"identifier": "TL-1",
				"title": "Test issue"
			}
		}
	}
}`

func TestCreateIssue_MutationRequestsCallerSettableFields(t *testing.T) {
	// Each of these fields is something the caller passes *into* create, so each
	// carries the "I set it, the response says null, did it work?" failure mode.
	//
	// The patterns are deliberately anchored to a whole line: a bare "labels"
	// substring would also match "labelIds" in the mutation input, so a naive
	// check would pass against the unfixed selection set and prove nothing.
	tests := []struct {
		field   string
		pattern *regexp.Regexp
	}{
		{"priority", regexp.MustCompile(`(?m)^\s*priority\s*$`)},
		{"estimate", regexp.MustCompile(`(?m)^\s*estimate\s*$`)},
		{"dueDate", regexp.MustCompile(`(?m)^\s*dueDate\s*$`)},
		{"labels", regexp.MustCompile(`(?m)^\s*labels\s*\{\s*$`)},
		{"cycle", regexp.MustCompile(`(?m)^\s*cycle\s*\{\s*$`)},
	}

	var capturedQuery string
	client := newCreateTestClient(t, func(w http.ResponseWriter, body []byte) {
		capturedQuery = graphQLQuery(t, body)
		_, _ = w.Write([]byte(minimalCreateResponse))
	})

	_, err := client.CreateIssue(&core.IssueCreateInput{
		Title:  "Test issue",
		TeamID: "team-uuid",
	})
	if err != nil {
		t.Fatalf("CreateIssue() returned unexpected error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if !tt.pattern.MatchString(capturedQuery) {
				t.Errorf("issueCreate selection set does not request %q\nmutation was:\n%s", tt.field, capturedQuery)
			}
		})
	}

	// The label sub-selection must match the read paths verbatim, so create and
	// get deserialize the same shape.
	for _, sub := range []string{"color", "parent"} {
		if !regexp.MustCompile(`(?m)^\s*` + sub + `[\s{]`).MatchString(capturedQuery) {
			t.Errorf("label sub-selection does not request %q\nmutation was:\n%s", sub, capturedQuery)
		}
	}
}

func TestCreateIssue_DeserializesLabels(t *testing.T) {
	// Two labels, one carrying a parent — asserting id, name, color AND parent
	// proves the deserializer handles the full selected shape, even though
	// LabelDTO later drops color/parent from the rendered JSON.
	const response = `{
		"data": {
			"issueCreate": {
				"success": true,
				"issue": {
					"id": "issue-uuid",
					"identifier": "TL-1",
					"title": "Test issue",
					"labels": {
						"nodes": [
							{"id": "label-1", "name": "Bugfix", "color": "#eb5757"},
							{"id": "label-2", "name": "iOS", "color": "#0f7488",
							 "parent": {"id": "label-parent", "name": "Platform"}}
						]
					}
				}
			}
		}
	}`

	client := newCreateTestClient(t, func(w http.ResponseWriter, body []byte) {
		_, _ = w.Write([]byte(response))
	})

	issue, err := client.CreateIssue(&core.IssueCreateInput{
		Title:    "Test issue",
		TeamID:   "team-uuid",
		LabelIDs: []string{"label-1", "label-2"},
	})
	if err != nil {
		t.Fatalf("CreateIssue() returned unexpected error: %v", err)
	}

	if issue.Labels == nil {
		t.Fatal("issue.Labels is nil — the create response's labels were not deserialized")
	}
	if got := len(issue.Labels.Nodes); got != 2 {
		t.Fatalf("expected 2 labels, got %d", got)
	}

	// Order is asserted only against this test's own canned response. Linear does
	// not document that live responses preserve labelIds order, so the live
	// verification compares labels as a set instead.
	first := issue.Labels.Nodes[0]
	if first.ID != "label-1" || first.Name != "Bugfix" || first.Color != "#eb5757" {
		t.Errorf("first label = {%s %s %s}, want {label-1 Bugfix #eb5757}", first.ID, first.Name, first.Color)
	}
	if first.Parent != nil {
		t.Errorf("first label should have no parent, got %+v", first.Parent)
	}

	second := issue.Labels.Nodes[1]
	if second.ID != "label-2" || second.Name != "iOS" || second.Color != "#0f7488" {
		t.Errorf("second label = {%s %s %s}, want {label-2 iOS #0f7488}", second.ID, second.Name, second.Color)
	}
	if second.Parent == nil {
		t.Fatal("second label's parent was not deserialized")
	}
	if second.Parent.ID != "label-parent" || second.Parent.Name != "Platform" {
		t.Errorf("second label parent = {%s %s}, want {label-parent Platform}", second.Parent.ID, second.Parent.Name)
	}
}

func TestCreateIssue_DeserializesSiblingFields(t *testing.T) {
	const response = `{
		"data": {
			"issueCreate": {
				"success": true,
				"issue": {
					"id": "issue-uuid",
					"identifier": "TL-1",
					"title": "Test issue",
					"priority": 2,
					"estimate": 3,
					"dueDate": "2026-03-01",
					"cycle": {"id": "cycle-uuid", "number": 65, "name": "Sprint 65"}
				}
			}
		}
	}`

	client := newCreateTestClient(t, func(w http.ResponseWriter, body []byte) {
		_, _ = w.Write([]byte(response))
	})

	issue, err := client.CreateIssue(&core.IssueCreateInput{
		Title:  "Test issue",
		TeamID: "team-uuid",
	})
	if err != nil {
		t.Fatalf("CreateIssue() returned unexpected error: %v", err)
	}

	if issue.Priority == nil || *issue.Priority != 2 {
		t.Errorf("Priority = %v, want 2", issue.Priority)
	}
	if issue.Estimate == nil || *issue.Estimate != 3 {
		t.Errorf("Estimate = %v, want 3", issue.Estimate)
	}
	if issue.DueDate == nil || *issue.DueDate != "2026-03-01" {
		t.Errorf("DueDate = %v, want 2026-03-01", issue.DueDate)
	}
	if issue.Cycle == nil {
		t.Fatal("Cycle was not deserialized")
	}
	if issue.Cycle.ID != "cycle-uuid" || issue.Cycle.Number != 65 || issue.Cycle.Name != "Sprint 65" {
		t.Errorf("Cycle = {%s %d %s}, want {cycle-uuid 65 Sprint 65}", issue.Cycle.ID, issue.Cycle.Number, issue.Cycle.Name)
	}
}

func TestCreateIssue_UnsuccessfulMutationReturnsError(t *testing.T) {
	const response = `{
		"data": {
			"issueCreate": {
				"success": false,
				"issue": {"id": "issue-uuid", "identifier": "TL-1", "title": "Test issue"}
			}
		}
	}`

	client := newCreateTestClient(t, func(w http.ResponseWriter, body []byte) {
		_, _ = w.Write([]byte(response))
	})

	issue, err := client.CreateIssue(&core.IssueCreateInput{
		Title:  "Test issue",
		TeamID: "team-uuid",
	})
	if err == nil {
		t.Fatal("CreateIssue() returned no error for an unsuccessful mutation")
	}
	if issue != nil {
		// A partially-populated issue would let a caller act on an issue that was
		// never created.
		t.Errorf("CreateIssue() returned a non-nil issue alongside the error: %+v", issue)
	}
}

func TestCreateIssue_ExtractsMetadataFromDescription(t *testing.T) {
	// Guards the post-mutation metadata handling against the selection-set edit:
	// the description must still be cleaned and its metadata lifted out.
	const response = `{
		"data": {
			"issueCreate": {
				"success": true,
				"issue": {
					"id": "issue-uuid",
					"identifier": "TL-1",
					"title": "Test issue",
					"description": "Real body.\n\n<details><summary>🤖 Metadata</summary>\n\n` + "```" + `json\n{\"depends_on\":[\"TL-9\"]}\n` + "```" + `\n\n</details>"
				}
			}
		}
	}`

	client := newCreateTestClient(t, func(w http.ResponseWriter, body []byte) {
		_, _ = w.Write([]byte(response))
	})

	issue, err := client.CreateIssue(&core.IssueCreateInput{
		Title:  "Test issue",
		TeamID: "team-uuid",
	})
	if err != nil {
		t.Fatalf("CreateIssue() returned unexpected error: %v", err)
	}

	if issue.Description != "Real body." {
		t.Errorf("Description = %q, want %q (metadata section not stripped)", issue.Description, "Real body.")
	}
	if issue.Metadata == nil {
		t.Fatal("Metadata was not extracted from the description")
	}
	deps, ok := issue.Metadata["depends_on"]
	if !ok {
		t.Fatalf("Metadata is missing depends_on: %+v", issue.Metadata)
	}
	list, ok := deps.([]interface{})
	if !ok || len(list) != 1 || list[0] != "TL-9" {
		t.Errorf("depends_on = %v, want [TL-9]", deps)
	}
}
