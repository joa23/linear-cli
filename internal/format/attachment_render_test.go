package format

import (
	"strings"
	"testing"

	"github.com/joa23/linear-cli/internal/linear/core"
)

func TestTextRenderer_RenderAttachmentList(t *testing.T) {
	renderer := &TextRenderer{}

	atts := []core.Attachment{
		{
			ID:         "att-1",
			URL:        "https://github.com/org/repo/pull/42",
			Title:      "PR #42: Fix auth",
			Subtitle:   "Merged",
			SourceType: "github",
			CreatedAt:  "2026-01-15T10:00:00Z",
		},
		{
			ID:         "att-2",
			URL:        "https://slack.com/archives/C123/p456",
			Title:      "Slack thread",
			SourceType: "slack",
			CreatedAt:  "2026-01-16T10:00:00Z",
		},
	}

	result := renderer.RenderAttachmentList(atts, VerbosityCompact)

	if !strings.Contains(result, "[github]") {
		t.Error("expected [github] source type tag")
	}
	if !strings.Contains(result, "PR #42: Fix auth") {
		t.Error("expected attachment title")
	}
	if !strings.Contains(result, "[slack]") {
		t.Error("expected [slack] source type tag")
	}
	if !strings.Contains(result, "Attachments (2)") {
		t.Error("expected 'Attachments (2)' header")
	}
}

func TestTextRenderer_RenderAttachment(t *testing.T) {
	renderer := &TextRenderer{}

	att := &core.Attachment{
		ID:         "att-1",
		URL:        "https://github.com/org/repo/pull/42",
		Title:      "PR #42: Fix auth",
		Subtitle:   "Merged",
		SourceType: "github",
		CreatedAt:  "2026-01-15T10:00:00Z",
		UpdatedAt:  "2026-01-15T12:00:00Z",
	}

	result := renderer.RenderAttachment(att, VerbosityFull)

	if !strings.Contains(result, "PR #42: Fix auth") {
		t.Error("expected title")
	}
	if !strings.Contains(result, "https://github.com/org/repo/pull/42") {
		t.Error("expected URL")
	}
	if !strings.Contains(result, "github") {
		t.Error("expected source type")
	}
	if !strings.Contains(result, "att-1") {
		t.Error("expected ID in full verbosity")
	}
}

func TestTextRenderer_RenderAttachmentList_Empty(t *testing.T) {
	renderer := &TextRenderer{}
	result := renderer.RenderAttachmentList(nil, VerbosityCompact)
	if !strings.Contains(result, "No attachments") {
		t.Error("expected empty message")
	}
}

func TestJSONRenderer_RenderAttachmentList(t *testing.T) {
	renderer := &JSONRenderer{}

	atts := []core.Attachment{
		{
			ID:         "att-1",
			URL:        "https://github.com/org/repo/pull/42",
			Title:      "PR #42",
			SourceType: "github",
			CreatedAt:  "2026-01-15T10:00:00Z",
		},
	}

	result := renderer.RenderAttachmentList(atts, VerbosityCompact)

	if !strings.Contains(result, `"id"`) {
		t.Error("expected JSON id field")
	}
	if !strings.Contains(result, `"att-1"`) {
		t.Error("expected attachment ID in JSON")
	}
	if !strings.Contains(result, `"sourceType"`) {
		t.Error("expected sourceType in JSON")
	}
}
