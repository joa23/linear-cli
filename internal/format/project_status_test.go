package format

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

func TestProjectStatusOutputKeepsNamedStatusAndStateAlias(t *testing.T) {
	project := &core.Project{
		ID:    "p1",
		Name:  "Planning",
		State: "started",
		Status: &core.ProjectStatus{
			ID:   "s1",
			Name: "On Hold",
			Type: "started",
		},
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte((&JSONRenderer{}).RenderProject(project, VerbosityCompact)), &output); err != nil {
		t.Fatalf("invalid project JSON: %v", err)
	}
	if output["state"] != "started" {
		t.Fatalf("state = %v, want started", output["state"])
	}
	status, ok := output["status"].(map[string]interface{})
	if !ok || status["name"] != "On Hold" {
		t.Fatalf("status = %#v, want named status", output["status"])
	}

	text := (&TextRenderer{}).RenderProject(project, VerbosityCompact)
	if !strings.Contains(text, "State: On Hold") {
		t.Fatalf("text = %q, want named status", text)
	}
}

func TestProjectStatusOutputFallsBackForLegacyProject(t *testing.T) {
	project := &core.Project{Name: "Legacy", State: "started"}
	text := (&TextRenderer{}).RenderProject(project, VerbosityCompact)
	if !strings.Contains(text, "State: started") {
		t.Fatalf("text = %q, want legacy state fallback", text)
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte((&JSONRenderer{}).RenderProject(project, VerbosityCompact)), &output); err != nil {
		t.Fatalf("invalid project JSON: %v", err)
	}
	if _, ok := output["status"]; ok {
		t.Fatal("legacy project unexpectedly contains status")
	}
}
