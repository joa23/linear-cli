// Package token provides secure storage and retrieval of Linear API access tokens.
// It handles token persistence across sessions with proper encryption and file
// permissions to protect sensitive authentication credentials.
//
// The package is designed to work seamlessly with the OAuth flow, storing tokens
// after successful authentication and providing them for API requests.
//
// # Token Storage
//
// Tokens are stored in a platform-specific secure location (XDG standard):
//
//	~/.config/linear/token
//
// The token file has restricted permissions (0600) to prevent unauthorized access.
// Only the owner can read or write the token file.
//
// # Security Features
//
//   - File permissions restricted to owner only (0600)
//   - Token validation before storage
//   - Automatic directory creation with proper permissions
//   - Safe file operations with atomic writes
//   - Clear error messages for permission issues
//
// # Usage
//
// Save a token after OAuth authentication:
//
//	storage := token.NewFileStorage()
//	err := storage.SaveToken("your-access-token")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Retrieve a stored token:
//
//	token, err := storage.GetToken()
//	if err != nil {
//	    if err == token.ErrTokenNotFound {
//	        // Handle missing token - user needs to authenticate
//	    }
//	    log.Fatal(err)
//	}
//
// Clear stored token on logout:
//
//	err := storage.ClearToken()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Error Handling
//
// The package defines specific errors for common scenarios:
//   - ErrTokenNotFound: No token file exists (user not authenticated)
//   - ErrTokenEmpty: Token file exists but is empty
//   - Permission errors: Token file has incorrect permissions
//   - I/O errors: File system issues during read/write operations
//
// # Token Lifecycle
//
// 1. Token Creation: OAuth flow generates access token
// 2. Token Storage: SaveToken() stores with secure permissions
// 3. Token Usage: GetToken() retrieves for API requests
// 4. Token Refresh: OAuth refresh flow updates stored token
// 5. Token Removal: ClearToken() removes on logout
//
// # Best Practices
//
//   - Always check for ErrTokenNotFound to detect unauthenticated state
//   - Handle token refresh proactively before expiration
//   - Clear tokens on logout to maintain security
//   - Never log or display token values
//   - Validate tokens before storage to prevent corruption
package token