package css

import (
	"strings"

	"github.com/Henelik/css-trimmer/internal/matcher"
)

// Parse analyzes the CSS and builds a class inventory of defined classes.
func ParseCSS(content string) (ClassInventory, error) {
	lines := strings.Split(content, "\n")

	var inRule bool
	var ruleStart int
	inventory := make(ClassInventory, len(lines))

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and pure comments
		if trimmed == "" || strings.HasPrefix(trimmed, "/*") {
			continue
		}

		// Start of rule - extract selector
		if strings.Contains(trimmed, "{") && !strings.HasPrefix(trimmed, "/*") {
			inRule = true
			ruleStart = lineNum

			// Extract selector (everything before the {)
			selectorPart := strings.Split(trimmed, "{")[0]
			classes := extractClassesFromSelector(selectorPart)

			for _, className := range classes {
				if _, exists := inventory[className]; !exists {
					inventory[className] = ClassInfo{StartLine: ruleStart, EndLine: lineNum}
				}
			}
		}

		// End of rule
		if strings.Contains(trimmed, "}") && inRule {
			inRule = false
		}
	}

	return inventory, nil
}

// extractClassesFromSelector finds all class names in a CSS selector.
func extractClassesFromSelector(selector string) []string {
	var classes []string
	seen := make(map[string]struct{})

	// Regex to find .classname patterns
	matches := matcher.MatchCSSClassDefinition(selector)

	for _, match := range matches {
		if _, ok := seen[match]; !ok {
			classes = append(classes, match)
			seen[match] = struct{}{}
		}
	}

	return classes
}

// AllClasses returns a slice of all class names defined in the CSS.
func (inv ClassInventory) AllClasses() []string {
	var classes []string
	for className := range inv {
		classes = append(classes, className)
	}
	return classes
}
