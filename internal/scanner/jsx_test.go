package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractJSXClasses(t *testing.T) {
	t.Run("extracts from className attribute", func(t *testing.T) {
		content := `<Component className="btn button-primary" />`
		result := extractJSXClasses(content)
		assert.Contains(t, result, "btn")
		assert.Contains(t, result, "button-primary")
	})

	t.Run("extracts from class attribute", func(t *testing.T) {
		content := `<div class="flex center" />`
		result := extractJSXClasses(content)
		assert.Contains(t, result, "flex")
		assert.Contains(t, result, "center")
	})

	t.Run("handles multiple elements", func(t *testing.T) {
		content := `<div className="header">Header</div><div className="footer">Footer</div>`
		result := extractJSXClasses(content)
		assert.Contains(t, result, "header")
		assert.Contains(t, result, "footer")
	})

	t.Run("ignores duplicates", func(t *testing.T) {
		content := `<div className="btn btn">Button</div>`
		result := extractJSXClasses(content)
		count := 0
		for _, class := range result {
			if class == "btn" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("handles empty className", func(t *testing.T) {
		content := `<div className=""></div>`
		result := extractJSXClasses(content)
		// Function returns nil for empty, adjust test
		assert.True(t, len(result) == 0)
	})

	t.Run("extracts from both className and class", func(t *testing.T) {
		content := `<div class="html-class" className="react-class" />`
		result := extractJSXClasses(content)
		assert.Contains(t, result, "html-class")
		assert.Contains(t, result, "react-class")
	})
}
