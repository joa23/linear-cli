package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// --- snapshotIssue ---------------------------------------------------------

func TestSnapshotIssue_CapturesAllWatchedFields(t *testing.T) {
	prio := 1
	est := 3.0
	issue := &core.Issue{
		Identifier:  "TEST-42",
		Title:       "Original title",
		Description: "hello world",
		UpdatedAt:   "2026-05-30T20:00:00Z",
		Assignee:    &core.User{Name: "Alice", Email: "alice@example.com"},
		Priority:    &prio,
		Estimate:    &est,
		Cycle:       &core.CycleReference{Number: 12, Name: "Sprint X"},
		Project:     &core.Project{Name: "Auth Revamp"},
		Parent:      &core.ParentIssue{Identifier: "TEST-1"},
		Labels: &core.LabelConnection{Nodes: []core.Label{
			{Name: "backend"}, {Name: "auth"},
		}},
		Comments: &core.CommentConnection{Nodes: []core.Comment{
			{ID: "c1"}, {ID: "c2"},
		}},
	}
	issue.State.Name = "In Progress"

	snap := snapshotIssue(issue)

	if snap.State != "In Progress" {
		t.Errorf("state: got %q", snap.State)
	}
	if snap.Assignee != "alice@example.com" {
		t.Errorf("assignee: expected email preference, got %q", snap.Assignee)
	}
	if snap.Priority != "Urgent" {
		t.Errorf("priority: expected Urgent label, got %q", snap.Priority)
	}
	if snap.Estimate != "3" {
		t.Errorf("estimate: got %q", snap.Estimate)
	}
	if snap.Cycle != "12 (Sprint X)" {
		t.Errorf("cycle: got %q", snap.Cycle)
	}
	if snap.Project != "Auth Revamp" {
		t.Errorf("project: got %q", snap.Project)
	}
	if snap.Parent != "TEST-1" {
		t.Errorf("parent: got %q", snap.Parent)
	}
	if snap.Labels != "auth,backend" {
		t.Errorf("labels: expected sorted, got %q", snap.Labels)
	}
	if snap.CommentCount != "2" {
		t.Errorf("comments: got %q", snap.CommentCount)
	}
	if snap.DescriptionHash == "" {
		t.Error("description hash should be non-empty when description set")
	}
	if snap.UpdatedAt != "2026-05-30T20:00:00Z" {
		t.Errorf("updatedAt: got %q", snap.UpdatedAt)
	}
}

func TestSnapshotIssue_NilOptionalFields(t *testing.T) {
	issue := &core.Issue{Identifier: "TEST-1", Title: "x"}
	issue.State.Name = "Backlog"

	snap := snapshotIssue(issue)

	if snap.Assignee != "" || snap.Priority != "" || snap.Estimate != "" ||
		snap.Cycle != "" || snap.Project != "" || snap.Parent != "" ||
		snap.Labels != "" || snap.CommentCount != "" || snap.DescriptionHash != "" {
		t.Errorf("nil-everything snapshot should be empty-valued, got %+v", snap)
	}
}

func TestSnapshotIssue_LabelsSortedRegardlessOfOrder(t *testing.T) {
	mk := func(names ...string) watchSnapshot {
		nodes := make([]core.Label, len(names))
		for i, n := range names {
			nodes[i] = core.Label{Name: n}
		}
		issue := &core.Issue{Labels: &core.LabelConnection{Nodes: nodes}}
		return snapshotIssue(issue)
	}
	if mk("b", "a", "c").Labels != mk("c", "a", "b").Labels {
		t.Error("label order should not affect snapshot equality")
	}
}

// --- diffSnapshots ---------------------------------------------------------

func TestDiffSnapshots_IdenticalReturnsNothing(t *testing.T) {
	a := watchSnapshot{State: "Todo", Title: "x"}
	if diff := diffSnapshots(a, a); len(diff) != 0 {
		t.Errorf("expected no changes, got %+v", diff)
	}
}

func TestDiffSnapshots_DetectsSingleField(t *testing.T) {
	a := watchSnapshot{State: "Todo"}
	b := watchSnapshot{State: "In Progress"}
	diff := diffSnapshots(a, b)
	if len(diff) != 1 || diff[0].Field != "state" ||
		diff[0].From != "Todo" || diff[0].To != "In Progress" {
		t.Errorf("unexpected diff: %+v", diff)
	}
}

func TestDiffSnapshots_DetectsMultipleFields(t *testing.T) {
	a := watchSnapshot{State: "Todo", Assignee: "alice", Priority: "Normal"}
	b := watchSnapshot{State: "Done", Assignee: "bob", Priority: "Normal"}
	diff := diffSnapshots(a, b)
	if len(diff) != 2 {
		t.Fatalf("expected 2 changes, got %d: %+v", len(diff), diff)
	}
	// Order must be deterministic: state before assignee per fields slice
	if diff[0].Field != "state" || diff[1].Field != "assignee" {
		t.Errorf("expected stable field order state,assignee; got %s,%s",
			diff[0].Field, diff[1].Field)
	}
}

