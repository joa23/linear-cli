package service

import "testing"

func TestWorktreeSlug(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		title      string
		maxLen     int
		want       string
	}{
		{
			name:       "basic",
			identifier: "CEN-123",
			title:      "Fix the login bug",
			maxLen:     60,
			want:       "cen-123_fix_the_login_bug",
		},
		{
			name:       "special characters collapsed",
			identifier: "CEN-7",
			title:      "  OAuth: token refresh (scope!) -- v2  ",
			maxLen:     60,
			want:       "cen-7_oauth_token_refresh_scope_v2",
		},
		{
			name:       "trims at word boundary",
			identifier: "CEN-123",
			title:      "Implement the brand new authentication subsystem today",
			maxLen:     30,
			// budget after "cen-123_" (8 chars) is 22; "_new" (would be 23) doesn't fit
			want: "cen-123_implement_the_brand",
		},
		{
			name:       "empty title returns prefix",
			identifier: "CEN-9",
			title:      "",
			maxLen:     60,
			want:       "cen-9",
		},
		{
			name:       "no limit when maxLen zero",
			identifier: "CEN-1",
			title:      "A very long title that keeps going and going",
			maxLen:     0,
			want:       "cen-1_a_very_long_title_that_keeps_going_and_going",
		},
		{
			name:       "first word too long is hard-cut",
			identifier: "AB-1",
			title:      "Supercalifragilisticexpialidocious",
			maxLen:     15,
			// budget after "ab-1_" (5 chars) is 10
			want: "ab-1_supercalif",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WorktreeSlug(tt.identifier, tt.title, tt.maxLen)
			if got != tt.want {
				t.Errorf("WorktreeSlug(%q, %q, %d) = %q, want %q",
					tt.identifier, tt.title, tt.maxLen, got, tt.want)
			}
			if tt.maxLen > 0 && len(got) > tt.maxLen {
				t.Errorf("result %q exceeds maxLen %d", got, tt.maxLen)
			}
		})
	}
}
