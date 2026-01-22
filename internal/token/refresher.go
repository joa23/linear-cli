package token

import (
	"fmt"
	"sync"
	"time"
)

// OAuthRefresher defines the interface for refreshing OAuth tokens.
// This allows the refresher to be independent of the oauth package implementation,
// avoiding circular dependencies.
type OAuthRefresher interface {
	RefreshAccessToken(refreshToken string) (*TokenData, error)
}

// Error types for specific refresh failure scenarios
type NoRefreshTokenError struct{}

func (e *NoRefreshTokenError) Error() string {
	return "no refresh token available - token cannot be refreshed"
}

type RefreshFailedError struct {
	Reason error
}

func (e *RefreshFailedError) Error() string {
	return fmt.Sprintf("token refresh failed: %v", e.Reason)
}

type SessionExpiredError struct{}

func (e *SessionExpiredError) Error() string {
	return "session expired - both access and refresh tokens are invalid"
}

// Refresher handles thread-safe automatic token refresh.
// It implements double-checked locking to prevent thundering herd on concurrent 401s.
type Refresher struct {
	storage      *Storage
	oauthHandler OAuthRefresher

	mu         sync.Mutex
	refreshing bool
	lastToken  string // For double-checked locking

	refreshBuffer time.Duration // Proactive refresh buffer (default: 5min)
}

// NewRefresher creates a new token refresher with default settings.
func NewRefresher(storage *Storage, oauthHandler OAuthRefresher) *Refresher {
	return &Refresher{
		storage:       storage,
		oauthHandler:  oauthHandler,
		refreshBuffer: 5 * time.Minute, // Default: refresh 5min before expiry
	}
}

// GetValidToken returns a valid access token, proactively refreshing if needed.
// This should be called before making API requests to ensure the token is fresh.
func (r *Refresher) GetValidToken() (string, error) {
	// Load current token
	tokenData, err := r.storage.LoadTokenData()
	if err != nil {
		return "", fmt.Errorf("failed to load token: %w", err)
	}

	// If no expiration set (old OAuth app), return immediately
	if tokenData.ExpiresAt.IsZero() {
		return tokenData.AccessToken, nil
	}

	// Check if token needs proactive refresh
	if NeedsRefresh(tokenData, r.refreshBuffer) {
		// Token is expiring soon, refresh it
		newToken, err := r.refreshToken(tokenData)
		if err != nil {
			// If refresh fails but token not yet expired, use current token
			if !IsExpired(tokenData) {
				return tokenData.AccessToken, nil
			}
			return "", err
		}
		return newToken, nil
	}

	return tokenData.AccessToken, nil
}

// RefreshIfNeeded handles reactive refresh when a 401 error occurs.
// Uses double-checked locking to prevent multiple concurrent refresh attempts.
//
// Parameters:
//   - originalToken: The token that was used in the failed request
//
// Returns:
//   - New access token if refresh succeeded
//   - Error if refresh failed or no refresh token available
func (r *Refresher) RefreshIfNeeded(originalToken string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-checked locking: Check if token already refreshed by another goroutine
	currentData, err := r.storage.LoadTokenData()
	if err != nil {
		return "", fmt.Errorf("failed to load token: %w", err)
	}

	// If token changed since the failed request, return new token immediately
	if currentData.AccessToken != originalToken {
		return currentData.AccessToken, nil
	}

	// Check if refresh token available
	if currentData.RefreshToken == "" {
		return "", &NoRefreshTokenError{}
	}

	// Perform refresh (max 1 attempt)
	newToken, err := r.refreshTokenLocked(currentData)
	if err != nil {
		return "", err
	}

	return newToken, nil
}

// refreshToken performs the actual token refresh with locking.
func (r *Refresher) refreshToken(oldToken *TokenData) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.refreshTokenLocked(oldToken)
}

// refreshTokenLocked performs token refresh while already holding the lock.
// Caller must hold r.mu.
func (r *Refresher) refreshTokenLocked(oldToken *TokenData) (string, error) {
	// Check if refresh token available
	if oldToken.RefreshToken == "" {
		return "", &NoRefreshTokenError{}
	}

	// Mark as refreshing
	r.refreshing = true
	defer func() {
		r.refreshing = false
	}()

	// Call OAuth handler to refresh
	newTokenData, err := r.oauthHandler.RefreshAccessToken(oldToken.RefreshToken)
	if err != nil {
		// Check if this is a session expiration (refresh token also invalid)
		// This would typically return a 400 or 401 from the OAuth endpoint
		return "", &RefreshFailedError{Reason: err}
	}

	// Preserve old refresh token if not in response (some OAuth servers don't return it)
	if newTokenData.RefreshToken == "" {
		newTokenData.RefreshToken = oldToken.RefreshToken
	}

	// Save new token to disk
	if err := r.storage.SaveTokenData(newTokenData); err != nil {
		return "", fmt.Errorf("failed to save refreshed token: %w", err)
	}

	// Update lastToken for double-checked locking
	r.lastToken = newTokenData.AccessToken

	return newTokenData.AccessToken, nil
}
