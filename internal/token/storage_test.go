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

	t.Run("save token with newlines sanitizes", func(t *testing.T) {
		sanitizedTokenPath := filepath.Join(tempDir, "sanitized_token")
		sanitizedStorage := NewStorage(sanitizedTokenPath)

		// Token with newlines
		tokenWithNewlines := "lin_api_token123\n\r"

		// Save token (should sanitize)
		err := sanitizedStorage.SaveToken(tokenWithNewlines)
		if err != nil {
			t.Fatalf("Failed to save token with newlines: %v", err)
		}

		// Load and verify sanitization
		loadedToken, err := sanitizedStorage.LoadToken()
		if err != nil {
			t.Fatalf("Failed to load token: %v", err)
		}

		expected := "lin_api_token123"
		if loadedToken != expected {
			t.Errorf("Expected sanitized token %q, got %q", expected, loadedToken)
		}
	})

	t.Run("load token data with newlines sanitizes", func(t *testing.T) {
		malformedTokenPath := filepath.Join(tempDir, "malformed_token")

		// Write token with newlines directly to file (bypassing SaveToken validation)
		malformedToken := "lin_api_token123\n\r"
		err := os.WriteFile(malformedTokenPath, []byte(malformedToken), 0600)
		if err != nil {
			t.Fatalf("Failed to write malformed token: %v", err)
		}

		// Load using storage (should sanitize)
		malformedStorage := NewStorage(malformedTokenPath)
		tokenData, err := malformedStorage.LoadTokenData()
		if err != nil {
			t.Fatalf("Failed to load token data: %v", err)
		}

		expected := "lin_api_token123"
		if tokenData.AccessToken != expected {
			t.Errorf("Expected sanitized token %q, got %q", expected, tokenData.AccessToken)
		}
	})

	t.Run("save token with invalid characters returns error", func(t *testing.T) {
		invalidTokenPath := filepath.Join(tempDir, "invalid_token")
		invalidStorage := NewStorage(invalidTokenPath)

		// Empty token should fail validation
		err := invalidStorage.SaveToken("")
		if err == nil {
			t.Error("Expected error when saving empty token")
		}
	})

	t.Run("save token data sanitizes access and refresh tokens", func(t *testing.T) {
		tokenDataPath := filepath.Join(tempDir, "token_data")
		tokenDataStorage := NewStorage(tokenDataPath)

		// Create token data with newlines
		tokenData := &TokenData{
			AccessToken:  "access_token\n123",
			RefreshToken: "refresh_token\r456",
			TokenType:    "Bearer",
		}

		// Save (should sanitize)
		err := tokenDataStorage.SaveTokenData(tokenData)
		if err != nil {
			t.Fatalf("Failed to save token data: %v", err)
		}

		// Load and verify sanitization
		loadedData, err := tokenDataStorage.LoadTokenData()
		if err != nil {
			t.Fatalf("Failed to load token data: %v", err)
		}

		if loadedData.AccessToken != "access_token123" {
			t.Errorf("Expected sanitized access token %q, got %q", "access_token123", loadedData.AccessToken)
		}

		if loadedData.RefreshToken != "refresh_token456" {
			t.Errorf("Expected sanitized refresh token %q, got %q", "refresh_token456", loadedData.RefreshToken)
		}
	})
}