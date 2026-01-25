// Package pagination provides utilities for offset-based pagination in Linear API queries.
//
// This package handles validation, normalization, and field mapping for
// paginated queries to ensure consistent behavior across the CLI.
//
// # Pagination Input
//
// Pagination uses offset-based navigation with limit and start parameters:
//
//	input := &core.PaginationInput{
//	    Limit:     10,        // Items per page
//	    Start:     20,        // Offset (skip first 20)
//	    Sort:      "updated", // Sort field
//	    Direction: "desc",    // Sort direction
//	}
//
// # Validation
//
// Validate and normalize pagination parameters:
//
//	input = pagination.ValidatePagination(input)
//	// Sets defaults: limit=10, start=0, sort="updated", direction="desc"
//	// Caps limit at 250 (Linear's maximum)
//	// Ensures non-negative start
//
// # Sort Field Mapping
//
// Map user-friendly sort names to Linear's API fields:
//
//	apiField := pagination.MapSortField("created")  // Returns "createdAt"
//	apiField := pagination.MapSortField("updated")  // Returns "updatedAt"
//	apiField := pagination.MapSortField("priority") // Returns "" (client-side sort)
//
// Linear's API doesn't support priority sorting, so it returns empty string
// to indicate client-side sorting is required.
//
// # Sort Direction Mapping
//
// Map sort directions to Linear's expected values:
//
//	direction := pagination.MapSortDirection("asc")  // Returns "asc"
//	direction := pagination.MapSortDirection("desc") // Returns "desc"
//	direction := pagination.MapSortDirection("")     // Returns "desc" (default)
//
// # Design Philosophy
//
// This package implements offset-based pagination instead of cursor-based
// pagination for better user experience:
//
//   - Offset-based: Users can jump to page 5 directly
//   - Cursor-based: Users must paginate through pages 1-4 first
//
// While cursor-based pagination is more efficient for very large datasets,
// offset-based pagination is more intuitive for CLI users who want to
// navigate directly to specific pages.
//
// The tradeoff is acceptable because:
//   - Most queries return < 1000 results
//   - CLI usage is interactive (not bulk processing)
//   - User experience matters more than microsecond efficiency gains
package pagination
