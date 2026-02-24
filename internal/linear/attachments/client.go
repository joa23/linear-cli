package attachments

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/joa23/linear-cli/internal/linear/guidance"
	"github.com/joa23/linear-cli/internal/token"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/draw"
)

// CacheEntry represents a cached attachment with expiration
type CacheEntry struct {
	Content     []byte
	ContentType string
	Size        int64
	Width       int
	Height      int
	Resized     bool
	ExpiresAt   time.Time
}

// AttachmentCache provides in-memory caching for processed attachments
type AttachmentCache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
}

// NewAttachmentCache creates a new attachment cache with specified TTL
func NewAttachmentCache(ttl time.Duration) *AttachmentCache {
	cache := &AttachmentCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// AttachmentClient handles attachment download and processing operations
type Client struct {
	base       *core.BaseClient
	httpClient *http.Client
	cache      *AttachmentCache
}

// NewAttachmentClient creates a new attachment client
func NewClient(base *core.BaseClient) *Client {
	// Use 30-minute TTL to align with Linear's URL expiration patterns
	// Linear attachment URLs typically expire after 1 hour, so 30 minutes
	// provides a good balance between performance and freshness
	cache := NewAttachmentCache(30 * time.Minute)

	return &Client{
		base:       base,
		httpClient: core.NewOptimizedHTTPClient(), // Reuse the optimized HTTP client
		cache:      cache,
	}
}

// AttachmentFormat defines the supported return formats for attachments
type AttachmentFormat string

const (
	FormatBase64   AttachmentFormat = "base64"   // Base64 encoded content (default for MCP)
	FormatURL      AttachmentFormat = "url"      // Direct URL (for large files)
	FormatMetadata AttachmentFormat = "metadata" // Metadata only, no download
)

// AttachmentResponse represents the response from GetAttachment
type AttachmentResponse struct {
	Format      AttachmentFormat `json:"format"`
	Content     string           `json:"content,omitempty"`     // Base64 content or URL
	URL         string           `json:"url,omitempty"`         // Original URL
	ContentType string           `json:"contentType,omitempty"` // MIME type
	Size        int64            `json:"size,omitempty"`        // Content size in bytes
	Width       int              `json:"width,omitempty"`       // Image width (if image)
	Height      int              `json:"height,omitempty"`      // Image height (if image)
	Resized     bool             `json:"resized,omitempty"`     // True if image was resized for MCP limits
	Error       string           `json:"error,omitempty"`       // Error message if download failed
}

// MCPSizeLimit is the 1MB limit for MCP content (Claude Desktop constraint)
const MCPSizeLimit = 1024 * 1024 // 1MB

// GetAttachment downloads and processes an attachment with the specified format
// Note: This is a simplified implementation that expects the full attachment URL to be passed as attachmentID
// In practice, this would be enhanced to properly resolve attachment IDs to URLs
func (ac *Client) GetAttachment(attachmentURL string, format AttachmentFormat) (*AttachmentResponse, error) {
	// Validate input
	if attachmentURL == "" {
		return nil, guidance.ValidationErrorWithExample("attachmentId", "cannot be empty",
			`// First list issue attachments
attachments = linear_list_issue_attachments("issue-id")
// Then get a specific attachment using its URL
content = linear_get_attachment(attachments[0].url, "base64")`)
	}

	response := &AttachmentResponse{
		Format: format,
		URL:    attachmentURL,
	}

	// For metadata-only format, return early
	if format == FormatMetadata {
		return response, nil
	}

	// For URL format, just return the URL
	if format == FormatURL {
		response.Content = attachmentURL
		return response, nil
	}

	// For base64 format, check cache first, then download and process
	cacheKey := ac.generateCacheKey(attachmentURL, format)
	if cached := ac.cache.Get(cacheKey); cached != nil {
		// Return cached version
		response.ContentType = cached.ContentType
		response.Size = cached.Size
		response.Width = cached.Width
		response.Height = cached.Height
		response.Resized = cached.Resized
		response.Content = base64.StdEncoding.EncodeToString(cached.Content)
		return response, nil
	}

	// Not in cache - download and process the content
	content, contentType, size, err := ac.downloadAttachment(attachmentURL)
	if err != nil {
		response.Error = fmt.Sprintf("Failed to download attachment: %v", err)
		return response, nil // Don't cache errors
	}

	response.ContentType = contentType
	response.Size = size

	// Check if it's an image and get dimensions
	if ac.isImageContent(contentType) {
		width, height, err := ac.getImageDimensions(content)
		if err == nil {
			response.Width = width
			response.Height = height
		}

		// Resize if over MCP limit
		if size > MCPSizeLimit {
			resizedContent, err := ac.resizeImageForMCP(content, contentType)
			if err != nil {
				// If resize fails, fall back to URL format - don't cache
				response.Format = FormatURL
				response.Content = attachmentURL
				response.Error = fmt.Sprintf("Image too large (%d bytes) and resize failed: %v", size, err)
				return response, nil
			}
			content = resizedContent
			response.Size = int64(len(content))
			response.Resized = true

			// Update dimensions after resize
			width, height, err := ac.getImageDimensions(content)
			if err == nil {
				response.Width = width
				response.Height = height
			}
		}

		// Cache the successfully processed image
		ac.cache.Set(cacheKey, &CacheEntry{
			Content:     content,
			ContentType: contentType,
			Size:        response.Size,
			Width:       response.Width,
			Height:      response.Height,
			Resized:     response.Resized,
			ExpiresAt:   time.Now().Add(ac.cache.ttl),
		})
	} else {
		// For non-images over the limit, return URL instead
		if size > MCPSizeLimit {
			response.Format = FormatURL
			response.Content = attachmentURL
			response.Error = fmt.Sprintf("File too large (%d bytes) for MCP transfer, returning URL instead", size)
			return response, nil
		}

		// Cache the non-image content
		ac.cache.Set(cacheKey, &CacheEntry{
			Content:     content,
			ContentType: contentType,
			Size:        size,
			ExpiresAt:   time.Now().Add(ac.cache.ttl),
		})
	}

	// Encode content as base64
	response.Content = base64.StdEncoding.EncodeToString(content)
	return response, nil
}

// getAttachmentMetadata retrieves metadata for a specific attachment
// Since Linear doesn't have a direct attachment(id:) query, we need to find it through issues
func (ac *Client) getAttachmentMetadata(attachmentID string) (*core.Attachment, error) {
	// For now, we'll create a basic attachment with just the ID
	// The URL will be constructed or retrieved when we need to download
	// This is a simplified approach - in a real implementation, we might cache
	// attachment metadata from previous list operations
	return &core.Attachment{
		ID: attachmentID,
		// URL will be set when we attempt download
		// Other fields will be populated from HTTP headers during download
	}, nil
}

// DownloadToTempFile downloads a private Linear URL with auth (adds Bearer header
// automatically for uploads.linear.app URLs), saves content to
// /tmp/linear-img-<sha256-of-url>.<ext>, and returns the file path.
func (ac *Client) DownloadToTempFile(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("url cannot be empty")
	}

	content, contentType, _, err := ac.downloadAttachment(url)
	if err != nil {
		return "", err
	}

	ext := extensionFromContentType(contentType)

	// Derive a stable filename from the URL so repeated calls reuse the same file.
	hasher := sha256.New()
	hasher.Write([]byte(url))
	hash := fmt.Sprintf("%x", hasher.Sum(nil))[:16]

	path := filepath.Join(os.TempDir(), fmt.Sprintf("linear-img-%s%s", hash, ext))
	if err := os.WriteFile(path, content, 0600); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	return path, nil
}

