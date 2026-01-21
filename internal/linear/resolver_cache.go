package linear

import (
	"sync"
	"time"
)

// cacheEntry represents a cached value with expiration time
type cacheEntry struct {
	value     string
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
}
