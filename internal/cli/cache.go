package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/joa23/linear-cli/internal/cache"
	"github.com/joa23/linear-cli/pkg/linear"
	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/spf13/cobra"
)

// newCacheCmd builds the `linear cache` command tree.
func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the durable team-context cache",
		Long: `Manage the on-disk cache of team enumeration data (workflow states,
labels, projects, members). The cache lives under $XDG_CACHE_HOME/linear
(typically ~/.cache/linear) and is shared across project checkouts on the
same machine.

The cache is read transparently by 'teams states', 'teams labels',
'users list --team', and 'projects list --team'. Use this command tree
to inspect, refresh, or clear it.`,
	}

	cmd.AddCommand(
		newCacheFetchCmd(),
		newCacheRefreshCmd(),
		newCacheShowCmd(),
		newCacheListCmd(),
		newCacheClearCmd(),
		newCachePathCmd(),
	)
	return cmd
}

// --- subcommands ---------------------------------------------------------

func newCacheFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch <team>",
		Short: "Fetch states/labels/projects/members for a team and cache them",
		Args:  cobra.ExactArgs(1),
		Example: `  linear cache fetch MTD
  linear cache fetch CEN`,
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}
			store, err := cache.NewStore()
			if err != nil {
				return err
			}
			tc, err := fetchTeamContext(deps.Client, args[0])
			if err != nil {
				return err
			}
			if err := store.Save(tc); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(),
				"Cached %s (%s): %d states, %d labels, %d projects, %d members\n",
				tc.Team.Key, tc.Team.Name,
				len(tc.States), len(tc.Labels), len(tc.Projects), len(tc.Members))
			return nil
		},
	}
}

func newCacheRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh [team]",
		Short: "Refresh all cached teams, or a single team",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}
			store, err := cache.NewStore()
			if err != nil {
				return err
			}

			var teamKeys []string
			if len(args) == 1 {
				teamKeys = []string{args[0]}
			} else {
				entries, err := store.List()
				if err != nil {
					return err
				}
				teamKeys = cache.SortedKeys(entries)
			}
			if len(teamKeys) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Cache is empty — nothing to refresh.")
				return nil
			}

			for _, key := range teamKeys {
				tc, err := fetchTeamContext(deps.Client, key)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warn: refresh %s: %v\n", key, err)
					continue
				}
				if err := store.Save(tc); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warn: save %s: %v\n", key, err)
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Refreshed %s (%s)\n", tc.Team.Key, tc.Team.Name)
			}
			return nil
		},
	}
}

func newCacheShowCmd() *cobra.Command {
	var outputType string
	cmd := &cobra.Command{
		Use:   "show <team>",
		Short: "Show cached data for a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}
			store, err := cache.NewStore()
			if err != nil {
				return err
			}
			fp := cache.TokenFingerprint(deps.Client.GetAPIToken())
			tc, err := store.Load(args[0], fp)
			if err != nil {
				if errors.Is(err, cache.ErrNotFound) {
					return fmt.Errorf("no cache entry for %q. Run: linear cache fetch %s", args[0], args[0])
				}
				return err
			}

			if strings.EqualFold(strings.TrimSpace(outputType), "json") {
				data, err := json.MarshalIndent(tc, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			renderCacheShow(cmd.OutOrStdout(), tc)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")
	return cmd
}

func newCacheListCmd() *cobra.Command {
	var outputType string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all cached teams with freshness",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := cache.NewStore()
			if err != nil {
				return err
			}
			entries, err := store.List()
			if err != nil {
				return err
			}

			if strings.EqualFold(strings.TrimSpace(outputType), "json") {
				data, err := json.MarshalIndent(entries, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			if len(entries) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(),
					"No teams cached. Run: linear cache fetch <TEAM> (or 'linear init')\n")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "CACHED TEAMS")
			fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("─", 40))
			for _, key := range cache.SortedKeys(entries) {
				e := entries[key]
				fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s (%s)\n",
					key, cache.FreshnessLabel(time.Since(e.FetchedAt)), e.Name)
			}
			fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("─", 40))
			fmt.Fprintf(cmd.OutOrStdout(), "Cache root: %s\n", store.Path())
			return nil
		},
	}
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")
	return cmd
}

func newCacheClearCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "clear [team]",
		Short: "Clear cache for one team, or all teams with --all",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := cache.NewStore()
			if err != nil {
				return err
			}
			if all {
				if err := store.ClearAll(); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Cleared all cached teams.")
				return nil
			}
			if len(args) == 0 {
				return fmt.Errorf("specify a team key, or pass --all to clear everything")
			}
			if err := store.Clear(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Cleared cache for %s.\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Clear every cached team")
	return cmd
}

func newCachePathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the on-disk cache root",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := cache.NewStore()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), store.Path())
			return nil
		},
	}
}

// --- shared fetch helper -------------------------------------------------

