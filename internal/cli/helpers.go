package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joa23/linear-cli/internal/linear"
	"github.com/spf13/cobra"
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

// getDescriptionFromFlagOrStdin returns description from flag or stdin.
// Stdin is only read when the flag value is explicitly "-" (e.g. -d -),
// matching the convention used by git, gh, curl, etc.
// Previously, stdin was read speculatively whenever a non-TTY pipe was
// detected, which caused commands to hang in CI, && chains, and
// subprocess contexts. See https://github.com/joa23/linear-cli/issues/10
func getDescriptionFromFlagOrStdin(flagValue string) (string, error) {
	if flagValue == "-" {
		if hasStdinPipe() {
			return readStdin()
		}
		return "", fmt.Errorf("'-' requires piped input (e.g. cat file.md | linear ... -d -)")
	}

	return flagValue, nil
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

// looksLikeCycleNumber returns true if the string appears to be a cycle number (all digits)
func looksLikeCycleNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// parsePriority parses a priority string that can be either:
// - An integer: "0", "1", "2", "3", "4"
// - A name: "none", "urgent", "high", "normal", "low"
// Returns the priority integer (0-4) or an error
func parsePriority(s string) (int, error) {
	if s == "" {
		return 0, nil
	}

	// Try parsing as integer first
	if p, err := strconv.Atoi(s); err == nil {
		if p < 0 || p > 4 {
			return 0, fmt.Errorf("priority must be 0-4, got %d", p)
		}
		return p, nil
	}

	// Parse as name (case-insensitive)
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "none", "no":
		return 0, nil
	case "urgent", "critical":
		return 1, nil
	case "high":
		return 2, nil
	case "normal", "medium":
		return 3, nil
	case "low":
		return 4, nil
	default:
		return 0, fmt.Errorf("invalid priority '%s': use 0-4 or none/urgent/high/normal/low", s)
	}
}

// Context key type for dependencies injection
type depsKey string

const dependenciesKey depsKey = "dependencies"

// getDeps extracts Dependencies from command context
// Returns an error if dependencies are not found, which indicates a programming bug
func getDeps(cmd *cobra.Command) (*Dependencies, error) {
	deps, ok := cmd.Context().Value(dependenciesKey).(*Dependencies)
	if !ok {
		return nil, fmt.Errorf("internal error: dependencies not initialized - this is a bug, please report it at https://github.com/joa23/linear-cli/issues")
	}
	return deps, nil
}
