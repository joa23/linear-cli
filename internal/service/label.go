package service

import (
	"encoding/json"
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
	"github.com/joa23/linear-cli/internal/linear/core"
)

// LabelService handles label CRUD operations
type LabelService struct {
	client    *linear.Client
	formatter *format.Formatter
}

// NewLabelService creates a new LabelService
func NewLabelService(client *linear.Client, formatter *format.Formatter) *LabelService {
	return &LabelService{
		client:    client,
		formatter: formatter,
	}
}

// List returns labels for a team
func (s *LabelService) List(teamID string, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	// Resolve team identifier
	resolvedTeamID, err := s.client.ResolveTeamIdentifier(teamID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", teamID, err)
	}

	labels, err := s.client.Teams.ListLabels(resolvedTeamID)
	if err != nil {
		return "", fmt.Errorf("failed to list labels: %w", err)
	}

	if len(labels) == 0 {
		return "No labels found.", nil
	}

	// JSON output
	if outputType.IsJSON() {
		data, err := json.MarshalIndent(labels, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal labels: %w", err)
		}
		return string(data), nil
	}

	// Text output
	output := fmt.Sprintf("LABELS (%d)\n────────────────────────────────────────\n", len(labels))
	for _, label := range labels {
		output += fmt.Sprintf("  %-30s %s  %s\n", label.Name, label.Color, label.ID)
		if label.Description != "" {
			output += fmt.Sprintf("    %s\n", label.Description)
		}
	}

	return output, nil
}

// errLabelMutationRequiresUser is returned when label mutations are attempted in agent mode
var errLabelMutationRequiresUser = fmt.Errorf("label create/update/delete requires user auth (linear auth login as user). OAuth app actors cannot manage labels")

// Create creates a new label
func (s *LabelService) Create(input *core.CreateLabelInput) (string, error) {
	if s.client.IsAgentMode() {
		return "", errLabelMutationRequiresUser
	}

	// Resolve team identifier
	resolvedTeamID, err := s.client.ResolveTeamIdentifier(input.TeamID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", input.TeamID, err)
	}
	input.TeamID = resolvedTeamID

	label, err := s.client.Teams.CreateLabel(input)
	if err != nil {
		return "", fmt.Errorf("failed to create label: %w", err)
	}

	return fmt.Sprintf("Created label: %s [%s] (%s)", label.Name, label.Color, label.ID), nil
}

// Update updates an existing label
func (s *LabelService) Update(id string, input *core.UpdateLabelInput) (string, error) {
	if s.client.IsAgentMode() {
		return "", errLabelMutationRequiresUser
	}

	label, err := s.client.Teams.UpdateLabel(id, input)
	if err != nil {
		return "", fmt.Errorf("failed to update label: %w", err)
	}

	return fmt.Sprintf("Updated label: %s [%s] (%s)", label.Name, label.Color, label.ID), nil
}

// Delete deletes a label
func (s *LabelService) Delete(id string) (string, error) {
	if s.client.IsAgentMode() {
		return "", errLabelMutationRequiresUser
	}

	err := s.client.Teams.DeleteLabel(id)
	if err != nil {
		return "", fmt.Errorf("failed to delete label: %w", err)
	}

	return fmt.Sprintf("Deleted label: %s", id), nil
}
