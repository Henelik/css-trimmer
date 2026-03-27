package css

import (
	"regexp"
	"strings"
)

// Parser builds a ClassInventory from CSS content.
type Parser struct {
	content   string
	inventory ClassInventory
}

// NewParser creates a new CSS parser.
func NewParser(content string) *Parser {
	return &Parser{
		content:   content,
		inventory: make(ClassInventory),
	}
}

// Parse analyzes the CSS and builds a class inventory of defined classes.
func (p *Parser) Parse() (ClassInventory, error) {
	lines := strings.Split(p.content, "\n")
	var inRule bool
	var ruleStart int

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
			classes := p.extractClassesFromSelector(selectorPart)

			for _, className := range classes {
				if _, exists := p.inventory[className]; !exists {
					p.inventory[className] = ClassInfo{StartLine: ruleStart, EndLine: lineNum}
				}
			}
		}

		// End of rule
		if strings.Contains(trimmed, "}") && inRule {
			inRule = false
		}
	}

	return p.inventory, nil
}

// extractClassesFromSelector finds all class names in a CSS selector.
func (p *Parser) extractClassesFromSelector(selector string) []string {
	var classes []string
	seen := make(map[string]bool)

	// Regex to find .classname patterns
	classRegex := regexp.MustCompile(`\.([a-zA-Z0-9_-]+)`)
	matches := classRegex.FindAllStringSubmatch(selector, -1)

	for _, match := range matches {
		if len(match) > 1 {
			className := match[1]
			if !seen[className] {
				classes = append(classes, className)
				seen[className] = true
			}
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
