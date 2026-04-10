package matcher

import "strings"

// FindSubMatches is a faster version of regex.FindAllStringSubmatch
// Returns only the submatches, and not the outer matches.
func FindSubMatches(prefix, postfix, content string) []string {
	// Pre-allocate typical capacity to reduce allocations
	result := make([]string, 0, 16)

	start := 0
	for {
		// Find next prefix occurrence
		idx := strings.Index(content[start:], prefix)
		if idx == -1 {
			break
		}

		// Convert relative index to absolute after prefix
		idx += start + len(prefix)

		// Find closing quote
		endIdx := strings.Index(content[idx:], postfix)
		if endIdx == -1 {
			break
		}

		// Extract the substring
		result = append(result, content[idx:idx+endIdx])

		// Continue searching after this match
		start = idx + endIdx + len(postfix)
	}

	return result
}

// MatchClassName matches the rules for a CSS class name definition
// considers a class to start with `.`, and end with one of ` .,:`
func MatchCSSClassDefinition(content string) []string {
	// Pre-allocate typical capacity to reduce allocations
	result := make([]string, 0, 16)

	start := 0
	for {
		// Find next class definition
		idx := strings.IndexByte(content[start:], '.')
		if idx == -1 {
			break
		}

		// Convert relative index to absolute after prefix
		idx += start + 1

		// Find the end of the definition
		endIdx := strings.IndexAny(content[idx:], " .,:")
		if endIdx == -1 {
			break
		}

		// Extract the class value
		result = append(result, content[idx:idx+endIdx])

		// Continue searching after this match
		start = idx + endIdx
	}

	return result
}
