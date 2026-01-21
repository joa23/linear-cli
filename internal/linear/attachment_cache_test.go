package linear

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
		
		// Should be expired now
		expired := cache.Get("short-lived")
		if expired != nil {
			t.Error("Expected cache miss after expiration")
		}
		
		// Cache should be cleaned up
		if cache.Size() != 0 {
			t.Errorf("Expected cache to be cleaned up, got size %d", cache.Size())
		}
	})
}

func TestAttachmentClientCaching(t *testing.T) {
	t.Run("caches downloaded content", func(t *testing.T) {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test content"))
		}))
		defer server.Close()
		
		client := NewClient("test-token")
		attachmentClient := client.Attachments
		
		// First request - should hit server
		response1, err := attachmentClient.GetAttachment(server.URL, FormatBase64)
		if err != nil {
			t.Fatalf("First request failed: %v", err)
		}
		
		if requests != 1 {
			t.Errorf("Expected 1 server request, got %d", requests)
		}
		
		// Second request - should use cache
		response2, err := attachmentClient.GetAttachment(server.URL, FormatBase64)
		if err != nil {
			t.Fatalf("Second request failed: %v", err)
		}
		
		if requests != 1 {
			t.Errorf("Expected 1 server request (cached), got %d", requests)
		}
		
		// Responses should be identical
		if response1.Content != response2.Content {
			t.Error("Cached response content differs from original")
		}
		
		if response1.ContentType != response2.ContentType {
			t.Error("Cached response content type differs from original")
		}
		
		if response1.Size != response2.Size {
			t.Error("Cached response size differs from original")
		}
	})
	
	t.Run("caches non-image content over size limit", func(t *testing.T) {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			
			// Create content over 1MB limit
			largeData := make([]byte, 1024*1024+1000) // Slightly over 1MB
			for i := range largeData {
				largeData[i] = byte('A' + (i % 26))
			}
			w.Write(largeData)
		}))
		defer server.Close()
		
		client := NewClient("test-token")
		attachmentClient := client.Attachments
		
		// First request - should hit server but fallback to URL format
		response1, err := attachmentClient.GetAttachment(server.URL, FormatBase64)
		if err != nil {
			t.Fatalf("First request failed: %v", err)
		}
		
		if requests != 1 {
			t.Errorf("Expected 1 server request, got %d", requests)
		}
		
		// Should fallback to URL format due to size
		if response1.Format != FormatURL {
			t.Errorf("Expected URL format for large file, got %s", response1.Format)
		}
		
		if response1.Content != server.URL {
			t.Error("Expected URL content for large file")
		}
		
		// Second request for same large file - should hit server again since large files aren't cached
		_, err = attachmentClient.GetAttachment(server.URL, FormatBase64)
		if err != nil {
			t.Fatalf("Second request failed: %v", err)
		}
		
		// Large files that fallback to URL aren't cached, so should hit server again
		if requests != 2 {
			t.Errorf("Expected 2 server requests (large files not cached), got %d", requests)
		}
	})
	
	t.Run("different formats have separate cache entries", func(t *testing.T) {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test content"))
		}))
		defer server.Close()
		
		client := NewClient("test-token")
		attachmentClient := client.Attachments
		
		// Request base64 format
		_, err := attachmentClient.GetAttachment(server.URL, FormatBase64)
		if err != nil {
			t.Fatalf("Base64 request failed: %v", err)
		}
		
		if requests != 1 {
			t.Errorf("Expected 1 server request, got %d", requests)
		}
		
		// Request URL format - should hit server again (different cache key)
		_, err = attachmentClient.GetAttachment(server.URL, FormatURL)
		if err != nil {
			t.Fatalf("URL request failed: %v", err)
		}
		
		// URL format doesn't cache since it doesn't download content
		// But base64 should still be cached
		_, err = attachmentClient.GetAttachment(server.URL, FormatBase64)
		if err != nil {
			t.Fatalf("Second base64 request failed: %v", err)
		}
		
		if requests != 1 {
			t.Errorf("Expected 1 server request (base64 cached), got %d", requests)
		}
	})
	
	t.Run("cache key generation", func(t *testing.T) {
		client := NewClient("test-token")
		attachmentClient := client.Attachments
		
		// Test consistent key generation
		key1 := attachmentClient.generateCacheKey("https://example.com/file.png", FormatBase64)
		key2 := attachmentClient.generateCacheKey("https://example.com/file.png", FormatBase64)
		
		if key1 != key2 {
			t.Error("Cache keys should be consistent for same URL and format")
		}
		
		// Test different keys for different formats
		key3 := attachmentClient.generateCacheKey("https://example.com/file.png", FormatURL)
		if key1 == key3 {
			t.Error("Cache keys should differ for different formats")
		}
		
		// Test different keys for different URLs
		key4 := attachmentClient.generateCacheKey("https://example.com/other.png", FormatBase64)
		if key1 == key4 {
			t.Error("Cache keys should differ for different URLs")
		}
		
		// Test key format (should be hex hash)
		if len(key1) != 64 { // SHA256 hex string length
			t.Errorf("Expected 64-character hex key, got %d characters", len(key1))
		}
		
		// Should only contain hex characters
		for _, char := range key1 {
			if !strings.ContainsRune("0123456789abcdef", char) {
				t.Errorf("Cache key contains non-hex character: %c", char)
			}
		}
	})
	
	t.Run("cache handles network errors gracefully", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts <= 2 { // Fail first 2 attempts
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Third attempt succeeds (within retry limit)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}))
		defer server.Close()
		
		client := NewClient("test-token")
		attachmentClient := client.Attachments
		
		// Request should succeed due to retry logic (3 retries max)
		response1, err := attachmentClient.GetAttachment(server.URL, FormatBase64)
		if err != nil {
			t.Fatalf("First request failed: %v", err)
		}
		
		// Should succeed after retries
		if response1.Error != "" {
			t.Errorf("Expected no error after successful retry, got: %s", response1.Error)
		}
		
		// Should be cached now  
		if attachmentClient.cache.Size() != 1 {
			t.Errorf("Expected 1 cached entry, got %d", attachmentClient.cache.Size())
		}
		
		// Second request should use cache (no additional server requests)
		response2, err := attachmentClient.GetAttachment(server.URL, FormatBase64)
		if err != nil {
			t.Fatalf("Second request failed: %v", err)
		}
		
		if response2.Error != "" {
			t.Errorf("Expected no error in cached response, got: %s", response2.Error)
		}
		
		// Should still be exactly 3 attempts (from first request only)
		if attempts != 3 {
			t.Errorf("Expected exactly 3 server attempts, got %d", attempts)
		}
	})
}