func TestDiffSnapshots_DescriptionRendersAsHashMarker(t *testing.T) {
	a := watchSnapshot{DescriptionHash: "aaa"}
	b := watchSnapshot{DescriptionHash: "bbb"}
	diff := diffSnapshots(a, b)
	if len(diff) != 1 || diff[0].Field != "description" {
		t.Fatalf("expected description diff, got %+v", diff)
	}
	if !strings.Contains(diff[0].From, "hash:") || !strings.Contains(diff[0].To, "hash:") {
		t.Errorf("description diff should mark hashes; got from=%q to=%q",
			diff[0].From, diff[0].To)
	}
}

// --- parseWatchOutput ------------------------------------------------------

func TestParseWatchOutput(t *testing.T) {
	cases := map[string]struct {
		in      string
		want    watchOutputType
		wantErr bool
	}{
		"empty defaults to text":  {"", watchOutputText, false},
		"text":                    {"text", watchOutputText, false},
		"TEXT case insensitive":   {"TEXT", watchOutputText, false},
		"json":                    {"json", watchOutputJSON, false},
		"trim whitespace":         {"  json ", watchOutputJSON, false},
		"unknown rejected":        {"yaml", 0, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := parseWatchOutput(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// --- buildExecEnv ----------------------------------------------------------

func TestBuildExecEnv_ContainsExpectedKeys(t *testing.T) {
	changes := []watchChange{
		{Field: "state", From: "Todo", To: "In Progress"},
		{Field: "assignee", From: "alice", To: "bob"},
	}
	snap := watchSnapshot{State: "In Progress"}
	env := buildExecEnv("TEST-1", changes, snap)
	joined := strings.Join(env, "\n")

	for _, want := range []string{
		"LINEAR_ISSUE_ID=TEST-1",
		"LINEAR_CHANGED_FIELDS=state,assignee",
		"LINEAR_STATE_FROM=Todo",
		"LINEAR_STATE_TO=In Progress",
		"LINEAR_ASSIGNEE_FROM=alice",
		"LINEAR_ASSIGNEE_TO=bob",
		"LINEAR_STATE=In Progress",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("env missing %q\nfull env:\n%s", want, joined)
		}
	}
}

// --- runWatch (integration with fake fetcher) ------------------------------

// fakeFetcher replays a queue of issues per GetIssue call.
type fakeFetcher struct {
	mu     sync.Mutex
	queue  []*core.Issue
	err    error
	calls  int32
}

func (f *fakeFetcher) GetIssue(id string) (*core.Issue, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.queue) == 0 {
		// Repeat the last issue forever so polling past the queue end is safe.
		return nil, errors.New("fetcher queue empty")
	}
	issue := f.queue[0]
	if len(f.queue) > 1 {
		f.queue = f.queue[1:]
	}
	return issue, nil
}

func issueWith(state string, updatedAt string) *core.Issue {
	i := &core.Issue{Identifier: "TEST-1", Title: "Sample", UpdatedAt: updatedAt}
	i.State.Name = state
	return i
}

func TestRunWatch_ExitsOnFirstChange(t *testing.T) {
	fake := &fakeFetcher{queue: []*core.Issue{
		issueWith("Todo", "t1"),
		issueWith("In Progress", "t2"),
	}}
	var out, errBuf bytes.Buffer

	res := runWatch(context.Background(), watchOptions{
		IssueID:  "TEST-1",
		Fetcher:  fake,
		Interval: 5 * time.Millisecond,
		Timeout:  500 * time.Millisecond,
		Output:   watchOutputText,
		Stdout:   &out,
		Stderr:   &errBuf,
		Now:      time.Now,
	})

	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if res.ExitCode != watchExitChanged {
		t.Errorf("expected exit %d (changed), got %d", watchExitChanged, res.ExitCode)
	}
	if !strings.Contains(out.String(), "state: Todo -> In Progress") {
		t.Errorf("output missing state transition, got: %s", out.String())
	}
}

func TestRunWatch_TimesOutWhenNothingChanges(t *testing.T) {
	steady := issueWith("Todo", "t1")
	fake := &fakeFetcher{queue: []*core.Issue{steady}}
	var out bytes.Buffer

	res := runWatch(context.Background(), watchOptions{
		IssueID:  "TEST-1",
		Fetcher:  fake,
		Interval: 5 * time.Millisecond,
		Timeout:  30 * time.Millisecond,
		Output:   watchOutputText,
		Quiet:    true,
		Stdout:   &out,
		Stderr:   &out,
		Now:      time.Now,
	})

	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if res.ExitCode != watchExitTimeout {
		t.Errorf("expected timeout exit %d, got %d", watchExitTimeout, res.ExitCode)
	}
}

func TestRunWatch_FastPathSkipsDiffWhenUpdatedAtUnchanged(t *testing.T) {
	steady := issueWith("Todo", "t1")
	fake := &fakeFetcher{queue: []*core.Issue{steady, steady, steady}}
	var out bytes.Buffer

	res := runWatch(context.Background(), watchOptions{
		IssueID:  "TEST-1",
		Fetcher:  fake,
		Interval: 2 * time.Millisecond,
		Timeout:  20 * time.Millisecond,
		Output:   watchOutputText,
		Quiet:    true,
		Stdout:   &out,
		Stderr:   &out,
		Now:      time.Now,
	})

	if res.ExitCode != watchExitTimeout {
		t.Errorf("expected timeout when no changes, got exit %d", res.ExitCode)
	}
	// No change lines should appear in output.
	if strings.Contains(out.String(), " -> ") {
		t.Errorf("fast path failed: spurious change emitted: %s", out.String())
	}
}

func TestRunWatch_KeepGoingLoopsThroughMultipleChanges(t *testing.T) {
	fake := &fakeFetcher{queue: []*core.Issue{
		issueWith("Todo", "t1"),
		issueWith("In Progress", "t2"),
		issueWith("In Review", "t3"),
		issueWith("Done", "t4"),
	}}
	var out bytes.Buffer

	res := runWatch(context.Background(), watchOptions{
		IssueID:   "TEST-1",
		Fetcher:   fake,
		Interval:  3 * time.Millisecond,
		Timeout:   200 * time.Millisecond,
		KeepGoing: true,
		Output:    watchOutputText,
		Quiet:     true,
		Stdout:    &out,
		Stderr:    &out,
		Now:       time.Now,
	})

	// --watch mode keeps polling after each change; once we exhaust the
	// queue the fake returns an error which is logged but doesn't exit.
	// Loop ends by timeout.
	if res.ExitCode != watchExitTimeout {
		t.Errorf("expected timeout exit in --watch mode, got %d", res.ExitCode)
	}

	output := out.String()
	for _, want := range []string{
		"Todo -> In Progress",
		"In Progress -> In Review",
		"In Review -> Done",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("missing transition %q in output:\n%s", want, output)
		}
	}
}

func TestRunWatch_InitialFetchErrorReturnsError(t *testing.T) {
	fake := &fakeFetcher{err: errors.New("auth failed")}
	res := runWatch(context.Background(), watchOptions{
		IssueID:  "TEST-1",
		Fetcher:  fake,
		Interval: 5 * time.Millisecond,
		Timeout:  20 * time.Millisecond,
		Output:   watchOutputText,
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Now:      time.Now,
	})

	if res.Err == nil {
		t.Fatal("expected error from initial fetch")
	}
	if res.ExitCode != watchExitError {
		t.Errorf("expected exit %d, got %d", watchExitError, res.ExitCode)
	}
}

func TestRunWatch_CancelledContextExitsSignal(t *testing.T) {
	fake := &fakeFetcher{queue: []*core.Issue{issueWith("Todo", "t1")}}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan watchResult, 1)
	go func() {
		done <- runWatch(ctx, watchOptions{
			IssueID:  "TEST-1",
			Fetcher:  fake,
			Interval: 5 * time.Millisecond,
			Timeout:  time.Second,
			Output:   watchOutputText,
			Quiet:    true,
			Stdout:   &bytes.Buffer{},
			Stderr:   &bytes.Buffer{},
			Now:      time.Now,
		})
	}()

	// Let the loop start, then cancel.
	time.Sleep(15 * time.Millisecond)
	cancel()

	select {
	case res := <-done:
		if res.ExitCode != watchExitSignal {
			t.Errorf("expected signal exit %d, got %d", watchExitSignal, res.ExitCode)
		}
	case <-time.After(time.Second):
		t.Fatal("runWatch did not return after cancellation")
	}
}

func TestRunWatch_JSONOutputProducesParseableLine(t *testing.T) {
	fake := &fakeFetcher{queue: []*core.Issue{
		issueWith("Todo", "t1"),
		issueWith("Done", "t2"),
	}}
	var out bytes.Buffer

	res := runWatch(context.Background(), watchOptions{
		IssueID:  "TEST-1",
		Fetcher:  fake,
		Interval: 5 * time.Millisecond,
		Timeout:  500 * time.Millisecond,
		Output:   watchOutputJSON,
		Quiet:    true,
		Stdout:   &out,
		Stderr:   &bytes.Buffer{},
		Now:      time.Now,
	})

	if res.ExitCode != watchExitChanged {
		t.Fatalf("expected change exit, got %d (err=%v)", res.ExitCode, res.Err)
	}
	// Must contain a JSON object with issue, at, changes fields.
	body := out.String()
	for _, want := range []string{`"issue":"TEST-1"`, `"changes":[`, `"field":"state"`} {
		if !strings.Contains(body, want) {
			t.Errorf("JSON output missing %q\nfull body:\n%s", want, body)
		}
	}
}
