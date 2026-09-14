package service

import (
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// TestConvertIssueDetails_CarriesDueDateEstimateDelegate is a regression test
// for TL-563: ListAllIssues's IssueWithDetails carried DueDate/Estimate/Delegate,
// but convertIssueDetails (used by ListAssignedWithPagination) dropped them
// when converting to core.Issue, which would have silently undone the fix one
// layer up. The fixture below starts with all three fields already populated
// (not both-nil, which would pass trivially) to catch a half-applied fix.
func TestConvertIssueDetails_CarriesDueDateEstimateDelegate(t *testing.T) {
	estimate := 3.5
	dueDate := "2026-09-01"
	details := []core.IssueWithDetails{
		{
			ID:         "issue-1",
			Identifier: "TL-1",
			Title:      "Issue with due date, estimate, and delegate",
			Priority:   2,
			Estimate:   &estimate,
			DueDate:    &dueDate,
			Delegate:   &core.User{ID: "delegate-1", Name: "Bot", Email: "bot@example.com"},
			CreatedAt:  "2026-01-01T00:00:00Z",
			UpdatedAt:  "2026-01-02T00:00:00Z",
		},
	}

	issues := convertIssueDetails(details)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	got := issues[0]

	if got.Estimate == nil || *got.Estimate != estimate {
		t.Errorf("Estimate = %v, want %v", got.Estimate, estimate)
	}
	if got.DueDate == nil || *got.DueDate != dueDate {
		t.Errorf("DueDate = %v, want %v", got.DueDate, dueDate)
	}
	if got.Delegate == nil || got.Delegate.ID != "delegate-1" {
		t.Errorf("Delegate = %v, want ID \"delegate-1\"", got.Delegate)
	}
}

// TestConvertIssueDetails_OmitsUnsetDueDateEstimateDelegate confirms issues
// without these fields set continue to convert to nil, not false positives.
func TestConvertIssueDetails_OmitsUnsetDueDateEstimateDelegate(t *testing.T) {
	details := []core.IssueWithDetails{
		{
			ID:         "issue-2",
			Identifier: "TL-2",
			Title:      "Issue without due date, estimate, or delegate",
			Priority:   1,
			CreatedAt:  "2026-01-01T00:00:00Z",
			UpdatedAt:  "2026-01-02T00:00:00Z",
		},
	}

	issues := convertIssueDetails(details)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	got := issues[0]

	if got.Estimate != nil {
		t.Errorf("Estimate = %v, want nil", got.Estimate)
	}
	if got.DueDate != nil {
		t.Errorf("DueDate = %v, want nil", got.DueDate)
	}
	if got.Delegate != nil {
		t.Errorf("Delegate = %v, want nil", got.Delegate)
	}
}
