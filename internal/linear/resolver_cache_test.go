package linear

import (
	"testing"
	"time"
)

func TestResolverCache_BasicOperations(t *testing.T) {
	cache := newResolverCache(5 * time.Minute)

	t.Run("set and get user by email", func(t *testing.T) {
		cache.setUserByEmail("john@company.com", "user-123")

		got, found := cache.getUserByEmail("john@company.com")
		if !found {
			t.Error("Expected to find cached user, but didn't")
		}
		if got != "user-123" {
			t.Errorf("Expected user-123, got %s", got)
		}
	})

	t.Run("get non-existent user by email", func(t *testing.T) {
		_, found := cache.getUserByEmail("nonexistent@company.com")
		if found {
			t.Error("Expected not to find user, but did")
		}
	})

	t.Run("set and get user by name", func(t *testing.T) {
		cache.setUserByName("John Doe", "user-123")

		got, found := cache.getUserByName("John Doe")
		if !found {
			t.Error("Expected to find cached user, but didn't")
		}
		if got != "user-123" {
			t.Errorf("Expected user-123, got %s", got)
		}
	})

	t.Run("set and get team by name", func(t *testing.T) {
		cache.setTeamByName("Engineering", "team-123")

		got, found := cache.getTeamByName("Engineering")
		if !found {
			t.Error("Expected to find cached team, but didn't")
		}
		if got != "team-123" {
			t.Errorf("Expected team-123, got %s", got)
		}
	})

	t.Run("set and get team by key", func(t *testing.T) {
		cache.setTeamByKey("ENG", "team-123")

		got, found := cache.getTeamByKey("ENG")
		if !found {
			t.Error("Expected to find cached team, but didn't")
		}
		if got != "team-123" {
			t.Errorf("Expected team-123, got %s", got)
		}
	})

	t.Run("set and get issue by identifier", func(t *testing.T) {
		cache.setIssueByIdentifier("CEN-123", "issue-uuid-123")

		got, found := cache.getIssueByIdentifier("CEN-123")
		if !found {
			t.Error("Expected to find cached issue, but didn't")
		}
		if got != "issue-uuid-123" {
			t.Errorf("Expected issue-uuid-123, got %s", got)
		}
	})
}

func TestResolverCache_Expiration(t *testing.T) {
	// Use a very short TTL for testing
	cache := newResolverCache(100 * time.Millisecond)

	t.Run("entry expires after TTL", func(t *testing.T) {
		cache.setUserByEmail("john@company.com", "user-123")

		// Should be found immediately
		_, found := cache.getUserByEmail("john@company.com")
		if !found {
			t.Error("Expected to find cached user immediately")
		}

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Should not be found after expiration
		_, found = cache.getUserByEmail("john@company.com")
		if found {
			t.Error("Expected cache entry to be expired, but it wasn't")
		}
	})
}

func TestResolverCache_ConcurrentAccess(t *testing.T) {
	cache := newResolverCache(5 * time.Minute)

	// Test concurrent writes and reads
	t.Run("concurrent operations are thread-safe", func(t *testing.T) {
		done := make(chan bool)

		// Writer goroutines
		for i := 0; i < 10; i++ {
			go func(n int) {
				for j := 0; j < 100; j++ {
					cache.setUserByEmail("user"+string(rune(n))+"@test.com", "user-id")
					cache.getUserByEmail("user"+string(rune(n))+"@test.com")
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to finish
		for i := 0; i < 10; i++ {
			<-done
		}

		// If we get here without deadlock or panic, the test passes
	})
}

func TestResolverCache_Cleanup(t *testing.T) {
	cache := newResolverCache(50 * time.Millisecond)

	t.Run("cleanup removes expired entries", func(t *testing.T) {
		// Add several entries
		cache.setUserByEmail("user1@test.com", "id1")
		cache.setUserByEmail("user2@test.com", "id2")
		cache.setTeamByName("Team1", "team1")

		// Wait for entries to expire
		time.Sleep(100 * time.Millisecond)

		// Trigger cleanup
		cache.cleanup()

		// All entries should be removed
		_, found1 := cache.getUserByEmail("user1@test.com")
		_, found2 := cache.getUserByEmail("user2@test.com")
		_, found3 := cache.getTeamByName("Team1")

		if found1 || found2 || found3 {
			t.Error("Expected all entries to be cleaned up, but some remain")
		}
	})
}

func TestResolverCache_Clear(t *testing.T) {
	cache := newResolverCache(5 * time.Minute)

	t.Run("clear removes all entries", func(t *testing.T) {
		// Add several entries
		cache.setUserByEmail("user@test.com", "id")
		cache.setTeamByName("Team", "team")
		cache.setIssueByIdentifier("CEN-123", "issue")

		// Clear all
		cache.clear()

		// All entries should be removed
		_, found1 := cache.getUserByEmail("user@test.com")
		_, found2 := cache.getTeamByName("Team")
		_, found3 := cache.getIssueByIdentifier("CEN-123")

		if found1 || found2 || found3 {
			t.Error("Expected all entries to be cleared, but some remain")
		}
	})
}