func TestCacheCleanup(t *testing.T) {
	t.Run("manual cleanup removes expired entries", func(t *testing.T) {
		// Create cache with short TTL
		cache := NewAttachmentCache(1 * time.Minute)
		
		// Add several entries with short expiration
		for i := 0; i < 5; i++ {
			entry := &CacheEntry{
				Content:   []byte(fmt.Sprintf("content %d", i)),
				ExpiresAt: time.Now().Add(25 * time.Millisecond), // Short expiration
			}
			cache.Set(fmt.Sprintf("key-%d", i), entry)
		}
		
		if cache.Size() != 5 {
			t.Errorf("Expected 5 entries, got %d", cache.Size())
		}
		
		// Wait for expiration
		time.Sleep(50 * time.Millisecond)
		
		// Manually trigger cleanup
		cache.removeExpiredEntries()
		
		// Cache should be cleaned up
		if cache.Size() != 0 {
			t.Errorf("Expected cache to be cleaned up, got size %d", cache.Size())
		}
	})
	
	t.Run("cleanup preserves non-expired entries", func(t *testing.T) {
		cache := NewAttachmentCache(1 * time.Minute)
		
		// Add mix of expired and non-expired entries
		expiredEntry := &CacheEntry{
			Content:   []byte("expired"),
			ExpiresAt: time.Now().Add(-1 * time.Minute), // Already expired
		}
		cache.Set("expired", expiredEntry)
		
		validEntry := &CacheEntry{
			Content:   []byte("valid"),
			ExpiresAt: time.Now().Add(1 * time.Minute), // Valid for 1 minute
		}
		cache.Set("valid", validEntry)
		
		if cache.Size() != 2 {
			t.Errorf("Expected 2 entries, got %d", cache.Size())
		}
		
		// Manual cleanup
		cache.removeExpiredEntries()
		
		// Should keep only the valid entry
		if cache.Size() != 1 {
			t.Errorf("Expected 1 entry after cleanup, got %d", cache.Size())
		}
		
		// Valid entry should still be accessible
		retrieved := cache.Get("valid")
		if retrieved == nil {
			t.Error("Expected valid entry to remain after cleanup")
		}
		
		// Expired entry should be gone
		expired := cache.Get("expired")
		if expired != nil {
			t.Error("Expected expired entry to be removed")
		}
	})
}