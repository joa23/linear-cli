// Package config manages configuration and environment settings for the Linear MCP server.
// It provides centralized configuration management with support for multiple sources
// including environment variables, configuration files, and default values.
//
// The package handles the complexity of configuration precedence, validation, and
// secure storage of sensitive configuration values like API credentials.
//
// # Configuration Sources
//
// Configuration is loaded from multiple sources in order of precedence:
//
// 1. Environment variables (highest precedence)
// 2. Configuration file (~/.config/linear/config.yaml)
// 3. Default values (lowest precedence)
//
// # Environment Variables
//
// The package recognizes the following environment variables:
//
//	LINEAR_CLIENT_ID     - OAuth2 client ID for Linear application
//	LINEAR_CLIENT_SECRET - OAuth2 client secret (stored securely)
//	LINEAR_API_KEY       - Direct API token for CLI usage
//	LINEAR_REDIRECT_URL  - OAuth2 callback URL (default: http://localhost:8080/callback)
//
// # Configuration File
//
// The configuration file is stored at ~/.config/linear/config.yaml with restricted
// permissions (0600) to prevent unauthorized access. The file structure:
//
//	{
//	    "client_id": "your-client-id",
//	    "client_secret": "encrypted-secret",
//	    "redirect_url": "http://localhost:8080/callback"
//	}
//
// # Security
//
// The package implements several security measures:
//   - Configuration files have restricted permissions (0600)
//   - Sensitive values are never logged
//   - Client secrets are stored encrypted when possible
//   - Environment variables are preferred for CI/CD environments
//
// # Usage
//
// Get configuration instance:
//
//	cfg := config.GetConfig()
//	clientID := cfg.GetClientID()
//	clientSecret := cfg.GetClientSecret()
//
// Update configuration:
//
//	err := cfg.SetClientID("new-client-id")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Error Handling
//
// The package provides detailed errors for configuration issues:
//   - Missing required configuration values
//   - Invalid configuration file format
//   - Permission errors when accessing configuration files
//   - Validation errors for configuration values
package config
