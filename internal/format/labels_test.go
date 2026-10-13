package format

import (
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

func TestFormatLabels_GroupsAndSortsIndependentOfAPIOrder(t *testing.T) {
	labels := []core.Label{
		{ID: "child-z", Name: "Zulu", Color: "#z", Description: "last", Parent: &core.LabelRef{ID: "group-2", Name: "Priority"}},
		{ID: "standalone", Name: "Bug", Color: "#b", Description: "standalone"},
		{ID: "child-a", Name: "Alpha", Color: "#a", Parent: &core.LabelRef{ID: "group-2", Name: "Priority"}},
		{ID: "group-2", Name: "Priority", Color: "#p", Description: "choose one"},
		{ID: "child-c", Name: "Gamma", Color: "#g", Parent: &core.LabelRef{ID: "missing", Name: "Severity"}},
	}

	got := FormatLabels(labels, LabelListOptions{IncludeIDs: true})
	if strings.Count(got, "GROUP: Priority") != 1 {
		t.Fatalf("expected one priority group header, got:\n%s", got)
	}
	if strings.Index(got, "    Alpha") > strings.Index(got, "    Zulu") {
		t.Errorf("children should be sorted: \n%s", got)
	}
	if strings.Index(got, "GROUP: Priority") > strings.Index(got, "GROUP: Severity") {
		t.Errorf("groups should be sorted: \n%s", got)
	}
	for _, want := range []string{"group-2", "child-a", "last", "standalone", "missing"} {
		if !strings.Contains(got, want) {
			t.Errorf("grouped output missing %q:\n%s", want, got)
		}
	}
}

func TestFormatLabels_CompactPreservesDescriptionsAndOmitsIDs(t *testing.T) {
	labels := []core.Label{
		{ID: "parent", Name: "Type", Color: "#fff"},
		{ID: "child", Name: "Feature", Color: "#000", Description: "a feature", Parent: &core.LabelRef{ID: "parent", Name: "Type"}},
	}

	got := New().Labels(labels)
	if strings.Contains(got, "parent") || strings.Contains(got, "child") {
		t.Errorf("compact output should omit IDs:\n%s", got)
	}
	if !strings.Contains(got, "GROUP: Type") || !strings.Contains(got, "    Feature [#000]") {
		t.Errorf("compact output should show group and child:\n%s", got)
	}
	if !strings.Contains(got, "      a feature") {
		t.Errorf("child description should be retained:\n%s", got)
	}
}

func TestFormatLabels_Empty(t *testing.T) {
	if got := FormatLabels(nil, LabelListOptions{}); got != "No labels found." {
		t.Errorf("unexpected empty output: %q", got)
	}
}
