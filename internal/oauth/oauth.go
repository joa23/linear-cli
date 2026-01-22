package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	// Linear OAuth endpoints
	linearAuthURL  = "https://linear.app/oauth/authorize"
	linearTokenURL = "https://api.linear.app/oauth/token"
)

// Handler handles OAuth flow for Linear.
// It manages the complete OAuth2 authorization code flow including:
// - Authorization URL generation with CSRF protection
// - Callback handling with state validation
// - Secure token exchange
type Handler struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// TokenResponse represents the response from Linear's token endpoint
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// NewHandler creates a new OAuth handler
func NewHandler(clientID, clientSecret string) *Handler {
	return &Handler{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   http.DefaultClient, // Use default client for backward compatibility
	}
}

// NewHandlerWithClient creates a new OAuth handler with a custom HTTP client
func NewHandlerWithClient(clientID, clientSecret string, httpClient *http.Client) *Handler {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Handler{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   httpClient,
	}
}

// GetAuthorizationURL generates the Linear OAuth authorization URL for user authentication.
//
// Why include state: The state parameter prevents CSRF attacks by ensuring
// the callback we receive corresponds to an authorization request we initiated.
// Linear will include this state in the callback, and we verify it matches.
//
// Scope explanation:
// - "read": Access to read Linear data (issues, projects, comments)
// - "write": Ability to create and modify Linear data
func (h *Handler) GetAuthorizationURL(redirectURI, state string) string {
	params := url.Values{
		"client_id":     []string{h.clientID},
		"redirect_uri":  []string{redirectURI},
		"response_type": []string{"code"},
		"state":         []string{state},
		"scope":         []string{"read write"},
	}
	
	return linearAuthURL + "?" + params.Encode()
}

// GetAppAuthorizationURL generates the Linear OAuth authorization URL for app authentication
func (h *Handler) GetAppAuthorizationURL(redirectURI, state string) string {
	params := url.Values{
		"client_id":     []string{h.clientID},
		"redirect_uri":  []string{redirectURI},
		"response_type": []string{"code"},
		"state":         []string{state},
		"scope":         []string{"app:assignable app:mentionable read write"},
		"actor":         []string{"app"},
		"prompt":        []string{"consent"}, // Force consent screen for multi-workspace support
	}

	return linearAuthURL + "?" + params.Encode()
}

// HandleCallback handles the OAuth callback and exchanges code for token.
// It starts a temporary HTTP server to receive the callback, validates the
// state parameter for CSRF protection, and exchanges the authorization code
// for an access token.
//
// Why temporary server: CLI applications can't receive HTTP callbacks directly.
// We start a local server just long enough to receive the callback, then shut
// it down immediately after getting the authorization code.
func (h *Handler) HandleCallback(port, expectedState string) (string, error) {
	// Channel to receive the authorization code
	// Why buffered channels: Prevents goroutine blocking if the handler
	// completes before the main function reads from the channel.
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)
	stateChan := make(chan string, 1)
	
	// Create HTTP server to handle callback
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth-callback", func(w http.ResponseWriter, r *http.Request) {
		// Extract code and state from callback
		query := r.URL.Query()
		code := query.Get("code")
		state := query.Get("state")
		errorParam := query.Get("error")
		
		if errorParam != "" {
			errChan <- fmt.Errorf("OAuth error: %s", errorParam)
			http.Error(w, "Authorization failed", http.StatusBadRequest)
			return
		}
		
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			http.Error(w, "No authorization code", http.StatusBadRequest)
			return
		}
		
		// Send code and state to channels
		codeChan <- code
		stateChan <- state
		
		// Send success response
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<html>
				<body>
					<h1>Authorization Successful!</h1>
					<p>You can close this window and return to the terminal.</p>
				</body>
			</html>
		`))
	})
	
	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start callback server: %w", err)
		}
	}()
	
	// Wait for callback or timeout
	var code, state string
	timeout := time.After(5 * time.Minute) // 5 minute timeout
	
	select {
	case code = <-codeChan:
		state = <-stateChan
		
		// Verify state parameter
		if state != expectedState {
			return "", fmt.Errorf("state parameter mismatch")
		}
		
		// Exchange code for token
		token, err := h.exchangeCodeForToken(code, fmt.Sprintf("http://localhost:%s/oauth-callback", port))
		if err != nil {
			return "", fmt.Errorf("failed to exchange code for token: %w", err)
		}
		
		// Shutdown server
		server.Close()
		
		return token, nil
		
	case err := <-errChan:
		server.Close()
		return "", err
		
	case <-timeout:
		server.Close()
		return "", fmt.Errorf("OAuth flow timed out")
	}
}

// exchangeCodeForToken exchanges authorization code for access token
func (h *Handler) exchangeCodeForToken(code, redirectURI string) (string, error) {
	// Prepare form data for token exchange
	data := url.Values{
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{h.clientID},
		"client_secret": []string{h.clientSecret},
		"code":          []string{code},
		"redirect_uri":  []string{redirectURI},
	}
	
	// Make request to Linear's token endpoint
	resp, err := h.httpClient.PostForm(linearTokenURL, data)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}
	
	// Parse token response
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}
	
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}
	
	return tokenResp.AccessToken, nil
}

// GenerateState generates a random state parameter for OAuth
func GenerateState() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}