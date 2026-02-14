package cli

import (
	"encoding/json"
	"testing"
)

func TestRenderDepsJSON_SingleIssue(t *testing.T) {
	nodes := map[string]*DepNode{
		"ENG-100": {ID: "uuid-100", Identifier: "ENG-100", Title: "Parent task", State: "In Progress"},
		"ENG-101": {ID: "uuid-101", Identifier: "ENG-101", Title: "Blocked task", State: "Backlog"},
	}
	edges := []DepEdge{
		{From: "ENG-100", To: "ENG-101", Type: "blocks"},
	}

	result := renderDepsJSON("ENG-100", nodes, edges)

	var parsed DepsGraphJSON
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// rootIssue present in single-issue mode
	if parsed.RootIssue != "ENG-100" {
		t.Errorf("expected rootIssue ENG-100, got %s", parsed.RootIssue)
	}

	if len(parsed.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(parsed.Nodes))
	}

	if len(parsed.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(parsed.Edges))
	}

	if parsed.Edges[0].From != "ENG-100" || parsed.Edges[0].To != "ENG-101" {
		t.Errorf("unexpected edge: %+v", parsed.Edges[0])
	}

	// cycles should be empty array, not nil
	if parsed.Cycles == nil {
		t.Error("cycles should be empty array, not nil")
	}
	if len(parsed.Cycles) != 0 {
		t.Errorf("expected 0 cycles, got %d", len(parsed.Cycles))
	}
}

func TestRenderDepsJSON_TeamMode(t *testing.T) {
	nodes := map[string]*DepNode{
		"ENG-100": {ID: "uuid-100", Identifier: "ENG-100", Title: "Parent task", State: "In Progress"},
		"ENG-101": {ID: "uuid-101", Identifier: "ENG-101", Title: "Child task", State: "Backlog"},
	}
	edges := []DepEdge{
		{From: "ENG-100", To: "ENG-101", Type: "blocks"},
	}

	// Team mode: rootIssue is empty string
	result := renderDepsJSON("", nodes, edges)

	var parsed DepsGraphJSON
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// rootIssue should be omitted (omitempty)
	if parsed.RootIssue != "" {
		t.Errorf("expected empty rootIssue in team mode, got %s", parsed.RootIssue)
	}

	if len(parsed.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(parsed.Nodes))
	}
}

func TestRenderDepsJSON_WithCycles(t *testing.T) {
	nodes := map[string]*DepNode{
		"ENG-100": {ID: "uuid-100", Identifier: "ENG-100", Title: "Issue A", State: "Todo"},
		"ENG-101": {ID: "uuid-101", Identifier: "ENG-101", Title: "Issue B", State: "Todo"},
	}
	edges := []DepEdge{
		{From: "ENG-100", To: "ENG-101", Type: "blocks"},
		{From: "ENG-101", To: "ENG-100", Type: "blocks"},
	}

	result := renderDepsJSON("ENG-100", nodes, edges)

	var parsed DepsGraphJSON
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(parsed.Cycles) == 0 {
		t.Error("expected cycles to be detected")
	}
}

func TestRenderDepsJSON_EmptyGraph(t *testing.T) {
	nodes := map[string]*DepNode{}
	var edges []DepEdge

	result := renderDepsJSON("", nodes, edges)

	var parsed DepsGraphJSON
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if parsed.Nodes == nil {
		t.Error("nodes should be empty array, not nil")
	}
	if parsed.Edges == nil {
		t.Error("edges should be empty array, not nil")
	}
	if parsed.Cycles == nil {
		t.Error("cycles should be empty array, not nil")
	}
}

func TestRenderDepsJSON_NodesSorted(t *testing.T) {
	nodes := map[string]*DepNode{
		"ENG-300": {ID: "uuid-300", Identifier: "ENG-300", Title: "Third", State: "Todo"},
		"ENG-100": {ID: "uuid-100", Identifier: "ENG-100", Title: "First", State: "Todo"},
		"ENG-200": {ID: "uuid-200", Identifier: "ENG-200", Title: "Second", State: "Todo"},
	}
	var edges []DepEdge

	result := renderDepsJSON("", nodes, edges)

	var parsed DepsGraphJSON
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Nodes should be sorted by identifier for deterministic output
	if parsed.Nodes[0].Identifier != "ENG-100" {
		t.Errorf("expected first node ENG-100, got %s", parsed.Nodes[0].Identifier)
	}
	if parsed.Nodes[1].Identifier != "ENG-200" {
		t.Errorf("expected second node ENG-200, got %s", parsed.Nodes[1].Identifier)
	}
	if parsed.Nodes[2].Identifier != "ENG-300" {
		t.Errorf("expected third node ENG-300, got %s", parsed.Nodes[2].Identifier)
	}
}

func TestDepsCmd_HasOutputFlag(t *testing.T) {
	cmd := newDepsCmd()
	flag := cmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("deps command should have --output flag")
	}
	if flag.DefValue != "text" {
		t.Errorf("expected default 'text', got '%s'", flag.DefValue)
	}
	if flag.Shorthand != "o" {
		t.Errorf("expected shorthand 'o', got '%s'", flag.Shorthand)
	}
}
