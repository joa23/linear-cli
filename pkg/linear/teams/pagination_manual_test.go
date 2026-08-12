package teams

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// sequentialTransport returns a different canned response body on each
// successive RoundTrip call, in order. Used to simulate multi-page
// GraphQL pagination responses.
type sequentialTransport struct {
	bodies []string
	calls  []string // captures request bodies for assertions
	idx    int
}

func (s *sequentialTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	buf, _ := io.ReadAll(req.Body)
	s.calls = append(s.calls, string(buf))

	if s.idx >= len(s.bodies) {
		panic("sequentialTransport: no more canned responses")
	}
	body := s.bodies[s.idx]
	s.idx++

	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}, nil
}

// TestGetTeams_PaginatesAcrossMultiplePages verifies that GetTeams follows
// pageInfo.hasNextPage/endCursor to fetch every page of teams, rather than
// silently truncating at the first page (regression test for
// joa23/linear-cli#71).
func TestGetTeams_PaginatesAcrossMultiplePages(t *testing.T) {
	page1 := `{"data":{"teams":{"nodes":[{"id":"t1","name":"Team One","key":"ONE"},{"id":"t2","name":"Team Two","key":"TWO"}],"pageInfo":{"hasNextPage":true,"endCursor":"cursor-abc"}}}}`
	page2 := `{"data":{"teams":{"nodes":[{"id":"t3","name":"Team Three","key":"THR"}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`

	transport := &sequentialTransport{bodies: []string{page1, page2}}

	base := core.NewBaseClient("fake-token")
	base.SetHTTPClient(&http.Client{Transport: transport})

	client := NewClient(base)

	teams, err := client.GetTeams()
	if err != nil {
		t.Fatalf("GetTeams returned error: %v", err)
	}

	if len(teams) != 3 {
		t.Fatalf("expected 3 teams across 2 pages, got %d: %+v", len(teams), teams)
	}

	wantIDs := map[string]bool{"t1": true, "t2": true, "t3": true}
	for _, tm := range teams {
		if !wantIDs[tm.ID] {
			t.Errorf("unexpected team ID in result: %s", tm.ID)
		}
		delete(wantIDs, tm.ID)
	}
	if len(wantIDs) != 0 {
		t.Errorf("missing expected team IDs: %+v", wantIDs)
	}

	// Verify exactly 2 HTTP calls were made (one per page)
	if len(transport.calls) != 2 {
		t.Fatalf("expected 2 GraphQL requests (one per page), got %d", len(transport.calls))
	}

	// First request must NOT include an "after" cursor
	var firstReq struct {
		Variables map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal([]byte(transport.calls[0]), &firstReq); err != nil {
		t.Fatalf("failed to unmarshal first request: %v", err)
	}
	if _, hasAfter := firstReq.Variables["after"]; hasAfter {
		t.Errorf("first request should not include 'after' variable, got: %+v", firstReq.Variables)
	}
	if firstReq.Variables["first"] == nil {
		t.Errorf("first request should include 'first' variable")
	}

	// Second request MUST include the cursor returned by page 1
	var secondReq struct {
		Variables map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal([]byte(transport.calls[1]), &secondReq); err != nil {
		t.Fatalf("failed to unmarshal second request: %v", err)
	}
	if secondReq.Variables["after"] != "cursor-abc" {
		t.Errorf("second request 'after' = %v, want %q", secondReq.Variables["after"], "cursor-abc")
	}

	// Sanity: query string should reference pageInfo/hasNextPage/endCursor
	if !strings.Contains(transport.calls[0], "pageInfo") {
		t.Errorf("query does not request pageInfo")
	}
}

// TestGetTeams_SinglePage verifies the common case (a single page, no
// pagination needed) still works and issues exactly one request.
func TestGetTeams_SinglePage(t *testing.T) {
	page1 := `{"data":{"teams":{"nodes":[{"id":"t1","name":"Team One","key":"ONE"}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`

	transport := &sequentialTransport{bodies: []string{page1}}

	base := core.NewBaseClient("fake-token")
	base.SetHTTPClient(&http.Client{Transport: transport})

	client := NewClient(base)

	teams, err := client.GetTeams()
	if err != nil {
		t.Fatalf("GetTeams returned error: %v", err)
	}
	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teams))
	}
	if len(transport.calls) != 1 {
		t.Fatalf("expected exactly 1 request for single page, got %d", len(transport.calls))
	}
}
