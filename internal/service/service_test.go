package service

import (
	"testing"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
)

func TestNew(t *testing.T) {
	// Create a mock client (just need non-nil for structure tests)
	client := &linear.Client{}

	services := New(client)

	if services == nil {
		t.Fatal("New() returned nil")
	}
	if services.Issues == nil {
		t.Error("Issues service is nil")
	}
	if services.Projects == nil {
		t.Error("Projects service is nil")
	}
	if services.Cycles == nil {
		t.Error("Cycles service is nil")
	}
	if services.Teams == nil {
		t.Error("Teams service is nil")
	}
	if services.Users == nil {
		t.Error("Users service is nil")
	}
}

func TestNewIssueService(t *testing.T) {
	client := &linear.Client{}
	formatter := format.New()

	svc := NewIssueService(client, formatter)

	if svc == nil {
		t.Fatal("NewIssueService() returned nil")
	}
	if svc.client != client {
		t.Error("client not set correctly")
	}
	if svc.formatter != formatter {
		t.Error("formatter not set correctly")
	}
}

func TestNewProjectService(t *testing.T) {
	client := &linear.Client{}
	formatter := format.New()

	svc := NewProjectService(client, formatter)

	if svc == nil {
		t.Fatal("NewProjectService() returned nil")
	}
}

func TestNewCycleService(t *testing.T) {
	client := &linear.Client{}
	formatter := format.New()

	svc := NewCycleService(client, formatter)

	if svc == nil {
		t.Fatal("NewCycleService() returned nil")
	}
}

func TestNewTeamService(t *testing.T) {
	client := &linear.Client{}
	formatter := format.New()

	svc := NewTeamService(client, formatter)

	if svc == nil {
		t.Fatal("NewTeamService() returned nil")
	}
}

func TestNewUserService(t *testing.T) {
	client := &linear.Client{}
	formatter := format.New()

	svc := NewUserService(client, formatter)

	if svc == nil {
		t.Fatal("NewUserService() returned nil")
	}
}

func TestSearchFilters_Defaults(t *testing.T) {
	filters := &SearchFilters{}

	// Check that default values can be set
	if filters.Limit != 0 {
		t.Error("Limit should default to 0 (caller sets default)")
	}
	if filters.Format != "" {
		t.Error("Format should default to empty (caller sets default)")
	}
}

func TestCreateIssueInput_Validation(t *testing.T) {
	input := &CreateIssueInput{
		Title:  "Test Issue",
		TeamID: "TEAM-123",
	}

	if input.Title != "Test Issue" {
		t.Error("Title not set correctly")
	}
	if input.TeamID != "TEAM-123" {
		t.Error("TeamID not set correctly")
	}
}

func TestCycleFilters_Fields(t *testing.T) {
	active := true
	filters := &CycleFilters{
		TeamID:   "team-123",
		IsActive: &active,
		Limit:    10,
		Format:   format.Compact,
	}

	if filters.TeamID != "team-123" {
		t.Error("TeamID not set correctly")
	}
	if *filters.IsActive != true {
		t.Error("IsActive not set correctly")
	}
	if filters.Limit != 10 {
		t.Error("Limit not set correctly")
	}
	if filters.Format != format.Compact {
		t.Error("Format not set correctly")
	}
}

func TestUserFilters_Fields(t *testing.T) {
	activeOnly := true
	filters := &UserFilters{
		TeamID:     "team-123",
		ActiveOnly: &activeOnly,
		Limit:      50,
		After:      "cursor123",
	}

	if filters.TeamID != "team-123" {
		t.Error("TeamID not set correctly")
	}
	if *filters.ActiveOnly != true {
		t.Error("ActiveOnly not set correctly")
	}
	if filters.Limit != 50 {
		t.Error("Limit not set correctly")
	}
	if filters.After != "cursor123" {
		t.Error("After not set correctly")
	}
}

func TestUpdateIssueInput_Fields(t *testing.T) {
	title := "Updated Title"
	priority := 2

	input := &UpdateIssueInput{
		Title:    &title,
		Priority: &priority,
	}

	if *input.Title != "Updated Title" {
		t.Error("Title not set correctly")
	}
	if *input.Priority != 2 {
		t.Error("Priority not set correctly")
	}
}

func TestCreateProjectInput_Fields(t *testing.T) {
	input := &CreateProjectInput{
		Name:        "Test Project",
		Description: "A test project",
		TeamID:      "team-123",
	}

	if input.Name != "Test Project" {
		t.Error("Name not set correctly")
	}
	if input.Description != "A test project" {
		t.Error("Description not set correctly")
	}
	if input.TeamID != "team-123" {
		t.Error("TeamID not set correctly")
	}
}

func TestUpdateProjectInput_Fields(t *testing.T) {
	desc := "Updated description"
	state := "completed"

	input := &UpdateProjectInput{
		Description: &desc,
		State:       &state,
	}

	if *input.Description != "Updated description" {
		t.Error("Description not set correctly")
	}
	if *input.State != "completed" {
		t.Error("State not set correctly")
	}
}

func TestCreateCycleInput_Fields(t *testing.T) {
	input := &CreateCycleInput{
		TeamID:   "team-123",
		Name:     "Sprint 1",
		StartsAt: "2025-01-15",
		EndsAt:   "2025-01-28",
	}

	if input.TeamID != "team-123" {
		t.Error("TeamID not set correctly")
	}
	if input.Name != "Sprint 1" {
		t.Error("Name not set correctly")
	}
	if input.StartsAt != "2025-01-15" {
		t.Error("StartsAt not set correctly")
	}
	if input.EndsAt != "2025-01-28" {
		t.Error("EndsAt not set correctly")
	}
}

func TestAnalyzeInput_Fields(t *testing.T) {
	input := &AnalyzeInput{
		TeamID:                "team-123",
		CycleCount:            10,
		AssigneeID:            "user-123",
		IncludeRecommendation: true,
	}

	if input.TeamID != "team-123" {
		t.Error("TeamID not set correctly")
	}
	if input.CycleCount != 10 {
		t.Error("CycleCount not set correctly")
	}
	if input.AssigneeID != "user-123" {
		t.Error("AssigneeID not set correctly")
	}
	if !input.IncludeRecommendation {
		t.Error("IncludeRecommendation not set correctly")
	}
}
