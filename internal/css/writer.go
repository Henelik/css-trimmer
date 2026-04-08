package css

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var classRegex = regexp.MustCompile(`\.([a-zA-Z0-9_-]+)`)

const removeSelector = "REMOVE_THIS_SELECTOR"

// Write applies removals and writes to the specified output file.
func Write(content string, toRemove []string, outputPath string, createBackup bool) error {
	removeSet := make(map[string]struct{})
	for _, className := range toRemove {
		removeSet[className] = struct{}{}
	}

	result := removeUnusedRules(content, removeSet)

	// Create backup if needed
	if createBackup && outputPath != "" {
		backupPath := outputPath + ".bak"
		if err := os.WriteFile(backupPath, []byte(content), 0644); err != nil {
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
// Blank lines after removed rules are skipped. Blank lines between kept rules are preserved.
func removeUnusedRules(content string, toRemove map[string]struct{}) string {
	lines := strings.Split(content, "\n")
	result := &strings.Builder{}
	var inRule bool
	var ruleBuffer []string
	var braceDepth int
	var lastWasRule bool

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Count braces in this line
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")

		// Detect start of a rule
		if !inRule && !strings.HasPrefix(trimmed, "/*") && trimmed != "" {
			if openBraces > 0 {
				inRule = true
				ruleBuffer = []string{line}
				braceDepth = openBraces - closeBraces
				continue
			}
			// Multi-line selector: line without brace but could be start
			if !strings.Contains(trimmed, "{") && !strings.Contains(trimmed, "}") {
				inRule = true
				ruleBuffer = []string{line}
				braceDepth = 0
				continue
			}
		}

		// Accumulate lines if in a rule
		if inRule {
			ruleBuffer = append(ruleBuffer, line)
			braceDepth += openBraces - closeBraces

			// Rule ends when braceDepth returns to 0 or below
			if braceDepth <= 0 && closeBraces > 0 {
				rule := strings.Join(ruleBuffer, "\n")
				inRule = false
				ruleBuffer = nil
				braceDepth = 0

				if shouldKeepRule(rule, toRemove) {
					filteredRule := filterSelectorsFromRule(rule, toRemove)
					if filteredRule != "" {
						// Add separator if needed
						if result.Len() > 0 {
							result.WriteString("\n")
						}
						result.WriteString(filteredRule)
						lastWasRule = true
					}
				} else {
					// Rule removed: skip subsequent blank lines
					for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
						i++
					}
				}
				continue
			}
			continue
		}

		// Not in a rule - handle comments and blank lines
		if trimmed == "" {
			// Blank line: only write if we have content and not after a removed rule
			if result.Len() > 0 && lastWasRule {
				result.WriteString("\n")
			}
		} else if strings.HasPrefix(trimmed, "/*") {
			// Comment: add separator if needed
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(line)
			lastWasRule = false
		}
	}

	// Handle incomplete rule at end (malformed CSS) - write out as-is
	if len(ruleBuffer) > 0 {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		for i, line := range ruleBuffer {
			if i > 0 {
				result.WriteString("\n")
			}
			result.WriteString(line)
		}
	}

	return result.String()
}

// shouldKeepRule determines if a CSS rule should be kept.
func shouldKeepRule(rule string, toRemove map[string]struct{}) bool {
	braceIdx := strings.Index(rule, "{")
	if braceIdx == -1 {
		return true
	}

	selector := strings.TrimSpace(rule[:braceIdx])

	// Extract classes from selector
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
					if _, ok := toRemove[className]; !ok {
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
		if _, ok := toRemove[className]; !ok {
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
func filterClassesInPseudoFunction(selector string, toRemove map[string]struct{}) string {
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
				filteredContent := filterPseudoFunctionContent(content, toRemove)
				if filteredContent == removeSelector {
					return removeSelector
				}
				result.WriteString(filteredContent)
			}
		}
	}

	return result.String()
}

// filterPseudoFunctionContent filters classes inside pseudo-function parentheses
// It splits by commas (respecting nested parens), filters out removed classes, and rejoins
func filterPseudoFunctionContent(content string, toRemove map[string]struct{}) string {
	if !strings.HasPrefix(content, "(") || !strings.HasSuffix(content, ")") {
		return content
	}

	inner := content[1 : len(content)-1]
	parts := splitByCommaRespectingParens(inner)

	var builder strings.Builder
	first := true
	builder.WriteString("(")

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		matches := classRegex.FindAllStringSubmatch(trimmed, -1)

		shouldKeep := true
		for _, match := range matches {
			if len(match) > 1 {
				className := match[1]
				if _, ok := toRemove[className]; ok {
					shouldKeep = false
					break
				}
			}
		}

		if !shouldKeep || trimmed == "" {
			continue
		}

		if !first {
			builder.WriteString(",")
		}
		first = false

		builder.WriteString(part)
	}

	if first {
		return removeSelector
	}

	builder.WriteString(")")
	return builder.String()
}

// splitByCommaRespectingParens is like splitSelectorsRespectingParens but for content inside parens
func splitByCommaRespectingParens(content string) []string {
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
func filterSelectorsFromRule(rule string, toRemove map[string]struct{}) string {
	braceIdx := strings.Index(rule, "{")
	if braceIdx == -1 {
		return rule
	}

	selector := rule[:braceIdx]
	body := rule[braceIdx:]

	selectors := splitSelectorsRespectingParens(selector)
	var builder strings.Builder
	first := true

	for _, sel := range selectors {
		trimmedSel := strings.TrimSpace(sel)
		if trimmedSel == "" {
			continue
		}

		filteredSel := filterClassesInPseudoFunction(trimmedSel, toRemove)
		if filteredSel == removeSelector {
			continue
		}

		matches := classRegex.FindAllStringSubmatch(filteredSel, -1)

		seen := make(map[string]bool)
		hasKeepableClass := false
		for _, match := range matches {
			if len(match) > 1 {
				className := match[1]
				if !seen[className] {
					seen[className] = true
					if _, ok := toRemove[className]; !ok {
						hasKeepableClass = true
						break
					}
				}
			}
		}

		shouldKeep := len(matches) == 0 || hasKeepableClass
		if !shouldKeep {
			continue
		}

		if !first {
			builder.WriteString(",")
		}
		first = false

		if trimmedSel == filteredSel {
			builder.WriteString(sel)
		} else {
			for _, ch := range sel {
				if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
					builder.WriteRune(ch)
				} else {
					break
				}
			}
			builder.WriteString(filteredSel)
		}
	}

	if first {
		return ""
	}

	builder.WriteString(body)
	return builder.String()
}
