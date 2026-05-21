package service

import (
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

func TestExtractMarkdownURLs(t *testing.T) {
	body := "See ![diagram](https://uploads.linear.app/abc/def) and " +
		"[spec](https://uploads.linear.app/x/y) plus [external](https://github.com/o/r/pull/1)."
	got := extractMarkdownURLs(body)
	want := []string{
		"https://uploads.linear.app/abc/def",
		"https://uploads.linear.app/x/y",
		"https://github.com/o/r/pull/1",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d urls, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("url[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsLinearUploadURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://uploads.linear.app/a/b", true},
		{"https://UPLOADS.linear.app/a/b", true},
		{"https://github.com/o/r", false},
		{"not a url", false},
	}
	for _, tt := range tests {
		if got := isLinearUploadURL(tt.url); got != tt.want {
			t.Errorf("isLinearUploadURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestUniqueName(t *testing.T) {
	used := map[string]bool{}
	if got := uniqueName(used, "img.png"); got != "img.png" {
		t.Errorf("first = %q, want img.png", got)
	}
	if got := uniqueName(used, "img.png"); got != "img-1.png" {
		t.Errorf("second = %q, want img-1.png", got)
	}
	if got := uniqueName(used, "img.png"); got != "img-2.png" {
		t.Errorf("third = %q, want img-2.png", got)
	}
}

func TestSanitizeFilename(t *testing.T) {
	if got := sanitizeFilename("a/b/c.png"); got != "c.png" {
		t.Errorf("got %q, want c.png", got)
	}
	if got := sanitizeFilename(""); got != "file" {
		t.Errorf("empty got %q, want file", got)
	}
}

func TestBuildMarkdown(t *testing.T) {
	prio := 1
	issue := &core.Issue{
		Identifier:  "CEN-1",
		Title:       "Fix login",
		Description: "Repro in ![shot](https://uploads.linear.app/a/b).",
		Priority:    &prio,
		URL:         "https://linear.app/x/issue/CEN-1",
	}
	issue.State.Name = "In Progress"
	comments := []core.Comment{
		{ID: "c1", Body: "Top comment", CreatedAt: "2026-05-01T10:00:00Z", User: core.User{Name: "Alice"}},
		{ID: "c2", Body: "A reply", CreatedAt: "2026-05-01T11:00:00Z", User: core.User{Name: "Bob"}, Parent: &core.CommentParent{ID: "c1"}},
	}
	attachments := []core.Attachment{
		{URL: "https://github.com/o/r/pull/2", Title: "PR #2", SourceType: "github"},
	}
	urlToLocal := map[string]string{
		"https://uploads.linear.app/a/b": "assets/shot.png",
	}

	md := buildMarkdown(issue, comments, attachments, urlToLocal)

	checks := []string{
		"# CEN-1: Fix login",
		"| Priority | Urgent |",
		"assets/shot.png",            // inline image rewritten
		"### @Alice",                 // top-level comment
		"#### ↳ @Bob",                // nested reply
		"## References",
		"[PR #2](https://github.com/o/r/pull/2)", // external link preserved
	}
	for _, want := range checks {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q\n---\n%s", want, md)
		}
	}
	if strings.Contains(md, "uploads.linear.app/a/b") {
		t.Errorf("rewritten URL should not remain:\n%s", md)
	}
}
