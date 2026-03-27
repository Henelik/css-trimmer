package scanner

import (
	"regexp"
	"strings"
)

// ExtractTemplClasses scans a .templ file and returns found class names.
func ExtractTemplClasses(content string) []string {
	var classes []string
	classSet := make(map[string]bool)

	// Pattern 1: class="foo bar baz"
	classAttrRegex := regexp.MustCompile(`class="([^"]*)"`)
	for _, match := range classAttrRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			parts := strings.Fields(match[1])
			for _, part := range parts {
				if part != "" && !classSet[part] {
					classes = append(classes, part)
					classSet[part] = true
				}
			}
		}
	}

	// Pattern 2: templ.Classes("foo", "bar")
	templClassesRegex := regexp.MustCompile(`templ\.Classes\(([^)]*)\)`)
	for _, match := range templClassesRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			// Extract strings from the argument list
			argContent := match[1]
			stringRegex := regexp.MustCompile(`"([^"]*)"`)
			for _, stringMatch := range stringRegex.FindAllStringSubmatch(argContent, -1) {
				if len(stringMatch) > 1 {
					className := stringMatch[1]
					if className != "" && !classSet[className] {
						classes = append(classes, className)
						classSet[className] = true
					}
				}
			}
		}
	}

	// Pattern 3: Fallback - scan for quoted identifiers that look like CSS classes
	// This is conservative and marks them as potentially used
	identifierRegex := regexp.MustCompile(`"([a-zA-Z0-9_-]+)"`)
	for _, match := range identifierRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			className := match[1]
			if className != "" && !classSet[className] && !e(className) {
				// Only add if looks like CSS (not common words)
				if isLikelyCSSIdentifier(className) {
					classes = append(classes, className)
					classSet[className] = true
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
	commonWords := map[string]bool{
		"the": true, "and": true, "or": true, "for": true, "is": true, "in": true, "of": true,
		"to": true, "a": true, "an": true, "on": true, "at": true, "by": true, "it": true,
	}
	return commonWords[strings.ToLower(s)]
}
