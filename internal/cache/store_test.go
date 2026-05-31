package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// helper: build a small valid TeamCache for tests.
func sampleCache(key string) *TeamCache {
	return &TeamCache{
		Version:          CacheVersion,
		TokenFingerprint: "deadbeefdeadbeef",
		FetchedAt:        time.Now(),
		Team:             TeamSummary{ID: "uuid-" + key, Key: key, Name: "Team " + key},
		States: []StateEntry{
			{ID: "s1", Name: "Backlog", Type: "backlog", Position: 0},
		},
		Labels:   []LabelEntry{{ID: "l1", Name: "bug", Color: "#ff0000"}},
		Projects: []ProjectEntry{{ID: "p1", Name: "Project A", State: "started"}},
		Members:  []MemberEntry{{ID: "u1", Name: "Alice", Email: "alice@example.com"}},
	}
}

func TestTokenFingerprint_StableAndShort(t *testing.T) {
	if got := TokenFingerprint(""); got != "" {
		t.Errorf("empty token should produce empty fingerprint, got %q", got)
	}
	a := TokenFingerprint("lin_api_abc")
	b := TokenFingerprint("lin_api_abc")
	if a != b {
		t.Errorf("fingerprint not deterministic: %q vs %q", a, b)
	}
	if len(a) != 16 {
		t.Errorf("expected 16-char fingerprint, got %d (%q)", len(a), a)
	}
	if a == TokenFingerprint("different") {
		t.Error("fingerprint collision for distinct tokens")
	}
}

func TestStore_SaveLoadRoundTrip(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	want := sampleCache("MTD")
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load("MTD", want.TokenFingerprint)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Team.ID != want.Team.ID || got.Team.Key != want.Team.Key {
		t.Errorf("team mismatch: %+v", got.Team)
	}
	if len(got.States) != 1 || got.States[0].Name != "Backlog" {
		t.Errorf("states mismatch: %+v", got.States)
	}
	if len(got.Labels) != 1 || got.Labels[0].Color != "#ff0000" {
		t.Errorf("labels mismatch: %+v", got.Labels)
	}
}

