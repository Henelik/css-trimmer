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
	toRemove map[string]struct{}
}

// NewWriter creates a CSS writer.
func NewWriter(content string, toRemove []string) *Writer {
	removeSet := make(map[string]struct{})
	for _, className := range toRemove {
		removeSet[className] = struct{}{}
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
	var i int

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Count braces in this line
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")

		// If we have a line with content and we're not in a rule, check if it could start one
		if !inRule && trimmed != "" && !strings.HasPrefix(trimmed, "/*") {
			// Look ahead to find if there's an opening brace coming up (for multi-line selectors)
			// Start buffering this line as a potential selector
			if !strings.Contains(trimmed, "{") && !strings.Contains(trimmed, "}") {
				// This line doesn't have braces, but might be a selector line
				// Buffer it and continue to the next line
				ruleBuffer = []string{line}
				inRule = true
				braceDepth = 0
				i++
				continue
			}
		}

		// Check for rule start - opening brace without closing
		if openBraces > 0 && !strings.HasPrefix(trimmed, "/*") && !inRule {
			inRule = true
			if len(ruleBuffer) == 0 {
				ruleBuffer = []string{line}
			} else {
				ruleBuffer = append(ruleBuffer, line)
			}
			braceDepth = openBraces - closeBraces
			i++
			continue
		}

		// Update brace depth if already in a rule
		if inRule {
			if len(ruleBuffer) == 0 {
				ruleBuffer = []string{line}
			} else {
				ruleBuffer = append(ruleBuffer, line)
			}
			braceDepth += openBraces - closeBraces
		}

		// Check for rule end (returning to depth 0)
		if braceDepth <= 0 && closeBraces > 0 && inRule && len(ruleBuffer) > 0 {
			inRule = false

			// Complete rule buffer - decide whether to keep it
			rule := strings.Join(ruleBuffer, "\n")
			if w.shouldKeepRule(rule) {
				// If the rule has an ignore comment, keep it as-is
				if strings.Contains(rule, "/* css-trimmer-ignore */") {
					result = append(result, ruleBuffer...)
				} else {
					// Otherwise, process the rule to filter out selectors that should be removed
					filteredRule := w.filterSelectorsFromRule(rule)
					if filteredRule != "" {
						result = append(result, strings.Split(filteredRule, "\n")...)
					}
				}
			}

			ruleBuffer = nil
			braceDepth = 0
			i++
			continue
		}

		// If not in rule and reached here, add line to result
		if !inRule {
			result = append(result, line)
		}

		i++
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
					if _, ok := w.toRemove[className]; !ok {
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
		if _, ok := w.toRemove[className]; !ok {
			return true
		}
	}

	// All classes in this selector should be removed
	return false
}

// filterSelectorsFromRule removes individual selectors from a comma-separated list
// if they contain classes that should be removed.
func (w *Writer) filterSelectorsFromRule(rule string) string {
	// Find the opening brace
	braceIdx := strings.Index(rule, "{")
	if braceIdx == -1 {
		return rule
	}

	selector := rule[:braceIdx]
	body := rule[braceIdx:]

	// Split selector by comma to get individual selectors
	selectors := strings.Split(selector, ",")
	var keptSelectors []string

	for _, sel := range selectors {
		trimmedSel := strings.TrimSpace(sel)
		if trimmedSel == "" {
			continue
		}

		// Extract classes from this individual selector
		classRegex := regexp.MustCompile(`\.([a-zA-Z0-9_-]+)`)
		matches := classRegex.FindAllStringSubmatch(trimmedSel, -1)

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

		// Selector without classes, or selector with at least one class not in removal list
		shouldKeep := false
		if len(selectorClasses) == 0 {
			// Non-class selectors (like div, p, :root, etc) should be kept
			shouldKeep = true
		} else {
			// Check if at least one class should be kept
			for _, className := range selectorClasses {
				if _, ok := w.toRemove[className]; !ok {
					shouldKeep = true
					break
				}
			}
		}

		if shouldKeep {
			keptSelectors = append(keptSelectors, sel)
		}
	}

	// If no selectors remain, return empty string to remove the entire rule
	if len(keptSelectors) == 0 {
		return ""
	}

	// Reconstruct the rule with only kept selectors
	newSelector := strings.Join(keptSelectors, ",")
	return newSelector + body
}
