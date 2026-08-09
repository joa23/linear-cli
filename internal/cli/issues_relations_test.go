package cli

import (
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// relatedIssue builds an IssueMinimal; State is an anonymous struct, so it has
// to be filled in after construction rather than in a literal.
func relatedIssue(identifier, state, title string) *core.IssueMinimal {
	issue := &core.IssueMinimal{Identifier: identifier, Title: title}
	issue.State.Name = state
	return issue
}

// blocker is a relation as it appears on InverseRelations: the other issue
// blocks the queried one.
func blocker(relType core.IssueRelationType, issue *core.IssueMinimal) core.IssueRelation {
	return core.IssueRelation{Type: relType, Issue: issue}
}

// blocked is a relation as it appears on Relations: the queried issue blocks
// the other one.
func blocked(relType core.IssueRelationType, issue *core.IssueMinimal) core.IssueRelation {
	return core.IssueRelation{Type: relType, RelatedIssue: issue}
}

func TestBlockerLines(t *testing.T) {
	issue := &core.IssueWithRelations{Identifier: "DEV-14"}
	issue.InverseRelations.Nodes = []core.IssueRelation{
		blocker(core.RelationBlocks, relatedIssue("DEV-12", "Backlog", "Metadata via library writer")),
		blocker(core.RelationBlocks, relatedIssue("DEV-9", "Done", "Set up the test workspace")),
	}

	lines := blockerLines(issue)
	want := []string{
		"DEV-12 [Backlog] Metadata via library writer",
		"DEV-9 [Done] Set up the test workspace",
	}
	assertLines(t, lines, want)
}

func TestBlockedLines(t *testing.T) {
	issue := &core.IssueWithRelations{Identifier: "DEV-14"}
	issue.Relations.Nodes = []core.IssueRelation{
		blocked(core.RelationBlocks, relatedIssue("DEV-11", "Backlog", "Decide what happens on partial failure")),
	}

	assertLines(t, blockedLines(issue), []string{"DEV-11 [Backlog] Decide what happens on partial failure"})
}

// Linear stores related/duplicate/similar on the same two connections. None of
// them means "blocks", so reporting them would invent blockers that don't exist.
func TestRelationLines_IgnoreNonBlockingTypes(t *testing.T) {
	issue := &core.IssueWithRelations{Identifier: "DEV-14"}
	issue.InverseRelations.Nodes = []core.IssueRelation{
		blocker(core.RelationRelated, relatedIssue("DEV-2", "Backlog", "Related work")),
		blocker(core.RelationDuplicate, relatedIssue("DEV-3", "Backlog", "Duplicate")),
		blocker(core.RelationBlocks, relatedIssue("DEV-12", "Backlog", "Real blocker")),
	}
	issue.Relations.Nodes = []core.IssueRelation{
		blocked(core.RelationRelated, relatedIssue("DEV-4", "Backlog", "Related work")),
	}

	assertLines(t, blockerLines(issue), []string{"DEV-12 [Backlog] Real blocker"})
	assertLines(t, blockedLines(issue), nil)
}

// The two directions must not be crossed: a blocker read off the wrong
// connection turns "blocked by" into "blocking" and inverts the answer.
func TestRelationLines_DirectionsDoNotLeak(t *testing.T) {
	issue := &core.IssueWithRelations{Identifier: "DEV-14"}
	issue.InverseRelations.Nodes = []core.IssueRelation{
		blocker(core.RelationBlocks, relatedIssue("DEV-12", "Backlog", "Blocks DEV-14")),
	}
	issue.Relations.Nodes = []core.IssueRelation{
		blocked(core.RelationBlocks, relatedIssue("DEV-11", "Backlog", "Blocked by DEV-14")),
	}

	assertLines(t, blockerLines(issue), []string{"DEV-12 [Backlog] Blocks DEV-14"})
	assertLines(t, blockedLines(issue), []string{"DEV-11 [Backlog] Blocked by DEV-14"})
}

// An issue with no relations at all must come back empty rather than panicking
// on the nil connections.
func TestRelationLines_NoRelations(t *testing.T) {
	issue := &core.IssueWithRelations{Identifier: "DEV-11"}

	assertLines(t, blockerLines(issue), nil)
	assertLines(t, blockedLines(issue), nil)
}

// A relation whose issue side is absent carries no identifier to print.
func TestRelationLines_SkipsNilIssues(t *testing.T) {
	issue := &core.IssueWithRelations{Identifier: "DEV-14"}
	issue.InverseRelations.Nodes = []core.IssueRelation{
		blocker(core.RelationBlocks, nil),
		blocker(core.RelationBlocks, relatedIssue("DEV-12", "Backlog", "Real blocker")),
	}
	// A relations-side node with only the inverse field set is equally unusable.
	issue.Relations.Nodes = []core.IssueRelation{
		blocker(core.RelationBlocks, relatedIssue("DEV-12", "Backlog", "Wrong side")),
	}

	assertLines(t, blockerLines(issue), []string{"DEV-12 [Backlog] Real blocker"})
	assertLines(t, blockedLines(issue), nil)
}

func TestRelationLine_TruncatesLongTitles(t *testing.T) {
	long := strings.Repeat("x", relationTitleWidth+20)
	line := relationLine(relatedIssue("DEV-1", "Backlog", long))

	title := strings.TrimPrefix(line, "DEV-1 [Backlog] ")
	if len(title) != relationTitleWidth {
		t.Errorf("title width = %d, want %d (line: %q)", len(title), relationTitleWidth, line)
	}
	if !strings.HasSuffix(title, "...") {
		t.Errorf("truncated title should end in an ellipsis, got %q", title)
	}
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d lines %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}
