package scanner

import (
	"regexp"
	"strings"

	"github.com/Henelik/css-trimmer/internal/matcher"
)

var (
	identifierRegex = regexp.MustCompile(`"([a-zA-Z0-9_-]+)"`)
	commonWords     = map[string]struct{}{
		"the": {}, "and": {}, "or": {}, "for": {}, "is": {}, "in": {}, "of": {},
		"to": {}, "a": {}, "an": {}, "on": {}, "at": {}, "by": {}, "it": {},
	}
)

// ExtractTemplClasses scans a .templ file and returns found class names.
func ExtractTemplClasses(content string) []string {
	var classes []string
	classSet := make(map[string]struct{})

	addClass := func(class string) {
		if class == "" {
			return
		}
		if _, ok := classSet[class]; !ok {
			classes = append(classes, class)
			classSet[class] = struct{}{}
		}
	}

	// Pattern 1: class="foo bar baz"
	for match := range matcher.FindSubMatches(`class="`, `"`, content) {
		for part := range strings.FieldsSeq(match) {
			addClass(part)
		}
	}

	// Pattern 2: templ.Classes("foo", "bar")
	for match := range matcher.FindSubMatches(`templ.Classes(`, ")", content) {
		// Extract strings from the argument list
		for className := range matcher.FindSubMatches(`"`, `"`, match) {
			addClass(className)
		}
	}

	// Pattern 3: Fallback - scan for quoted identifiers that look like CSS
	// classes, but only inside templ component blocks to avoid matching Go
	// string literals in regular function bodies.
	blocks := extractTemplBlocks(content)
	for _, match := range identifierRegex.FindAllStringSubmatchIndex(blocks, -1) {
		if len(match) > 3 {
			className := blocks[match[2]:match[3]]
			if _, ok := classSet[className]; !ok && !isExcludedWord(className) {
				if isLikelyCSSIdentifier(className) {
					addClass(className)
				}
			}
		}
	}

	return classes
}

// extractTemplBlocks returns the concatenated content of all templ component
// blocks, i.e. regions between "templ " and the matching closing brace, while
// skipping ordinary Go functions and other source regions.
func extractTemplBlocks(content string) string {
	var result strings.Builder
	i := 0
	for i < len(content) {
		idx := strings.Index(content[i:], "templ ")
		if idx == -1 {
			break
		}
		idx += i

		// Advance past the "templ " keyword and the component signature to find
		// the opening brace of the component body.
		brace := strings.Index(content[idx:], "{")
		if brace == -1 {
			break
		}
		bodyStart := idx + brace

		// Find the matching closing brace, accounting for nested braces.
		depth := 1
		bodyEnd := -1
		for j := bodyStart + 1; j < len(content); j++ {
			switch content[j] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					bodyEnd = j
					goto done
				}
			}
		}
	done:
		if bodyEnd == -1 {
			break
		}

		result.WriteString(content[bodyStart : bodyEnd+1])
		i = bodyEnd + 1
	}
	return result.String()
}

// isLikelyCSSIdentifier checks if a string looks like a CSS class name
// (has dashes or underscores, or is relatively short and descriptive).
func isLikelyCSSIdentifier(s string) bool {
	// Must have dashes or underscores to be conservative
	return strings.ContainsAny(s, "-_")
}

// isExcludedWord checks whether a string is a common English word that should be
// excluded from CSS identifier detection.
func isExcludedWord(s string) bool {
	_, ok := commonWords[strings.ToLower(s)]
	return ok
}
