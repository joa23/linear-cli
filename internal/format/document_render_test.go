package format

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

func sampleDocument() *core.Document {
	return &core.Document{
		ID:        "965bab1e-b1a0-4353-943e-ec14571cc1d2",
		SlugID:    "913afa4d594c",
		Title:     "API Spec",
		Content:   strings.Repeat("x", 600),
		URL:       "https://linear.app/acme/document/api-spec-913afa4d594c",
		CreatedAt: "2026-01-15T10:00:00Z",
		UpdatedAt: "2026-01-16T10:00:00Z",
		Creator:   &core.User{ID: "u-1", Name: "Alice Smith", DisplayName: "alice"},
		Project:   &core.Project{ID: "p-1", Name: "Platform"},
	}
}

func TestTextRenderer_RenderDocument_Verbosity(t *testing.T) {
	r := &TextRenderer{}
	doc := sampleDocument()

	t.Run("nil", func(t *testing.T) {
		if got := r.RenderDocument(nil, VerbosityFull); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("minimal", func(t *testing.T) {
		out := r.RenderDocument(doc, VerbosityMinimal)
		if !strings.Contains(out, "API Spec") || !strings.Contains(out, "ID: 965bab1e") {
			t.Errorf("minimal missing title/id: %q", out)
		}
		if strings.Contains(out, "Slug:") || strings.Contains(out, "xxxx") {
			t.Errorf("minimal must not include slug or content: %q", out)
		}
	})

	t.Run("compact", func(t *testing.T) {
		out := r.RenderDocument(doc, VerbosityCompact)
		for _, want := range []string{"Slug: 913afa4d594c", "Parent: Project Platform", "Updated: 2026-01-16", "URL: https://"} {
			if !strings.Contains(out, want) {
				t.Errorf("compact missing %q: %q", want, out)
			}
		}
		if strings.Contains(out, "Created:") || strings.Contains(out, "xxxx") {
			t.Errorf("compact must not include created/content: %q", out)
		}
	})

	t.Run("detailed truncates content", func(t *testing.T) {
		out := r.RenderDocument(doc, VerbosityDetailed)
		if !strings.Contains(out, "Created: 2026-01-15 by alice") {
			t.Errorf("detailed missing creator line: %q", out)
		}
		if !strings.Contains(out, "...") || strings.Count(out, "x") >= 600 {
			t.Errorf("detailed should truncate content: len=%d", strings.Count(out, "x"))
		}
	})

	t.Run("full has all content", func(t *testing.T) {
		out := r.RenderDocument(doc, VerbosityFull)
		if strings.Count(out, "x") != 600 {
			t.Errorf("full should include all content, got %d x", strings.Count(out, "x"))
		}
	})

	t.Run("archived", func(t *testing.T) {
		archived := "2026-02-01T10:00:00Z"
		d := sampleDocument()
		d.ArchivedAt = &archived
		out := r.RenderDocument(d, VerbosityCompact)
		if !strings.Contains(out, "Archived: 2026-02-01") {
			t.Errorf("expected archived line: %q", out)
		}
	})
}

func TestDocumentParentLabel(t *testing.T) {
	tests := []struct {
		name string
		doc  core.Document
		want string
	}{
		{"project", core.Document{Project: &core.Project{Name: "Platform"}}, "Project Platform"},
		{"issue", core.Document{Issue: &core.Issue{Identifier: "ENG-42"}}, "Issue ENG-42"},
		{"team", core.Document{Team: &core.Team{Key: "ENG"}}, "Team ENG"},
		{"none", core.Document{}, "-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := documentParentLabel(&tt.doc); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTextRenderer_RenderDocumentList(t *testing.T) {
	r := &TextRenderer{}

	t.Run("empty", func(t *testing.T) {
		if got := r.RenderDocumentList(nil, VerbosityCompact, nil); got != "No documents found.\n" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("rows", func(t *testing.T) {
		docs := []core.Document{
			*sampleDocument(),
			{ID: "abcdefgh-1234", Title: "No slug", UpdatedAt: "2026-01-01T00:00:00Z", Team: &core.Team{Key: "QA"}},
		}
		out := r.RenderDocumentList(docs, VerbosityCompact, &Pagination{HasNextPage: true, EndCursor: "cur-1"})
		if !strings.Contains(out, "DOCUMENTS (2)") {
			t.Errorf("missing header: %q", out)
		}
		if !strings.Contains(out, "913afa4d594c  API Spec  [Project Platform]  2026-01-16") {
			t.Errorf("missing first row: %q", out)
		}
		if !strings.Contains(out, "abcdefgh  No slug  [Team QA]  2026-01-01") {
			t.Errorf("missing short-id fallback row: %q", out)
		}
		if strings.Contains(out, "xxxx") {
			t.Error("compact list must not include content")
		}
		if !strings.Contains(out, "Next: cursor=cur-1") {
			t.Errorf("missing pagination footer: %q", out)
		}
	})

	t.Run("detailed renders full blocks", func(t *testing.T) {
		out := r.RenderDocumentList([]core.Document{*sampleDocument()}, VerbosityDetailed, nil)
		if !strings.Contains(out, "Parent: Project Platform") {
			t.Errorf("detailed list should render document blocks: %q", out)
		}
	})
}

func TestJSONRenderer_RenderDocument(t *testing.T) {
	r := &JSONRenderer{}

	t.Run("nil", func(t *testing.T) {
		out := r.RenderDocument(nil, VerbosityFull)
		if !strings.Contains(out, "error") {
			t.Errorf("expected error JSON, got %q", out)
		}
	})

	t.Run("single includes content and refs", func(t *testing.T) {
		out := r.RenderDocument(sampleDocument(), VerbosityCompact)
		var dto DocumentDTO
		if err := json.Unmarshal([]byte(out), &dto); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if dto.Content == "" || dto.SlugID != "913afa4d594c" {
			t.Errorf("unexpected dto: %+v", dto)
		}
		if dto.Project == nil || dto.Project.Name != "Platform" {
			t.Error("expected project ref")
		}
		if dto.Creator == nil || dto.Creator.DisplayName != "alice" {
			t.Error("expected creator ref")
		}
		if dto.Issue != nil || dto.Team != nil {
			t.Error("expected nil issue/team")
		}
	})

	t.Run("list omits content at compact, includes at detailed", func(t *testing.T) {
		docs := []core.Document{*sampleDocument()}
		compact := r.RenderDocumentList(docs, VerbosityCompact, nil)
		if strings.Contains(compact, "\"content\"") {
			t.Error("compact list should omit content")
		}
		detailed := r.RenderDocumentList(docs, VerbosityDetailed, nil)
		if !strings.Contains(detailed, "\"content\"") {
			t.Error("detailed list should include content")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		if got := r.RenderDocumentList(nil, VerbosityCompact, nil); got != "[]" {
			t.Errorf("got %q", got)
		}
	})
}

func TestFormatter_RenderDocument_Dispatch(t *testing.T) {
	f := New()
	doc := sampleDocument()

	text := f.RenderDocument(doc, VerbosityCompact, OutputText)
	if !strings.HasPrefix(text, "API Spec\n") {
		t.Errorf("text output unexpected: %q", text)
	}

	jsonOut := f.RenderDocumentList([]core.Document{*doc}, VerbosityCompact, OutputJSON, nil)
	if !strings.HasPrefix(jsonOut, "[") {
		t.Errorf("json output unexpected: %q", jsonOut)
	}
}
