// Package metadata provides HTML-based metadata storage for Linear issues and projects.
//
// Linear doesn't provide native custom fields, so this package implements
// metadata storage by embedding HTML comment blocks in markdown descriptions.
// The metadata is invisible in Linear's UI but accessible via the API.
//
// # Metadata Format
//
// Metadata is stored as JSON in an HTML comment block:
//
//	<!-- linear-metadata
//	{
//	  "customField1": "value",
//	  "customField2": 42,
//	  "nested": {"key": "value"}
//	}
//	-->
//
// # Extraction
//
// Extract metadata from a description:
//
//	description := "Issue description\n<!-- linear-metadata\n{\"key\":\"value\"}\n-->"
//	metadata, cleanDesc := metadata.ExtractMetadataFromDescription(description)
//	// metadata = map[string]interface{}{"key": "value"}
//	// cleanDesc = "Issue description"
//
// # Injection
//
// Inject metadata into a description:
//
//	metadata := map[string]interface{}{"priority": "high", "customer": "Acme Corp"}
//	newDesc := metadata.InjectMetadataIntoDescription("User description", metadata)
//	// Adds HTML comment block with metadata to description
//
// # Preservation During Updates
//
// When updating descriptions, preserve existing metadata:
//
//	oldDesc := "Old text\n<!-- linear-metadata\n{\"key\":\"value\"}\n-->"
//	newDesc := "New text"
//	finalDesc := metadata.UpdateDescriptionPreservingMetadata(oldDesc, newDesc)
//	// finalDesc = "New text\n<!-- linear-metadata\n{\"key\":\"value\"}\n-->"
//
// # Use Cases
//
// Common metadata use cases:
//   - Custom priority systems
//   - Customer/stakeholder tracking
//   - External system IDs
//   - AI agent state tracking
//   - Custom workflow flags
//
// # Design Rationale
//
// This approach has several advantages:
//   - Works within Linear's existing API (no schema changes needed)
//   - Invisible to users in Linear's UI (clean UX)
//   - Survives copy/paste operations
//   - Compatible with Linear's markdown rendering
//   - Type-safe JSON storage
//
// The HTML comment format was chosen because:
//   - Linear renders it invisibly (unlike code blocks)
//   - It's valid markdown
//   - It doesn't interfere with descriptions
//   - It's easily parsable and detectable
package metadata
