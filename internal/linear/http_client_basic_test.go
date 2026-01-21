package linear

import (
	"net/http"
	"testing"
	"time"
)

// TestBaseClientHTTPClientConfiguration verifies the HTTP client has proper connection pooling settings
func TestBaseClientHTTPClientConfiguration(t *testing.T) {
	// Create a base client
	base := NewBaseClient("test-token")
	
	// Verify HTTP client is not nil
	if base.httpClient == nil {
		t.Fatal("HTTP client should not be nil")
	}
	
	// Get the transport from the HTTP client
	transport, ok := base.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("HTTP client should use http.Transport")
	}
	
	// Verify connection pooling settings
	if transport.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns should be 100, got %d", transport.MaxIdleConns)
	}
	
	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost should be 10, got %d", transport.MaxIdleConnsPerHost)
	}
	
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout should be 90s, got %v", transport.IdleConnTimeout)
	}
	
	if transport.DisableCompression != false {
		t.Error("Compression should be enabled")
	}
	
	// Verify client timeout
	if base.httpClient.Timeout != 30*time.Second {
		t.Errorf("Client timeout should be 30s, got %v", base.httpClient.Timeout)
	}
}

// TestMultipleBaseClients verifies that different base clients have separate HTTP clients
func TestMultipleBaseClients(t *testing.T) {
	// Create two different base clients
	base1 := NewBaseClient("token1")
	base2 := NewBaseClient("token2")
	
	// Verify they have different HTTP client instances
	if base1.httpClient == base2.httpClient {
		t.Error("Different base clients should have different HTTP client instances")
	}
	
	// But each should have proper configuration
	transport1, _ := base1.httpClient.Transport.(*http.Transport)
	transport2, _ := base2.httpClient.Transport.(*http.Transport)
	
	if transport1.MaxIdleConns != transport2.MaxIdleConns {
		t.Error("Both transports should have same MaxIdleConns configuration")
	}
}