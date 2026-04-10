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

	// Pattern 1: class="foo bar baz"
	for _, match := range matcher.FindSubMatches(`class="`, `"`, content) {
		for part := range strings.FieldsSeq(match) {
			if part != "" {
				if _, ok := classSet[part]; !ok {
					classes = append(classes, part)
					classSet[part] = struct{}{}
				}
			}
		}
	}

	// Pattern 2: templ.Classes("foo", "bar")
	for _, match := range matcher.FindSubMatches(`templ.Classes(`, ")", content) {
		// Extract strings from the argument list
		for _, className := range matcher.FindSubMatches(`"`, `"`, match) {
			if className != "" {
				if _, ok := classSet[className]; !ok {
					classes = append(classes, className)
					classSet[className] = struct{}{}
				}
			}
		}
	}

	// Pattern 3: Fallback - scan for quoted identifiers that look like CSS classes
	// This is conservative and marks them as potentially used
	for _, match := range identifierRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			className := match[1]
			if className != "" {
				if _, ok := classSet[className]; !ok && !e(className) {
					// Only add if looks like CSS (not common words)
					if isLikelyCSSIdentifier(className) {
						classes = append(classes, className)
						classSet[className] = struct{}{}
					}
				}
			}
		}
	}

	return classes
}

// isLikelyCSSIdentifier checks if a string looks like a CSS class name
// (has dashes or underscores, or is relatively short and descriptive).
func isLikelyCSSIdentifier(s string) bool {
	// Must have dashes or underscores to be conservative
	return strings.ContainsAny(s, "-_")
}

// e is a helper to check for common English words to exclude from CSS identifier detection
func e(s string) bool {
	_, ok := commonWords[strings.ToLower(s)]
	return ok
}
