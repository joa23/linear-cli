// Package cache provides a durable on-disk cache of "team context" data
// (workflow states, labels, projects, members) keyed by team UUID under
// $XDG_CACHE_HOME/linear (defaults to ~/.cache/linear on Linux/macOS).
//
// The cache is a write-through enrichment layer for enumeration commands
// (teams states, teams labels, users list, projects list). Every entry
// carries a token fingerprint so logging in as a different OAuth identity
// transparently invalidates the previous identity's cache without mixing
// data across workspaces.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Schema/freshness constants. Kept here so they can be tuned with one edit.
const (
	// CacheVersion bumps whenever TeamCache changes incompatibly. Loaders
	// treat any mismatched version as ErrNotFound, so old caches are
	// silently re-fetched on the next call.
	CacheVersion = 1

	// FreshTTL: under this age the cache is used silently.
	FreshTTL = 24 * time.Hour

	// StaleTTL: at or beyond this age the cache is treated as missing
	// and re-fetched. Between FreshTTL and StaleTTL the cache is still
	// usable but a footer hint is printed.
	StaleTTL = 7 * 24 * time.Hour

	dirMode  os.FileMode = 0o700
	fileMode os.FileMode = 0o600
)

// Errors returned by the store.
var (
	ErrNotFound = errors.New("cache entry not found")
)

// TeamSummary identifies which team a cache entry belongs to.
type TeamSummary struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// StateEntry mirrors core.WorkflowState with only the fields we need to keep.
type StateEntry struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Position float64 `json:"position"`
}

// LabelEntry mirrors core.Label.
type LabelEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ProjectEntry mirrors core.Project.
type ProjectEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// MemberEntry mirrors core.User for team membership.
type MemberEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// TeamCache is the on-disk payload for one team.
type TeamCache struct {
	Version          int            `json:"version"`
	TokenFingerprint string         `json:"token_fingerprint"`
	FetchedAt        time.Time      `json:"fetched_at"`
	Team             TeamSummary    `json:"team"`
	States           []StateEntry   `json:"states"`
	Labels           []LabelEntry   `json:"labels"`
	Projects         []ProjectEntry `json:"projects"`
	Members          []MemberEntry  `json:"members"`
}

// Age returns how long since the cache was fetched.
func (c *TeamCache) Age() time.Duration { return time.Since(c.FetchedAt) }

// IsFresh reports whether the cache is younger than FreshTTL.
func (c *TeamCache) IsFresh() bool { return c.Age() < FreshTTL }

// IsHardExpired reports whether the cache is older than StaleTTL and
// should be treated as missing.
func (c *TeamCache) IsHardExpired() bool { return c.Age() >= StaleTTL }

// IndexEntry is one record in index.json. Lets us resolve a team key
// (e.g. "MTD") to the per-team file under teams/<uuid>.json without
// scanning every file on disk.
type IndexEntry struct {
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	FetchedAt time.Time `json:"fetched_at"`
}

// indexFile is the on-disk schema for ~/.cache/linear/index.json. The map
// key is the team key (e.g. "MTD").
type indexFile struct {
	Entries map[string]IndexEntry `json:"entries"`
}

// Store is the file-system-backed cache. All methods are safe for
// concurrent use within a single process; cross-process writes are made
// safe via atomic write-then-rename.
type Store struct {
	dir string
	mu  sync.Mutex // serializes index updates within this process
}

// NewStore returns a Store rooted at $XDG_CACHE_HOME/linear (or
// ~/.cache/linear, etc.) — wherever os.UserCacheDir() points.
func NewStore() (*Store, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("cache: cannot resolve user cache dir: %w", err)
	}
	return NewStoreAt(filepath.Join(base, "linear")), nil
}

// NewStoreAt returns a Store rooted at the given directory. Used by tests
// to operate on a t.TempDir(); production callers use NewStore.
func NewStoreAt(dir string) *Store {
	return &Store{dir: dir}
}

// Path returns the root directory of this store. Useful for showing the
// user where to look or pointing rm -rf at.
func (s *Store) Path() string { return s.dir }

// TokenFingerprint reduces an OAuth/API token to 16 hex chars so we can
// detect identity changes without storing the raw token alongside the
// cache. An empty token returns the empty string, which matches caches
// written before this field existed (treated as compatible).
func TokenFingerprint(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:8])
}

// teamFilePath is the JSON file for the given team UUID.
func (s *Store) teamFilePath(uuid string) string {
	return filepath.Join(s.dir, "teams", uuid+".json")
}

func (s *Store) indexFilePath() string {
	return filepath.Join(s.dir, "index.json")
}

// Load returns the cache entry for the given team key. The currentFingerprint
// parameter invalidates the cache transparently when the OAuth identity has
// changed since the cache was written; pass an empty string to disable that
// check. Returns ErrNotFound when missing, version-mismatched, fingerprint-
// mismatched, or hard-expired (caller treats all three the same).
func (s *Store) Load(teamKey, currentFingerprint string) (*TeamCache, error) {
	idx, err := s.readIndex()
	if err != nil {
		return nil, err
	}
	entry, ok := idx.Entries[teamKey]
	if !ok {
		return nil, ErrNotFound
	}
	return s.LoadByUUID(entry.UUID, currentFingerprint)
}

