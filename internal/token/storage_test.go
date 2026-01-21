package token

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTokenStorage(t *testing.T) {
	// Use temporary directory for testing
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "token")
	
	// Create storage with test path
	storage := NewStorage(tokenPath)
	
	t.Run("save and load token", func(t *testing.T) {
		testToken := "test-linear-token-123"
		
		// Save token
		err := storage.SaveToken(testToken)
		if err != nil {
			t.Fatalf("Failed to save token: %v", err)
		}
		
		// Verify file exists and has correct permissions
		info, err := os.Stat(tokenPath)
		if err != nil {
			t.Fatalf("Token file not created: %v", err)
		}
		
		// Check file permissions (should be 0600)
		if info.Mode().Perm() != 0600 {
			t.Errorf("Expected file permissions 0600, got %v", info.Mode().Perm())
		}
		
		// Load token
		loadedToken, err := storage.LoadToken()
		if err != nil {
			t.Fatalf("Failed to load token: %v", err)
		}
		
		if loadedToken != testToken {
			t.Errorf("Expected token %s, got %s", testToken, loadedToken)
		}
	})
	
	t.Run("load non-existent token", func(t *testing.T) {
		// Use different path that doesn't exist
		nonExistentPath := filepath.Join(tempDir, "nonexistent")
		storage := NewStorage(nonExistentPath)
		
		token, err := storage.LoadToken()
		if err == nil {
			t.Error("Expected error when loading non-existent token")
		}
		
		if token != "" {
			t.Errorf("Expected empty token, got %s", token)
		}
	})
	
	t.Run("token exists check", func(t *testing.T) {
		// Use a fresh path for this test
		freshTokenPath := filepath.Join(tempDir, "fresh_token")
		freshStorage := NewStorage(freshTokenPath)
		
		testToken := "another-test-token"
		
		// Initially should not exist
		if freshStorage.TokenExists() {
			t.Error("Token should not exist initially")
		}
		
		// Save token
		err := freshStorage.SaveToken(testToken)
		if err != nil {
			t.Fatalf("Failed to save token: %v", err)
		}
		
		// Now should exist
		if !freshStorage.TokenExists() {
			t.Error("Token should exist after saving")
		}
	})
}