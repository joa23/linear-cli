package format

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

func strPtr(s string) *string { return &s }

func sampleInitiative() *core.Initiative {
	return &core.Initiative{
		ID:          "965bab1e-b1a0-4353-943e-ec14571cc1d2",
		SlugID:      "3c7a2b1d9e0f",
		Name:        "Q1 Platform",
		Description: "Stabilize the platform",
		Content:     strings.Repeat("x", 600),
		URL:         "https://linear.app/acme/initiative/q1-platform-3c7a2b1d9e0f",
		Status:      "Active",
		Health:      "atRisk",
		TargetDate:  strPtr("2026-03-31"),
		CreatedAt:   "2026-01-01T10:00:00Z",
		UpdatedAt:   "2026-01-16T10:00:00Z",
		Owner:       &core.User{ID: "u-1", Name: "Alice Smith", DisplayName: "alice"},
		Creator:     &core.User{ID: "u-2", Name: "Bob", DisplayName: "bob"},
		ParentInitiative: &core.Initiative{
			ID: "ini-0", SlugID: "000000000000", Name: "FY26",
		},
		SubInitiatives: &core.InitiativeConnection{Nodes: []core.Initiative{
			{ID: "ini-3", SlugID: "bbbbbbbbbbbb", Name: "Auth", Status: "Planned"},
		}},
		Projects: &core.InitiativeProjectConnection{Nodes: []core.InitiativeProject{
			{
				ID: "p-1", Name: "Auth Revamp", Progress: 0.426, TargetDate: strPtr("2026-02-15"),
				Status: &core.ProjectStatusSummary{Name: "In Progress", Type: "started"},
				Lead:   &core.User{ID: "u-1", Name: "Alice Smith", DisplayName: "alice"},
			},
			{ID: "p-2", Name: "Billing v2"},
		}},
		Updates: &core.InitiativeUpdateConnection{Nodes: []core.InitiativeUpdate{
			{ID: "up-1", Body: strings.Repeat("~", 300), Health: "onTrack", CreatedAt: "2026-01-15T10:00:00Z",
				User: &core.User{ID: "u-1", Name: "Alice Smith", DisplayName: "alice"}},
		}},
	}
}

