package service

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joa23/linear-cli/pkg/linear"
	"github.com/joa23/linear-cli/pkg/linear/core"
)

// IssueExportServiceInterface defines the operation for exporting a single
// issue (description, comments, and downloadable attachments) to a folder.
type IssueExportServiceInterface interface {
	Export(identifier, folder string) (*IssueExportResult, error)
}

// IssueExportService exports a Linear issue into a self-contained, LLM-friendly
// folder: a markdown file plus an assets/ subfolder holding every downloaded
// image and uploaded file, referenced by relative path from the markdown.
type IssueExportService struct {
	client *linear.Client
}

// NewIssueExportService creates a new IssueExportService.
func NewIssueExportService(client *linear.Client) *IssueExportService {
	return &IssueExportService{client: client}
}

// IssueExportResult summarizes a completed export for CLI feedback.
type IssueExportResult struct {
	Identifier   string
	MarkdownPath string
	AssetCount   int
	CommentCount int
	FailedAssets []string // URLs that could not be downloaded
}

const assetsDir = "assets"

// markdownLinkRe matches markdown links and images: [text](url) and ![alt](url).
// Group 1 captures the URL (everything up to the first space or closing paren).
var markdownLinkRe = regexp.MustCompile(`!?\[[^\]]*\]\(([^)\s]+)`)

// Export fetches the issue, its comments and attachment cards, downloads every
// uploads.linear.app asset into folder/assets/, and writes folder/<Identifier>.md
// with those URLs rewritten to local relative paths.
func (s *IssueExportService) Export(identifier, folder string) (*IssueExportResult, error) {
	issue, err := s.client.GetIssue(identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", identifier, err)
	}

	comments, err := s.client.Comments.GetIssueComments(issue.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments for %s: %w", issue.Identifier, err)
	}

	attachments, err := s.client.Attachments.ListAttachments(issue.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments for %s: %w", issue.Identifier, err)
	}

	if err := os.MkdirAll(filepath.Join(folder, assetsDir), 0755); err != nil {
		return nil, fmt.Errorf("failed to create export folder: %w", err)
	}

	// Collect downloadable URLs in encounter order, deduped.
	var urls []string
	seen := map[string]bool{}
	addURL := func(u string) {
		if u != "" && isLinearUploadURL(u) && !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	for _, u := range extractMarkdownURLs(issue.Description) {
		addURL(u)
	}
	for _, c := range comments {
		for _, u := range extractMarkdownURLs(c.Body) {
			addURL(u)
		}
	}
	for _, a := range attachments {
		addURL(a.URL)
	}

	// Download each unique URL into assets/, mapping URL -> relative path.
	urlToLocal := map[string]string{}
	usedNames := map[string]bool{}
	var failed []string
	for _, u := range urls {
		content, name, derr := s.client.Attachments.DownloadBytes(u)
		if derr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not download %s: %v\n", u, derr)
			failed = append(failed, u)
			continue
		}
		name = uniqueName(usedNames, sanitizeFilename(name))
		if werr := os.WriteFile(filepath.Join(folder, assetsDir, name), content, 0644); werr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write asset for %s: %v\n", u, werr)
			failed = append(failed, u)
			continue
		}
		urlToLocal[u] = assetsDir + "/" + name
	}

	md := buildMarkdown(issue, comments, attachments, urlToLocal)

	mdPath := filepath.Join(folder, issue.Identifier+".md")
	if err := os.WriteFile(mdPath, []byte(md), 0644); err != nil {
		return nil, fmt.Errorf("failed to write markdown file: %w", err)
	}

	return &IssueExportResult{
		Identifier:   issue.Identifier,
		MarkdownPath: mdPath,
		AssetCount:   len(urlToLocal),
		CommentCount: len(comments),
		FailedAssets: failed,
	}, nil
}

// extractMarkdownURLs returns the URLs referenced by markdown links/images in text.
func extractMarkdownURLs(text string) []string {
	matches := markdownLinkRe.FindAllStringSubmatch(text, -1)
	urls := make([]string, 0, len(matches))
	for _, m := range matches {
		urls = append(urls, m[1])
	}
	return urls
}

// isLinearUploadURL reports whether u points at Linear's private upload host.
func isLinearUploadURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), "uploads.linear.app")
}

// sanitizeFilename strips path separators and trims to a safe base name.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "file"
	}
	return name
}

