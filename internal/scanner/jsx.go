package scanner

import (
	"strings"

	"github.com/Henelik/css-trimmer/internal/matcher"
)

// extractJSXClasses extracts classes from JSX/TSX content.
func extractJSXClasses(content string) []string {
	var classes []string
	classSet := make(map[string]struct{})

	// Pattern: className="foo bar"
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

	// Pattern: class="foo bar" (sometimes JSX uses class too)
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

	return classes
}