// LoadByUUID is like Load but takes the team UUID directly.
func (s *Store) LoadByUUID(uuid, currentFingerprint string) (*TeamCache, error) {
	path := s.teamFilePath(uuid)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cache: read %s: %w", path, err)
	}

	var tc TeamCache
	if err := json.Unmarshal(data, &tc); err != nil {
		// Corrupt or partial file — treat as missing so next call refetches.
		return nil, ErrNotFound
	}

	if tc.Version != CacheVersion {
		return nil, ErrNotFound
	}
	if currentFingerprint != "" && tc.TokenFingerprint != "" &&
		tc.TokenFingerprint != currentFingerprint {
		return nil, ErrNotFound
	}
	if tc.IsHardExpired() {
		return nil, ErrNotFound
	}
	return &tc, nil
}

// Save persists the cache and updates the index. Caller is responsible
// for populating Version (use cache.CacheVersion) and FetchedAt
// (use time.Now()).
func (s *Store) Save(c *TeamCache) error {
	if c == nil {
		return errors.New("cache: cannot save nil TeamCache")
	}
	if c.Team.ID == "" || c.Team.Key == "" {
		return errors.New("cache: TeamCache.Team.ID and Team.Key are required")
	}
	if c.Version == 0 {
		c.Version = CacheVersion
	}
	if c.FetchedAt.IsZero() {
		c.FetchedAt = time.Now()
	}

	if err := os.MkdirAll(filepath.Join(s.dir, "teams"), dirMode); err != nil {
		return fmt.Errorf("cache: mkdir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}
	if err := atomicWrite(s.teamFilePath(c.Team.ID), data, fileMode); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.readIndex()
	if err != nil {
		return err
	}
	idx.Entries[c.Team.Key] = IndexEntry{
		UUID:      c.Team.ID,
		Name:      c.Team.Name,
		FetchedAt: c.FetchedAt,
	}
	return s.writeIndex(idx)
}

// Clear removes the cache entry for one team. Returns nil if it didn't
// exist — Clear is idempotent.
func (s *Store) Clear(teamKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.readIndex()
	if err != nil {
		return err
	}
	entry, ok := idx.Entries[teamKey]
	if !ok {
		return nil
	}
	delete(idx.Entries, teamKey)
	if err := s.writeIndex(idx); err != nil {
		return err
	}
	if err := os.Remove(s.teamFilePath(entry.UUID)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("cache: remove team file: %w", err)
	}
	return nil
}

// ClearAll wipes the entire cache directory. Used by `linear cache clear`
// without a team argument.
func (s *Store) ClearAll() error {
	if err := os.RemoveAll(s.dir); err != nil {
		return fmt.Errorf("cache: clear all: %w", err)
	}
	return nil
}

// List returns all index entries sorted by team key, so `cache list`
// output is stable across runs.
func (s *Store) List() (map[string]IndexEntry, error) {
	idx, err := s.readIndex()
	if err != nil {
		return nil, err
	}
	// Return a copy so callers can mutate freely.
	out := make(map[string]IndexEntry, len(idx.Entries))
	for k, v := range idx.Entries {
		out[k] = v
	}
	return out, nil
}

// SortedKeys returns index keys in a deterministic order. Helper for
// rendering `cache list` so tests and humans see a consistent layout.
func SortedKeys(entries map[string]IndexEntry) []string {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// readIndex returns the current index file. A missing file is not an
// error — it just returns an empty index, which is correct for an
// uninitialized cache.
func (s *Store) readIndex() (*indexFile, error) {
	path := s.indexFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &indexFile{Entries: map[string]IndexEntry{}}, nil
		}
		return nil, fmt.Errorf("cache: read index: %w", err)
	}
	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		// Corrupt index — start over rather than fail. The data files
		// are still on disk and will be re-indexed as they're rewritten.
		return &indexFile{Entries: map[string]IndexEntry{}}, nil
	}
	if idx.Entries == nil {
		idx.Entries = map[string]IndexEntry{}
	}
	return &idx, nil
}

func (s *Store) writeIndex(idx *indexFile) error {
	if err := os.MkdirAll(s.dir, dirMode); err != nil {
		return fmt.Errorf("cache: mkdir index: %w", err)
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal index: %w", err)
	}
	return atomicWrite(s.indexFilePath(), data, fileMode)
}

// atomicWrite writes data to a sibling temp file then renames it over
// the target. On POSIX filesystems rename is atomic within the same
// directory, so concurrent readers always see a complete file.
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return fmt.Errorf("cache: mkdir for %s: %w", path, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("cache: temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("cache: write temp: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("cache: chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("cache: close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("cache: rename to %s: %w", path, err)
	}
	return nil
}

// FreshnessLabel returns a short human string like "3h ago" or
// "fetched just now" suitable for the stderr footer.
func FreshnessLabel(age time.Duration) string {
	switch {
	case age < time.Minute:
		return "fetched just now"
	case age < time.Hour:
		return fmt.Sprintf("cached %dm ago", int(age.Minutes()))
	case age < 24*time.Hour:
		return fmt.Sprintf("cached %dh ago", int(age.Hours()))
	default:
		return fmt.Sprintf("cached %dd ago", int(age.Hours()/24))
	}
}
