package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/spf13/cobra"
)

// Exit codes for the watch command.
const (
	watchExitChanged = 0
	watchExitError   = 1
	watchExitTimeout = 2
	watchExitSignal  = 130
)

// minWatchInterval clamps --interval to a polite floor so we don't
// hammer the Linear API even if the user passes something tiny.
const minWatchInterval = 5 * time.Second

// issueFetcher abstracts the single call needed by the watch loop so tests
// can inject a fake without standing up the full Linear client.
type issueFetcher interface {
	GetIssue(identifierOrID string) (*core.Issue, error)
}

// watchSnapshot captures every field the watcher diffs between polls.
// String-valued for easy comparison and printing; description is hashed
// so we don't print huge multi-kilobyte diffs.
type watchSnapshot struct {
	UpdatedAt       string
	State           string
	Assignee        string
	Priority        string
	Title           string
	DescriptionHash string
	Estimate        string
	Cycle           string
	Project         string
	Parent          string
	Labels          string // sorted, comma-separated
	CommentCount    string
}

// watchChange is a single field transition.
type watchChange struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// watchEvent is the JSON payload emitted on each detected change set.
type watchEvent struct {
	Issue   string        `json:"issue"`
	At      string        `json:"at"`
	Changes []watchChange `json:"changes"`
}

