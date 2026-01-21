package linear

import (
	"net"
	"net/http"
	"time"
)

// defaultHTTPTransport provides optimized transport settings for Linear API
var defaultHTTPTransport = &http.Transport{
	// Connection pooling settings
	MaxIdleConns:        100,              // Maximum idle connections across all hosts
	MaxIdleConnsPerHost: 10,               // Maximum idle connections per host
	MaxConnsPerHost:     10,               // Maximum total connections per host
	IdleConnTimeout:     90 * time.Second, // How long idle connections are kept alive
	
	// Connection settings
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second, // Connection timeout
		KeepAlive: 30 * time.Second, // TCP keepalive interval
	}).DialContext,
	
	// TLS and HTTP/2 settings
	ForceAttemptHTTP2:     true,              // Use HTTP/2 when available
	TLSHandshakeTimeout:   10 * time.Second,  // TLS handshake timeout
	ExpectContinueTimeout: 1 * time.Second,   // 100-continue timeout
	ResponseHeaderTimeout: 10 * time.Second,  // Time to wait for response headers
	
	// Compression
	DisableCompression: false, // Enable gzip compression
}

// NewOptimizedHTTPClient creates an HTTP client optimized for Linear API usage
func NewOptimizedHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: defaultHTTPTransport.Clone(),
	}
}

// sharedHTTPClient is a package-level HTTP client for OAuth and other non-API usage
var sharedHTTPClient = NewOptimizedHTTPClient()

// GetSharedHTTPClient returns the shared HTTP client instance
// This should be used for OAuth and other non-API operations that don't need
// per-client authentication headers
func GetSharedHTTPClient() *http.Client {
	return sharedHTTPClient
}