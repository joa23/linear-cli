package service

import (
	"testing"

	"github.com/joa23/linear-cli/internal/format"
)

func TestAttachmentCreateParams_Validation(t *testing.T) {
	tests := []struct {
		name    string
		params  AttachmentCreateParams
		wantErr string
	}{
		{
			name:    "no url or file",
			params:  AttachmentCreateParams{IssueID: "TEC-1", Title: "test"},
			wantErr: "either --url or --file is required",
		},
		{
			name:    "both url and file",
			params:  AttachmentCreateParams{IssueID: "TEC-1", URL: "https://x.com", FilePath: "/tmp/f.png", Title: "test"},
			wantErr: "--url and --file are mutually exclusive",
		},
		{
			name:    "missing title",
			params:  AttachmentCreateParams{IssueID: "TEC-1", URL: "https://x.com"},
			wantErr: "--title is required",
		},
	}

	svc := NewAttachmentService(nil, format.New())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(&tt.params)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestAttachmentUpdateParams_Validation(t *testing.T) {
	svc := NewAttachmentService(nil, format.New())

	_, err := svc.Update("some-uuid", &AttachmentUpdateParams{Title: ""})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if err.Error() != "--title is required" {
		t.Errorf("error = %q, want %q", err.Error(), "--title is required")
	}
}
