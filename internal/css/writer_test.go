package css

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toRemoveSet(toRemove []string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, className := range toRemove {
		result[className] = struct{}{}
	}
	return result
}

func TestWriterWrite(t *testing.T) {
	t.Run("writes modified CSS to file without backup", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "css*.css")
		require.NoError(t, err)
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())

		content := `.keep {
  color: blue;
}
.remove {
  color: red;
}`
		err = Write(content, []string{"remove"}, tmpfile.Name(), false)

		require.NoError(t, err)

		result, err := os.ReadFile(tmpfile.Name())
		require.NoError(t, err)
		assert.Contains(t, string(result), ".keep")
		assert.NotContains(t, string(result), ".remove")
	})

	t.Run("creates backup file when requested", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "css*.css")
		require.NoError(t, err)
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())
		defer os.Remove(tmpfile.Name() + ".bak")

		content := `.test { color: red; }`
		err = Write(content, []string{}, tmpfile.Name(), true)

		require.NoError(t, err)

		backup, err := os.ReadFile(tmpfile.Name() + ".bak")
		require.NoError(t, err)
		assert.Equal(t, content, string(backup))
	})

	t.Run("handles file write errors gracefully", func(t *testing.T) {
		content := ".test { color: red; }"
		err := Write(content, []string{}, "/invalid/nonexistent/path/file.css", false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write output file")
	})

	t.Run("handles backup creation errors gracefully", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "css*.css")
		require.NoError(t, err)
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())

		content := ".test { color: red; }"
		err = Write(content, []string{}, "/invalid/path/file.css", true)

		assert.Error(t, err)
	})
}

func TestWriterRemoveUnusedRules(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		toRemove     []string
		shouldKeep   []string
		shouldRemove []string
	}{
		{
			name: "removes single unused class",
			content: `.keep {
  color: blue;
}
.remove {
  color: red;
}`,
			toRemove:     []string{"remove"},
			shouldKeep:   []string{".keep"},
			shouldRemove: []string{".remove"},
		},
		{
			name: "keeps multiple classes when some are not removed",
			content: `.btn {
  padding: 5px;
}
.btn-primary {
  background: blue;
}
.btn-secondary {
  background: gray;
}`,
			toRemove:     []string{"btn-secondary"},
			shouldKeep:   []string{".btn", ".btn-primary"},
			shouldRemove: []string{".btn-secondary"},
		},
		{
			name: "removes rules with multiple classes all marked for removal",
			content: `.remove1.remove2 {
  color: red;
}
.keep {
  color: blue;
}`,
			toRemove:     []string{"remove1", "remove2"},
			shouldKeep:   []string{".keep"},
			shouldRemove: []string{".remove1", ".remove2"},
		},
		{
			name: "keeps rules with mixed removal status",
			content: `.keep.remove {
  color: red;
}`,
			toRemove:     []string{"remove"},
			shouldKeep:   []string{".keep"},
			shouldRemove: nil,
		},
		{
			name: "preserves non-class selectors",
			content: `div {
  margin: 0;
}
.remove {
  padding: 0;
}
p {
  color: black;
}`,
			toRemove:     []string{"remove"},
			shouldKeep:   []string{"div {", "p {"},
			shouldRemove: []string{".remove"},
		},
		{
			name: "handles comments and preserves them",
			content: `/* Important style */
.remove {
  color: red;
}
/* Another comment */
.keep {
  color: blue;
}`,
			toRemove:     []string{"remove"},
			shouldKeep:   []string{".keep", "/* Important style */"},
			shouldRemove: []string{".remove"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeUnusedRules(tt.content, toRemoveSet(tt.toRemove))

			for _, keep := range tt.shouldKeep {
				assert.Contains(t, result, keep)
			}
			for _, remove := range tt.shouldRemove {
				if remove != "" {
					assert.NotContains(t, result, remove)
				}
			}
		})
	}
}

func TestWriterShouldKeepRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		toRemove []string
		expected bool
	}{
		{
			name:     "removes rule when all classes should be removed",
			rule:     ".remove { color: red; }",
			toRemove: []string{"remove"},
			expected: false,
		},
		{
			name:     "keeps rule when class is not in removal list",
			rule:     ".keep { color: blue; }",
			toRemove: []string{"remove"},
			expected: true,
		},
		{
			name:     "keeps rule with multiple classes if at least one is kept",
			rule:     ".keep.remove { color: blue; }",
			toRemove: []string{"remove"},
			expected: true,
		},
		{
			name:     "removes rule when all classes are in removal list",
			rule:     ".remove1.remove2 { color: red; }",
			toRemove: []string{"remove1", "remove2"},
			expected: false,
		},
		{
			name:     "keeps rule without class selectors",
			rule:     "div { margin: 0; }",
			toRemove: []string{"remove"},
			expected: true,
		},
		{
			name:     "handles selector with pseudo-classes",
			rule:     ".keep:hover { color: red; }",
			toRemove: []string{"remove"},
			expected: true,
		},
		{
			name:     "handles simple selector",
			rule:     ".container { padding: 10px; }",
			toRemove: []string{"remove"},
			expected: true,
		},
		{
			name:     "keeps rule without brace",
			rule:     ".remove color: red;",
			toRemove: []string{"remove"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldKeepRule(tt.rule, toRemoveSet(tt.toRemove))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriter_RealWorldScenarios(t *testing.T) {
	t.Run("processes Bootstrap-like CSS", func(t *testing.T) {
		content := `.container {
  max-width: 1200px;
}
.row {
  display: flex;
}
.unused-class {
  color: red;
}
.col-md-6 {
  width: 50%;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"unused-class"}))

		assert.Contains(t, result, ".container")
		assert.Contains(t, result, ".row")
		assert.Contains(t, result, ".col-md-6")
		assert.NotContains(t, result, ".unused-class")
	})

	t.Run("processes CSS with media queries", func(t *testing.T) {
		content := `.mobile-only {
   display: block;
}
@media (max-width: 768px) {
   .hide-mobile {
     display: none;
   }
}
.desktop {
   display: block;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"hide-mobile"}))

		assert.Contains(t, result, ".mobile-only")
		assert.Contains(t, result, ".desktop")
		assert.NotContains(t, result, ".hide-mobile")
	})

	t.Run("bug: removes @media header but leaves content when no class selector", func(t *testing.T) {
		content := `@media (prefers-color-scheme: dark) {
  :root {
    --bulma-white-on-scheme-l: 100%;
  }
}`
		result := removeUnusedRules(content, toRemoveSet([]string{}))

		assert.Contains(t, result, "@media (prefers-color-scheme: dark)")
		assert.Contains(t, result, ":root")
		lines := strings.Split(strings.TrimSpace(result), "\n")
		assert.GreaterOrEqual(t, len(lines), 4)
	})

	t.Run("handles large CSS files efficiently", func(t *testing.T) {
		content := ""
		for i := range 1000 {
			if i%10 == 0 {
				content += fmt.Sprintf(".unused-%d { color: red; }\n", i)
			} else {
				content += fmt.Sprintf(".keep-%d { color: blue; }\n", i)
			}
		}

		toRemove := []string{}
		for i := 0; i < 1000; i += 10 {
			toRemove = append(toRemove, fmt.Sprintf("unused-%d", i))
		}

		result := removeUnusedRules(content, toRemoveSet(toRemove))

		for i := range 1000 {
			if i%10 == 0 {
				assert.NotContains(t, result, fmt.Sprintf(".unused-%d", i))
			}
		}
	})
}

func TestWriter_CommaSeparatedSelectors(t *testing.T) {
	t.Run("removes all classes when both comma-separated selectors are unused", func(t *testing.T) {
		content := `.modal-content,
.modal-card {
  overflow: auto;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"modal-content", "modal-card"}))

		assert.NotContains(t, result, ".modal-content")
		assert.NotContains(t, result, ".modal-card")
		assert.NotContains(t, result, "overflow")
	})

	t.Run("keeps rule with at least one used selector when comma-separated", func(t *testing.T) {
		content := `.modal-content,
.modal-card {
  overflow: auto;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"modal-content"}))

		assert.NotContains(t, result, ".modal-content")
		assert.Contains(t, result, ".modal-card")
		assert.Contains(t, result, "overflow")
	})
}

