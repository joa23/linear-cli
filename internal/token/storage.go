package token

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TokenData represents structured OAuth token information with expiration tracking.
// Supports both refresh-capable tokens (new OAuth apps created after Oct 1, 2025)
// and legacy tokens without expiration.
type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Scope        string    `json:"scope"`
	AuthMode     string    `json:"auth_mode,omitempty"` // "user" or "agent" - determines how "me" resolves
}

// Storage handles token storage and retrieval with secure file permissions.
// Tokens are sensitive credentials that must be protected from unauthorized access.
type Storage struct {
	tokenPath string
}

// NewStorage creates a new token storage instance
func NewStorage(tokenPath string) *Storage {
	return &Storage{
		tokenPath: tokenPath,
	}
}

// SaveToken saves a token to the file system with secure permissions.
//
// Security measures:
// - Directory created with 0700 (owner only access)
// - Token file created with 0600 (owner read/write only)
// - Atomic write operation to prevent partial token storage
//
// Why these permissions: OAuth tokens are bearer tokens - anyone with the
// token can act as the authenticated user. Restricting file permissions
// prevents other users on the system from reading the token.
func (s *Storage) SaveToken(token string) error {
	// Sanitize and validate token before saving
	sanitized := SanitizeToken(token)
	if err := ValidateToken(sanitized); err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Ensure directory exists with secure permissions
	// 0700 = rwx------ (only owner can read, write, or access directory)
	dir := filepath.Dir(s.tokenPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// Write token with secure permissions
	// 0600 = rw------- (only owner can read or write the file)
	// WriteFile is atomic - it writes to a temp file then renames
	if err := os.WriteFile(s.tokenPath, []byte(sanitized), 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// LoadToken loads a token from the file system
func (s *Storage) LoadToken() (string, error) {
	data, err := os.ReadFile(s.tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read token file: %w", err)
	}

	return string(data), nil
}

// TokenExists checks if a token file exists.
// Returns false for file not found and logs errors for other issues like permission denied.
//
// Why log but not fail: This maintains backward compatibility while still
// alerting developers to potential issues. Permission errors might indicate
// security problems that should be investigated.
//
// Deprecated: Use TokenExistsWithError for better error handling.
func (s *Storage) TokenExists() bool {
	exists, err := s.TokenExistsWithError()
	if err != nil {
		// Log the error but maintain backward compatibility by returning false
		fmt.Fprintf(os.Stderr, "Warning: Error checking token file existence: %v\n", err)
	}
	return exists
}

// TokenExistsWithError checks if a token file exists and returns detailed error information.
// Returns (false, nil) if file doesn't exist - this is not an error condition.
// Returns (false, error) if there's an actual error like permission denied.
//
// Why distinguish between "not found" and other errors:
// - File not found is expected when user hasn't authenticated yet
// - Permission errors indicate a security or configuration problem
// - This allows callers to handle these cases differently
func (s *Storage) TokenExistsWithError() (bool, error) {
	_, err := os.Stat(s.tokenPath)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil // File not existing is expected and not an error
	}

	// Other errors (permission denied, etc.) indicate real problems
	return false, fmt.Errorf("failed to check token file: %w", err)
}

// DeleteToken removes the token file
func (s *Storage) DeleteToken() error {
	if !s.TokenExists() {
		return nil // Token doesn't exist, nothing to delete
	}

	if err := os.Remove(s.tokenPath); err != nil {
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	return nil
}

// GetDefaultTokenPath returns the default path for storing tokens.
//
// Token location strategy:
// - Primary: ~/.config/linear/token (XDG standard)
// - Fallback: .config/linear/token (current directory)
//
// Why home directory: Tokens are user-specific credentials. Storing them
// in the home directory ensures they're not accidentally committed to
// version control and are isolated per user on multi-user systems.
func GetDefaultTokenPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home dir unavailable
		// This might happen in restricted environments or containers
		return ".config/linear/token"
	}

	return filepath.Join(homeDir, ".config", "linear", "token")
}

// LoadTokenWithFallback loads token from default location with env var fallback
func LoadTokenWithFallback() string {
	// Try to load from default token storage
	storage := NewStorage(GetDefaultTokenPath())
	if storage.TokenExists() {
		token, err := storage.LoadToken()
		if err == nil && token != "" {
			return token
		}
	}

	// Fall back to environment variables
	return LoadTokenFromEnv()
}

// LoadTokenFromEnv loads a Linear token from environment variables.
// Uses LINEAR_API_KEY as the canonical env var for direct API-key auth.
func LoadTokenFromEnv() string {
	return SanitizeToken(os.Getenv("LINEAR_API_KEY"))
}

// SaveTokenData saves structured token data as JSON with secure permissions.
// This replaces the legacy plain-string token format and enables automatic refresh.
func (s *Storage) SaveTokenData(data *TokenData) error {
	// Sanitize tokens before saving
	data.AccessToken = SanitizeToken(data.AccessToken)
	if data.RefreshToken != "" {
		data.RefreshToken = SanitizeToken(data.RefreshToken)
	}

	// Ensure directory exists with secure permissions
	dir := filepath.Dir(s.tokenPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// Marshal to JSON with indentation for human readability
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	// Write with secure permissions (0600 = rw-------)
	if err := os.WriteFile(s.tokenPath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// LoadTokenData loads structured token data with backward compatibility.
//
// Behavior:
// - If file contains JSON: parse and return TokenData
// - If file contains plain string: treat as legacy access token (no refresh)
// - Returns error only on file read failure or invalid JSON
//
// This enables seamless migration from legacy token format to new format.
func (s *Storage) LoadTokenData() (*TokenData, error) {
	data, err := os.ReadFile(s.tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	content := SanitizeToken(string(data))

	// Try parsing as JSON first (new format)
	var tokenData TokenData
	if err := json.Unmarshal([]byte(content), &tokenData); err == nil {
		// Successfully parsed as JSON
		return &tokenData, nil
	}

	// Fall back to legacy plain string format
	// Treat as access token with no refresh capability
	return &TokenData{
		AccessToken: content,
		TokenType:   "Bearer",
		// No RefreshToken, no ExpiresAt - indicates legacy token
	}, nil
}

// IsExpired checks if the token has expired.
// Returns false for tokens without expiration (legacy tokens or long-lived tokens).
func IsExpired(data *TokenData) bool {
	if data.ExpiresAt.IsZero() {
		return false // No expiration set
	}
	return time.Now().After(data.ExpiresAt)
}

// NeedsRefresh checks if the token should be proactively refreshed.
// Returns true if the token will expire within the buffer duration.
// Returns false for tokens without expiration.
func NeedsRefresh(data *TokenData, buffer time.Duration) bool {
	if data.ExpiresAt.IsZero() {
		return false // No expiration set
	}
	return time.Now().Add(buffer).After(data.ExpiresAt)
}
