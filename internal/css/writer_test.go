package css

import (
	"os"
	"path/filepath"
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

	t.Run("handles large CSS files efficiently", func(t *testing.T) {
		// Create a large CSS content
		content := ""
		for i := 0; i < 1000; i++ {
			if i%10 == 0 {
				content += ".unused-" + string(rune(i)) + " { color: red; }\n"
			} else {
				content += ".keep-" + string(rune(i)) + " { color: blue; }\n"
			}
		}

		toRemove := []string{}
		for i := 0; i < 1000; i += 10 {
			toRemove = append(toRemove, "unused-"+string(rune(i)))
		}

		writer := NewWriter(content, toRemove)
		result := writer.removeUnusedRules()

		for i := 0; i < 1000; i++ {
			if i%10 == 0 {
				assert.NotContains(t, result, ".unused-"+string(rune(i)))
			}
		}
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
