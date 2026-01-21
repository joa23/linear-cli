package linear

// MapSortField converts user-facing sort names to Linear's PaginationOrderBy enum
// Returns empty string for priority (client-side sorting required)
func MapSortField(sort string) string {
	switch sort {
	case "created":
		return "createdAt"
	case "updated":
		return "updatedAt"
	case "priority":
		// Priority sorting must be done client-side
		return ""
	default:
		return "updatedAt"
	}
}

// MapSortDirection converts user-facing direction to GraphQL direction
func MapSortDirection(direction string) string {
	if direction == "asc" {
		return "asc"
	}
	return "desc" // Default to descending
}

// ValidatePagination validates and normalizes pagination input
func ValidatePagination(input *PaginationInput) *PaginationInput {
	if input == nil {
		input = &PaginationInput{}
	}

	// Set defaults
	if input.Limit <= 0 {
		input.Limit = 10
	}
	if input.Limit > 250 {
		input.Limit = 250 // Cap at Linear's maximum
	}
	if input.Start < 0 {
		input.Start = 0
	}
	if input.Sort == "" {
		input.Sort = "updated"
	}
	if input.Direction == "" {
		input.Direction = "desc"
	}

	return input
}
