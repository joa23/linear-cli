package attachments

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joa23/linear-cli/internal/linear/core"
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
		{"https://example.com/uploads.linear.app.fake", true}, // contains the string
	}
	for _, c := range cases {
		got := isPrivateLinearURL(c.url)
		if got != c.want {
			t.Errorf("isPrivateLinearURL(%q) = %v, want %v", c.url, got, c.want)
		}
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
