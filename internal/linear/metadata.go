package linear

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Metadata pattern for description-based storage
const metadataPattern = `(?s)<details><summary>🤖 Metadata</summary>\s*` + "```json\n(.*?)\n```" + `\s*</details>`

var metadataRegex = regexp.MustCompile(metadataPattern)

// extractMetadataFromDescription extracts metadata JSON from a description and returns both
// the metadata and the description without the metadata section.
//
// Why this approach: We store metadata as a hidden collapsible section in descriptions
// to avoid cluttering the UI while preserving structured data. This allows metadata to
// travel with issues and projects without requiring separate API calls.
func extractMetadataFromDescription(description string) (map[string]interface{}, string) {
	if description == "" {
		return make(map[string]interface{}), ""
	}

	// Find metadata section
	matches := metadataRegex.FindStringSubmatch(description)
	if len(matches) < 2 {
		// No metadata found, return original description
		return make(map[string]interface{}), description
	}

	// Parse the JSON metadata
	var metadata map[string]interface{}
	jsonStr := matches[1]
	if err := json.Unmarshal([]byte(jsonStr), &metadata); err != nil {
		// If JSON is malformed, log it but still remove the metadata section
		// Why: We don't want malformed metadata to break issue retrieval.
		// The description is still valid even if metadata parsing fails.
		fmt.Printf("Warning: Failed to parse metadata JSON: %v\n", err)
		// Remove the malformed metadata section from description
		cleanDescription := metadataRegex.ReplaceAllString(description, "")
		// Clean up extra newlines that might be left after removal
		cleanDescription = strings.ReplaceAll(cleanDescription, "\n\n\n\n", "\n\n")
		cleanDescription = strings.TrimSpace(cleanDescription)
		return make(map[string]interface{}), cleanDescription
	}

	// Remove metadata section from description
	// Why: We want to present a clean description to users without the
	// technical metadata markup. The metadata is available separately.
	cleanDescription := metadataRegex.ReplaceAllString(description, "")
	// Clean up extra newlines that might be left after removal
	cleanDescription = strings.ReplaceAll(cleanDescription, "\n\n\n\n", "\n\n")
	cleanDescription = strings.TrimSpace(cleanDescription)

	return metadata, cleanDescription
}

// injectMetadataIntoDescription adds metadata to a description as a collapsible section.
// If the description already contains metadata, it will be replaced.
//
// Why this approach: By always replacing existing metadata, we ensure there's only
// one metadata section and it's always up-to-date. The collapsible format keeps
// the description readable while preserving the data.
func injectMetadataIntoDescription(description string, metadata map[string]interface{}) string {
	if metadata == nil || len(metadata) == 0 {
		// No metadata to inject
		return description
	}
	
	// Convert metadata to pretty-printed JSON
	// Why: Pretty printing makes the metadata human-readable if someone
	// expands the collapsible section in the Linear UI.
	jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		// If we can't marshal the metadata, return original description
		// Why: Better to preserve the description than fail the operation
		// due to metadata serialization issues.
		fmt.Printf("Warning: Failed to marshal metadata: %v\n", err)
		return description
	}
	
	// Remove any existing metadata section
	// Why: We want to avoid duplicate metadata sections which could
	// cause confusion and parsing issues.
	cleanDescription := description
	if metadataRegex.MatchString(description) {
		cleanDescription = metadataRegex.ReplaceAllString(description, "")
		cleanDescription = strings.TrimSpace(cleanDescription)
	}
	
	// Create the metadata section
	// Why: The specific format with emoji and markdown ensures consistent
	// rendering across different contexts and makes it easily identifiable.
	metadataSection := fmt.Sprintf("<details><summary>🤖 Metadata</summary>\n\n```json\n%s\n```\n</details>", string(jsonBytes))
	
	// Append metadata section to description
	// Why: Appending at the end keeps the main description content at the
	// top where it's most visible and relevant to users.
	if cleanDescription == "" {
		return metadataSection
	}
	return cleanDescription + "\n\n" + metadataSection
}

// updateDescriptionPreservingMetadata updates a description while preserving any existing metadata.
// This is useful when updating description content without losing metadata.
//
// Why: Users often want to update the human-readable description without worrying
// about preserving technical metadata. This function handles that automatically.
func updateDescriptionPreservingMetadata(oldDescription, newDescription string) string {
	// Extract metadata from old description
	metadata, _ := extractMetadataFromDescription(oldDescription)
	
	// If there was metadata, inject it into the new description
	// Why: This ensures metadata isn't accidentally lost when users update
	// descriptions through various interfaces.
	if metadata != nil && len(metadata) > 0 {
		return injectMetadataIntoDescription(newDescription, metadata)
	}
	
	// No metadata to preserve
	return newDescription
}