// uniqueName returns name (or name-1, name-2, ...) not present in used, and
// records the chosen name in used.
func uniqueName(used map[string]bool, name string) string {
	candidate := name
	if !used[candidate] {
		used[candidate] = true
		return candidate
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		candidate = fmt.Sprintf("%s-%d%s", base, i, ext)
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
}

// rewriteURLs replaces every mapped source URL with its local relative path.
func rewriteURLs(text string, urlToLocal map[string]string) string {
	for src, local := range urlToLocal {
		text = strings.ReplaceAll(text, src, local)
	}
	return text
}

// buildMarkdown assembles the export document.
func buildMarkdown(issue *core.Issue, comments []core.Comment, attachments []core.Attachment, urlToLocal map[string]string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s: %s\n\n", issue.Identifier, issue.Title)

	// Metadata block.
	b.WriteString("| Field | Value |\n|---|---|\n")
	fmt.Fprintf(&b, "| State | %s |\n", issue.State.Name)
	if issue.Assignee != nil {
		fmt.Fprintf(&b, "| Assignee | %s |\n", displayName(issue.Assignee))
	}
	if issue.Priority != nil {
		fmt.Fprintf(&b, "| Priority | %s |\n", priorityName(issue.Priority))
	}
	if issue.Estimate != nil {
		fmt.Fprintf(&b, "| Estimate | %g |\n", *issue.Estimate)
	}
	if issue.Cycle != nil {
		fmt.Fprintf(&b, "| Cycle | %s |\n", cycleLabel(issue.Cycle))
	}
	if issue.Project != nil {
		fmt.Fprintf(&b, "| Project | %s |\n", issue.Project.Name)
	}
	if labels := labelNames(issue.Labels); labels != "" {
		fmt.Fprintf(&b, "| Labels | %s |\n", labels)
	}
	if issue.URL != "" {
		fmt.Fprintf(&b, "| URL | %s |\n", issue.URL)
	}
	fmt.Fprintf(&b, "| Created | %s |\n", issue.CreatedAt)
	fmt.Fprintf(&b, "| Updated | %s |\n", issue.UpdatedAt)
	b.WriteString("\n")

	// Description.
	b.WriteString("## Description\n\n")
	if strings.TrimSpace(issue.Description) == "" {
		b.WriteString("_No description._\n\n")
	} else {
		b.WriteString(rewriteURLs(issue.Description, urlToLocal))
		b.WriteString("\n\n")
	}

	// Comments, with replies nested under their parent.
	b.WriteString("## Comments\n\n")
	if len(comments) == 0 {
		b.WriteString("_No comments._\n\n")
	} else {
		writeComments(&b, comments, urlToLocal)
	}

	// References: sidebar attachment cards.
	if len(attachments) > 0 {
		b.WriteString("## References\n\n")
		for _, a := range attachments {
			title := a.Title
			if title == "" {
				title = a.URL
			}
			target := a.URL
			if local, ok := urlToLocal[a.URL]; ok {
				target = local
			}
			if a.SourceType != "" {
				fmt.Fprintf(&b, "- [%s](%s) _(%s)_\n", title, target, a.SourceType)
			} else {
				fmt.Fprintf(&b, "- [%s](%s)\n", title, target)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// writeComments renders top-level comments followed by their replies (indented).
func writeComments(b *strings.Builder, comments []core.Comment, urlToLocal map[string]string) {
	childrenOf := map[string][]core.Comment{}
	var topLevel []core.Comment
	for _, c := range comments {
		if c.Parent != nil && c.Parent.ID != "" {
			childrenOf[c.Parent.ID] = append(childrenOf[c.Parent.ID], c)
		} else {
			topLevel = append(topLevel, c)
		}
	}

	for _, c := range topLevel {
		writeComment(b, c, urlToLocal, false)
		for _, reply := range childrenOf[c.ID] {
			writeComment(b, reply, urlToLocal, true)
		}
	}
}

func writeComment(b *strings.Builder, c core.Comment, urlToLocal map[string]string, isReply bool) {
	heading := "###"
	prefix := ""
	if isReply {
		heading = "####"
		prefix = "↳ "
	}
	fmt.Fprintf(b, "%s %s@%s — %s\n\n", heading, prefix, displayName(&c.User), c.CreatedAt)
	body := strings.TrimSpace(rewriteURLs(c.Body, urlToLocal))
	if body == "" {
		body = "_(empty)_"
	}
	b.WriteString(body)
	b.WriteString("\n\n")
}

func displayName(u *core.User) string {
	if u == nil {
		return "unknown"
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.Name != "" {
		return u.Name
	}
	if u.Email != "" {
		return u.Email
	}
	return "unknown"
}

func priorityName(p *int) string {
	if p == nil {
		return ""
	}
	switch *p {
	case 0:
		return "No priority"
	case 1:
		return "Urgent"
	case 2:
		return "High"
	case 3:
		return "Normal"
	case 4:
		return "Low"
	default:
		return fmt.Sprintf("%d", *p)
	}
}

func cycleLabel(c *core.CycleReference) string {
	if c == nil {
		return ""
	}
	if c.Name != "" {
		return fmt.Sprintf("%d (%s)", c.Number, c.Name)
	}
	return fmt.Sprintf("%d", c.Number)
}

func labelNames(lc *core.LabelConnection) string {
	if lc == nil || len(lc.Nodes) == 0 {
		return ""
	}
	names := make([]string, 0, len(lc.Nodes))
	for _, l := range lc.Nodes {
		names = append(names, l.Name)
	}
	return strings.Join(names, ", ")
}
