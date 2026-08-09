package scanner

import (
	"regexp"
	"strings"

	"github.com/Henelik/css-trimmer/internal/matcher"
)

// extractJSXClasses extracts classes from JSX/TSX content.
func extractJSXClasses(content string) []string {
	var classes []string
	classSet := make(map[string]struct{})

	addClass := func(class string) {
		if class == "" {
			return
		}
		if _, ok := classSet[class]; !ok {
			classes = append(classes, class)
			classSet[class] = struct{}{}
		}
	}

	// Pattern: className="foo bar"
	for match := range matcher.FindSubMatches(`className="`, `"`, content) {
		for part := range strings.FieldsSeq(match) {
			addClass(part)
		}
	}

	// Pattern: class="foo bar" (sometimes JSX uses class too)
	for match := range matcher.FindSubMatches(`class="`, `"`, content) {
		for part := range strings.FieldsSeq(match) {
			addClass(part)
		}
	}

	// Pattern: className={`foo bar ${...}`} and class={`foo bar ${...}`}
	for _, prefix := range []string{"className={`", "class={`"} {
		for match := range matcher.FindSubMatches(prefix, "`", content) {
			for _, class := range extractClassesFromTemplateLiteral(match) {
				addClass(class)
			}
		}
	}

	return classes
}

// extractClassesFromTemplateLiteral returns static class names found inside a
// JSX template literal. It splits whitespace-delimited static text and also
// extracts string literals (single or double quoted) inside ${...} expressions,
// which commonly contribute dynamic class names.
func extractClassesFromTemplateLiteral(content string) []string {
	var classes []string
	seen := make(map[string]struct{})

	add := func(class string) {
		if class == "" {
			return
		}
		if _, ok := seen[class]; !ok {
			seen[class] = struct{}{}
			classes = append(classes, class)
		}
	}

	// Extract static text (with ${...} expressions removed) as whitespace tokens.
	staticText := removeTemplateExpressions(content)
	for _, part := range strings.Fields(staticText) {
		add(part)
	}

	// Conservatively pull string literals out of ${...} expressions.
	for _, expr := range findTemplateExpressions(content) {
		for _, lit := range findStringLiterals(expr) {
			for _, part := range strings.Fields(lit) {
				add(part)
			}
		}
	}

	return classes
}

// findTemplateExpressions returns the bodies of every ${...} expression in a
// template literal, handling nested braces.
func findTemplateExpressions(content string) []string {
	var exprs []string
	depth := 0
	start := -1
	for i := 0; i < len(content); {
		ch := content[i]
		if depth == 0 {
			if ch == '$' && i+1 < len(content) && content[i+1] == '{' {
				depth = 1
				start = i + 2
				i += 2
				continue
			}
		} else {
			switch ch {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					exprs = append(exprs, content[start:i])
					start = -1
				}
			}
		}
		i++
	}
	return exprs
}

// findStringLiterals returns the contents of every single- or double-quoted
// string literal in the given text.
func findStringLiterals(text string) []string {
	var lits []string
	re := regexp.MustCompile(`'([^']*)'|"([^"]*)"`)
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		if m[1] != "" {
			lits = append(lits, m[1])
		} else if m[2] != "" {
			lits = append(lits, m[2])
		}
	}
	return lits
}

// removeTemplateExpressions strips ${...} expressions from a template literal
// body, replacing them with a single space so surrounding static text remains
// separated.
func removeTemplateExpressions(content string) string {
	var result strings.Builder
	depth := 0
	for i := 0; i < len(content); {
		ch := content[i]
		if depth == 0 {
			if ch == '$' && i+1 < len(content) && content[i+1] == '{' {
				depth = 1
				i += 2
				continue
			}
			result.WriteByte(ch)
		} else {
			switch ch {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					result.WriteByte(' ')
				}
			}
		}
		i++
	}
	return result.String()
}