func TestStore_LoadMissingReturnsErrNotFound(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	_, err := s.Load("NOPE", "")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_FingerprintMismatchInvalidates(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	tc := sampleCache("MTD")
	tc.TokenFingerprint = "aaaaaaaaaaaaaaaa"
	if err := s.Save(tc); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Same fingerprint: hit.
	if _, err := s.Load("MTD", "aaaaaaaaaaaaaaaa"); err != nil {
		t.Errorf("matching fingerprint should load, got %v", err)
	}
	// Different fingerprint: invalidates.
	if _, err := s.Load("MTD", "bbbbbbbbbbbbbbbb"); err != ErrNotFound {
		t.Errorf("mismatched fingerprint should return ErrNotFound, got %v", err)
	}
	// Empty fingerprint: bypass check.
	if _, err := s.Load("MTD", ""); err != nil {
		t.Errorf("empty fingerprint should bypass check, got %v", err)
	}
}

func TestStore_VersionMismatchInvalidates(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	tc := sampleCache("MTD")
	if err := s.Save(tc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Manually corrupt the version on disk.
	path := s.teamFilePath(tc.Team.ID)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw["version"] = float64(CacheVersion + 99)
	out, _ := json.Marshal(raw)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Load("MTD", ""); err != ErrNotFound {
		t.Errorf("version mismatch should return ErrNotFound, got %v", err)
	}
}

func TestStore_HardExpiredTreatedAsMissing(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	tc := sampleCache("MTD")
	tc.FetchedAt = time.Now().Add(-StaleTTL - time.Hour)
	if err := s.Save(tc); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := s.Load("MTD", tc.TokenFingerprint); err != ErrNotFound {
		t.Errorf("hard-expired cache should return ErrNotFound, got %v", err)
	}
}

func TestTeamCache_FreshnessHelpers(t *testing.T) {
	c := &TeamCache{FetchedAt: time.Now().Add(-1 * time.Hour)}
	if !c.IsFresh() {
		t.Error("1h-old cache should be fresh")
	}
	if c.IsHardExpired() {
		t.Error("1h-old cache should not be hard-expired")
	}

	stale := &TeamCache{FetchedAt: time.Now().Add(-(FreshTTL + time.Hour))}
	if stale.IsFresh() {
		t.Error("cache older than FreshTTL should not be fresh")
	}
	if stale.IsHardExpired() {
		t.Error("cache between FreshTTL and StaleTTL should not be hard-expired")
	}

	dead := &TeamCache{FetchedAt: time.Now().Add(-(StaleTTL + time.Hour))}
	if !dead.IsHardExpired() {
		t.Error("cache older than StaleTTL should be hard-expired")
	}
}

func TestStore_ClearOne(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	if err := s.Save(sampleCache("MTD")); err != nil {
		t.Fatalf("Save MTD: %v", err)
	}
	if err := s.Save(sampleCache("TEST")); err != nil {
		t.Fatalf("Save TEST: %v", err)
	}

	if err := s.Clear("MTD"); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := s.Load("MTD", ""); err != ErrNotFound {
		t.Errorf("MTD should be gone, got %v", err)
	}
	if _, err := s.Load("TEST", ""); err != nil {
		t.Errorf("TEST should still exist, got %v", err)
	}
}

func TestStore_ClearMissingIsNoop(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	if err := s.Clear("NEVER"); err != nil {
		t.Errorf("clearing missing team should be a no-op, got %v", err)
	}
}

func TestStore_ClearAll(t *testing.T) {
	dir := t.TempDir()
	s := NewStoreAt(dir)
	if err := s.Save(sampleCache("MTD")); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearAll(); err != nil {
		t.Fatalf("ClearAll: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.json")); !os.IsNotExist(err) {
		t.Errorf("expected index.json gone, got stat err=%v", err)
	}
}

func TestStore_List_ReturnsAllEntries(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	if err := s.Save(sampleCache("MTD")); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(sampleCache("TEST")); err != nil {
		t.Fatal(err)
	}

	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d: %+v", len(entries), entries)
	}
	keys := SortedKeys(entries)
	if len(keys) != 2 || keys[0] != "MTD" || keys[1] != "TEST" {
		t.Errorf("sorted keys mismatch: %v", keys)
	}
}

func TestStore_FileModeIs0600(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	tc := sampleCache("MTD")
	if err := s.Save(tc); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		s.teamFilePath(tc.Team.ID),
		s.indexFilePath(),
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("%s: expected mode 0600, got %o", path, info.Mode().Perm())
		}
	}
}

func TestStore_ConcurrentSaves_NoCorruption(t *testing.T) {
	s := NewStoreAt(t.TempDir())

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			tc := sampleCache("MTD")
			tc.States[0].Position = float64(i)
			if err := s.Save(tc); err != nil {
				t.Errorf("concurrent save %d: %v", i, err)
			}
		}()
	}
	wg.Wait()

	// Should still load cleanly — last write wins, but file must not be corrupt.
	got, err := s.Load("MTD", "deadbeefdeadbeef")
	if err != nil {
		t.Fatalf("post-concurrent Load: %v", err)
	}
	if got.Team.Key != "MTD" {
		t.Errorf("unexpected team: %+v", got.Team)
	}
}

func TestStore_CorruptIndexRecovers(t *testing.T) {
	dir := t.TempDir()
	s := NewStoreAt(dir)

	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Save should succeed (corrupt index is silently reset).
	if err := s.Save(sampleCache("MTD")); err != nil {
		t.Fatalf("Save with corrupt index: %v", err)
	}
	if _, err := s.Load("MTD", ""); err != nil {
		t.Errorf("Load after corrupt-index recovery: %v", err)
	}
}

func TestFreshnessLabel(t *testing.T) {
	cases := []struct {
		age  time.Duration
		want string
	}{
		{30 * time.Second, "fetched just now"},
		{5 * time.Minute, "cached 5m ago"},
		{3 * time.Hour, "cached 3h ago"},
		{50 * time.Hour, "cached 2d ago"},
	}
	for _, c := range cases {
		if got := FreshnessLabel(c.age); got != c.want {
			t.Errorf("FreshnessLabel(%v) = %q, want %q", c.age, got, c.want)
		}
	}
}
