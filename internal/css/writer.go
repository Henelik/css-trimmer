package css

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Writer removes CSS rules and writes the result to a file.
type Writer struct {
	content  string
	toRemove map[string]bool
}

// NewWriter creates a CSS writer.
func NewWriter(content string, toRemove []string) *Writer {
	removeSet := make(map[string]bool)
	for _, className := range toRemove {
		removeSet[className] = true
	}

	return &Writer{
		content:  content,
		toRemove: removeSet,
	}
}

// Write applies removals and writes to the specified output file.
func (w *Writer) Write(outputPath string, createBackup bool) error {
	result := w.removeUnusedRules()

	// Create backup if needed
	if createBackup && outputPath != "" {
		backupPath := outputPath + ".bak"
		if err := os.WriteFile(backupPath, []byte(w.content), 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Write result
	if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// removeUnusedRules processes the CSS and removes rules with classes in toRemove.
func (w *Writer) removeUnusedRules() string {
	lines := strings.Split(w.content, "\n")
	var result []string
	var inRule bool
	var ruleBuffer []string
	var braceDepth int

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Count braces in this line
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")

		// Check for rule start - only start a new rule if we're not already in one
		if openBraces > 0 && !strings.HasPrefix(trimmed, "/*") && !inRule {
			inRule = true
			ruleBuffer = []string{line}
			braceDepth = openBraces - closeBraces
			continue
		}

		// Update brace depth if already in a rule
		if inRule {
			braceDepth += openBraces - closeBraces
		}

		// Check for rule end (returning to depth 0)
		if braceDepth <= 0 && closeBraces > 0 && inRule {
			inRule = false
			ruleBuffer = append(ruleBuffer, line)

			// Complete rule buffer - decide whether to keep it
			rule := strings.Join(ruleBuffer, "\n")
			if w.shouldKeepRule(rule) {
				result = append(result, ruleBuffer...)
			}

			ruleBuffer = nil
			braceDepth = 0
			continue
		}

		// If in rule, buffer the line
		if inRule && len(ruleBuffer) > 0 {
			ruleBuffer = append(ruleBuffer, line)
		} else if !inRule {
			// Outside rules - keep all content (comments, at-rules, etc)
			result = append(result, line)
		}
	}

	// Handle any incomplete rule at end
	if len(ruleBuffer) > 0 {
		result = append(result, ruleBuffer...)
	}

	return strings.Join(result, "\n")
}

// shouldKeepRule determines if a CSS rule should be kept.
func (w *Writer) shouldKeepRule(rule string) bool {
	// Check for ignore comment
	if strings.Contains(rule, "/* css-trimmer-ignore */") {
		return true
	}

	// Extract selector from rule
	braceIdx := strings.Index(rule, "{")
	if braceIdx == -1 {
		return true
	}

	selector := strings.TrimSpace(rule[:braceIdx])

	// Extract classes from selector
	classRegex := regexp.MustCompile(`\.([a-zA-Z0-9_-]+)`)
	matches := classRegex.FindAllStringSubmatch(selector, -1)

	var selectorClasses []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			className := match[1]
			if !seen[className] {
				selectorClasses = append(selectorClasses, className)
				seen[className] = true
			}
		}
	}

	// If no classes in selector, also check for classes in the entire rule body
	// (this handles nested rules in @media, @supports, etc)
	if len(selectorClasses) == 0 {
		// Look for any class selectors in the nested content
		nestedMatches := classRegex.FindAllStringSubmatch(rule, -1)
		if len(nestedMatches) == 0 {
			// No classes anywhere in the rule, keep it
			return true
		}

		// There are classes in nested content, check if ANY of them should be kept
		seenNested := make(map[string]bool)
		for _, match := range nestedMatches {
			if len(match) > 1 {
				className := match[1]
				if !seenNested[className] {
					seenNested[className] = true
					// If we find a class that should NOT be removed, keep the entire rule
					if !w.toRemove[className] {
						return true
					}
				}
			}
		}
		// All nested classes should be removed, so remove the rule
		return false
	}

	// If all classes should be removed, remove the rule
	// Otherwise keep it (at least one class should be kept)
	for _, className := range selectorClasses {
		if !w.toRemove[className] {
			return true
		}
	}

	// All classes in this selector should be removed
	return false
}
