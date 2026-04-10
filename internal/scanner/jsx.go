package scanner

import "strings"

// extractJSXClasses extracts classes from JSX/TSX content.
func extractJSXClasses(content string) []string {
	var classes []string
	classSet := make(map[string]bool)

	// Pattern: className="foo bar"
	for _, match := range classNameRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			for part := range strings.FieldsSeq(match[1]) {
				if part != "" && !classSet[part] {
					classes = append(classes, part)
					classSet[part] = true
				}
			}
		}
	}

	// Pattern: class="foo bar" (sometimes JSX uses class too)
	for _, match := range classRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			for part := range strings.FieldsSeq(match[1]) {
				if part != "" && !classSet[part] {
					classes = append(classes, part)
					classSet[part] = true
				}
			}
		}
	}

	return classes
}
