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
		writer := NewWriter(content, []string{"remove"})
		err = writer.Write(tmpfile.Name(), false)

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
		writer := NewWriter(content, []string{})
		err = writer.Write(tmpfile.Name(), true)

		require.NoError(t, err)

		// Check backup file exists and has original content
		backup, err := os.ReadFile(tmpfile.Name() + ".bak")
		require.NoError(t, err)
		assert.Equal(t, content, string(backup))
	})

	t.Run("handles file write errors gracefully", func(t *testing.T) {
		content := ".test { color: red; }"
		writer := NewWriter(content, []string{})
		err := writer.Write("/invalid/nonexistent/path/file.css", false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write output file")
	})

	t.Run("handles backup creation errors gracefully", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "css*.css")
		require.NoError(t, err)
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())

		content := ".test { color: red; }"
		writer := NewWriter(content, []string{})
		// Try to create backup in non-existent directory
		err = writer.Write("/invalid/path/file.css", true)

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
			shouldRemove: nil, // Rule is kept if any class is kept
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
			writer := NewWriter(tt.content, tt.toRemove)
			result := writer.removeUnusedRules()

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
			name:     "keeps rule with ignore comment",
			rule:     ".remove { /* css-trimmer-ignore */ color: red; }",
			toRemove: []string{"remove"},
			expected: true,
		},
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
			expected: true, // container is not in removal list
		},
		{
			name:     "keeps rule without brace",
			rule:     ".remove color: red;",
			toRemove: []string{"remove"},
			expected: true, // No brace, so returned as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewWriter("", tt.toRemove)
			result := writer.shouldKeepRule(tt.rule)
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
		writer := NewWriter(content, []string{"unused-class"})
		result := writer.removeUnusedRules()

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
		writer := NewWriter(content, []string{"hide-mobile"})
		result := writer.removeUnusedRules()

		assert.Contains(t, result, ".mobile-only")
		assert.Contains(t, result, ".desktop")
		assert.NotContains(t, result, ".hide-mobile")
	})

	t.Run("bug: removes @media header but leaves content when no class selector", func(t *testing.T) {
		// This is the bug from the report: @media block with NON-class selectors are treated as rules
		// When we encounter "@media (prefers-color-scheme: dark) {" on line 1:
		// - inRule = true, ruleBuffer = ["@media ..."]
		// OLD BUG: When we get to ":root {" on line 2:
		//   - It ALSO contains {, so we'd reset inRule and overwrite ruleBuffer!
		//   - This causes the @media line to be lost
		// FIXED: We should stay in the same rule and count braces properly
		content := `@media (prefers-color-scheme: dark) {
  :root {
    --bulma-white-on-scheme-l: 100%;
  }
}`
		writer := NewWriter(content, []string{})
		result := writer.removeUnusedRules()

		// Expected: The entire block should be preserved as-is
		// The @media line and all content should remain
		assert.Contains(t, result, "@media (prefers-color-scheme: dark)")
		assert.Contains(t, result, ":root")
		// Verify the structure is intact (not orphaned content)
		lines := strings.Split(strings.TrimSpace(result), "\n")
		assert.GreaterOrEqual(t, len(lines), 4) // Should have all 4+ lines
	})

	t.Run("handles large CSS files efficiently", func(t *testing.T) {
		// Create a large CSS content with proper class names (not using Unicode conversion which breaks CSS)
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

		writer := NewWriter(content, toRemove)
		result := writer.removeUnusedRules()

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
		writer := NewWriter(content, []string{"modal-content", "modal-card"})
		result := writer.removeUnusedRules()

		// Bug: The rule should be completely removed, not left with orphaned selectors
		assert.NotContains(t, result, ".modal-content")
		assert.NotContains(t, result, ".modal-card")
		assert.NotContains(t, result, "overflow")
	})

	t.Run("keeps rule with at least one used selector when comma-separated", func(t *testing.T) {
		content := `.modal-content,
.modal-card {
  overflow: auto;
}`
		writer := NewWriter(content, []string{"modal-content"})
		result := writer.removeUnusedRules()

		// Should keep the rule because .modal-card is not in removal list
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
		// This is the exact bug from the report
		content := `.navbar-dropdown .navbar-item:not(.is-active, .is-selected) {
  background-color: #fff;
}`
		writer := NewWriter(content, []string{"is-selected"})
		result := writer.removeUnusedRules()

		// The closing parenthesis should be preserved
		assert.Contains(t, result, ".navbar-item:not(.is-active)")
		assert.NotContains(t, result, ".is-selected")
		// Ensure the rule is properly formatted
		assert.Contains(t, result, "background-color")
	})

	t.Run(":not() with all classes removed - keeps the rule but filters selector", func(t *testing.T) {
		content := `.navbar-item:not(.is-active, .is-selected) {
  color: red;
}
.other {
  color: blue;
}`
		writer := NewWriter(content, []string{"is-active", "is-selected"})
		result := writer.removeUnusedRules()

		// The .navbar-item:not(...) rule should be kept because it has the .navbar-item selector
		// Since the :not() function has no remaining classes, behavior depends on implementation
		// but the parentheses must be balanced
		assert.NotContains(t, result, ".is-active")
		assert.NotContains(t, result, ".is-selected")
		assert.Contains(t, result, ".other")
	})

	t.Run(":is() with multiple classes - removes one", func(t *testing.T) {
		content := `.button:is(.primary, .secondary) {
  padding: 10px;
}`
		writer := NewWriter(content, []string{"secondary"})
		result := writer.removeUnusedRules()

		// Should have the first class in :is()
		assert.Contains(t, result, ".button:is(.primary)")
		assert.NotContains(t, result, ".secondary")
	})

	t.Run(":where() with multiple classes - removes one", func(t *testing.T) {
		content := `.element:where(.active, .inactive) {
  opacity: 1;
}`
		writer := NewWriter(content, []string{"inactive"})
		result := writer.removeUnusedRules()

		assert.Contains(t, result, ".element:where(.active)")
		assert.NotContains(t, result, ".inactive")
	})

	t.Run("complex selector with multiple pseudo-functions", func(t *testing.T) {
		content := `.item:not(.disabled, .archived):is(.visible, .hidden) {
  display: block;
}`
		writer := NewWriter(content, []string{"archived", "hidden"})
		result := writer.removeUnusedRules()

		// Both pseudo-functions should still be present with balanced parentheses
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
		writer := NewWriter(content, []string{"warning", "secondary"})
		result := writer.removeUnusedRules()

		// Both rules should be in the result with proper parentheses
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
		writer := NewWriter(content, []string{"a", "b"})
		result := writer.removeUnusedRules()

		// The first selector should be removed, but .other should remain
		// The rule should still exist because of .other
		assert.NotContains(t, result, ".item:not")
		assert.Contains(t, result, ".other")
		assert.Contains(t, result, "color: red")
	})
}

