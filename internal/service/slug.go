package service

import (
	"regexp"
	"strings"
)

// DefaultWorktreeSlugMaxLen is the default maximum length for a worktree slug.
const DefaultWorktreeSlugMaxLen = 60

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// WorktreeSlug builds a filesystem-friendly worktree/branch name from an issue
// identifier and title, e.g. "cen-123_fix_login_bug". The result is lowercase
// and underscore-separated, capped at maxLen characters. When it must shorten,
// it keeps the full "<identifier>_" prefix and drops whole trailing title words
// so it never ends mid-word — unless the first title word alone exceeds the
// budget, in which case that word is hard-cut.
func WorktreeSlug(identifier, title string, maxLen int) string {
	prefix := strings.ToLower(strings.TrimSpace(identifier))
	titleSlug := slugify(title)

	if titleSlug == "" {
		return clampLen(prefix, maxLen)
	}

	full := prefix + "_" + titleSlug
	if maxLen <= 0 || len(full) <= maxLen {
		return full
	}

	// Budget for the title portion after "<prefix>_".
	budget := maxLen - len(prefix) - 1
	if budget <= 0 {
		return clampLen(prefix, maxLen)
	}

	var b strings.Builder
	for _, w := range strings.Split(titleSlug, "_") {
		switch {
		case b.Len() == 0 && len(w) <= budget:
			b.WriteString(w)
		case b.Len() == 0:
			// First word alone exceeds the budget: hard-cut it.
			b.WriteString(w[:budget])
		case b.Len()+1+len(w) <= budget:
			b.WriteString("_")
			b.WriteString(w)
		default:
			// No more whole words fit.
			return prefix + "_" + b.String()
		}
	}
	return prefix + "_" + b.String()
}

// slugify lowercases s and collapses runs of non-alphanumeric characters into
// single underscores, trimming leading/trailing underscores.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = slugNonAlnum.ReplaceAllString(s, "_")
	return strings.Trim(s, "_")
}

// clampLen truncates s to maxLen characters (maxLen <= 0 means no limit).
func clampLen(s string, maxLen int) string {
	if maxLen > 0 && len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
