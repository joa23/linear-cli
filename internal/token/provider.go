package token

// TokenProvider provides access to authentication tokens with optional refresh capability.
// This interface abstracts token management for the Linear API client.
type TokenProvider interface {
	// GetToken returns a valid access token, refreshing if needed (proactive refresh)
	GetToken() (string, error)

	// RefreshIfNeeded attempts to refresh the token when a 401 error occurs (reactive refresh)
	// Takes the token that failed as a parameter to enable double-checked locking
	RefreshIfNeeded(failedToken string) (string, error)
}

// StaticProvider provides a static token without refresh capability.
// Used for:
// - Legacy OAuth apps with long-lived tokens (10 year expiration)
// - Environment variable tokens
// - API tokens without OAuth refresh
type StaticProvider struct {
	token string
}

// NewStaticProvider creates a provider for non-refreshable tokens
func NewStaticProvider(token string) *StaticProvider {
	return &StaticProvider{token: token}
}

// GetToken returns the static token (sanitized for safety)
func (p *StaticProvider) GetToken() (string, error) {
	return SanitizeToken(p.token), nil
}

// RefreshIfNeeded always fails for static tokens
func (p *StaticProvider) RefreshIfNeeded(failedToken string) (string, error) {
	return "", &NoRefreshTokenError{}
}

// RefreshingProvider provides tokens with automatic refresh capability.
// Used for new OAuth apps (created after Oct 1, 2025) with 24-hour token expiration.
type RefreshingProvider struct {
	refresher *Refresher
}

// NewRefreshingProvider creates a provider that automatically refreshes tokens
func NewRefreshingProvider(refresher *Refresher) *RefreshingProvider {
	return &RefreshingProvider{refresher: refresher}
}

// GetToken returns a valid token, proactively refreshing if expiring soon
func (p *RefreshingProvider) GetToken() (string, error) {
	return p.refresher.GetValidToken()
}

// RefreshIfNeeded reactively refreshes the token when a 401 occurs
func (p *RefreshingProvider) RefreshIfNeeded(failedToken string) (string, error) {
	return p.refresher.RefreshIfNeeded(failedToken)
}