// fetchTeamContext resolves teamKey to a UUID then concurrently fetches
// states, labels, projects, and members. Returns a TeamCache populated
// with everything needed for write-through.
//
// Concurrency: each slice runs in its own goroutine. Errors are collected
// per-slice; if any slice fails we abort and return the first error so the
// caller doesn't silently cache a partially-empty team.
func fetchTeamContext(client *linear.Client, teamKey string) (*cache.TeamCache, error) {
	team, err := resolveTeam(client, teamKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve team %q: %w", teamKey, err)
	}

	tc := &cache.TeamCache{
		Version:          cache.CacheVersion,
		TokenFingerprint: cache.TokenFingerprint(client.GetAPIToken()),
		FetchedAt:        time.Now(),
		Team: cache.TeamSummary{
			ID:   team.ID,
			Key:  team.Key,
			Name: team.Name,
		},
	}

	var (
		wg      sync.WaitGroup
		errMu   sync.Mutex
		errList []error
	)
	record := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		errList = append(errList, err)
		errMu.Unlock()
	}

	wg.Add(4)

	go func() {
		defer wg.Done()
		states, err := client.GetWorkflowStates(team.ID)
		if err != nil {
			record(fmt.Errorf("states: %w", err))
			return
		}
		for _, s := range states {
			tc.States = append(tc.States, cache.StateEntry{
				ID: s.ID, Name: s.Name, Type: s.Type, Position: s.Position,
			})
		}
	}()

	go func() {
		defer wg.Done()
		labels, err := client.Teams.ListLabels(team.ID)
		if err != nil {
			record(fmt.Errorf("labels: %w", err))
			return
		}
		for _, l := range labels {
			tc.Labels = append(tc.Labels, cache.LabelEntry{
				ID: l.ID, Name: l.Name, Color: l.Color,
			})
		}
	}()

	go func() {
		defer wg.Done()
		projects, err := client.Projects.ListByTeam(team.ID, 250)
		if err != nil {
			record(fmt.Errorf("projects: %w", err))
			return
		}
		for _, p := range projects {
			tc.Projects = append(tc.Projects, cache.ProjectEntry{
				ID: p.ID, Name: p.Name, State: p.State,
			})
		}
	}()

	go func() {
		defer wg.Done()
		members, err := client.ListUsers(&core.UserFilter{TeamID: team.ID, First: 250})
		if err != nil {
			record(fmt.Errorf("members: %w", err))
			return
		}
		for _, m := range members {
			tc.Members = append(tc.Members, cache.MemberEntry{
				ID: m.ID, Name: preferredName(m.DisplayName, m.Name), Email: m.Email,
			})
		}
	}()

	wg.Wait()
	if len(errList) > 0 {
		return nil, errList[0]
	}
	return tc, nil
}

// resolveTeam takes a key/name and returns the canonical core.Team. Uses
// GetTeams (cached by the in-process resolver) so we get the full Team
// struct including the canonical Name.
func resolveTeam(client *linear.Client, keyOrName string) (*core.Team, error) {
	teams, err := client.GetTeams()
	if err != nil {
		return nil, err
	}
	keyOrNameLower := strings.ToLower(strings.TrimSpace(keyOrName))
	for i := range teams {
		t := &teams[i]
		if strings.EqualFold(t.Key, keyOrName) ||
			strings.EqualFold(t.Name, keyOrName) ||
			t.ID == keyOrName ||
			strings.ToLower(t.Key) == keyOrNameLower {
			return t, nil
		}
	}
	return nil, fmt.Errorf("team not found: %s", keyOrName)
}

func preferredName(candidates ...string) string {
	for _, c := range candidates {
		if c != "" {
			return c
		}
	}
	return ""
}

// --- per-slice renderers (used when reading from cache) -----------------

