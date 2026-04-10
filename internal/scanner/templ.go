package scanner

import (
	"regexp"
	"strings"
)

var (
	classAttrRegex    = regexp.MustCompile(`class="([^"]*)"`)
	templClassesRegex = regexp.MustCompile(`templ\.Classes\(([^)]*)\)`)
	identifierRegex   = regexp.MustCompile(`"([a-zA-Z0-9_-]+)"`)
	stringRegex       = regexp.MustCompile(`"([^"]*)"`)
	commonWords       = map[string]struct{}{
		"the": {}, "and": {}, "or": {}, "for": {}, "is": {}, "in": {}, "of": {},
		"to": {}, "a": {}, "an": {}, "on": {}, "at": {}, "by": {}, "it": {},
	}
)

// ExtractTemplClasses scans a .templ file and returns found class names.
func ExtractTemplClasses(content string) []string {
	var classes []string
	classSet := make(map[string]struct{})

	// Pattern 1: class="foo bar baz"
	for _, match := range classAttrRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			for part := range strings.FieldsSeq(match[1]) {
				if part != "" {
					if _, ok := classSet[part]; !ok {
						classes = append(classes, part)
						classSet[part] = struct{}{}
					}
				}
			}
		}
	}

	// Pattern 2: templ.Classes("foo", "bar")
	for _, match := range templClassesRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			// Extract strings from the argument list
			argContent := match[1]
			for _, stringMatch := range stringRegex.FindAllStringSubmatch(argContent, -1) {
				if len(stringMatch) > 1 {
					className := stringMatch[1]
					if className != "" {
						if _, ok := classSet[className]; !ok {
							classes = append(classes, className)
							classSet[className] = struct{}{}
						}
					}
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
