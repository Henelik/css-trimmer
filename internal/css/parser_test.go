package css

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserParse(t *testing.T) {
	t.Run("parses single class selector", func(t *testing.T) {
		content := `.test {
  color: red;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		require.NotNil(t, inventory)
		assert.Contains(t, inventory, "test")
		assert.Equal(t, 1, inventory["test"].StartLine)
	})

	t.Run("parses multiple class selectors", func(t *testing.T) {
		content := `.header {
  background: blue;
}
.footer {
  background: gray;
}
.button {
  padding: 10px;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Equal(t, 3, len(inventory))
		assert.Contains(t, inventory, "header")
		assert.Contains(t, inventory, "footer")
		assert.Contains(t, inventory, "button")
	})

	t.Run("parses multiple classes in single selector", func(t *testing.T) {
		content := `.header.active.sticky {
  position: fixed;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Equal(t, 3, len(inventory))
		assert.Contains(t, inventory, "header")
		assert.Contains(t, inventory, "active")
		assert.Contains(t, inventory, "sticky")
	})

	t.Run("parses complex selectors", func(t *testing.T) {
		content := `.button:hover {
  background: red;
}
.header > .nav {
  display: flex;
}
.form-group + .form-group {
  margin-top: 10px;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Contains(t, inventory, "button")
		assert.Contains(t, inventory, "header")
		assert.Contains(t, inventory, "nav")
		assert.Contains(t, inventory, "form-group")
	})

	t.Run("skips empty lines and comments", func(t *testing.T) {
		content := `/* This is a comment */

.test {
  color: red;
}

/* Another comment */`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Equal(t, 1, len(inventory))
		assert.Contains(t, inventory, "test")
	})

	t.Run("handles classes with hyphens and underscores", func(t *testing.T) {
		content := `.btn-primary {
  color: blue;
}
.form_group {
  margin: 5px;
}
.label-primary_active {
  font-weight: bold;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Contains(t, inventory, "btn-primary")
		assert.Contains(t, inventory, "form_group")
		assert.Contains(t, inventory, "label-primary_active")
	})

	t.Run("handles classes with numbers", func(t *testing.T) {
		content := `.col-md-12 {
  width: 100%;
}
.text-size-2xl {
  font-size: 24px;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Contains(t, inventory, "col-md-12")
		assert.Contains(t, inventory, "text-size-2xl")
	})

	t.Run("tracks line numbers correctly", func(t *testing.T) {
		content := `.first {
  color: red;
}
.second {
  color: blue;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Equal(t, 1, inventory["first"].StartLine)
		assert.Equal(t, 4, inventory["second"].StartLine)
	})

	t.Run("returns empty inventory for CSS without classes", func(t *testing.T) {
		content := `* {
  margin: 0;
  padding: 0;
}
div {
  display: block;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Equal(t, 0, len(inventory))
	})

	t.Run("handles multiline selectors", func(t *testing.T) {
		content := `.container,
.wrapper {
	 max-width: 1200px;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		// Parser only extracts from the line with {, so only wrapper on second line
		assert.Contains(t, inventory, "wrapper")
	})

	t.Run("ignores duplicate class definitions", func(t *testing.T) {
		content := `.test {
  color: red;
}
.test {
  color: blue;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Equal(t, 1, len(inventory))
		// Should keep first definition
		assert.Equal(t, 1, inventory["test"].StartLine)
	})

	t.Run("handles empty CSS", func(t *testing.T) {
		parser := NewParser("")
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Equal(t, 0, len(inventory))
	})

	t.Run("parses CSS with at-rules", func(t *testing.T) {
		content := `@media (max-width: 768px) {
  .mobile-only {
    display: block;
  }
}
.desktop {
  display: none;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Contains(t, inventory, "mobile-only")
		assert.Contains(t, inventory, "desktop")
	})
}

func TestExtractClassesFromSelector(t *testing.T) {
	tests := []struct {
		name            string
		selector        string
		expectedClasses []string
	}{
		{
			name:            "single class",
			selector:        ".test",
			expectedClasses: []string{"test"},
		},
		{
			name:            "multiple classes",
			selector:        ".header.sticky.active",
			expectedClasses: []string{"header", "sticky", "active"},
		},
		{
			name:            "class with descendant combinator",
			selector:        ".container > .item",
			expectedClasses: []string{"container", "item"},
		},
		{
			name:            "class with adjacent combinator",
			selector:        ".header + .nav",
			expectedClasses: []string{"header", "nav"},
		},
		{
			name:            "class with pseudo-class",
			selector:        ".button:hover",
			expectedClasses: []string{"button"},
		},
		{
			name:            "class with pseudo-element",
			selector:        ".text::before",
			expectedClasses: []string{"text"},
		},
		{
			name:            "class with attribute selector",
			selector:        `.form[type="text"]`,
			expectedClasses: []string{"form"},
		},
		{
			name:            "hyphenated class names",
			selector:        ".btn-primary.btn-lg",
			expectedClasses: []string{"btn-primary", "btn-lg"},
		},
		{
			name:            "underscored class names",
			selector:        ".form_control.form_input",
			expectedClasses: []string{"form_control", "form_input"},
		},
		{
			name:            "numbered class names",
			selector:        ".col-md-12.text-2xl",
			expectedClasses: []string{"col-md-12", "text-2xl"},
		},
		{
			name:            "no duplicates in same selector",
			selector:        ".item.item.item",
			expectedClasses: []string{"item"},
		},
		{
			name:            "complex selector with multiple operators",
			selector:        ".container > .row.active ~ .column",
			expectedClasses: []string{"container", "row", "active", "column"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser("")
			result := parser.extractClassesFromSelector(tt.selector)
			assert.Equal(t, tt.expectedClasses, result)
		})
	}
}

func TestClassInventoryAllClasses(t *testing.T) {
	t.Run("returns all class names", func(t *testing.T) {
		inventory := ClassInventory{
			"button": ClassInfo{StartLine: 1, EndLine: 3},
			"header": ClassInfo{StartLine: 5, EndLine: 7},
			"footer": ClassInfo{StartLine: 9, EndLine: 11},
		}

		classes := inventory.AllClasses()

		require.NotNil(t, classes)
		assert.Equal(t, 3, len(classes))
		assert.Contains(t, classes, "button")
		assert.Contains(t, classes, "header")
		assert.Contains(t, classes, "footer")
	})

	t.Run("returns empty slice for empty inventory", func(t *testing.T) {
		inventory := ClassInventory{}

		classes := inventory.AllClasses()

		// Empty inventory may return nil slice
		assert.True(t, classes == nil || len(classes) == 0)
	})

	t.Run("returns unique classes", func(t *testing.T) {
		inventory := ClassInventory{
			"test1": ClassInfo{StartLine: 1, EndLine: 1},
			"test2": ClassInfo{StartLine: 2, EndLine: 2},
		}

		classes := inventory.AllClasses()

		// Convert to map to check uniqueness
		classMap := make(map[string]bool)
		for _, class := range classes {
			if classMap[class] {
				t.Fatalf("Duplicate class found: %s", class)
			}
			classMap[class] = true
		}
	})
}

func TestParser_RealWorldScenarios(t *testing.T) {
	t.Run("parses Bootstrap-like CSS", func(t *testing.T) {
		content := `.container {
  max-width: 1200px;
}
.row {
  display: flex;
}
.col-md-6 {
  width: 50%;
}
.btn.btn-primary {
  background: blue;
}
.text-center {
  text-align: center;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Contains(t, inventory, "container")
		assert.Contains(t, inventory, "row")
		assert.Contains(t, inventory, "col-md-6")
		assert.Contains(t, inventory, "btn")
		assert.Contains(t, inventory, "btn-primary")
		assert.Contains(t, inventory, "text-center")
	})

	t.Run("parses Tailwind-like utility classes", func(t *testing.T) {
		content := `.m-0 {
  margin: 0;
}
.p-4 {
  padding: 1rem;
}
.text-lg {
  font-size: 1.125rem;
}
.bg-white {
  background-color: white;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Equal(t, 4, len(inventory))
	})

	t.Run("parses CSS with comments and whitespace", func(t *testing.T) {
		content := `/* Header Styles */
.header {
  background: #f0f0f0;
  /* Header background */
  padding: 20px;
}

/* Footer Styles */
.footer {
  background: #333;
  color: white;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Contains(t, inventory, "header")
		assert.Contains(t, inventory, "footer")
	})
}

func TestParser_EdgeCases(t *testing.T) {
	t.Run("handles class names with uppercase letters", func(t *testing.T) {
		content := `.MyComponent {
  color: red;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Contains(t, inventory, "MyComponent")
	})

	t.Run("handles selectors with line breaks", func(t *testing.T) {
		content := `.test {
  color: red;
}
`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Contains(t, inventory, "test")
	})

	t.Run("handles CSS with no line breaks", func(t *testing.T) {
		content := ".test { color: red; }"
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Contains(t, inventory, "test")
	})

	t.Run("handles deeply nested selectors", func(t *testing.T) {
		content := `.a > .b > .c > .d {
  margin: 0;
}`
		parser := NewParser(content)
		inventory, err := parser.Parse()

		require.NoError(t, err)
		assert.Contains(t, inventory, "a")
		assert.Contains(t, inventory, "b")
		assert.Contains(t, inventory, "c")
		assert.Contains(t, inventory, "d")
	})
}
