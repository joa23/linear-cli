package linear

import (
	"fmt"
	"regexp"
	"unicode"
)

// Constants for validation limits
const (
	// MaxTitleLength is the maximum length for issue titles in Linear
	MaxTitleLength = 255
	// MaxDescriptionLength is the maximum length for descriptions in Linear
	MaxDescriptionLength = 100000
	// MaxNotificationLimit is the maximum number of notifications that can be fetched
	MaxNotificationLimit = 100
)

// isValidMetadataKey validates that a metadata key follows proper naming conventions
// Valid keys must:
// - Not be empty
// - Start with a letter or underscore
// - Contain only letters, numbers, underscores, or hyphens
func isValidMetadataKey(key string) bool {
	if key == "" {
		return false
	}
	
	// Must start with letter or underscore
	firstRune := rune(key[0])
	if !unicode.IsLetter(firstRune) && firstRune != '_' {
		return false
	}
	
	// Rest must be alphanumeric, underscore, or hyphen
	validKeyRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)
	return validKeyRegex.MatchString(key)
}

// isValidEmoji checks if a string is a single valid emoji
func isValidEmoji(emoji string) bool {
	if emoji == "" {
		return false
	}
	
	// Simple heuristic: most single emojis are 4-8 bytes
	// Double emojis are typically 8+ bytes
	if len(emoji) > 8 {
		return false
	}
	
	runes := []rune(emoji)
	
	// Count actual emoji characters (not modifiers)
	emojiCount := 0
	for _, r := range runes {
		if isEmojiRune(r) {
			emojiCount++
		}
	}
	
	// Should have exactly one emoji character
	// (may have additional modifier runes)
	return emojiCount == 1
}

// isEmojiRune checks if a rune is in common emoji ranges
func isEmojiRune(r rune) bool {
	return (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
		(r >= 0x1F300 && r <= 0x1F5FF) || // Misc Symbols and Pictographs
		(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
		(r >= 0x2600 && r <= 0x26FF) ||   // Misc symbols
		(r >= 0x2700 && r <= 0x27BF) ||   // Dingbats
		(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols and Pictographs
		(r >= 0x1F1E6 && r <= 0x1F1FF) ||  // Regional indicators
		r == 0x2705 || r == 0x274C         // Check mark and X mark
}


// validateStringLength validates that a string is within acceptable length limits
func validateStringLength(value, fieldName string, maxLength int) error {
	if len(value) > maxLength {
		return &ValidationError{
			Field:  fieldName,
			Value:  value,
			Reason: fmt.Sprintf("exceeds maximum length of %d characters", maxLength),
		}
	}
	return nil
}

// validatePositiveIntWithRange validates that an integer is positive and within range
func validatePositiveIntWithRange(value int, fieldName string, min, max int) error {
	if value < min {
		return &ValidationError{
			Field:  fieldName,
			Value:  value,
			Reason: fmt.Sprintf("must be at least %d", min),
		}
	}
	if value > max {
		return &ValidationError{
			Field:  fieldName,
			Value:  value,
			Reason: fmt.Sprintf("cannot exceed %d", max),
		}
	}
	return nil
}