func TestSplitSelectorsRespectingParens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "splits simple comma-separated selectors",
			input:    ".class1, .class2, .class3",
			expected: []string{".class1", " .class2", " .class3"},
		},
		{
			name:     "preserves commas inside :not() parentheses",
			input:    ".navbar-item:not(.is-active, .is-selected), .other",
			expected: []string{".navbar-item:not(.is-active, .is-selected)", " .other"},
		},
		{
			name:     "preserves commas inside :is() parentheses",
			input:    ".button:is(.primary, .secondary), .link",
			expected: []string{".button:is(.primary, .secondary)", " .link"},
		},
		{
			name:     "preserves commas inside :where() parentheses",
			input:    ".element:where(.active, .hover), .default",
			expected: []string{".element:where(.active, .hover)", " .default"},
		},
		{
			name:     "preserves commas inside :has() parentheses",
			input:    ".container:has(> .child1, > .child2), .other",
			expected: []string{".container:has(> .child1, > .child2)", " .other"},
		},
		{
			name:     "handles nested parentheses",
			input:    ".a:not(.b:is(.c, .d)), .e",
			expected: []string{".a:not(.b:is(.c, .d))", " .e"},
		},
		{
			name:     "handles empty parentheses",
			input:    ".class(), .other",
			expected: []string{".class()", " .other"},
		},
		{
			name:     "handles multiple pseudo-functions with commas",
			input:    ".item:not(.a, .b):is(.c, .d), .other",
			expected: []string{".item:not(.a, .b):is(.c, .d)", " .other"},
		},
		{
			name:     "single selector without commas",
			input:    ".single",
			expected: []string{".single"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitSelectorsRespectingParens(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriter_PseudoFunctionSelectors(t *testing.T) {
	t.Run("bug: :not() with comma-separated selectors - removes one class", func(t *testing.T) {
		content := `.navbar-dropdown .navbar-item:not(.is-active, .is-selected) {
  background-color: #fff;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"is-selected"}))

		assert.Contains(t, result, ".navbar-item:not(.is-active)")
		assert.NotContains(t, result, ".is-selected")
		assert.Contains(t, result, "background-color")
	})

	t.Run(":not() with all classes removed - keeps the rule but filters selector", func(t *testing.T) {
		content := `.navbar-item:not(.is-active, .is-selected) {
  color: red;
}
.other {
  color: blue;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"is-active", "is-selected"}))

		assert.NotContains(t, result, ".is-active")
		assert.NotContains(t, result, ".is-selected")
		assert.Contains(t, result, ".other")
	})

	t.Run(":is() with multiple classes - removes one", func(t *testing.T) {
		content := `.button:is(.primary, .secondary) {
  padding: 10px;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"secondary"}))

		assert.Contains(t, result, ".button:is(.primary)")
		assert.NotContains(t, result, ".secondary")
	})

	t.Run(":where() with multiple classes - removes one", func(t *testing.T) {
		content := `.element:where(.active, .inactive) {
  opacity: 1;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"inactive"}))

		assert.Contains(t, result, ".element:where(.active)")
		assert.NotContains(t, result, ".inactive")
	})

	t.Run("complex selector with multiple pseudo-functions", func(t *testing.T) {
		content := `.item:not(.disabled, .archived):is(.visible, .hidden) {
  display: block;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"archived", "hidden"}))

		assert.Contains(t, result, ".item:not(.disabled)")
		assert.Contains(t, result, ":is(.visible)")
		assert.NotContains(t, result, ".hidden")
		assert.NotContains(t, result, ".archived")
	})

	t.Run("multiple comma-separated selectors with pseudo-functions", func(t *testing.T) {
		content := `.btn:not(.danger, .warning),
.button:is(.primary, .secondary) {
  cursor: pointer;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"warning", "secondary"}))

		assert.Contains(t, result, ".btn:not(.danger)")
		assert.Contains(t, result, ".button:is(.primary)")
		assert.NotContains(t, result, ".warning")
		assert.NotContains(t, result, ".secondary")
		assert.Contains(t, result, "cursor: pointer")
	})

	t.Run("selector removed entirely when all classes in :not() are removed", func(t *testing.T) {
		content := `.item:not(.a, .b), .other {
  color: red;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"a", "b"}))

		assert.NotContains(t, result, ".item:not")
		assert.Contains(t, result, ".other")
		assert.Contains(t, result, "color: red")
	})
}

func TestWriter_MultilineSelectorWithPseudoFunctions(t *testing.T) {
	t.Run("multiline selector with :not() - no character duplication", func(t *testing.T) {
		content := `.navbar-dropdown 
.navbar-item:not(.is-active, .is-selected) {
  background-color: #fff;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"is-selected"}))

		assert.Contains(t, result, ".navbar-item:not(.is-active)")
		assert.NotContains(t, result, ".is-selected")
		assert.NotContains(t, result, "..")
		assert.NotContains(t, result, "..navbar")
		assert.Contains(t, result, "background-color")
	})

	t.Run("multiline selector with :is() - preserves formatting", func(t *testing.T) {
		content := `.button 
.link:is(.primary, .secondary) {
  padding: 10px;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"secondary"}))

		assert.Contains(t, result, ".link:is(.primary)")
		assert.NotContains(t, result, ".secondary")
		assert.NotContains(t, result, "..")
		assert.Contains(t, result, "padding")
	})

	t.Run("multiline selector with multiple pseudo-functions", func(t *testing.T) {
		content := `.container 
.item:not(.disabled, .archived):is(.visible, .hidden) {
  display: block;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"archived", "hidden"}))

		assert.Contains(t, result, ".item:not(.disabled)")
		assert.Contains(t, result, ":is(.visible)")
		assert.NotContains(t, result, ".archived")
		assert.NotContains(t, result, ".hidden")
		assert.NotContains(t, result, "..")
		assert.Contains(t, result, "display: block")
	})

	t.Run("multiline selector with trailing whitespace", func(t *testing.T) {
		content := `.navbar-dropdown 
.navbar-item:not(.is-active, .is-selected) {
  background-color: #fff;
}
`
		result := removeUnusedRules(content, toRemoveSet([]string{"is-selected"}))

		lines := strings.Split(strings.TrimSpace(result), "\n")
		for _, line := range lines {
			assert.NotContains(t, line, "..")
		}
		assert.Contains(t, result, ".navbar-item:not(.is-active)")
	})

	t.Run("single-line selector still works correctly", func(t *testing.T) {
		content := `.navbar-dropdown .navbar-item:not(.is-active, .is-selected) {
  background-color: #fff;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"is-selected"}))

		assert.Contains(t, result, ".navbar-item:not(.is-active)")
		assert.NotContains(t, result, ".is-selected")
		assert.NotContains(t, result, "..")
		assert.Contains(t, result, "background-color")
	})
}

func TestWriter_EdgeCases(t *testing.T) {
	t.Run("handles CSS with no rules", func(t *testing.T) {
		content := `/* Just a comment */
`
		result := removeUnusedRules(content, toRemoveSet([]string{"unused"}))

		assert.Contains(t, result, "/* Just a comment */")
	})

	t.Run("handles malformed CSS gracefully", func(t *testing.T) {
		content := `.incomplete {
  color: red;
.next {
  color: blue;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"incomplete"}))

		assert.NotEmpty(t, result)
	})

	t.Run("handles empty content", func(t *testing.T) {
		result := removeUnusedRules("", toRemoveSet([]string{"test"}))

		assert.Equal(t, "", result)
	})

	t.Run("handles class names with special characters", func(t *testing.T) {
		content := `.btn-primary {
  color: blue;
}
.form-control-lg {
  width: 100%;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"form-control-lg"}))

		assert.Contains(t, result, ".btn-primary")
		assert.NotContains(t, result, ".form-control-lg")
	})
}

func TestWriter_WriteAndBackup(t *testing.T) {
	t.Run("backup file is identical to original", func(t *testing.T) {
		tmpdir := t.TempDir()
		outputPath := filepath.Join(tmpdir, "output.css")

		originalContent := `.test {
  color: red;
}
.remove {
  color: blue;
}`
		err := Write(originalContent, []string{"remove"}, outputPath, true)

		require.NoError(t, err)

		backupPath := outputPath + ".bak"
		backupContent, err := os.ReadFile(backupPath)
		require.NoError(t, err)
		assert.Equal(t, originalContent, string(backupContent))
	})

	t.Run("output file has modifications", func(t *testing.T) {
		tmpdir := t.TempDir()
		outputPath := filepath.Join(tmpdir, "output.css")

		originalContent := `.keep {
  color: blue;
}
.remove {
  color: red;
}`
		err := Write(originalContent, []string{"remove"}, outputPath, true)

		require.NoError(t, err)

		outputContent, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		assert.NotContains(t, string(outputContent), ".remove")
		assert.Contains(t, string(outputContent), ".keep")
	})

	t.Run("skip backup when flag is false", func(t *testing.T) {
		tmpdir := t.TempDir()
		outputPath := filepath.Join(tmpdir, "output.css")
		backupPath := outputPath + ".bak"

		err := Write(".test { color: red; }", []string{}, outputPath, false)

		require.NoError(t, err)

		_, err = os.Stat(backupPath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestWriter_BlankLineRemoval(t *testing.T) {
	t.Run("removes blank line after removed rule", func(t *testing.T) {
		content := `.remove {
  color: red;
}

.keep {
  color: blue;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"remove"}))

		assert.NotContains(t, result, ".remove")
		assert.Contains(t, result, ".keep")
		lines := strings.Split(result, "\n")
		assert.True(t, len(lines) > 0)
		firstNonEmpty := ""
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				firstNonEmpty = line
				break
			}
		}
		assert.Contains(t, firstNonEmpty, ".keep")
	})

	t.Run("removes multiple blank lines after removed rule", func(t *testing.T) {
		content := `.remove {
	  color: red;
	}

	.keep {
	  color: blue;
	}`
		result := removeUnusedRules(content, toRemoveSet([]string{"remove"}))

		assert.NotContains(t, result, ".remove")
		assert.Contains(t, result, ".keep")
		lines := strings.Split(result, "\n")
		blankCount := 0
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				blankCount++
			} else {
				break
			}
		}
		assert.Equal(t, 0, blankCount)
	})

	t.Run("preserves blank lines between kept rules", func(t *testing.T) {
		content := `.keep1 {
  color: red;
}

.keep2 {
  color: blue;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{}))

		assert.Contains(t, result, ".keep1")
		assert.Contains(t, result, ".keep2")
		assert.Contains(t, result, ".keep1 {\n  color: red;\n}\n\n.keep2")
	})

	t.Run("removes blank lines between removed rules", func(t *testing.T) {
		content := `.remove1 {
  color: red;
}

.remove2 {
  color: blue;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"remove1", "remove2"}))

		assert.NotContains(t, result, ".remove1")
		assert.NotContains(t, result, ".remove2")
		assert.Equal(t, "", strings.TrimSpace(result))
	})

	t.Run("removes blank line after removed rule with kept rules on both sides", func(t *testing.T) {
		content := `.keep1 {
  color: red;
}

.remove {
  color: yellow;
}

.keep2 {
  color: blue;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"remove"}))

		assert.Contains(t, result, ".keep1")
		assert.Contains(t, result, ".keep2")
		assert.NotContains(t, result, ".remove")

		lines := strings.Split(result, "\n")
		keep1Found := false
		keep2Found := false

		for i, line := range lines {
			if strings.Contains(line, ".keep1") {
				keep1Found = true
			}
			if strings.Contains(line, ".keep2") {
				keep2Found = true
				for j := i - 1; j >= 0; j-- {
					if strings.Contains(lines[j], ".remove") {
						t.Fatal("Found .remove between .keep1 and .keep2")
					}
					if strings.TrimSpace(lines[j]) != "" {
						break
					}
				}
			}
		}

		assert.True(t, keep1Found, ".keep1 should be in result")
		assert.True(t, keep2Found, ".keep2 should be in result")
	})

	t.Run("handles removed rule with no trailing blank line", func(t *testing.T) {
		content := `.remove {
  color: red;
}
.keep {
  color: blue;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"remove"}))

		assert.NotContains(t, result, ".remove")
		assert.Contains(t, result, ".keep")
		assert.NotEmpty(t, result)
	})

	t.Run("handles comments and blank lines correctly", func(t *testing.T) {
		content := `/* Comment */

.remove {
  color: red;
}

.keep {
  color: blue;
}`
		result := removeUnusedRules(content, toRemoveSet([]string{"remove"}))

		assert.Contains(t, result, "/* Comment */")
		assert.Contains(t, result, ".keep")
		assert.NotContains(t, result, ".remove")
	})
}

func BenchmarkRemoveUnusedRules(b *testing.B) {
	content := `.container {
  max-width: 1200px;
}
.row {
  display: flex;
}
.unused-class {
  color: red;
}
.col-md-6 {
  width: 50%;
}
.btn {
  padding: 5px;
}
.btn-primary {
  background: blue;
}
.btn-secondary {
  background: gray;
}
.modal-content {
  overflow: hidden;
}
.modal-header {
  border-bottom: 1px solid #ccc;
}
.navbar-dropdown .navbar-item:not(.is-active, .is-selected) {
  background-color: #fff;
}
@media (max-width: 768px) {
  .hide-mobile {
    display: none;
  }
}
.desktop {
  display: block;
}`

	toRemove := []string{"unused-class", "modal-content", "modal-header", "btn-secondary", "hide-mobile"}

	b.ResetTimer()
	for b.Loop() {
		removeUnusedRules(content, toRemoveSet(toRemove))
	}
}
