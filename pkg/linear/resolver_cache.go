package linear

import (
	"strings"
	"sync"
	"time"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// cacheEntry represents a cached value with expiration time
type cacheEntry struct {
	value     string
	expiresAt time.Time
}

type labelCacheEntry struct {
	label     core.Label
	expiresAt time.Time
}

// isExpired checks if the cache entry has expired
func (e *cacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// resolverCache caches resolution results to minimize API calls
// All methods are thread-safe for concurrent access
//
// Why: Resolution lookups (user by email, team by name, etc.) can be expensive.
// Caching reduces API calls and improves performance for repeated resolutions.
type resolverCache struct {
	// User resolution caches
	userByEmail map[string]*cacheEntry // email → userID
	userByName  map[string]*cacheEntry // name → userID

	// Team resolution caches
	teamByName map[string]*cacheEntry // team name → teamID
	teamByKey  map[string]*cacheEntry // team key → teamID

	// Issue resolution cache
	issueByIdentifier map[string]*cacheEntry // CEN-123 → issueID

	// Label resolution caches are keyed by normalized team and label values.
	labelByName map[string]*cacheEntry      // teamID:normalized-name → labelID
	labelData   map[string]*labelCacheEntry // teamID:normalized-id → label metadata

	// Project resolution cache
	projectByName map[string]*cacheEntry // project name → projectID

	mu  sync.RWMutex
	ttl time.Duration
}

// newResolverCache creates a new resolver cache with the specified TTL
func newResolverCache(ttl time.Duration) *resolverCache {
	cache := &resolverCache{
		userByEmail:       make(map[string]*cacheEntry),
		userByName:        make(map[string]*cacheEntry),
		teamByName:        make(map[string]*cacheEntry),
		teamByKey:         make(map[string]*cacheEntry),
		issueByIdentifier: make(map[string]*cacheEntry),
		labelByName:       make(map[string]*cacheEntry),
		labelData:         make(map[string]*labelCacheEntry),
		projectByName:     make(map[string]*cacheEntry),
		ttl:               ttl,
	}

	// Start background cleanup goroutine
	go cache.runCleanup()

	return cache
}

// User cache methods

func (rc *resolverCache) getUserByEmail(email string) (string, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.userByEmail[email]
	if !exists || entry.isExpired() {
		return "", false
	}
	return entry.value, true
}

func (rc *resolverCache) setUserByEmail(email, userID string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.userByEmail[email] = &cacheEntry{
		value:     userID,
		expiresAt: time.Now().Add(rc.ttl),
	}
}

func (rc *resolverCache) getUserByName(name string) (string, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.userByName[name]
	if !exists || entry.isExpired() {
		return "", false
	}
	return entry.value, true
}

func (rc *resolverCache) setUserByName(name, userID string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.userByName[name] = &cacheEntry{
		value:     userID,
		expiresAt: time.Now().Add(rc.ttl),
	}
}

// Team cache methods

func (rc *resolverCache) getTeamByName(name string) (string, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.teamByName[name]
	if !exists || entry.isExpired() {
		return "", false
	}
	return entry.value, true
}

func (rc *resolverCache) setTeamByName(name, teamID string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.teamByName[name] = &cacheEntry{
		value:     teamID,
		expiresAt: time.Now().Add(rc.ttl),
	}
}

func (rc *resolverCache) getTeamByKey(key string) (string, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.teamByKey[key]
	if !exists || entry.isExpired() {
		return "", false
	}
	return entry.value, true
}

func (rc *resolverCache) setTeamByKey(key, teamID string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.teamByKey[key] = &cacheEntry{
		value:     teamID,
		expiresAt: time.Now().Add(rc.ttl),
	}
}

// Issue cache methods

func (rc *resolverCache) getIssueByIdentifier(identifier string) (string, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.issueByIdentifier[identifier]
	if !exists || entry.isExpired() {
		return "", false
	}
	return entry.value, true
}

func (rc *resolverCache) setIssueByIdentifier(identifier, issueID string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.issueByIdentifier[identifier] = &cacheEntry{
		value:     issueID,
		expiresAt: time.Now().Add(rc.ttl),
	}
}

// Label cache methods

func labelKey(teamID, value string) string {
	return strings.ToLower(strings.TrimSpace(teamID)) + ":" + strings.ToLower(strings.TrimSpace(value))
}

func cloneLabel(label core.Label) core.Label {
	copy := label
	if label.Parent != nil {
		parent := *label.Parent
		copy.Parent = &parent
	}
	return copy
}

func (rc *resolverCache) getLabelByName(teamID, labelName string) (string, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.labelByName[labelKey(teamID, labelName)]
	if !exists || entry.isExpired() {
		return "", false
	}
	return entry.value, true
}

func (rc *resolverCache) getLabelByID(teamID, labelID string) (core.Label, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.labelData[labelKey(teamID, labelID)]
	if !exists || entry.expiresAt.Before(time.Now()) {
		return core.Label{}, false
	}
	return cloneLabel(entry.label), true
}

func (rc *resolverCache) setLabels(teamID string, labels []core.Label) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	teamPrefix := strings.ToLower(strings.TrimSpace(teamID)) + ":"
	for key := range rc.labelByName {
		if strings.HasPrefix(key, teamPrefix) {
			delete(rc.labelByName, key)
		}
	}
	for key := range rc.labelData {
		if strings.HasPrefix(key, teamPrefix) {
			delete(rc.labelData, key)
		}
	}

	expiresAt := time.Now().Add(rc.ttl)
	nameCounts := make(map[string]int, len(labels))
	for _, label := range labels {
		nameCounts[labelKey(teamID, label.Name)]++
	}
	for _, label := range labels {
		copy := cloneLabel(label)
		idKey := labelKey(teamID, label.ID)
		rc.labelData[idKey] = &labelCacheEntry{label: copy, expiresAt: expiresAt}
		if nameCounts[labelKey(teamID, label.Name)] == 1 {
			rc.labelByName[labelKey(teamID, label.Name)] = &cacheEntry{value: label.ID, expiresAt: expiresAt}
		}
	}
}

// Project cache methods

func (rc *resolverCache) getProjectByName(name string) (string, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, exists := rc.projectByName[name]
	if !exists || entry.isExpired() {
		return "", false
	}
	return entry.value, true
}

func (rc *resolverCache) setProjectByName(name, projectID string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.projectByName[name] = &cacheEntry{
		value:     projectID,
		expiresAt: time.Now().Add(rc.ttl),
	}
}

// Utility methods

// cleanup removes expired entries from the cache
// This is called periodically by the background goroutine
func (rc *resolverCache) cleanup() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()

	// Clean up user caches
	for email, entry := range rc.userByEmail {
		if entry.expiresAt.Before(now) {
			delete(rc.userByEmail, email)
		}
	}

	for name, entry := range rc.userByName {
		if entry.expiresAt.Before(now) {
			delete(rc.userByName, name)
		}
	}

	// Clean up team caches
	for name, entry := range rc.teamByName {
		if entry.expiresAt.Before(now) {
			delete(rc.teamByName, name)
		}
	}

	for key, entry := range rc.teamByKey {
		if entry.expiresAt.Before(now) {
			delete(rc.teamByKey, key)
		}
	}

	// Clean up issue cache
	for identifier, entry := range rc.issueByIdentifier {
		if entry.expiresAt.Before(now) {
			delete(rc.issueByIdentifier, identifier)
		}
	}

	for key, entry := range rc.labelByName {
		if entry.expiresAt.Before(now) {
			delete(rc.labelByName, key)
		}
	}
	for key, entry := range rc.labelData {
		if entry.expiresAt.Before(now) {
			delete(rc.labelData, key)
		}
	}

	// Clean up project cache
	for name, entry := range rc.projectByName {
		if entry.expiresAt.Before(now) {
			delete(rc.projectByName, name)
		}
	}
}

// runCleanup runs periodic cleanup in a background goroutine
// Runs every TTL/2 to remove expired entries
func (rc *resolverCache) runCleanup() {
	// Run cleanup at half the TTL interval
	ticker := time.NewTicker(rc.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		rc.cleanup()
	}
}

// clear removes all entries from the cache
// Useful for testing or when wanting to force fresh lookups
func (rc *resolverCache) clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.userByEmail = make(map[string]*cacheEntry)
	rc.userByName = make(map[string]*cacheEntry)
	rc.teamByName = make(map[string]*cacheEntry)
	rc.teamByKey = make(map[string]*cacheEntry)
	rc.issueByIdentifier = make(map[string]*cacheEntry)
	rc.labelByName = make(map[string]*cacheEntry)
	rc.labelData = make(map[string]*labelCacheEntry)
	rc.projectByName = make(map[string]*cacheEntry)
}
