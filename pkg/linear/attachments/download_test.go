package attachments

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

func TestAttemptDownload_SetsAuthHeaderForPrivateLinearURL(t *testing.T) {
	const fakeToken = "test-token-abc"
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "image/png")
		// Minimal 1x1 PNG so content-type detection works.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-image-data"))
	}))
	defer srv.Close()

	base := core.NewTestBaseClient(fakeToken, "http://unused", srv.Client())
	ac := NewClient(base)
	ac.httpClient = srv.Client()

	// Simulate an uploads.linear.app URL by substituting the test server host
	// via a custom transport that rewrites the host.
	privateURL := "https://uploads.linear.app/some/path/image.png"

	// Override httpClient to redirect uploads.linear.app to our test server.
	ac.httpClient = &http.Client{
		Transport: &rewriteTransport{
			realURL: srv.URL,
			target:  "uploads.linear.app",
			inner:   srv.Client().Transport,
		},
	}

	_, _, _, _ = ac.attemptDownload(privateURL)

	if capturedAuth != "Bearer "+fakeToken {
		t.Errorf("expected Authorization header %q, got %q", "Bearer "+fakeToken, capturedAuth)
	}
}

func TestAttemptDownload_NoAuthHeaderForPublicURL(t *testing.T) {
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("public content"))
	}))
	defer srv.Close()

	base := core.NewTestBaseClient("secret-token", "http://unused", srv.Client())
	ac := NewClient(base)
	ac.httpClient = srv.Client()

	_, _, _, _ = ac.attemptDownload(srv.URL + "/public/file.txt")

	if capturedAuth != "" {
		t.Errorf("expected no Authorization header for public URL, got %q", capturedAuth)
	}
}

func TestIsPrivateLinearURL(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://uploads.linear.app/abc/image.png", true},
		{"https://uploads.linear.app/", true},
		{"https://linear.app/team/issue", false},
		{"https://github.com/org/repo", false},
		{"https://example.com/uploads.linear.app.fake", false},
		{"not-a-valid-url", false},
	}
	for _, c := range cases {
		got := isPrivateLinearURL(c.url)
		if got != c.want {
			t.Errorf("isPrivateLinearURL(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}

func TestAttemptDownload_FailsFastWhenTokenLookupFailsForPrivateLinearURL(t *testing.T) {
	hitServer := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitServer = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("unused"))
	}))
	defer srv.Close()

	base := core.NewBaseClientWithProvider(&failingTokenProvider{})
	ac := NewClient(base)
	ac.httpClient = &http.Client{
		Transport: &rewriteTransport{
			realURL: srv.URL,
			target:  "uploads.linear.app",
			inner:   srv.Client().Transport,
		},
	}

	_, _, _, err := ac.attemptDownload("https://uploads.linear.app/private/image.png")
	if err == nil {
		t.Fatal("expected token retrieval error, got nil")
	}
	if !strings.Contains(err.Error(), "linear auth login") {
		t.Fatalf("expected re-auth guidance in error, got: %v", err)
	}
	if hitServer {
		t.Fatal("expected no outbound request when token lookup fails")
	}
}

// rewriteTransport redirects requests whose host matches target to realURL.
type rewriteTransport struct {
	realURL string
	target  string
	inner   http.RoundTripper
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == rt.target {
		rewritten := req.Clone(req.Context())
		parsed, _ := http.NewRequest(req.Method, rt.realURL+req.URL.Path, req.Body)
		rewritten.URL = parsed.URL
		return rt.inner.RoundTrip(rewritten)
	}
	return rt.inner.RoundTrip(req)
}

type failingTokenProvider struct{}

func (p *failingTokenProvider) GetToken() (string, error) {
	return "", errors.New("token unavailable")
}

func (p *failingTokenProvider) RefreshIfNeeded(string) (string, error) {
	return "", errors.New("refresh unsupported")
}

func TestFilenameFromDisposition(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{`attachment; filename="report.xlsx"`, "report.xlsx"},
		{`attachment; filename="image.png"`, "image.png"},
		{`attachment; filename="file with spaces.json"`, "file with spaces.json"},
		{`inline; filename="doc.pdf"`, "doc.pdf"},
		{"", ""},
		{"attachment", ""},
		{`attachment; filename="../../../etc/passwd"`, ".._.._.._etc_passwd"},
	}
	for _, c := range cases {
		got := filenameFromDisposition(c.header)
		if got != c.want {
			t.Errorf("filenameFromDisposition(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}

func TestExtensionFromContentType(t *testing.T) {
	cases := []struct {
		ct   string
		want string
	}{
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"application/json", ".json"},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ".xlsx"},
		{"text/plain; charset=utf-8", ".txt"},
		{"application/pdf", ".pdf"},
		{"application/zip", ".zip"},
		{"video/mp4", ".mp4"},
		{"application/octet-stream", ".bin"},
		{"something/unknown", ".bin"},
	}
	for _, c := range cases {
		got := extensionFromContentType(c.ct)
		if got != c.want {
			t.Errorf("extensionFromContentType(%q) = %q, want %q", c.ct, got, c.want)
		}
	}
}

func TestDownloadToFile_UsesContentDisposition(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="debug.json"`)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"test": true}`))
	}))
	defer srv.Close()

	base := core.NewTestBaseClient("tok", "http://unused", srv.Client())
	ac := NewClient(base)
	ac.httpClient = srv.Client()

	path, err := ac.DownloadToFile(srv.URL+"/file", t.TempDir(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(path, "debug.json") {
		t.Errorf("expected path to contain original filename, got %q", path)
	}
}

func TestDownloadToFile_FallsBackToContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake pdf"))
	}))
	defer srv.Close()

	base := core.NewTestBaseClient("tok", "http://unused", srv.Client())
	ac := NewClient(base)
	ac.httpClient = srv.Client()

	path, err := ac.DownloadToFile(srv.URL+"/file", t.TempDir(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, ".pdf") {
		t.Errorf("expected path to end with .pdf, got %q", path)
	}
}

func TestDownloadToFile_FilenameOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", `attachment; filename="original.txt"`)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("content"))
	}))
	defer srv.Close()

	base := core.NewTestBaseClient("tok", "http://unused", srv.Client())
	ac := NewClient(base)
	ac.httpClient = srv.Client()

	path, err := ac.DownloadToFile(srv.URL+"/file", t.TempDir(), "override.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(path, "override.txt") {
		t.Errorf("expected filename override in path, got %q", path)
	}
}
