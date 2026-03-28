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

// splitSelectorsRespectingParens splits a comma-separated selector list while
// preserving parentheses boundaries. This prevents splitting inside :not(), :is(), etc.
func splitSelectorsRespectingParens(selector string) []string {
	var result []string
	var current strings.Builder
	var parenDepth int

	for _, ch := range selector {
		switch ch {
		case '(':
			parenDepth++
			current.WriteRune(ch)
		case ')':
			parenDepth--
			current.WriteRune(ch)
		case ',':
			if parenDepth == 0 {
				// This comma is a top-level separator
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
			} else {
				// This comma is inside parentheses, keep it
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	// Add the last selector
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// filterClassesInPseudoFunction removes classes from inside pseudo-function parentheses
// while preserving the pseudo-function structure. For example:
// .item:not(.removed, .kept) becomes .item:not(.kept) if .removed is in toRemove
func (w *Writer) filterClassesInPseudoFunction(selector string) string {
	var result strings.Builder
	var parenDepth int
	var parenStart int
	runes := []rune(selector)

	for i := range runes {
		ch := runes[i]

		if ch == '(' {
			if parenDepth == 0 {
				parenStart = i
			}
			parenDepth++
		}

		if parenDepth == 0 {
			// Outside of any parentheses, just add the character
			result.WriteRune(ch)
		}

		if ch == ')' {
			parenDepth--
			if parenDepth == 0 {
				// Exiting parenthesis group - process the content
				content := string(runes[parenStart : i+1])
				filteredContent := w.filterPseudoFunctionContent(content)
				if filteredContent == "REMOVE_THIS_SELECTOR" {
					return "REMOVE_THIS_SELECTOR"
				}
				result.WriteString(filteredContent)
			}
		}
	}

	return result.String()
}

// filterPseudoFunctionContent filters classes inside pseudo-function parentheses
// It splits by commas (respecting nested parens), filters out removed classes, and rejoins
func (w *Writer) filterPseudoFunctionContent(content string) string {
	// content is like "(.class1, .class2)"
	if !strings.HasPrefix(content, "(") || !strings.HasSuffix(content, ")") {
		return content
	}

	// Extract the inner content
	inner := content[1 : len(content)-1]

	// Split by top-level commas
	parts := w.splitByCommaRespectingParens(inner)

	var keptParts []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		// Check if this part contains any classes that should be removed
		classRegex := regexp.MustCompile(`\.([a-zA-Z0-9_-]+)`)
		matches := classRegex.FindAllStringSubmatch(trimmed, -1)

		shouldKeep := true
		for _, match := range matches {
			if len(match) > 1 {
				className := match[1]
				if _, ok := w.toRemove[className]; ok {
					// This part contains a class to be removed
					shouldKeep = false
					break
				}
			}
		}

		if shouldKeep && trimmed != "" {
			keptParts = append(keptParts, part)
		}
	}

	if len(keptParts) == 0 {
		// If nothing remains in the pseudo-function, remove the entire selector
		return "REMOVE_THIS_SELECTOR"
	}

	return "(" + strings.Join(keptParts, ",") + ")"
}

// splitByCommaRespectingParens is like splitSelectorsRespectingParens but for content inside parens
func (w *Writer) splitByCommaRespectingParens(content string) []string {
	var result []string
	var current strings.Builder
	var parenDepth int

	for _, ch := range content {
		switch ch {
		case '(':
			parenDepth++
			current.WriteRune(ch)
		case ')':
			parenDepth--
			current.WriteRune(ch)
		case ',':
			if parenDepth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
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

	// Split selector by comma to get individual selectors, respecting parentheses
	selectors := splitSelectorsRespectingParens(selector)
	var keptSelectors []string

	for _, sel := range selectors {
		trimmedSel := strings.TrimSpace(sel)
		if trimmedSel == "" {
			continue
		}

		// First, filter classes inside pseudo-functions like :not(), :is(), etc.
		filteredSel := w.filterClassesInPseudoFunction(trimmedSel)

		// Check if the entire selector was marked for removal
		if filteredSel == "REMOVE_THIS_SELECTOR" {
			continue
		}

		// Extract classes from this individual selector (excluding those in pseudo-functions)
		classRegex := regexp.MustCompile(`\.([a-zA-Z0-9_-]+)`)
		matches := classRegex.FindAllStringSubmatch(filteredSel, -1)

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
			// Preserve any leading/trailing whitespace from the original selector
			// by using filteredSel but with original padding if they're the same after trimming
			if trimmedSel == filteredSel {
				// No changes were made, preserve original spacing
				keptSelectors = append(keptSelectors, sel)
			} else {
				// Changes were made (classes filtered), preserve original leading space and use filtered content
				leadingSpaces := len(sel) - len(trimmedSel)
				result := sel[:leadingSpaces] + filteredSel
				keptSelectors = append(keptSelectors, result)
			}
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