func TestTextRenderer_RenderInitiative_Verbosity(t *testing.T) {
	r := &TextRenderer{}
	ini := sampleInitiative()

	t.Run("nil", func(t *testing.T) {
		if got := r.RenderInitiative(nil, VerbosityFull); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("minimal", func(t *testing.T) {
		out := r.RenderInitiative(ini, VerbosityMinimal)
		if !strings.Contains(out, "Q1 Platform") || !strings.Contains(out, "ID: 965bab1e") {
			t.Errorf("minimal missing name/id: %q", out)
		}
		if strings.Contains(out, "Status:") || strings.Contains(out, "PROJECTS") {
			t.Errorf("minimal must stay minimal: %q", out)
		}
	})

	t.Run("compact", func(t *testing.T) {
		out := r.RenderInitiative(ini, VerbosityCompact)
		for _, want := range []string{"Slug: 3c7a2b1d9e0f", "Status: Active, At risk", "Owner: alice", "Target: 2026-03-31",
			"Parent: FY26", "Projects: 2", "Updated: 2026-01-16", "URL: https://"} {
			if !strings.Contains(out, want) {
				t.Errorf("compact missing %q: %q", want, out)
			}
		}
		if strings.Contains(out, "xxxx") || strings.Contains(out, "PROJECTS (") || strings.Contains(out, "Created:") {
			t.Errorf("compact must not include content/projects table/created: %q", out)
		}
	})

	t.Run("detailed", func(t *testing.T) {
		out := r.RenderInitiative(ini, VerbosityDetailed)
		for _, want := range []string{
			"Created: 2026-01-01 by bob",
			"Description: Stabilize the platform",
			"PROJECTS (2)",
			"Auth Revamp [In Progress] 43%  target 2026-02-15  @alice",
			"Billing v2 [-] 0%",
			"SUB-INITIATIVES (1)",
			"bbbbbbbbbbbb  Auth  [Planned]",
			"RECENT UPDATES (1)",
			"[On track] 2026-01-15 by alice",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("detailed missing %q:\n%s", want, out)
			}
		}
		if strings.Count(out, "x") >= 600 || strings.Count(out, "~") >= 300 {
			t.Error("detailed should truncate content and update bodies")
		}
	})

	t.Run("full", func(t *testing.T) {
		out := r.RenderInitiative(ini, VerbosityFull)
		if strings.Count(out, "x") != 600 {
			t.Errorf("full should include all content, got %d x", strings.Count(out, "x"))
		}
		if strings.Count(out, "~") != 300 {
			t.Errorf("full should include full update body, got %d ~", strings.Count(out, "~"))
		}
	})

	t.Run("archived and no health", func(t *testing.T) {
		i := sampleInitiative()
		i.Health = ""
		i.ArchivedAt = strPtr("2026-02-01T10:00:00Z")
		out := r.RenderInitiative(i, VerbosityCompact)
		if !strings.Contains(out, "Archived: 2026-02-01") || !strings.Contains(out, "Status: Active\n") {
			t.Errorf("unexpected output: %q", out)
		}
	})

	t.Run("list-shaped projects are counted, not tabulated", func(t *testing.T) {
		i := sampleInitiative()
		i.Projects = &core.InitiativeProjectConnection{Nodes: []core.InitiativeProject{{ID: "p-1"}}}
		out := r.RenderInitiative(i, VerbosityDetailed)
		if strings.Contains(out, "PROJECTS (") || !strings.Contains(out, "Projects: 1") {
			t.Errorf("unexpected output: %q", out)
		}
	})
}

func TestTextRenderer_RenderInitiativeList(t *testing.T) {
	r := &TextRenderer{}

	t.Run("empty", func(t *testing.T) {
		if got := r.RenderInitiativeList(nil, VerbosityCompact, nil); got != "No initiatives found.\n" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("rows", func(t *testing.T) {
		list := []core.Initiative{
			*sampleInitiative(),
			{ID: "abcdefgh-1234", Name: "No slug", Status: "Planned"},
		}
		out := r.RenderInitiativeList(list, VerbosityCompact, &Pagination{HasNextPage: true, EndCursor: "cur-1"})
		for _, want := range []string{
			"INITIATIVES (2)",
			"3c7a2b1d9e0f  Q1 Platform  [Active, At risk]  @alice  target 2026-03-31  2 projects",
			"abcdefgh  No slug  [Planned]\n",
			"Next: cursor=cur-1",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("minimal rows", func(t *testing.T) {
		out := r.RenderInitiativeList([]core.Initiative{*sampleInitiative()}, VerbosityMinimal, nil)
		if !strings.Contains(out, "3c7a2b1d9e0f  Q1 Platform  [Active]\n") || strings.Contains(out, "@alice") {
			t.Errorf("unexpected minimal row: %q", out)
		}
	})

	t.Run("detailed renders blocks", func(t *testing.T) {
		out := r.RenderInitiativeList([]core.Initiative{*sampleInitiative()}, VerbosityDetailed, nil)
		if !strings.Contains(out, "Owner: alice") {
			t.Errorf("detailed list should render blocks: %q", out)
		}
	})
}

func TestTextRenderer_InitiativeUpdates(t *testing.T) {
	r := &TextRenderer{}
	updates := sampleInitiative().GetUpdates()

	if got := r.RenderInitiativeUpdateList(nil, VerbosityCompact); got != "No initiative updates found.\n" {
		t.Errorf("got %q", got)
	}

	out := r.RenderInitiativeUpdateList(updates, VerbosityMinimal)
	if !strings.Contains(out, "INITIATIVE UPDATES (1)") || strings.Contains(out, "~~~") {
		t.Errorf("minimal should list headers only: %q", out)
	}

	single := updates[0]
	single.Initiative = &core.Initiative{Name: "Q1 Platform"}
	single.URL = "https://linear.app/acme/initiative/x/updates#u"
	out = r.RenderInitiativeUpdate(&single, VerbosityFull)
	for _, want := range []string{"Initiative: Q1 Platform", "[On track]", "URL: https://"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q: %q", want, out)
		}
	}
	if r.RenderInitiativeUpdate(nil, VerbosityFull) != "" {
		t.Error("nil update should render empty")
	}
}

func TestHealthLabel(t *testing.T) {
	for in, want := range map[string]string{"onTrack": "On track", "atRisk": "At risk", "offTrack": "Off track", "": "No health", "weird": "weird"} {
		if got := healthLabel(in); got != want {
			t.Errorf("healthLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJSONRenderer_RenderInitiative(t *testing.T) {
	r := &JSONRenderer{}

	t.Run("nil", func(t *testing.T) {
		if out := r.RenderInitiative(nil, VerbosityFull); !strings.Contains(out, "error") {
			t.Errorf("expected error JSON, got %q", out)
		}
	})

	t.Run("single includes details", func(t *testing.T) {
		var dto InitiativeDTO
		if err := json.Unmarshal([]byte(r.RenderInitiative(sampleInitiative(), VerbosityCompact)), &dto); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if dto.Content == "" || dto.ProjectCount != 2 || len(dto.Projects) != 2 || len(dto.SubInitiatives) != 1 || len(dto.Updates) != 1 {
			t.Errorf("unexpected dto: %+v", dto)
		}
		p := dto.Projects[0]
		if p.Status != "In Progress" || p.StatusType != "started" || p.Progress != 0.426 || p.Lead == nil {
			t.Errorf("unexpected project dto: %+v", p)
		}
		if dto.Owner == nil || dto.Owner.DisplayName != "alice" || dto.ParentInitiative == nil || dto.ParentInitiative.Name != "FY26" {
			t.Errorf("missing refs: %+v", dto)
		}
		if dto.TargetDate == nil || *dto.TargetDate != "2026-03-31" {
			t.Error("expected targetDate")
		}
	})

	t.Run("list omits details but keeps project count", func(t *testing.T) {
		out := r.RenderInitiativeList([]core.Initiative{*sampleInitiative()}, VerbosityFull, nil)
		var dtos []InitiativeDTO
		if err := json.Unmarshal([]byte(out), &dtos); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if len(dtos) != 1 || dtos[0].ProjectCount != 2 {
			t.Errorf("unexpected list: %+v", dtos)
		}
		for _, key := range []string{`"content"`, `"projects"`, `"updates"`, `"subInitiatives"`} {
			if strings.Contains(out, key) {
				t.Errorf("list should omit %s", key)
			}
		}
	})

	t.Run("empty lists", func(t *testing.T) {
		if got := r.RenderInitiativeList(nil, VerbosityCompact, nil); got != "[]" {
			t.Errorf("got %q", got)
		}
		if got := r.RenderInitiativeUpdateList(nil, VerbosityCompact); got != "[]" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("update", func(t *testing.T) {
		u := sampleInitiative().GetUpdates()[0]
		u.Initiative = &core.Initiative{ID: "ini-1", Name: "Q1 Platform"}
		var dto InitiativeUpdateDTO
		if err := json.Unmarshal([]byte(r.RenderInitiativeUpdate(&u, VerbosityCompact)), &dto); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if dto.Health != "onTrack" || dto.User == nil || dto.Initiative == nil || dto.Initiative.Name != "Q1 Platform" {
			t.Errorf("unexpected dto: %+v", dto)
		}
	})
}

func TestProjectRendering_Initiatives(t *testing.T) {
	p := &core.Project{
		ID: "p-1", Name: "Auth Revamp", State: "started",
		Initiatives: &core.InitiativeConnection{Nodes: []core.Initiative{
			{ID: "ini-1", SlugID: "3c7a2b1d9e0f", Name: "Q1 Platform", Status: "Active"},
			{ID: "ini-2", Name: "Security"},
		}},
	}

	text := (&TextRenderer{}).RenderProject(p, VerbosityFull)
	if !strings.Contains(text, "Initiatives: Q1 Platform, Security") {
		t.Errorf("text project should list initiatives: %q", text)
	}

	var dto ProjectDTO
	if err := json.Unmarshal([]byte((&JSONRenderer{}).RenderProject(p, VerbosityFull)), &dto); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(dto.Initiatives) != 2 || dto.Initiatives[0].SlugID != "3c7a2b1d9e0f" {
		t.Errorf("unexpected initiatives: %+v", dto.Initiatives)
	}

	p.Initiatives = nil
	if strings.Contains((&TextRenderer{}).RenderProject(p, VerbosityFull), "Initiatives:") {
		t.Error("no initiatives line when none")
	}
	if strings.Contains((&JSONRenderer{}).RenderProject(p, VerbosityFull), `"initiatives"`) {
		t.Error("initiatives omitted from JSON when none")
	}
}

func TestFormatter_RenderInitiative_Dispatch(t *testing.T) {
	f := New()
	ini := sampleInitiative()
	if text := f.RenderInitiative(ini, VerbosityCompact, OutputText); !strings.HasPrefix(text, "Q1 Platform\n") {
		t.Errorf("text output unexpected: %q", text)
	}
	if out := f.RenderInitiativeList([]core.Initiative{*ini}, VerbosityCompact, OutputJSON, nil); !strings.HasPrefix(out, "[") {
		t.Errorf("json output unexpected: %q", out)
	}
	if out := f.RenderInitiativeUpdateList(ini.GetUpdates(), VerbosityCompact, OutputJSON); !strings.HasPrefix(out, "[") {
		t.Errorf("json output unexpected: %q", out)
	}
	if out := f.RenderInitiativeUpdate(&ini.GetUpdates()[0], VerbosityCompact, OutputText); !strings.Contains(out, "[On track]") {
		t.Errorf("text output unexpected: %q", out)
	}
}
