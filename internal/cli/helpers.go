package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/joa23/linear-cli/internal/linear"
)

const (
	// DefaultLimit is the default number of results to return
	DefaultLimit = 25
	// MaxLimit is the maximum number of results allowed by the Linear API
	MaxLimit = 250
)

// hasStdinPipe detects if content is piped to stdin
func hasStdinPipe() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// readStdin reads all piped content from stdin
func readStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var builder strings.Builder

	for {
		line, err := reader.ReadString('\n')
		builder.WriteString(line)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
	}

	return strings.TrimSpace(builder.String()), nil
}

// parseCommaSeparated splits a comma-separated string into a slice
// Trims whitespace from each element and filters empty strings
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// getDescriptionFromFlagOrStdin returns description from flag or stdin
// Flag takes precedence over stdin
func getDescriptionFromFlagOrStdin(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if hasStdinPipe() {
		return readStdin()
	}

	return "", nil
}

// uploadAndAppendAttachments uploads files and appends markdown image links to body
// Returns the updated body string or an error if any upload fails
func uploadAndAppendAttachments(client *linear.Client, body string, filePaths []string) (string, error) {
	if len(filePaths) == 0 {
		return body, nil
	}

	for _, filePath := range filePaths {
		assetURL, err := client.Attachments.UploadFileFromPath(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to upload %s: %w", filePath, err)
		}
		// Append image markdown to body
		if body != "" {
			body += "\n\n"
		}
		body += fmt.Sprintf("![%s](%s)", filepath.Base(filePath), assetURL)
	}
	return body, nil
}

// validateAndNormalizeLimit validates and normalizes a limit parameter
// Returns DefaultLimit if limit <= 0, returns error if limit > MaxLimit
func validateAndNormalizeLimit(limit int) (int, error) {
	if limit <= 0 {
		return DefaultLimit, nil
	}
	if limit > MaxLimit {
		return 0, fmt.Errorf("--limit cannot exceed %d (Linear API maximum), got %d", MaxLimit, limit)
	}
	return limit, nil
}
