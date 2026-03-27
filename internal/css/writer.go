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

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for rule start
		if strings.Contains(line, "{") && !strings.HasPrefix(trimmed, "/*") {
			inRule = true
			ruleBuffer = []string{line}
			continue
		}

		// Check for rule end
		if strings.Contains(line, "}") && inRule {
			inRule = false
			ruleBuffer = append(ruleBuffer, line)

			// Complete rule buffer - decide whether to keep it
			rule := strings.Join(ruleBuffer, "\n")
			if w.shouldKeepRule(rule) {
				result = append(result, ruleBuffer...)
			}

			ruleBuffer = nil
			continue
		}

		// If in rule, buffer the line
		if inRule && len(ruleBuffer) > 0 {
			ruleBuffer = append(ruleBuffer, line)
		} else {
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

	// If no classes in selector, keep it
	if len(selectorClasses) == 0 {
		return true
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
