// Package oauth implements OAuth2 authentication flow for Linear API access.
// It provides a complete OAuth2 implementation including authorization URL
// generation, callback handling, and secure token exchange.
//
// The package supports both individual user authentication and application-level
// authentication flows, handling the complexities of OAuth2 state management
// and PKCE (Proof Key for Code Exchange) for enhanced security.
//
// # OAuth2 Flow
//
// The typical OAuth2 flow with this package:
//
// 1. Generate authorization URL:
//
//	handler := oauth.NewHandler(clientID, clientSecret, redirectURL)
//	authURL := handler.GetAuthorizationURL(state)
//
// 2. User authorizes in browser and is redirected to callback URL
//
// 3. Handle callback to exchange code for token:
//
//	token, err := handler.HandleCallback(code, state)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Security Features
//
//   - State parameter validation to prevent CSRF attacks
//   - Secure token exchange with client credentials
//   - Automatic cleanup of temporary resources
//   - Timeout handling for authorization flows
//
// # Local Server Mode
//
// For CLI applications, the package can start a temporary local server
// to handle OAuth callbacks:
//
//	token, err := handler.StartOAuthFlow()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// This automatically:
//   - Starts a local HTTP server
//   - Opens the browser to Linear's authorization page
//   - Handles the callback
//   - Returns the access token
//   - Cleans up the server
//
// # Error Handling
//
// The package provides detailed error messages for common OAuth issues:
//   - Invalid authorization codes
//   - State mismatch (CSRF protection)
//   - Network failures during token exchange
//   - Timeout during user authorization
package oauth