// renderStatesFromCache mirrors the live "teams states" output so callers
// can't tell whether the data came from cache or API by glancing at the
// shape of the result.
func renderStatesFromCache(states []cache.StateEntry, asJSON bool) (string, error) {
	if asJSON {
		data, err := json.MarshalIndent(states, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	if len(states) == 0 {
		return "No workflow states found.", nil
	}
	out := fmt.Sprintf("WORKFLOW STATES (%d)\n%s\n", len(states), strings.Repeat("─", 40))
	for _, s := range states {
		out += fmt.Sprintf("  %s [%s] - %s\n", s.Name, s.Type, s.ID)
	}
	return out, nil
}

func renderLabelsFromCache(labels []cache.LabelEntry, asJSON bool) (string, error) {
	if asJSON {
		data, err := json.MarshalIndent(labels, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	if len(labels) == 0 {
		return "No labels found.", nil
	}
	out := fmt.Sprintf("LABELS (%d)\n%s\n", len(labels), strings.Repeat("─", 40))
	for _, l := range labels {
		out += fmt.Sprintf("  %s\n", l.Name)
	}
	return out, nil
}

func renderProjectsFromCache(projects []cache.ProjectEntry, asJSON bool) (string, error) {
	if asJSON {
		data, err := json.MarshalIndent(projects, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	if len(projects) == 0 {
		return "No projects found.", nil
	}
	out := fmt.Sprintf("PROJECTS (%d)\n%s\n", len(projects), strings.Repeat("─", 40))
	for _, p := range projects {
		out += fmt.Sprintf("  %-30s [%s]\n", p.Name, p.State)
	}
	return out, nil
}

func renderMembersFromCache(members []cache.MemberEntry, asJSON bool) (string, error) {
	if asJSON {
		data, err := json.MarshalIndent(members, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	if len(members) == 0 {
		return "No members found.", nil
	}
	out := fmt.Sprintf("MEMBERS (%d)\n%s\n", len(members), strings.Repeat("─", 40))
	for _, m := range members {
		out += fmt.Sprintf("  %-25s %s\n", m.Name, m.Email)
	}
	return out, nil
}

// printFreshnessFooter writes a "cached Nh ago" hint to stderr so the user
// knows the data came from cache without polluting stdout (so JSON output
// stays pipe-clean).
func printFreshnessFooter(w io.Writer, age time.Duration, stale bool) {
	if stale {
		fmt.Fprintf(w, "(%s — STALE, run with --refresh to update)\n",
			cache.FreshnessLabel(age))
		return
	}
	fmt.Fprintf(w, "(%s. --refresh to update, --no-cache to bypass)\n",
		cache.FreshnessLabel(age))
}

// cacheReadOptions captures the standard --cached/--refresh/--no-cache flags.
type cacheReadOptions struct {
	UseOnly  bool // --cached: error if missing/stale, never hit API
	Refresh  bool // --refresh: force live fetch + write-through
	Bypass   bool // --no-cache: live only, no write-through
}

// loadFromCache returns the cache entry for teamKey if usable for reads.
// Returns (nil, nil) when the caller should fall back to the live path
// (cache missing, stale, or --no-cache/--refresh set). Returns an error
// only for --cached when the entry is missing.
func loadFromCache(deps *Dependencies, teamKey string, opts cacheReadOptions) (*cache.TeamCache, error) {
	if opts.Bypass || opts.Refresh {
		return nil, nil
	}
	store, err := cache.NewStore()
	if err != nil {
		return nil, err
	}
	fp := cache.TokenFingerprint(deps.Client.GetAPIToken())
	tc, err := store.Load(teamKey, fp)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			if opts.UseOnly {
				return nil, fmt.Errorf("--cached set but no cache entry for %q. Run: linear cache fetch %s",
					teamKey, teamKey)
			}
			return nil, nil
		}
		return nil, err
	}
	return tc, nil
}

// writeThroughCache fetches a fresh team context and stores it. Errors are
// non-fatal: the calling command already produced its primary output, the
// cache write is best-effort, so a warning to stderr is sufficient.
func writeThroughCache(deps *Dependencies, teamKey string, stderr io.Writer) {
	tc, err := fetchTeamContext(deps.Client, teamKey)
	if err != nil {
		fmt.Fprintf(stderr, "warn: cache write-through failed: %v\n", err)
		return
	}
	store, err := cache.NewStore()
	if err != nil {
		fmt.Fprintf(stderr, "warn: cache write-through failed: %v\n", err)
		return
	}
	if err := store.Save(tc); err != nil {
		fmt.Fprintf(stderr, "warn: cache write-through failed: %v\n", err)
	}
}

// renderCacheShow prints a human-friendly summary of one team cache.
func renderCacheShow(w io.Writer, tc *cache.TeamCache) {
	fmt.Fprintf(w, "TEAM %s — %s\n", tc.Team.Key, tc.Team.Name)
	fmt.Fprintln(w, strings.Repeat("─", 60))
	fmt.Fprintf(w, "Fetched: %s (%s)\n",
		tc.FetchedAt.Format(time.RFC3339),
		cache.FreshnessLabel(time.Since(tc.FetchedAt)))
	fmt.Fprintln(w)

	fmt.Fprintf(w, "STATES (%d)\n", len(tc.States))
	for _, s := range tc.States {
		fmt.Fprintf(w, "  %-22s [%s]\n", s.Name, s.Type)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "LABELS (%d)\n", len(tc.Labels))
	for _, l := range tc.Labels {
		fmt.Fprintf(w, "  %s\n", l.Name)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "PROJECTS (%d)\n", len(tc.Projects))
	for _, p := range tc.Projects {
		fmt.Fprintf(w, "  %-30s [%s]\n", p.Name, p.State)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "MEMBERS (%d)\n", len(tc.Members))
	for _, m := range tc.Members {
		fmt.Fprintf(w, "  %-25s %s\n", m.Name, m.Email)
	}
}
