package cli

import (
	"bufio"
	"io"
	"os"
	"strings"
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
