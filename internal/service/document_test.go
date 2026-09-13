package service

import (
	"strings"
	"testing"

	"github.com/joa23/linear-cli/internal/format"
)

func TestParseDocumentRef(t *testing.T) {
	const uuid = "965bab1e-b1a0-4353-943e-ec14571cc1d2"

	tests := []struct {
		name     string
		ref      string
		wantUUID string
		wantSlug string
	}{
		{name: "empty", ref: "", wantUUID: "", wantSlug: ""},
		{name: "uuid", ref: uuid, wantUUID: uuid, wantSlug: ""},
		{name: "uuid with spaces", ref: "  " + uuid + " ", wantUUID: uuid, wantSlug: ""},
		{name: "bare slug", ref: "913afa4d594c", wantSlug: "913afa4d594c"},
		{name: "title slug", ref: "cli-smoke-doc-913afa4d594c", wantSlug: "913afa4d594c"},
		{name: "url", ref: "https://linear.app/acme/document/cli-smoke-doc-913afa4d594c", wantSlug: "913afa4d594c"},
		{name: "url trailing slash", ref: "https://linear.app/acme/document/cli-smoke-doc-913afa4d594c/", wantSlug: "913afa4d594c"},
		{name: "url with query", ref: "https://linear.app/acme/document/cli-smoke-doc-913afa4d594c?tab=x", wantSlug: "913afa4d594c"},
		{name: "url with fragment", ref: "https://linear.app/acme/document/doc-913afa4d594c#heading", wantSlug: "913afa4d594c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUUID, gotSlug := parseDocumentRef(tt.ref)
			if gotUUID != tt.wantUUID || gotSlug != tt.wantSlug {
				t.Errorf("parseDocumentRef(%q) = (%q, %q), want (%q, %q)", tt.ref, gotUUID, gotSlug, tt.wantUUID, tt.wantSlug)
			}
		})
	}
}

func TestDocumentCreateParams_Validation(t *testing.T) {
	tests := []struct {
		name    string
		params  DocumentCreateParams
		wantErr string
	}{
		{name: "missing title", params: DocumentCreateParams{Team: "ENG"}, wantErr: "--title is required"},
		{name: "blank title", params: DocumentCreateParams{Title: "  ", Team: "ENG"}, wantErr: "--title is required"},
		{name: "no parent", params: DocumentCreateParams{Title: "Spec"}, wantErr: "a parent is required"},
		{name: "project and issue", params: DocumentCreateParams{Title: "Spec", Project: "P", Issue: "ENG-1"}, wantErr: "only one parent is allowed"},
	}

	svc := NewDocumentService(nil, format.New())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(&tt.params)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDocumentUpdateParams_Validation(t *testing.T) {
	str := func(s string) *string { return &s }

	tests := []struct {
		name    string
		params  DocumentUpdateParams
		wantErr string
	}{
		{name: "nothing set", params: DocumentUpdateParams{}, wantErr: "nothing to update"},
		{name: "empty title", params: DocumentUpdateParams{Title: str(" ")}, wantErr: "--title cannot be empty"},
	}

	svc := NewDocumentService(nil, format.New())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Update("913afa4d594c", &tt.params)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}
