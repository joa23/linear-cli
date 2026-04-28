package teams

import (
	"encoding/json"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// TestListLabelsResponseStruct_ParentField verifies that the ListLabels response
// struct correctly deserializes Parent on core.Label: non-nil when present, nil
// when absent.
func TestListLabelsResponseStruct_ParentField(t *testing.T) {
	// Simulated GraphQL response for GetTeamLabels
	graphqlResponse := `{
		"team": {
			"labels": {
				"nodes": [
					{
						"id": "label-child-1",
						"name": "Bug",
						"color": "#ff0000",
						"description": "A bug report",
						"parent": {
							"id": "label-parent-1",
							"name": "Type"
						}
					},
					{
						"id": "label-orphan-1",
						"name": "Urgent",
						"color": "#ff9900",
						"description": "Needs immediate attention"
					}
				]
			}
		}
	}`

	var response struct {
		Team *struct {
			Labels struct {
				Nodes []core.Label `json:"nodes"`
			} `json:"labels"`
		} `json:"team"`
	}

	if err := json.Unmarshal([]byte(graphqlResponse), &response); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if response.Team == nil {
		t.Fatal("Team is nil")
	}

	nodes := response.Team.Labels.Nodes
	if len(nodes) != 2 {
		t.Fatalf("expected 2 label nodes, got %d", len(nodes))
	}

	// First label has a parent
	first := nodes[0]
	if first.Parent == nil {
		t.Fatal("nodes[0].Parent is nil, want non-nil")
	}
	if first.Parent.ID != "label-parent-1" {
		t.Errorf("nodes[0].Parent.ID = %q, want %q", first.Parent.ID, "label-parent-1")
	}
	if first.Parent.Name != "Type" {
		t.Errorf("nodes[0].Parent.Name = %q, want %q", first.Parent.Name, "Type")
	}

	// Second label has no parent
	second := nodes[1]
	if second.Parent != nil {
		t.Errorf("nodes[1].Parent = %+v, want nil", second.Parent)
	}
}
