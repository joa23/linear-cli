package service

import (
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

type mockIssueClientForUpdate struct {
	*mockIssueClientForCreate
	issue       *core.Issue
	teamIDs     map[string]string
	labels      map[string]*core.Label
	updateCalls int
	updateInput core.UpdateIssueInput
}

func (m *mockIssueClientForUpdate) GetIssue(string) (*core.Issue, error) {
	return m.issue, nil
}

func (m *mockIssueClientForUpdate) ResolveTeamIdentifier(key string) (string, error) {
	return m.teamIDs[key], nil
}

func (m *mockIssueClientForUpdate) ResolveLabelMetadata(name, _ string) (*core.Label, error) {
	return m.labels[name], nil
}

func (m *mockIssueClientForUpdate) UpdateIssue(_ string, input core.UpdateIssueInput) (*core.Issue, error) {
	m.updateCalls++
	m.updateInput = input
	return m.issue, nil
}

func makeIssueServiceForUpdate(mock *mockIssueClientForUpdate) *IssueService {
	return NewIssueService(mock, makeIssueServiceForCreate(mock.mockIssueClientForCreate).formatter)
}

func TestIssueService_Update_AddWithUnchangedTeamSucceeds(t *testing.T) {
	mock := newUpdateMock()
	mock.labels["Bug"] = &core.Label{ID: "label-bug", Name: "Bug"}

	_, err := makeIssueServiceForUpdate(mock).Update("TL-1", &UpdateIssueInput{
		TeamID:      stringPtr("TL"),
		AddLabelIDs: []string{"Bug"},
	})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if mock.updateCalls != 1 {
		t.Fatalf("UpdateIssue calls = %d, want 1", mock.updateCalls)
	}
	if len(mock.updateInput.LabelIDs) != 1 || mock.updateInput.LabelIDs[0] != "label-bug" {
		t.Fatalf("label IDs = %#v, want [label-bug]", mock.updateInput.LabelIDs)
	}
}

func TestIssueService_Update_AddWithTeamChangeFailsBeforeMutation(t *testing.T) {
	mock := newUpdateMock()
	mock.teamIDs["NEW"] = "team-2"

	_, err := makeIssueServiceForUpdate(mock).Update("TL-1", &UpdateIssueInput{
		TeamID:      stringPtr("NEW"),
		AddLabelIDs: []string{"Bug"},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot add or remove labels while changing") {
		t.Fatalf("error = %v, want team-change validation", err)
	}
	if mock.updateCalls != 0 {
		t.Fatal("UpdateIssue called despite team-change validation")
	}
}

func TestIssueService_Update_ReplaceWithTeamChangeSucceeds(t *testing.T) {
	mock := newUpdateMock()
	mock.teamIDs["NEW"] = "team-2"
	mock.labels["Bug"] = &core.Label{ID: "label-bug", Name: "Bug"}

	_, err := makeIssueServiceForUpdate(mock).Update("TL-1", &UpdateIssueInput{
		TeamID:   stringPtr("NEW"),
		LabelIDs: []string{"Bug"},
	})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if mock.updateCalls != 1 {
		t.Fatalf("UpdateIssue calls = %d, want 1", mock.updateCalls)
	}
}

func TestIssueService_Update_RemoveOnlySendsExplicitEmptyLabels(t *testing.T) {
	mock := newUpdateMock()

	_, err := makeIssueServiceForUpdate(mock).Update("TL-1", &UpdateIssueInput{
		RemoveLabelIDs: []string{"Bug"},
	})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}
	if mock.updateCalls != 1 {
		t.Fatalf("UpdateIssue calls = %d, want 1", mock.updateCalls)
	}
	if mock.updateInput.LabelIDs == nil || len(mock.updateInput.LabelIDs) != 0 {
		t.Fatalf("label IDs = %#v, want non-nil empty slice", mock.updateInput.LabelIDs)
	}
}

func TestIssueService_Update_AddIncludesExistingConflict(t *testing.T) {
	mock := newUpdateMock()
	mock.issue.Labels = &core.LabelConnection{Nodes: []core.Label{
		{ID: "existing-a", Name: "Alpha", Parent: &core.LabelRef{ID: "group", Name: "Type"}},
	}}
	mock.labels["Beta"] = &core.Label{ID: "existing-b", Name: "Beta", Parent: &core.LabelRef{ID: "group", Name: "Type"}}

	_, err := makeIssueServiceForUpdate(mock).Update("TL-1", &UpdateIssueInput{
		AddLabelIDs: []string{"Beta"},
	})
	if err == nil || !strings.Contains(err.Error(), "exclusive group") {
		t.Fatalf("error = %v, want existing-label conflict", err)
	}
	if mock.updateCalls != 0 {
		t.Fatal("UpdateIssue called despite additive conflict")
	}
}

func TestIssueService_Update_NilResolverResultIsSafe(t *testing.T) {
	mock := newUpdateMock()

	_, err := makeIssueServiceForUpdate(mock).Update("TL-1", &UpdateIssueInput{
		AddLabelIDs: []string{"Missing"},
	})
	if err == nil || !strings.Contains(err.Error(), "resolver returned no label") {
		t.Fatalf("error = %v, want nil resolver validation", err)
	}
	if mock.updateCalls != 0 {
		t.Fatal("UpdateIssue called despite resolver failure")
	}
}

func newUpdateMock() *mockIssueClientForUpdate {
	base := &mockIssueClientForCreate{}
	return &mockIssueClientForUpdate{
		mockIssueClientForCreate: base,
		issue: &core.Issue{
			ID:         "issue-1",
			Identifier: "TL-1",
			Title:      "Issue",
		},
		teamIDs: map[string]string{"TL": "team-1"},
		labels:  map[string]*core.Label{"Bug": {ID: "label-bug", Name: "Bug"}},
	}
}

func stringPtr(value string) *string {
	return &value
}
