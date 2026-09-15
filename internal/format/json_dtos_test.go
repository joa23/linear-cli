package format

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// emptyCollectionKeys are the JSON keys that must render as `[]` rather than
// `null` when the issue carries none of that collection.
//
// comments is included even though create never requests it, so a created issue
// reports "comments": [] for a field the server was never asked about. That is
// the accepted cost of a uniform contract: the DTO describes the shape it
// renders, not the wire response it was built from.
var emptyCollectionKeys = []string{"labels", "children", "attachments", "comments"}

// minimalIssue is an issue with every collection left unset — the shape a freshly
// created, label-less issue arrives in.
func minimalIssue() *core.Issue {
	return &core.Issue{
		ID:         "issue-uuid",
		Identifier: "TL-1",
		Title:      "Test issue",
		URL:        "https://linear.app/team/issue/TL-1",
	}
}

// marshalJSON renders a DTO and returns the raw string, so tests can assert on
// the wire shape rather than on the Go value.
func marshalJSON(t *testing.T, v interface{}) string {
	t.Helper()

	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal DTO: %v", err)
	}
	return string(out)
}

func TestIssueToFullDTO_EmptyCollectionsRenderAsArrays(t *testing.T) {
	got := marshalJSON(t, IssueToFullDTO(minimalIssue()))

	for _, key := range emptyCollectionKeys {
		t.Run(key, func(t *testing.T) {
			if !strings.Contains(got, `"`+key+`":[]`) {
				t.Errorf("expected %q to render as an empty array\ngot: %s", key, got)
			}
			if strings.Contains(got, `"`+key+`":null`) {
				// A null collection is ambiguous between "none" and "not reported",
				// which is the defect TL-572 fixed.
				t.Errorf("%q rendered as null\ngot: %s", key, got)
			}
		})
	}
}

func TestIssueToDetailedDTO_EmptyCollectionsRenderAsArrays(t *testing.T) {
	// IssueDetailedDTO shares populateIssueBase, so it must behave identically.
	got := marshalJSON(t, IssueToDetailedDTO(minimalIssue()))

	for _, key := range emptyCollectionKeys {
		t.Run(key, func(t *testing.T) {
			if !strings.Contains(got, `"`+key+`":[]`) {
				t.Errorf("expected %q to render as an empty array\ngot: %s", key, got)
			}
			if strings.Contains(got, `"`+key+`":null`) {
				t.Errorf("%q rendered as null\ngot: %s", key, got)
			}
		})
	}
}

func TestIssueToFullDTO_NilAndEmptyLabelConnectionsBothRenderAsArray(t *testing.T) {
	// populateIssueBase collapses "no connection" and "connection with no nodes"
	// into one branch; both must reach the same wire shape.
	tests := []struct {
		name  string
		setup func(*core.Issue)
	}{
		{"nil connection", func(i *core.Issue) { i.Labels = nil }},
		{"empty nodes", func(i *core.Issue) { i.Labels = &core.LabelConnection{Nodes: []core.Label{}} }},
		{"nil nodes", func(i *core.Issue) { i.Labels = &core.LabelConnection{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := minimalIssue()
			tt.setup(issue)

			got := marshalJSON(t, IssueToFullDTO(issue))
			if !strings.Contains(got, `"labels":[]`) {
				t.Errorf("expected labels to render as [], got: %s", got)
			}
		})
	}
}

func TestIssueToFullDTO_LabelsReportIDAndNameOnly(t *testing.T) {
	issue := minimalIssue()
	issue.Labels = &core.LabelConnection{
		Nodes: []core.Label{
			{ID: "label-1", Name: "Bugfix", Color: "#eb5757"},
			{
				ID:     "label-2",
				Name:   "iOS",
				Color:  "#0f7488",
				Parent: &core.LabelRef{ID: "label-parent", Name: "Platform"},
			},
		},
	}

	dto := IssueToFullDTO(issue)

	// Order is legitimate to assert here: this test supplies the core.Issue, and
	// the DTO loop preserves slice order.
	if len(dto.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(dto.Labels))
	}
	if dto.Labels[0].ID != "label-1" || dto.Labels[0].Name != "Bugfix" {
		t.Errorf("labels[0] = {%s %s}, want {label-1 Bugfix}", dto.Labels[0].ID, dto.Labels[0].Name)
	}
	if dto.Labels[1].ID != "label-2" || dto.Labels[1].Name != "iOS" {
		t.Errorf("labels[1] = {%s %s}, want {label-2 iOS}", dto.Labels[1].ID, dto.Labels[1].Name)
	}

	// LabelDTO deliberately bounds the output surface: color and parent are
	// selected from the API but must not reach the rendered JSON.
	//
	// Asserted against the labels elements' own key sets rather than by scanning
	// the whole document, so the test keeps guarding what it means to guard if
	// minimalIssue() ever gains a field whose text contains "color" or "Platform".
	got := marshalJSON(t, dto)
	for i, keys := range unmarshalKeySets(t, got, "labels") {
		for key := range keys {
			if key != "id" && key != "name" {
				t.Errorf("labels[%d] leaked %q — LabelDTO should expose only id and name\ngot: %s", i, key, got)
			}
		}
	}
}

func TestIssueToFullDTO_NilDelegateOmitsTheKey(t *testing.T) {
	// Guards the deliberate asymmetry documented on populateIssueBase against a
	// later "cleanup" that harmonises Delegate with the collections.
	got := marshalJSON(t, IssueToFullDTO(minimalIssue()))

	// Checked against the top-level key set, not the raw document, so unrelated
	// content that happens to spell "delegate" cannot fail this test.
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(got), &top); err != nil {
		t.Fatalf("DTO output is not a JSON object: %v\n%s", err, got)
	}
	if _, present := top["delegate"]; present {
		t.Errorf("expected the delegate key to be omitted entirely for a nil delegate\ngot: %s", got)
	}
}

// unmarshalKeySets returns the key set of every element of the named array field,
// so tests can assert on a field's actual shape instead of substring-scanning the
// whole marshalled document.
func unmarshalKeySets(t *testing.T, doc, field string) []map[string]json.RawMessage {
	t.Helper()

	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(doc), &top); err != nil {
		t.Fatalf("DTO output is not a JSON object: %v\n%s", err, doc)
	}

	raw, present := top[field]
	if !present {
		t.Fatalf("DTO output has no %q key\n%s", field, doc)
	}

	var elements []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &elements); err != nil {
		t.Fatalf("%q is not an array of objects: %v\n%s", field, err, doc)
	}
	return elements
}
