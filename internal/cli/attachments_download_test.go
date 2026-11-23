package cli

import "testing"

func TestIsLinearUploadsURL(t *testing.T) {
	cases := map[string]bool{
		"https://uploads.linear.app/a/b":        true,
		"https://UPLOADS.linear.app/a":          true,
		"http://uploads.linear.app/a/b":         false,
		"https://uploads.linear.app.evil.com/a": false,
		"https://evil.com/uploads.linear.app":   false,
		"file:///etc/passwd":                    false,
		"https://user@uploads.linear.app/a":     true,
	}
	for in, want := range cases {
		if got := isLinearUploadsURL(in); got != want {
			t.Errorf("isLinearUploadsURL(%q) = %v, want %v", in, got, want)
		}
	}
}
