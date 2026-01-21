package linear

import (
	"net/http"
	"testing"
	"time"
)

// TestOptimizedHTTPClient verifies the optimized HTTP client configuration
func TestOptimizedHTTPClient(t *testing.T) {
	client := NewOptimizedHTTPClient()
	
	// Verify client is not nil
	if client == nil {
		t.Fatal("Optimized HTTP client should not be nil")
	}
	
	// Verify timeout
	if client.Timeout != 30*time.Second {
		t.Errorf("Client timeout should be 30s, got %v", client.Timeout)
	}
	
	// Verify transport
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Client should use http.Transport")
	}
	
	// Verify connection pooling settings
	expectedSettings := map[string]interface{}{
		"MaxIdleConns":        100,
		"MaxIdleConnsPerHost": 10,
		"MaxConnsPerHost":     10,
		"IdleConnTimeout":     90 * time.Second,
		"DisableCompression":  false,
		"ForceAttemptHTTP2":   true,
		"TLSHandshakeTimeout": 10 * time.Second,
	}
	
	if transport.MaxIdleConns != expectedSettings["MaxIdleConns"] {
		t.Errorf("MaxIdleConns should be %v, got %d", expectedSettings["MaxIdleConns"], transport.MaxIdleConns)
	}
	
	if transport.MaxIdleConnsPerHost != expectedSettings["MaxIdleConnsPerHost"] {
		t.Errorf("MaxIdleConnsPerHost should be %v, got %d", expectedSettings["MaxIdleConnsPerHost"], transport.MaxIdleConnsPerHost)
	}
	
	if transport.MaxConnsPerHost != expectedSettings["MaxConnsPerHost"] {
		t.Errorf("MaxConnsPerHost should be %v, got %d", expectedSettings["MaxConnsPerHost"], transport.MaxConnsPerHost)
	}
	
	if transport.IdleConnTimeout != expectedSettings["IdleConnTimeout"] {
		t.Errorf("IdleConnTimeout should be %v, got %v", expectedSettings["IdleConnTimeout"], transport.IdleConnTimeout)
	}
	
	if transport.DisableCompression != expectedSettings["DisableCompression"] {
		t.Errorf("DisableCompression should be %v, got %v", expectedSettings["DisableCompression"], transport.DisableCompression)
	}
	
	if transport.ForceAttemptHTTP2 != expectedSettings["ForceAttemptHTTP2"] {
		t.Errorf("ForceAttemptHTTP2 should be %v, got %v", expectedSettings["ForceAttemptHTTP2"], transport.ForceAttemptHTTP2)
	}
	
	if transport.TLSHandshakeTimeout != expectedSettings["TLSHandshakeTimeout"] {
		t.Errorf("TLSHandshakeTimeout should be %v, got %v", expectedSettings["TLSHandshakeTimeout"], transport.TLSHandshakeTimeout)
	}
}

// TestSharedHTTPClient verifies the shared HTTP client is a singleton
func TestSharedHTTPClient(t *testing.T) {
	// Get shared client multiple times
	client1 := GetSharedHTTPClient()
	client2 := GetSharedHTTPClient()
	
	// Verify they are the same instance
	if client1 != client2 {
		t.Error("GetSharedHTTPClient should return the same instance")
	}
	
	// Verify it's properly configured
	if client1.Timeout != 30*time.Second {
		t.Errorf("Shared client timeout should be 30s, got %v", client1.Timeout)
	}
}

// TestTransportCloning verifies that each optimized client gets its own transport
func TestTransportCloning(t *testing.T) {
	// Create two optimized clients
	client1 := NewOptimizedHTTPClient()
	client2 := NewOptimizedHTTPClient()
	
	// Verify they have different transports (cloned)
	if client1.Transport == client2.Transport {
		t.Error("Each optimized client should have its own transport instance")
	}
	
	// But configuration should be the same
	transport1 := client1.Transport.(*http.Transport)
	transport2 := client2.Transport.(*http.Transport)
	
	if transport1.MaxIdleConns != transport2.MaxIdleConns {
		t.Error("Cloned transports should have same configuration")
	}
}