func TestWriter_EdgeCases(t *testing.T) {
	t.Run("handles CSS with no rules", func(t *testing.T) {
		content := `/* Just a comment */
`
		writer := NewWriter(content, []string{"unused"})
		result := writer.removeUnusedRules()

		assert.Contains(t, result, "/* Just a comment */")
	})

	t.Run("handles malformed CSS gracefully", func(t *testing.T) {
		content := `.incomplete {
  color: red;
.next {
  color: blue;
}`
		writer := NewWriter(content, []string{"incomplete"})
		result := writer.removeUnusedRules()

		// Should handle gracefully without panicking
		assert.NotEmpty(t, result)
	})

	t.Run("handles CSS with multiple ignore comments", func(t *testing.T) {
		content := `.remove1 {
  /* css-trimmer-ignore */
  color: red;
}
.remove2 { /* css-trimmer-ignore */ color: blue; }`
		writer := NewWriter(content, []string{"remove1", "remove2"})
		result := writer.removeUnusedRules()

		assert.Contains(t, result, ".remove1")
		assert.Contains(t, result, ".remove2")
	})

	t.Run("handles empty content", func(t *testing.T) {
		writer := NewWriter("", []string{"test"})
		result := writer.removeUnusedRules()

		assert.Equal(t, "", result)
	})

	t.Run("handles class names with special characters", func(t *testing.T) {
		content := `.btn-primary {
  color: blue;
}
.form-control-lg {
  width: 100%;
}`
		writer := NewWriter(content, []string{"form-control-lg"})
		result := writer.removeUnusedRules()

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
		writer := NewWriter(originalContent, []string{"remove"})
		err := writer.Write(outputPath, true)

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
		writer := NewWriter(originalContent, []string{"remove"})
		err := writer.Write(outputPath, true)

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

		content := ".test { color: red; }"
		writer := NewWriter(content, []string{})
		err := writer.Write(outputPath, false)

		require.NoError(t, err)

		_, err = os.Stat(backupPath)
		assert.True(t, os.IsNotExist(err))
	})
}
