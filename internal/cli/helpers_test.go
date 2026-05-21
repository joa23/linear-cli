package cli

import (
	"strings"
	"testing"
	"time"
)

func TestParseCreatedSince(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantAgo time.Duration // how far back the resulting timestamp should be
		wantErr string        // substring expected in error message; empty = no error
	}{
		{name: "empty returns empty", input: "", wantAgo: 0},
		{name: "24h", input: "24h", wantAgo: 24 * time.Hour},
		{name: "7d", input: "7d", wantAgo: 7 * 24 * time.Hour},
		{name: "2w", input: "2w", wantAgo: 14 * 24 * time.Hour},
		{name: "30m", input: "30m", wantAgo: 30 * time.Minute},
		{name: "0.5d", input: "0.5d", wantAgo: 12 * time.Hour},
		{name: "negative rejected", input: "-1h", wantErr: "must be positive"},
		{name: "zero rejected", input: "0h", wantErr: "must be positive"},
		{name: "garbage rejected", input: "foo", wantErr: "invalid --created-since"},
		{name: "missing unit rejected", input: "24", wantErr: "invalid --created-since"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now().UTC()
			got, err := parseCreatedSince(tt.input)
			after := time.Now().UTC()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (result: %q)", tt.wantErr, got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.input == "" {
				if got != "" {
					t.Fatalf("expected empty result for empty input, got %q", got)
				}
				return
			}

			parsed, parseErr := time.Parse(time.RFC3339, got)
			if parseErr != nil {
				t.Fatalf("result %q is not valid RFC3339: %v", got, parseErr)
			}

			// Allow a small window for the actual time-of-call within parseCreatedSince.
			earliest := before.Add(-tt.wantAgo).Add(-time.Second)
			latest := after.Add(-tt.wantAgo).Add(time.Second)
			if parsed.Before(earliest) || parsed.After(latest) {
				t.Fatalf("parsed timestamp %v outside expected window [%v, %v]", parsed, earliest, latest)
			}
		})
	}
}
