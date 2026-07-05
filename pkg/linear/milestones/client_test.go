package milestones

import (
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

func strPtr(s string) *string { return &s }

func TestBuildCreateInput(t *testing.T) {
	tests := []struct {
		name              string
		input             *core.CreateProjectMilestoneInput
		expectDescription bool
		expectTargetDate  bool
	}{
		{
			name:              "required fields only",
			input:             &core.CreateProjectMilestoneInput{Name: "Beta", ProjectID: "project-1"},
			expectDescription: false,
			expectTargetDate:  false,
		},
		{
			name: "with description and target date",
			input: &core.CreateProjectMilestoneInput{
				Name:        "Beta",
				ProjectID:   "project-1",
				Description: "Public beta",
				TargetDate:  "2026-08-01",
			},
			expectDescription: true,
			expectTargetDate:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCreateInput(tt.input)

			if result["name"] != tt.input.Name {
				t.Errorf("name = %v, want %v", result["name"], tt.input.Name)
			}
			if result["projectId"] != tt.input.ProjectID {
				t.Errorf("projectId = %v, want %v", result["projectId"], tt.input.ProjectID)
			}

			_, hasDescription := result["description"]
			if hasDescription != tt.expectDescription {
				t.Errorf("description presence = %v, want %v", hasDescription, tt.expectDescription)
			}

			_, hasTargetDate := result["targetDate"]
			if hasTargetDate != tt.expectTargetDate {
				t.Errorf("targetDate presence = %v, want %v", hasTargetDate, tt.expectTargetDate)
			}
		})
	}
}

func TestBuildUpdateInput(t *testing.T) {
	tests := []struct {
		name              string
		input             *core.UpdateProjectMilestoneInput
		expectName        bool
		expectDescription bool
		expectTargetDate  bool
	}{
		{
			name:       "no fields set",
			input:      &core.UpdateProjectMilestoneInput{},
			expectName: false,
		},
		{
			name:       "rename only",
			input:      &core.UpdateProjectMilestoneInput{Name: strPtr("Private beta")},
			expectName: true,
		},
		{
			name: "all fields set",
			input: &core.UpdateProjectMilestoneInput{
				Name:        strPtr("Private beta"),
				Description: strPtr("Updated description"),
				TargetDate:  strPtr("2026-08-15"),
			},
			expectName:        true,
			expectDescription: true,
			expectTargetDate:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildUpdateInput(tt.input)

			_, hasName := result["name"]
			if hasName != tt.expectName {
				t.Errorf("name presence = %v, want %v", hasName, tt.expectName)
			}

			_, hasDescription := result["description"]
			if hasDescription != tt.expectDescription {
				t.Errorf("description presence = %v, want %v", hasDescription, tt.expectDescription)
			}

			_, hasTargetDate := result["targetDate"]
			if hasTargetDate != tt.expectTargetDate {
				t.Errorf("targetDate presence = %v, want %v", hasTargetDate, tt.expectTargetDate)
			}
		})
	}
}

func TestList_ValidationError(t *testing.T) {
	client := NewClient(nil)

	_, err := client.List("", 50)
	if err == nil {
		t.Fatal("expected validation error for empty projectID")
	}
	if _, ok := err.(*core.ValidationError); !ok {
		t.Errorf("expected *core.ValidationError, got %T", err)
	}
}

func TestGet_ValidationError(t *testing.T) {
	client := NewClient(nil)

	_, err := client.Get("")
	if err == nil {
		t.Fatal("expected validation error for empty id")
	}
	if _, ok := err.(*core.ValidationError); !ok {
		t.Errorf("expected *core.ValidationError, got %T", err)
	}
}

func TestCreate_ValidationErrors(t *testing.T) {
	client := NewClient(nil)

	tests := []struct {
		name  string
		input *core.CreateProjectMilestoneInput
	}{
		{"nil input", nil},
		{"empty name", &core.CreateProjectMilestoneInput{ProjectID: "project-1"}},
		{"empty projectID", &core.CreateProjectMilestoneInput{Name: "Beta"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Create(tt.input)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if _, ok := err.(*core.ValidationError); !ok {
				t.Errorf("expected *core.ValidationError, got %T", err)
			}
		})
	}
}

func TestUpdate_ValidationErrors(t *testing.T) {
	client := NewClient(nil)

	t.Run("empty id", func(t *testing.T) {
		_, err := client.Update("", &core.UpdateProjectMilestoneInput{Name: strPtr("Beta")})
		if err == nil {
			t.Fatal("expected validation error for empty id")
		}
	})

	t.Run("nil input", func(t *testing.T) {
		_, err := client.Update("milestone-1", nil)
		if err == nil {
			t.Fatal("expected validation error for nil input")
		}
	})

	t.Run("no fields set", func(t *testing.T) {
		_, err := client.Update("milestone-1", &core.UpdateProjectMilestoneInput{})
		if err == nil {
			t.Fatal("expected validation error when no fields are set")
		}
	})
}

func TestDelete_ValidationError(t *testing.T) {
	client := NewClient(nil)

	err := client.Delete("")
	if err == nil {
		t.Fatal("expected validation error for empty id")
	}
	if _, ok := err.(*core.ValidationError); !ok {
		t.Errorf("expected *core.ValidationError, got %T", err)
	}
}
