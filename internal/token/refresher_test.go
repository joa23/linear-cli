package token

import (
	"path/filepath"
	"testing"
)

// mockOAuthRefresher is a test double for OAuthRefresher.
type mockOAuthRefresher struct {
	returnData *TokenData
	returnErr  error
}

func (m *mockOAuthRefresher) RefreshAccessToken(_ string) (*TokenData, error) {
	return m.returnData, m.returnErr
}

func TestRefresher_PreservesAuthModeOnRefresh(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewStorage(filepath.Join(tempDir, "token"))

	// Seed storage with a token that has AuthMode set
	err := storage.SaveTokenData(&TokenData{
		AccessToken:  "old",
		RefreshToken: "rt",
		TokenType:    "Bearer",
		AuthMode:     "agent",
	})
	if err != nil {
		t.Fatalf("failed to seed token: %v", err)
	}

	// Mock returns new tokens but no AuthMode (OAuth server never returns it)
	mock := &mockOAuthRefresher{
		returnData: &TokenData{
			AccessToken:  "new",
			RefreshToken: "rt2",
			TokenType:    "Bearer",
		},
	}

	refresher := NewRefresher(storage, mock)
	newToken, err := refresher.RefreshIfNeeded("old")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newToken != "new" {
		t.Errorf("expected returned token %q, got %q", "new", newToken)
	}

	// Verify AuthMode was preserved on disk
	saved, err := storage.LoadTokenData()
	if err != nil {
		t.Fatalf("failed to load token: %v", err)
	}
	if saved.AuthMode != "agent" {
		t.Errorf("expected AuthMode %q, got %q", "agent", saved.AuthMode)
	}
}

func TestRefresher_DoesNotOverwriteAuthModeWhenPresent(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewStorage(filepath.Join(tempDir, "token"))

	err := storage.SaveTokenData(&TokenData{
		AccessToken:  "old",
		RefreshToken: "rt",
		TokenType:    "Bearer",
		AuthMode:     "agent",
	})
	if err != nil {
		t.Fatalf("failed to seed token: %v", err)
	}

	// Mock returns a response that includes AuthMode — new value should win
	mock := &mockOAuthRefresher{
		returnData: &TokenData{
			AccessToken:  "new",
			RefreshToken: "rt2",
			TokenType:    "Bearer",
			AuthMode:     "user",
		},
	}

	refresher := NewRefresher(storage, mock)
	_, err = refresher.RefreshIfNeeded("old")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	saved, err := storage.LoadTokenData()
	if err != nil {
		t.Fatalf("failed to load token: %v", err)
	}
	if saved.AuthMode != "user" {
		t.Errorf("expected AuthMode %q (new value wins), got %q", "user", saved.AuthMode)
	}
}

func TestRefresher_PreservesRefreshTokenWhenAbsent(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewStorage(filepath.Join(tempDir, "token"))

	err := storage.SaveTokenData(&TokenData{
		AccessToken:  "old",
		RefreshToken: "rt-original",
		TokenType:    "Bearer",
		AuthMode:     "agent",
	})
	if err != nil {
		t.Fatalf("failed to seed token: %v", err)
	}

	// Mock returns no RefreshToken
	mock := &mockOAuthRefresher{
		returnData: &TokenData{
			AccessToken: "new",
			TokenType:   "Bearer",
		},
	}

	refresher := NewRefresher(storage, mock)
	_, err = refresher.RefreshIfNeeded("old")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	saved, err := storage.LoadTokenData()
	if err != nil {
		t.Fatalf("failed to load token: %v", err)
	}
	if saved.RefreshToken != "rt-original" {
		t.Errorf("expected RefreshToken %q to be preserved, got %q", "rt-original", saved.RefreshToken)
	}
}
