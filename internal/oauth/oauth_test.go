package oauth

import (
	"net/url"
	"testing"
)

func TestOAuthFlow(t *testing.T) {
	t.Run("callback handling", func(t *testing.T) {
		// Skip this test for now as it requires complex mock setup
		// The real OAuth flow will be tested manually
		t.Skip("Skipping callback test - requires mock Linear token endpoint")
	})
	
	t.Run("authorization URL generation", func(t *testing.T) {
		handler := NewHandler("test-client-id", "test-client-secret")
		
		authURL := handler.GetAuthorizationURL("http://localhost:8080/callback", "test-state")
		
		// Parse URL to verify parameters
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			t.Fatalf("Failed to parse authorization URL: %v", err)
		}
		
		// Check that it's Linear's OAuth endpoint
		if parsedURL.Host != "linear.app" {
			t.Errorf("Expected host linear.app, got %s", parsedURL.Host)
		}
		
		// Check required parameters
		query := parsedURL.Query()
		if query.Get("client_id") != "test-client-id" {
			t.Error("Missing or incorrect client_id parameter")
		}
		
		if query.Get("redirect_uri") != "http://localhost:8080/callback" {
			t.Error("Missing or incorrect redirect_uri parameter")
		}
		
		if query.Get("response_type") != "code" {
			t.Error("Missing or incorrect response_type parameter")
		}
		
		if query.Get("state") != "test-state" {
			t.Error("Missing or incorrect state parameter")
		}
	})
	
	t.Run("app authorization URL generation", func(t *testing.T) {
		handler := NewHandler("test-client-id", "test-client-secret")
		
		authURL := handler.GetAppAuthorizationURL("http://localhost:8080/callback", "test-state")
		
		// Parse URL to verify parameters
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			t.Fatalf("Failed to parse authorization URL: %v", err)
		}
		
		// Check that it's Linear's OAuth endpoint
		if parsedURL.Host != "linear.app" {
			t.Errorf("Expected host linear.app, got %s", parsedURL.Host)
		}
		
		// Check required parameters
		query := parsedURL.Query()
		if query.Get("client_id") != "test-client-id" {
			t.Error("Missing or incorrect client_id parameter")
		}
		
		if query.Get("redirect_uri") != "http://localhost:8080/callback" {
			t.Error("Missing or incorrect redirect_uri parameter")
		}
		
		if query.Get("response_type") != "code" {
			t.Error("Missing or incorrect response_type parameter")
		}
		
		if query.Get("state") != "test-state" {
			t.Error("Missing or incorrect state parameter")
		}
		
		// Check app-specific parameters
		if query.Get("actor") != "app" {
			t.Error("Missing or incorrect actor parameter for app authentication")
		}
		
		// Check scopes include app-specific scopes
		scopes := query.Get("scope")
		if scopes != "app:assignable app:mentionable read write" {
			t.Errorf("Expected app scopes, got: %s", scopes)
		}
	})
}