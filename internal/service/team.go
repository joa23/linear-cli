package service

import (
	"encoding/json"
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
)

// TeamService handles team-related operations
type TeamService struct {
	client    *linear.Client
	formatter *format.Formatter
}

// NewTeamService creates a new TeamService
func NewTeamService(client *linear.Client, formatter *format.Formatter) *TeamService {
	return &TeamService{
		client:    client,
		formatter: formatter,
	}
}

// Get retrieves a single team by ID or key (legacy method)
func (s *TeamService) Get(identifier string) (string, error) {
	// Try to resolve the identifier first
	teamID, err := s.client.ResolveTeamIdentifier(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", identifier, err)
	}

	team, err := s.client.Teams.GetTeam(teamID)
	if err != nil {
		return "", fmt.Errorf("failed to get team: %w", err)
	}

	return s.formatter.Team(team), nil
}

// GetWithOutput retrieves a single team with new renderer architecture
func (s *TeamService) GetWithOutput(identifier string, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	// Try to resolve the identifier first
	teamID, err := s.client.ResolveTeamIdentifier(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", identifier, err)
	}

	team, err := s.client.Teams.GetTeam(teamID)
	if err != nil {
		return "", fmt.Errorf("failed to get team: %w", err)
	}

	return s.formatter.RenderTeam(team, verbosity, outputType), nil
}

// ListAll lists all teams in the workspace (legacy method)
func (s *TeamService) ListAll() (string, error) {
	teams, err := s.client.GetTeams()
	if err != nil {
		return "", fmt.Errorf("failed to list teams: %w", err)
	}

	return s.formatter.TeamList(teams, nil), nil
}

// ListAllWithOutput lists all teams with new renderer architecture
func (s *TeamService) ListAllWithOutput(verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	teams, err := s.client.GetTeams()
	if err != nil {
		return "", fmt.Errorf("failed to list teams: %w", err)
	}

	return s.formatter.RenderTeamList(teams, verbosity, outputType), nil
}

// GetLabels returns labels for a team (legacy method)
func (s *TeamService) GetLabels(identifier string) (string, error) {
	// Resolve team identifier
	teamID, err := s.client.ResolveTeamIdentifier(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", identifier, err)
	}

	labels, err := s.client.Teams.ListLabels(teamID)
	if err != nil {
		return "", fmt.Errorf("failed to list labels: %w", err)
	}

	if len(labels) == 0 {
		return "No labels found.", nil
	}

	// Format labels as simple list
	output := fmt.Sprintf("LABELS (%d)\n────────────────────────────────────────\n", len(labels))
	for _, label := range labels {
		output += fmt.Sprintf("  %s [%s]\n", label.Name, label.Color)
		if label.Description != "" {
			output += fmt.Sprintf("    %s\n", label.Description)
		}
	}

	return output, nil
}

// GetLabelsWithOutput returns labels for a team with new renderer architecture
func (s *TeamService) GetLabelsWithOutput(identifier string, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	// Resolve team identifier
	teamID, err := s.client.ResolveTeamIdentifier(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", identifier, err)
	}

	labels, err := s.client.Teams.ListLabels(teamID)
	if err != nil {
		return "", fmt.Errorf("failed to list labels: %w", err)
	}

	if len(labels) == 0 {
		return "No labels found.", nil
	}

	if outputType.IsJSON() {
		data, err := json.MarshalIndent(labels, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal labels: %w", err)
		}
		return string(data), nil
	}

	// Format labels as simple list
	output := fmt.Sprintf("LABELS (%d)\n────────────────────────────────────────\n", len(labels))
	for _, label := range labels {
		output += fmt.Sprintf("  %s [%s]\n", label.Name, label.Color)
		if label.Description != "" {
			output += fmt.Sprintf("    %s\n", label.Description)
		}
	}

	return output, nil
}

// GetWorkflowStates returns workflow states for a team (legacy method)
func (s *TeamService) GetWorkflowStates(identifier string) (string, error) {
	// Resolve team identifier
	teamID, err := s.client.ResolveTeamIdentifier(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", identifier, err)
	}

	states, err := s.client.GetWorkflowStates(teamID)
	if err != nil {
		return "", fmt.Errorf("failed to list workflow states: %w", err)
	}

	if len(states) == 0 {
		return "No workflow states found.", nil
	}

	// Format states
	output := fmt.Sprintf("WORKFLOW STATES (%d)\n────────────────────────────────────────\n", len(states))
	for _, state := range states {
		output += fmt.Sprintf("  %s [%s] - %s\n", state.Name, state.Type, state.ID)
	}

	return output, nil
}

// GetWorkflowStatesWithOutput returns workflow states with new renderer architecture
func (s *TeamService) GetWorkflowStatesWithOutput(identifier string, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	// Resolve team identifier
	teamID, err := s.client.ResolveTeamIdentifier(identifier)
	if err != nil {
		return "", fmt.Errorf("failed to resolve team '%s': %w", identifier, err)
	}

	states, err := s.client.GetWorkflowStates(teamID)
	if err != nil {
		return "", fmt.Errorf("failed to list workflow states: %w", err)
	}

	if len(states) == 0 {
		return "No workflow states found.", nil
	}

	if outputType.IsJSON() {
		data, err := json.MarshalIndent(states, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal states: %w", err)
		}
		return string(data), nil
	}

	// Format states
	output := fmt.Sprintf("WORKFLOW STATES (%d)\n────────────────────────────────────────\n", len(states))
	for _, state := range states {
		output += fmt.Sprintf("  %s [%s] - %s\n", state.Name, state.Type, state.ID)
	}

	return output, nil
}