// extensionFromContentType returns a file extension (with dot) for a MIME type.
func extensionFromContentType(ct string) string {
	// Strip parameters (e.g. "image/png; charset=utf-8")
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	switch ct {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "application/pdf":
		return ".pdf"
	default:
		return ".bin"
	}
}

// ListAttachments queries all attachments for an issue.
// issueID must be a UUID (resolve identifiers like "TEC-123" before calling).
func (ac *Client) ListAttachments(issueID string) ([]core.Attachment, error) {
	const query = `
		query IssueAttachments($id: String!) {
			issue(id: $id) {
				attachments(first: 50) {
					nodes {
						id
						url
						title
						subtitle
						sourceType
						createdAt
						updatedAt
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id": issueID,
	}

	var response struct {
		Issue struct {
			Attachments struct {
				Nodes []core.Attachment `json:"nodes"`
			} `json:"attachments"`
		} `json:"issue"`
	}

	err := ac.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}

	return response.Issue.Attachments.Nodes, nil
}

// AttachmentCreateInput holds parameters for creating a Linear attachment object.
type AttachmentCreateInput struct {
	IssueID  string // Required — UUID of the issue
	URL      string // Required — attachment URL (also unique key per issue)
	Title    string // Required — display title
	Subtitle string // Optional — display subtitle
}

// AttachmentUpdateInput holds parameters for updating an attachment.
type AttachmentUpdateInput struct {
	Title    string // Required by API
	Subtitle string // Optional
}

// CreateAttachment creates a new Linear attachment object on an issue.
func (ac *Client) CreateAttachment(input *AttachmentCreateInput) (*core.Attachment, error) {
	const mutation = `
		mutation AttachmentCreate($input: AttachmentCreateInput!) {
			attachmentCreate(input: $input) {
				success
				attachment {
					id
					url
					title
					subtitle
					sourceType
					createdAt
					updatedAt
				}
			}
		}
	`

	inputMap := map[string]interface{}{
		"issueId": input.IssueID,
		"url":     input.URL,
		"title":   input.Title,
	}
	if input.Subtitle != "" {
		inputMap["subtitle"] = input.Subtitle
	}

	variables := map[string]interface{}{
		"input": inputMap,
	}

	var response struct {
		AttachmentCreate struct {
			Success    bool            `json:"success"`
			Attachment core.Attachment `json:"attachment"`
		} `json:"attachmentCreate"`
	}

	err := ac.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create attachment: %w", err)
	}

	if !response.AttachmentCreate.Success {
		return nil, fmt.Errorf("attachmentCreate returned success=false")
	}

	return &response.AttachmentCreate.Attachment, nil
}

// UpdateAttachment updates an existing attachment's title and subtitle.
func (ac *Client) UpdateAttachment(id string, input *AttachmentUpdateInput) (*core.Attachment, error) {
	const mutation = `
		mutation AttachmentUpdate($id: String!, $input: AttachmentUpdateInput!) {
			attachmentUpdate(id: $id, input: $input) {
				success
				attachment {
					id
					url
					title
					subtitle
					sourceType
					updatedAt
				}
			}
		}
	`

	inputMap := map[string]interface{}{
		"title": input.Title,
	}
	if input.Subtitle != "" {
		inputMap["subtitle"] = input.Subtitle
	}

	variables := map[string]interface{}{
		"id":    id,
		"input": inputMap,
	}

	var response struct {
		AttachmentUpdate struct {
			Success    bool            `json:"success"`
			Attachment core.Attachment `json:"attachment"`
		} `json:"attachmentUpdate"`
	}

	err := ac.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to update attachment: %w", err)
	}

	if !response.AttachmentUpdate.Success {
		return nil, fmt.Errorf("attachmentUpdate returned success=false")
	}

	return &response.AttachmentUpdate.Attachment, nil
}

// DeleteAttachment deletes an attachment by UUID.
func (ac *Client) DeleteAttachment(id string) error {
	const mutation = `
		mutation AttachmentDelete($id: String!) {
			attachmentDelete(id: $id) {
				success
			}
		}
	`

	variables := map[string]interface{}{
		"id": id,
	}

	var response struct {
		AttachmentDelete struct {
			Success bool `json:"success"`
		} `json:"attachmentDelete"`
	}

	err := ac.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	if !response.AttachmentDelete.Success {
		return fmt.Errorf("attachmentDelete returned success=false")
	}

	return nil
}

// downloadAttachment downloads the actual attachment content with retry logic and robust error handling
func (ac *Client) downloadAttachment(url string) ([]byte, string, int64, error) {
	return ac.downloadAttachmentWithRetry(url, 3)
}

// downloadAttachmentWithRetry implements robust download with exponential backoff retry logic
func (ac *Client) downloadAttachmentWithRetry(url string, maxRetries int) ([]byte, string, int64, error) {
	const baseDelay = 200 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		content, contentType, size, err := ac.attemptDownload(url)

		// Success case
		if err == nil {
			return content, contentType, size, nil
		}

		// Store the error for final return
		lastErr = err

		// Don't retry on final attempt
		if attempt == maxRetries {
			break
		}

		// Check if error is retryable
		if !ac.isRetryableError(err) {
			return nil, "", 0, err
		}

		// Exponential backoff with jitter
		delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
		// Add jitter to prevent thundering herd
		jitter := time.Duration(rand.Int63n(int64(delay / 4)))
		time.Sleep(delay + jitter)
	}

	return nil, "", 0, fmt.Errorf("download failed after %d retries: %w", maxRetries+1, lastErr)
}

// isPrivateLinearURL reports whether the URL requires a Linear auth token.
func isPrivateLinearURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), "uploads.linear.app")
}

// attemptDownload performs a single download attempt
func (ac *Client) attemptDownload(url string) ([]byte, string, int64, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Private Linear upload URLs require a Bearer token.
	if isPrivateLinearURL(url) {
		tok, err := ac.base.GetToken()
		if err != nil {
			return nil, "", 0, fmt.Errorf("failed to get Linear auth token for private upload URL (run 'linear auth login'): %w", err)
		}
		req.Header.Set("Authorization", token.FormatAuthHeader(tok))
	}

	// Set timeout for individual requests
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := ac.httpClient.Do(req)
	if err != nil {
		// Enhance error context for better debugging
		if ctx.Err() == context.DeadlineExceeded {
			return nil, "", 0, fmt.Errorf("download timeout after 30s: %w", err)
		}
		return nil, "", 0, fmt.Errorf("network error during download: %w", err)
	}
	defer resp.Body.Close()

	// Handle various HTTP status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Success case - continue to read content
	case http.StatusNotFound:
		return nil, "", 0, fmt.Errorf("attachment not found (404) - URL may have expired: %s", url)
	case http.StatusForbidden:
		return nil, "", 0, fmt.Errorf("access denied (403) - insufficient permissions for attachment: %s", url)
	case http.StatusUnauthorized:
		return nil, "", 0, fmt.Errorf("authentication required (401) - Linear token may be invalid")
	case http.StatusTooManyRequests:
		// Rate limited - this is retryable
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			return nil, "", 0, fmt.Errorf("rate limited (429) - retry after %s seconds", retryAfter)
		}
		return nil, "", 0, fmt.Errorf("rate limited (429) - too many requests")
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		// Server errors - these are retryable
		return nil, "", 0, fmt.Errorf("server error (%d) - Linear service temporarily unavailable", resp.StatusCode)
	default:
		return nil, "", 0, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Limit content size to prevent memory issues
	const maxContentSize = 100 * 1024 * 1024 // 100MB limit
	limitedReader := &io.LimitedReader{R: resp.Body, N: maxContentSize}

	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to read content: %w", err)
	}

	// Check if content was truncated
	if limitedReader.N == 0 && len(content) == maxContentSize {
		return nil, "", 0, fmt.Errorf("content too large (>%d MB) - use URL format for large files", maxContentSize/(1024*1024))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		// Try to detect content type from content
		contentType = http.DetectContentType(content)
	}

	size := int64(len(content))
	return content, contentType, size, nil
}

// isRetryableError determines if an error should trigger a retry
func (ac *Client) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors are usually retryable
	if strings.Contains(errStr, "network error") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection refused") {
		return true
	}

	// Server errors are retryable
	if strings.Contains(errStr, "server error") ||
		strings.Contains(errStr, "rate limited") ||
		strings.Contains(errStr, "service temporarily unavailable") {
		return true
	}

	// URL expiration might be recoverable if we refresh the URL
	if strings.Contains(errStr, "404") || strings.Contains(errStr, "not found") {
		// In a full implementation, we could try to refresh the attachment URL
		// by re-querying the issue's attachments
		return false // For now, don't retry 404s as they likely won't resolve
	}

	// Permission and auth errors are not retryable
	if strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "access denied") ||
		strings.Contains(errStr, "authentication required") {
		return false
	}

	// Content too large errors are not retryable
	if strings.Contains(errStr, "content too large") {
		return false
	}

	// Default to not retryable for unknown errors
	return false
}

// isImageContent checks if the content type represents an image
func (ac *Client) isImageContent(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// getImageDimensions returns the width and height of an image
func (ac *Client) getImageDimensions(imageData []byte) (width, height int, err error) {
	img, _, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image config: %w", err)
	}
	return img.Width, img.Height, nil
}

// resizeImageForMCP resizes an image to fit under the MCP size limit while maintaining aspect ratio
func (ac *Client) resizeImageForMCP(imageData []byte, contentType string) ([]byte, error) {
	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	// Start with 80% of original size and reduce until under limit
	scale := 0.8
	var resizedData []byte

	for scale > 0.1 {
		newWidth := int(float64(originalWidth) * scale)
		newHeight := int(float64(originalHeight) * scale)

		// Create resized image
		resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		draw.BiLinear.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

		// Encode the resized image
		var buf bytes.Buffer
		switch format {
		case "jpeg":
			err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85})
		case "png":
			err = png.Encode(&buf, resized)
		default:
			// Default to JPEG for unknown formats (including WebP)
			err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85})
		}

		if err != nil {
			return nil, fmt.Errorf("failed to encode resized image: %w", err)
		}

		resizedData = buf.Bytes()

		// Check if it's under the limit
		if len(resizedData) <= MCPSizeLimit {
			break
		}

		// Reduce scale further
		scale -= 0.1
	}

	if len(resizedData) > MCPSizeLimit {
		return nil, fmt.Errorf("unable to resize image under %d bytes", MCPSizeLimit)
	}

	return resizedData, nil
}

// generateCacheKey creates a cache key from URL and format
func (ac *Client) generateCacheKey(url string, format AttachmentFormat) string {
	// Use SHA256 hash of URL + format to create consistent cache key
	// This handles long URLs and special characters safely
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s", url, format)))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// Cache management methods

// Get retrieves a cache entry if it exists and hasn't expired.
// Expired entries are left for the background cleanup goroutine to remove.
func (cache *AttachmentCache) Get(key string) *CacheEntry {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	entry, exists := cache.entries[key]
	if !exists {
		return nil
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil
	}

	return entry
}

// Set stores a cache entry
func (cache *AttachmentCache) Set(key string, entry *CacheEntry) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.entries[key] = entry
}

// Clear removes all entries from the cache
func (cache *AttachmentCache) Clear() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.entries = make(map[string]*CacheEntry)
}

// Size returns the current number of cached entries
func (cache *AttachmentCache) Size() int {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return len(cache.entries)
}

// cleanupExpired runs periodically to remove expired entries
func (cache *AttachmentCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute) // Run cleanup every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cache.removeExpiredEntries()
		}
	}
}

// removeExpiredEntries removes all expired entries from the cache
// This is separated for testing purposes
func (cache *AttachmentCache) removeExpiredEntries() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	now := time.Now()
	for key, entry := range cache.entries {
		if now.After(entry.ExpiresAt) {
			delete(cache.entries, key)
		}
	}
}

// FileUploadResponse represents the response from the fileUpload mutation
type FileUploadResponse struct {
	UploadURL string            `json:"uploadUrl"`
	AssetURL  string            `json:"assetUrl"`
	Headers   map[string]string `json:"headers"`
}

// UploadFile uploads a file to Linear and returns the asset URL
// This implements the full upload flow:
// 1. Call fileUpload mutation to get upload URL and headers
// 2. PUT the file content to the upload URL
// 3. Return the asset URL for use in markdown
func (ac *Client) UploadFile(filename string, content []byte, contentType string) (string, error) {
	if filename == "" {
		return "", &core.ValidationError{Field: "filename", Message: "filename cannot be empty"}
	}
	if len(content) == 0 {
		return "", &core.ValidationError{Field: "content", Message: "content cannot be empty"}
	}
	if contentType == "" {
		contentType = http.DetectContentType(content)
	}

	// Step 1: Get upload URL from Linear
	uploadResp, err := ac.getUploadURL(len(content), filename, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to get upload URL: %w", err)
	}

	// Step 2: Upload the file to the upload URL
	err = ac.uploadToURL(uploadResp.UploadURL, content, contentType, uploadResp.Headers)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Step 3: Return the asset URL
	return uploadResp.AssetURL, nil
}

// UploadFileFromPath uploads a file from a filesystem path
func (ac *Client) UploadFileFromPath(filepath string) (string, error) {
	content, err := ac.readFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	filename := ac.extractFilename(filepath)
	contentType := ac.detectContentType(filepath, content)

	return ac.UploadFile(filename, content, contentType)
}

// getUploadURL calls the fileUpload mutation to get an upload URL
func (ac *Client) getUploadURL(size int, filename, contentType string) (*FileUploadResponse, error) {
	const mutation = `
		mutation FileUpload($size: Int!, $filename: String!, $contentType: String!) {
			fileUpload(size: $size, filename: $filename, contentType: $contentType) {
				success
				uploadFile {
					uploadUrl
					assetUrl
					headers {
						key
						value
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"size":        size,
		"filename":    filename,
		"contentType": contentType,
	}

	var response struct {
		FileUpload struct {
			Success    bool `json:"success"`
			UploadFile struct {
				UploadURL string `json:"uploadUrl"`
				AssetURL  string `json:"assetUrl"`
				Headers   []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"headers"`
			} `json:"uploadFile"`
		} `json:"fileUpload"`
	}

	err := ac.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return nil, err
	}

	if !response.FileUpload.Success {
		return nil, fmt.Errorf("fileUpload mutation failed")
	}

	// Convert headers array to map
	headers := make(map[string]string)
	for _, h := range response.FileUpload.UploadFile.Headers {
		headers[h.Key] = h.Value
	}

	return &FileUploadResponse{
		UploadURL: response.FileUpload.UploadFile.UploadURL,
		AssetURL:  response.FileUpload.UploadFile.AssetURL,
		Headers:   headers,
	}, nil
}

// uploadToURL performs the actual file upload via PUT request
func (ac *Client) uploadToURL(uploadURL string, content []byte, contentType string, headers map[string]string) error {
	req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	// Set content type
	req.Header.Set("Content-Type", contentType)

	// Set headers from Linear's fileUpload response
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(req.Context(), 60*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := ac.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// readFile reads a file from the filesystem
func (ac *Client) readFile(filepath string) ([]byte, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Check file size first
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	const maxSize = 100 * 1024 * 1024 // 100MB limit
	if stat.Size() > maxSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", stat.Size(), maxSize)
	}

	return io.ReadAll(file)
}

// extractFilename gets the filename from a path
func (ac *Client) extractFilename(filepath string) string {
	// Find last / or \
	for i := len(filepath) - 1; i >= 0; i-- {
		if filepath[i] == '/' || filepath[i] == '\\' {
			return filepath[i+1:]
		}
	}
	return filepath
}

// detectContentType determines the MIME type from filename and content
func (ac *Client) detectContentType(filepath string, content []byte) string {
	// Try to detect from content first
	contentType := http.DetectContentType(content)
	if contentType != "application/octet-stream" {
		return contentType
	}

	// Fall back to extension-based detection
	ext := ""
	for i := len(filepath) - 1; i >= 0; i-- {
		if filepath[i] == '.' {
			ext = filepath[i:]
			break
		}
	}

	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	default:
		return contentType
	}
}

// Helper functions for image analysis

// IsImageAttachment checks if an attachment is an image based on content type
func IsImageAttachment(attachment core.Attachment) bool {
	return strings.HasPrefix(attachment.SourceType, "image/") ||
		strings.HasPrefix(attachment.ContentType, "image/")
}

// FilterImageAttachments returns only image attachments from a slice
func FilterImageAttachments(attachments []core.Attachment) []core.Attachment {
	var images []core.Attachment
	for _, attachment := range attachments {
		if IsImageAttachment(attachment) {
			images = append(images, attachment)
		}
	}
	return images
}

// GetImageDimensions returns dimensions of image data
func GetImageDimensions(imageData []byte) (width, height int, err error) {
	img, _, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image config: %w", err)
	}
	return img.Width, img.Height, nil
}
