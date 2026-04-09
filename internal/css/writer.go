package css

import (
	"fmt"
	"io"
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

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	defer outputFile.Close()

	if err := streamRemoveUnusedRules(outputFile, content, removeSet); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	if err := outputFile.Close(); err != nil {
		return fmt.Errorf("failed to close output file: %w", err)
	}

	if createBackup && outputPath != "" {
		backupPath := outputPath + ".bak"
		if err := os.WriteFile(backupPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	return nil
}

// streamRemoveUnusedRules processes the CSS and removes rules with classes in toRemove,
// writing directly to w. Blank lines after removed rules are skipped.
func streamRemoveUnusedRules(w io.Writer, content string, toRemove map[string]struct{}) error {
	lines := strings.Split(content, "\n")
	var inRule bool
	var ruleBuffer []string
	var braceDepth int
	var lastWasRule bool
	var prevWasRule bool

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")

		if !inRule && !strings.HasPrefix(trimmed, "/*") && trimmed != "" {
			if openBraces > 0 {
				inRule = true
				ruleBuffer = []string{line}
				braceDepth = openBraces - closeBraces
				continue
			}
			if !strings.Contains(trimmed, "{") && !strings.Contains(trimmed, "}") {
				inRule = true
				ruleBuffer = []string{line}
				braceDepth = 0
				continue
			}
		}

		if inRule {
			ruleBuffer = append(ruleBuffer, line)
			braceDepth += openBraces - closeBraces

			if braceDepth <= 0 && closeBraces > 0 {
				rule := strings.Join(ruleBuffer, "\n")
				inRule = false
				ruleBuffer = nil
				braceDepth = 0

				if shouldKeepRule(rule, toRemove) {
					if err := filterSelectorsFromRule(w, rule, toRemove); err != nil {
						return err
					}
					lastWasRule = true
					prevWasRule = true
				} else {
					for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
						i++
					}
				}
				continue
			}
			continue
		}

		if trimmed == "" {
			if prevWasRule {
				if _, err := fmt.Fprintln(w); err != nil {
					return err
				}
				prevWasRule = false
			}
		} else if strings.HasPrefix(trimmed, "/*") {
			if lastWasRule || resultLen(w) > 0 {
				if _, err := fmt.Fprint(w, "\n"); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
			lastWasRule = false
		}
	}

	if len(ruleBuffer) > 0 {
		rule := strings.Join(ruleBuffer, "\n")
		if _, err := fmt.Fprint(w, "\n", rule); err != nil {
			return err
		}
	}

	return nil
}

func resultLen(w io.Writer) int {
	if sb, ok := w.(*strings.Builder); ok {
		return sb.Len()
	}
	return 0
}

// shouldKeepRule determines if a CSS rule should be kept.
func shouldKeepRule(rule string, toRemove map[string]struct{}) bool {
	braceIdx := strings.Index(rule, "{")
	if braceIdx == -1 {
		return true
	}

	selector := strings.TrimSpace(rule[:braceIdx])

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

	if len(selectorClasses) == 0 {
		nestedMatches := classRegex.FindAllStringSubmatch(rule, -1)
		if len(nestedMatches) == 0 {
			return true
		}

		seenNested := make(map[string]bool)
		for _, match := range nestedMatches {
			if len(match) > 1 {
				className := match[1]
				if !seenNested[className] {
					seenNested[className] = true
					if _, ok := toRemove[className]; !ok {
						return true
					}
				}
			}
		}
		return false
	}

	for _, className := range selectorClasses {
		if _, ok := toRemove[className]; !ok {
			return true
		}
	}

	return false
}

// splitSelectorsRespectingParens splits a comma-separated selector list while
// preserving parentheses boundaries.
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
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
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

// filterClassesInPseudoFunction removes classes from inside pseudo-function parentheses
// while preserving the pseudo-function structure.
func filterClassesInPseudoFunction(selector string, toRemove map[string]struct{}) string {
	var result strings.Builder
	var parenDepth int
	var parenStart int
	var leadingSpace string
	var foundNonSpace bool
	var trailingSpace string
	runes := []rune(selector)
	pendingTrailing := ""

	for i := range runes {
		ch := runes[i]

		if ch == '(' {
			if parenDepth == 0 {
				parenStart = i
				if pendingTrailing != "" {
					result.WriteString(pendingTrailing)
					pendingTrailing = ""
				}
			}
			parenDepth++
		}

		if parenDepth == 0 {
			if foundNonSpace {
				if ch == ' ' || ch == '\t' {
					pendingTrailing += string(ch)
				} else {
					trailingSpace = pendingTrailing
					pendingTrailing = ""
					result.WriteRune(ch)
				}
			} else if ch == ' ' || ch == '\t' {
				leadingSpace += string(ch)
			} else {
				result.WriteRune(ch)
				foundNonSpace = true
			}
		}

		if ch == ')' {
			parenDepth--
			if parenDepth == 0 {
				content := string(runes[parenStart : i+1])
				filteredContent := filterPseudoFunctionContent(content, toRemove)
				if filteredContent == removeSelector {
					return removeSelector
				}
				result.WriteString(filteredContent)
			}
		}
	}

	if pendingTrailing != "" {
		trailingSpace = pendingTrailing
	}

	return leadingSpace + result.String() + trailingSpace
}

// filterPseudoFunctionContent filters classes inside pseudo-function parentheses.
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

// splitByCommaRespectingParens splits content by commas while respecting parentheses.
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
func filterSelectorsFromRule(w io.Writer, rule string, toRemove map[string]struct{}) error {
	braceIdx := strings.Index(rule, "{")
	if braceIdx == -1 {
		_, err := fmt.Fprintln(w, rule)
		return err
	}

	selector := rule[:braceIdx]
	body := rule[braceIdx:]

	selectors := splitSelectorsRespectingParens(selector)
	first := true

	for _, sel := range selectors {
		trimmedSel := strings.TrimSpace(sel)
		if trimmedSel == "" {
			continue
		}

		filteredSel := filterClassesInPseudoFunction(sel, toRemove)
		if filteredSel == removeSelector {
			continue
		}

		filteredTrimmed := strings.TrimSpace(filteredSel)
		matches := classRegex.FindAllStringSubmatch(filteredTrimmed, -1)

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
			if _, err := fmt.Fprint(w, ","); err != nil {
				return err
			}
		}
		first = false

		if trimmedSel == filteredTrimmed {
			if _, err := fmt.Fprint(w, sel); err != nil {
				return err
			}
		} else {
			leadingSpace := getLeadingWhitespace(sel)
			if _, err := fmt.Fprint(w, leadingSpace, filteredSel); err != nil {
				return err
			}
		}
	}

	if !first {
		if _, err := fmt.Fprint(w, body, "\n"); err != nil {
			return err
		}
	}

	return nil
}

func getLeadingWhitespace(s string) string {
	i := 0
	for i < len(s) {
		ch := rune(s[i])
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			i++
		} else {
			break
		}
	}
	return s[:i]
}

// removeUnusedRules is a convenience wrapper that calls streamRemoveUnusedRules
// and returns the result as a string for backward compatibility.
func removeUnusedRules(content string, toRemove map[string]struct{}) string {
	var sb strings.Builder
	streamRemoveUnusedRules(&sb, content, toRemove)
	return sb.String()
}