func newIssuesWatchCmd() *cobra.Command {
	var (
		interval   time.Duration
		timeout    time.Duration
		keepGoing  bool
		execCmd    string
		outputType string
		quiet      bool
	)

	cmd := &cobra.Command{
		Use:   "watch <issue-id>",
		Short: "Poll an issue and exit when any field changes",
		Long: `Watch a Linear issue and exit when any tracked field changes.

Tracked fields: state, assignee, priority, title, description, estimate,
cycle, project, parent, labels, comment count.

Polls every --interval (default 30s, minimum 5s) until a change is detected
or --timeout is reached. By default exits on the first change; pass --watch
to keep watching indefinitely.

Exit codes:
  0  change detected
  1  error
  2  timeout reached with no changes
  130 interrupted (ctrl-c)

NOTE: For a refreshing on-screen dashboard, prefer Unix watch(1):
  watch -n 30 "linear issues list --priority 1"
Use 'linear issues watch' when you need diff detection, exit-on-change for
chaining (e.g. 'linear issues watch CEN-123 && deploy.sh'), or --exec hooks.`,
		Example: `  # Wait for any change on CEN-123 (30s polling, 1h timeout)
  linear issues watch CEN-123

  # Poll every 10s, give up after 5 minutes
  linear issues watch CEN-123 --interval 10s --timeout 5m

  # Run forever, printing each change as it happens
  linear issues watch CEN-123 --watch

  # Trigger a shell command on every change (env vars: LINEAR_ISSUE_ID,
  # LINEAR_CHANGED_FIELDS, LINEAR_STATE_FROM, LINEAR_STATE_TO, etc.)
  linear issues watch CEN-123 --watch --exec 'say "issue updated: $LINEAR_CHANGED_FIELDS"'

  # JSON output for scripting
  linear issues watch CEN-123 --output json | jq '.changes'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			if interval < minWatchInterval {
				interval = minWatchInterval
			}

			outType, err := parseWatchOutput(outputType)
			if err != nil {
				return err
			}

			ctx, cancel := signal.NotifyContext(
				cmd.Context(),
				os.Interrupt, syscall.SIGTERM,
			)
			defer cancel()

			result := runWatch(ctx, watchOptions{
				IssueID:    args[0],
				Fetcher:    deps.Client,
				Interval:   interval,
				Timeout:    timeout,
				KeepGoing:  keepGoing,
				ExecCmd:    execCmd,
				Output:     outType,
				Quiet:      quiet,
				Stdout:     cmd.OutOrStdout(),
				Stderr:     cmd.ErrOrStderr(),
				Now:        time.Now,
				ExecRunner: runShellCmd,
			})

			if result.Err != nil {
				return result.Err
			}
			// Cobra would otherwise exit 0. Use os.Exit for the
			// timeout/signal codes so callers can branch on them.
			switch result.ExitCode {
			case watchExitChanged:
				return nil
			case watchExitTimeout, watchExitSignal:
				os.Exit(result.ExitCode)
			}
			return nil
		},
	}

	cmd.Flags().DurationVarP(&interval, "interval", "i", 30*time.Second, "Poll interval (minimum 5s)")
	cmd.Flags().DurationVarP(&timeout, "timeout", "T", time.Hour, "Give up after this long (0 = forever)")
	cmd.Flags().BoolVarP(&keepGoing, "watch", "w", false, "Keep watching after first change (loop forever)")
	cmd.Flags().StringVarP(&execCmd, "exec", "x", "", "Shell command to run on each change")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output format: text|json")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Suppress heartbeat output, only print changes")

	return cmd
}

type watchOutputType int

const (
	watchOutputText watchOutputType = iota
	watchOutputJSON
)

func parseWatchOutput(s string) (watchOutputType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "text":
		return watchOutputText, nil
	case "json":
		return watchOutputJSON, nil
	default:
		return 0, fmt.Errorf("invalid --output %q: must be text or json", s)
	}
}

// watchOptions bundles everything runWatch needs. Splitting this out
// keeps the cobra wiring above tiny and makes the loop testable.
type watchOptions struct {
	IssueID    string
	Fetcher    issueFetcher
	Interval   time.Duration
	Timeout    time.Duration // 0 means no timeout
	KeepGoing  bool
	ExecCmd    string
	Output     watchOutputType
	Quiet      bool
	Stdout     io.Writer
	Stderr     io.Writer
	Now        func() time.Time
	ExecRunner func(shell string, env []string) error
}

// watchResult is what runWatch returns. ExitCode mirrors the documented
// CLI exit codes; Err is for unexpected failures (auth, network, etc.).
type watchResult struct {
	ExitCode int
	Err      error
}

func runWatch(ctx context.Context, opts watchOptions) watchResult {
	if opts.Now == nil {
		opts.Now = time.Now
	}

	initialIssue, err := opts.Fetcher.GetIssue(opts.IssueID)
	if err != nil {
		return watchResult{ExitCode: watchExitError, Err: fmt.Errorf("failed to fetch issue: %w", err)}
	}

	baseline := snapshotIssue(initialIssue)
	issueLabel := initialIssue.Identifier
	if issueLabel == "" {
		issueLabel = opts.IssueID
	}

	if opts.Output == watchOutputText && !opts.Quiet {
		fmt.Fprintf(opts.Stdout, "Watching %s — %s\n", issueLabel, initialIssue.Title)
		fmt.Fprintf(opts.Stdout, "Baseline: state=%s assignee=%s priority=%s labels=[%s]\n",
			baseline.State, orDash(baseline.Assignee),
			orDash(baseline.Priority), baseline.Labels)
	}

	var deadline time.Time
	if opts.Timeout > 0 {
		deadline = opts.Now().Add(opts.Timeout)
	}

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	for {
		// Stop on cancellation (ctrl-c) or deadline.
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return watchResult{ExitCode: watchExitSignal}
			}
			return watchResult{ExitCode: watchExitTimeout}
		case <-ticker.C:
		}

		if !deadline.IsZero() && !opts.Now().Before(deadline) {
			if opts.Output == watchOutputText && !opts.Quiet {
				fmt.Fprintf(opts.Stdout, "Timeout reached, no changes detected.\n")
			}
			return watchResult{ExitCode: watchExitTimeout}
		}

		current, err := opts.Fetcher.GetIssue(opts.IssueID)
		if err != nil {
			// Transient errors shouldn't kill a long-running watch; log
			// and keep polling so a flaky network doesn't lose state.
			fmt.Fprintf(opts.Stderr, "warn: fetch failed: %v\n", err)
			continue
		}

		// Fast path: if updatedAt hasn't moved and the comment count
		// hasn't changed either, nothing we care about did.
		nextSnap := snapshotIssue(current)
		if nextSnap.UpdatedAt == baseline.UpdatedAt &&
			nextSnap.CommentCount == baseline.CommentCount {
			continue
		}

		changes := diffSnapshots(baseline, nextSnap)
		if len(changes) == 0 {
			// updatedAt advanced for a field we don't track (e.g. a
			// metadata-only Linear-side update). Re-baseline so we
			// don't keep printing a no-op diff.
			baseline = nextSnap
			continue
		}

		emitChanges(opts, issueLabel, changes)

		if opts.ExecCmd != "" && opts.ExecRunner != nil {
			env := buildExecEnv(issueLabel, changes, nextSnap)
			if err := opts.ExecRunner(opts.ExecCmd, env); err != nil {
				fmt.Fprintf(opts.Stderr, "warn: --exec failed: %v\n", err)
			}
		}

		if !opts.KeepGoing {
			return watchResult{ExitCode: watchExitChanged}
		}

		// In --watch mode, reset baseline and keep going.
		baseline = nextSnap
	}
}

// snapshotIssue extracts every watched field into a flat struct that's
// trivial to compare.
func snapshotIssue(issue *core.Issue) watchSnapshot {
	snap := watchSnapshot{
		UpdatedAt: issue.UpdatedAt,
		State:     issue.State.Name,
		Title:     issue.Title,
	}

	if issue.Assignee != nil {
		snap.Assignee = preferred(issue.Assignee.Email, issue.Assignee.Name, issue.Assignee.DisplayName)
	}
	if issue.Priority != nil {
		snap.Priority = priorityLabel(*issue.Priority)
	}
	if issue.Estimate != nil {
		snap.Estimate = strconv.FormatFloat(*issue.Estimate, 'f', -1, 64)
	}
	if issue.Cycle != nil {
		snap.Cycle = strconv.Itoa(issue.Cycle.Number)
		if issue.Cycle.Name != "" {
			snap.Cycle += " (" + issue.Cycle.Name + ")"
		}
	}
	if issue.Project != nil {
		snap.Project = issue.Project.Name
	}
	if issue.Parent != nil {
		snap.Parent = issue.Parent.Identifier
	}
	if issue.Labels != nil {
		names := make([]string, 0, len(issue.Labels.Nodes))
		for _, l := range issue.Labels.Nodes {
			names = append(names, l.Name)
		}
		sort.Strings(names)
		snap.Labels = strings.Join(names, ",")
	}
	if issue.Comments != nil {
		snap.CommentCount = strconv.Itoa(len(issue.Comments.Nodes))
	}

	if issue.Description != "" {
		sum := sha256.Sum256([]byte(issue.Description))
		snap.DescriptionHash = hex.EncodeToString(sum[:8]) // first 16 hex chars is plenty for change detection
	}

	return snap
}

// diffSnapshots returns one watchChange per field that differs between a and b.
// Order is deterministic so tests and output are stable.
func diffSnapshots(a, b watchSnapshot) []watchChange {
	fields := []struct {
		name string
		from string
		to   string
	}{
		{"state", a.State, b.State},
		{"assignee", a.Assignee, b.Assignee},
		{"priority", a.Priority, b.Priority},
		{"title", a.Title, b.Title},
		{"description", a.DescriptionHash, b.DescriptionHash},
		{"estimate", a.Estimate, b.Estimate},
		{"cycle", a.Cycle, b.Cycle},
		{"project", a.Project, b.Project},
		{"parent", a.Parent, b.Parent},
		{"labels", a.Labels, b.Labels},
		{"comments", a.CommentCount, b.CommentCount},
	}

	var out []watchChange
	for _, f := range fields {
		if f.from != f.to {
			from, to := f.from, f.to
			if f.name == "description" {
				// Hashes aren't useful to a human; just say "changed".
				from = "<hash:" + from + ">"
				to = "<hash:" + to + ">"
			}
			out = append(out, watchChange{Field: f.name, From: from, To: to})
		}
	}
	return out
}

func emitChanges(opts watchOptions, issueLabel string, changes []watchChange) {
	at := opts.Now().Format(time.RFC3339)
	switch opts.Output {
	case watchOutputJSON:
		_ = json.NewEncoder(opts.Stdout).Encode(watchEvent{
			Issue:   issueLabel,
			At:      at,
			Changes: changes,
		})
	default:
		ts := opts.Now().Format("15:04:05")
		for _, c := range changes {
			fmt.Fprintf(opts.Stdout, "[%s] %s: %s -> %s\n",
				ts, c.Field, orDash(c.From), orDash(c.To))
		}
	}
}

// buildExecEnv constructs the environment passed to --exec scripts. Using env
// vars (not args) avoids shell-escaping issues when field values contain
// spaces or quotes.
func buildExecEnv(issueLabel string, changes []watchChange, snap watchSnapshot) []string {
	env := os.Environ()
	env = append(env, "LINEAR_ISSUE_ID="+issueLabel)

	fields := make([]string, 0, len(changes))
	for _, c := range changes {
		fields = append(fields, c.Field)
		upper := strings.ToUpper(c.Field)
		env = append(env,
			"LINEAR_"+upper+"_FROM="+c.From,
			"LINEAR_"+upper+"_TO="+c.To,
		)
	}
	env = append(env, "LINEAR_CHANGED_FIELDS="+strings.Join(fields, ","))
	env = append(env, "LINEAR_STATE="+snap.State)
	return env
}

// runShellCmd runs a command via `sh -c` with the given env. Output is
// streamed straight through so the user sees `--exec` results inline.
func runShellCmd(shell string, env []string) error {
	c := exec.Command("sh", "-c", shell)
	c.Env = env
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// priorityLabel converts Linear's 0-4 priority int to the same labels used
// elsewhere in the CLI so diffs read "Urgent -> High" rather than "1 -> 2".
func priorityLabel(p int) string {
	switch p {
	case 0:
		return "None"
	case 1:
		return "Urgent"
	case 2:
		return "High"
	case 3:
		return "Normal"
	case 4:
		return "Low"
	default:
		return strconv.Itoa(p)
	}
}

func preferred(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
