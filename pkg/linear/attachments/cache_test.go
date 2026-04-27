package attachments

import (
	"testing"
	"time"
)

func TestAttachmentCache(t *testing.T) {
	t.Run("basic cache operations", func(t *testing.T) {
		cache := NewAttachmentCache(1 * time.Minute)
		
		// Test empty cache
		if cache.Size() != 0 {
			t.Errorf("Expected empty cache, got size %d", cache.Size())
		}
		
		// Test cache miss
		entry := cache.Get("nonexistent")
		if entry != nil {
			t.Error("Expected cache miss, got entry")
		}
		
		// Test cache set and get
		testEntry := &CacheEntry{
			Content:     []byte("test content"),
			ContentType: "text/plain",
			Size:        12,
			ExpiresAt:   time.Now().Add(1 * time.Minute),
		}
		
		cache.Set("test-key", testEntry)
		
		if cache.Size() != 1 {
			t.Errorf("Expected cache size 1, got %d", cache.Size())
		}
		
		retrieved := cache.Get("test-key")
		if retrieved == nil {
			t.Fatal("Expected cache hit, got miss")
		}
		
		if string(retrieved.Content) != "test content" {
			t.Errorf("Expected 'test content', got '%s'", string(retrieved.Content))
		}
		
		// Test cache clear
		cache.Clear()
		if cache.Size() != 0 {
			t.Errorf("Expected empty cache after clear, got size %d", cache.Size())
		}
	})
	
	t.Run("expiration handling", func(t *testing.T) {
		cache := NewAttachmentCache(100 * time.Millisecond)
		
		// Add entry with short TTL
		testEntry := &CacheEntry{
			Content:     []byte("test content"),
			ContentType: "text/plain",
			Size:        12,
			ExpiresAt:   time.Now().Add(50 * time.Millisecond), // Expires in 50ms
		}
		
		cache.Set("short-lived", testEntry)
		
		// Should be available immediately
		retrieved := cache.Get("short-lived")
		if retrieved == nil {
			t.Fatal("Expected cache hit immediately after set")
		}
		
		// Wait for expiration
		time.Sleep(100 * time.Millisecond)
		
		// Should be expired now — Get returns nil but doesn't eagerly delete
		expired := cache.Get("short-lived")
		if expired != nil {
			t.Error("Expected cache miss after expiration")
		}

		// Expired entry remains in map until background cleanup runs;
		// verify removeExpiredEntries cleans it up
		cache.removeExpiredEntries()
		if cache.Size() != 0 {
			t.Errorf("Expected cache to be cleaned up after removeExpiredEntries, got size %d", cache.Size())
		}
	})
}
