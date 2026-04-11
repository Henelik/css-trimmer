package matcher

import (
	"iter"
	"strings"
)

// FindSubMatches is a faster version of regex.FindAllStringSubmatch
// Returns only the submatches, and not the outer matches.
func FindSubMatches(prefix, postfix, content string) iter.Seq[string] {
	return func(yield func(string) bool) {
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
			if !yield(content[idx : idx+endIdx]) {
				return
			}

			// Continue searching after this match
			start = idx + endIdx + len(postfix)
		}
	}
}

// MatchCSSClassDefinition matches the rules for a CSS class name definition
// considers a class to start with `.`, and end with one of ` .,:[`
func MatchCSSClassDefinition(content string) iter.Seq[string] {
	return func(yield func(string) bool) {
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
			endIdx := strings.IndexAny(content[idx:], " .,:[")
			if endIdx == -1 {
				// No delimiter found - capture until end of string
				yield(content[idx:])
				return
			}

			// Extract the class value
			if !yield(content[idx : idx+endIdx]) {
				return
			}

			// Continue searching after this match
			start = idx + endIdx
		}
	}
}
