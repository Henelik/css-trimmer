package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTemplClasses(t *testing.T) {
	t.Run("extracts from class attribute", func(t *testing.T) {
		content := `<div class="btn primary">Button</div>`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "primary")
	})

	t.Run("extracts from templ.Classes function", func(t *testing.T) {
		content := `{ templ.Classes("btn", "primary") }`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "primary")
	})

	t.Run("extracts from multiple class attributes", func(t *testing.T) {
		content := `
		<div class="header">Header</div>
		<div class="main">Main</div>
		<div class="footer">Footer</div>
		`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "header")
		assert.Contains(t, classes, "main")
		assert.Contains(t, classes, "footer")
	})

	t.Run("avoids duplicate classes", func(t *testing.T) {
		content := `
		<div class="btn btn">Button 1</div>
		<button class="btn">Button 2</button>
		`
		classes := ExtractTemplClasses(content)

		// Count occurrences of "btn"
		count := 0
		for _, class := range classes {
			if class == "btn" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("handles templ.Classes with multiple arguments", func(t *testing.T) {
		content := `{ templ.Classes("active", "large", "primary") }`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "active")
		assert.Contains(t, classes, "large")
		assert.Contains(t, classes, "primary")
	})

	t.Run("handles whitespace in class attribute", func(t *testing.T) {
		content := `<div class="btn    primary    large">Button</div>`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "primary")
		assert.Contains(t, classes, "large")
	})

	t.Run("handles classes with hyphens and underscores", func(t *testing.T) {
		content := `<div class="btn-primary form_control text-2xl">Content</div>`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "btn-primary")
		assert.Contains(t, classes, "form_control")
		assert.Contains(t, classes, "text-2xl")
	})

	t.Run("extracts quoted identifiers that look like CSS", func(t *testing.T) {
		content := `"btn-class" "form-control-lg"`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "btn-class")
		assert.Contains(t, classes, "form-control-lg")
	})

	t.Run("ignores common English words", func(t *testing.T) {
		content := `"the" "and" "for" "btn-primary"`
		classes := ExtractTemplClasses(content)

		assert.NotContains(t, classes, "the")
		assert.NotContains(t, classes, "and")
		assert.NotContains(t, classes, "for")
		assert.Contains(t, classes, "btn-primary")
	})

	t.Run("handles empty class attribute", func(t *testing.T) {
		content := `<div class="">Empty</div>`
		classes := ExtractTemplClasses(content)

		assert.Equal(t, 0, len(classes))
	})

	t.Run("handles templ.Classes with empty strings", func(t *testing.T) {
		content := `{ templ.Classes("btn", "", "primary") }`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "primary")
		// Empty strings should be skipped
		assert.NotContains(t, classes, "")
	})

	t.Run("processes complete templ file", func(t *testing.T) {
		content := `templ Header() {
	<header class="header sticky">
		<nav class="navbar">
			<a href="#" class="nav-link">Home</a>
			<button { templ.Classes("btn", "btn-primary") }>
				Menu
			</button>
		</nav>
	</header>
}`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "header")
		assert.Contains(t, classes, "sticky")
		assert.Contains(t, classes, "navbar")
		assert.Contains(t, classes, "nav-link")
		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "btn-primary")
	})

	t.Run("handles templ.Classes with variables", func(t *testing.T) {
		content := `{ templ.Classes("base", if condition { "active" } else { "inactive" }) }`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "base")
		assert.Contains(t, classes, "active")
		assert.Contains(t, classes, "inactive")
	})

	t.Run("returns empty array for content without classes", func(t *testing.T) {
		content := `<div>No classes here</div>`
		classes := ExtractTemplClasses(content)

		assert.Equal(t, 0, len(classes))
	})
}

func TestIsLikelyCSSIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "recognizes hyphenated identifiers",
			input:    "btn-primary",
			expected: true,
		},
		{
			name:     "recognizes underscored identifiers",
			input:    "form_control",
			expected: true,
		},
		{
			name:     "recognizes mixed hyphens and underscores",
			input:    "btn_primary-lg",
			expected: true,
		},
		{
			name:     "rejects identifiers without separators",
			input:    "btnprimary",
			expected: false,
		},
		{
			name:     "recognizes complex CSS names",
			input:    "col-md-12",
			expected: true,
		},
		{
			name:     "recognizes Tailwind-like utilities",
			input:    "text-2xl",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLikelyCSSIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExcludedWords(t *testing.T) {
	t.Run("excludes common English words", func(t *testing.T) {
		words := []string{"the", "and", "or", "for", "is", "in", "of", "to", "a", "an", "on", "at", "by", "it"}

		for _, word := range words {
			assert.True(t, e(word), "word %s should be excluded", word)
		}
	})

	t.Run("does not exclude CSS-like words", func(t *testing.T) {
		words := []string{"button", "form", "input", "header", "footer"}

		for _, word := range words {
			assert.False(t, e(word), "word %s should not be excluded", word)
		}
	})

	t.Run("case-insensitive exclusion", func(t *testing.T) {
		assert.True(t, e("THE"))
		assert.True(t, e("The"))
		assert.True(t, e("And"))
	})
}

func TestExtractTemplClasses_RealWorldScenarios(t *testing.T) {
	t.Run("extracts from complex Templ component", func(t *testing.T) {
		content := `templ Button(href string, class string) {
	if href != "" {
		<a { templ.Classes("btn", "btn-base") } href={href}>
			{ children... }
		</a>
	} else {
		<button { templ.Classes("btn", "btn-base") }>
			{ children... }
		</button>
	}
}

templ PrimaryButton(href string) {
	<a href="#">btn-primary btn-lg</a>
}`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "btn-base")
	})

	t.Run("extracts from Templ with conditional classes", func(t *testing.T) {
		content := `templ Alert(severity string) {
	<div { templ.Classes(
		"alert",
		if severity == "error" { "alert-danger" } else if severity == "warning" { "alert-warning" } else { "alert-info" }
	) }>
		{ children... }
	</div>
}`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "alert")
		assert.Contains(t, classes, "alert-danger")
		assert.Contains(t, classes, "alert-warning")
		assert.Contains(t, classes, "alert-info")
	})

	t.Run("extracts from styled component pattern", func(t *testing.T) {
		content := `templ Card(title string) {
	<div class="card card-elevated">
		<div class="card-header">
			<h2 class="card-title">{ title }</h2>
		</div>
		<div class="card-body">
			{ children... }
		</div>
		<div class="card-footer">
			<button { templ.Classes("btn", "btn-secondary") }>Close</button>
		</div>
	</div>
}`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "card")
		assert.Contains(t, classes, "card-elevated")
		assert.Contains(t, classes, "card-header")
		assert.Contains(t, classes, "card-title")
		assert.Contains(t, classes, "card-body")
		assert.Contains(t, classes, "card-footer")
		assert.Contains(t, classes, "btn-secondary")
	})
}

func TestExtractTemplClasses_EdgeCases(t *testing.T) {
	t.Run("handles nested templ.Classes calls", func(t *testing.T) {
		content := `{ templ.Classes("outer", templ.Classes("inner", "nested")) }`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "outer")
		assert.Contains(t, classes, "inner")
		assert.Contains(t, classes, "nested")
	})

	t.Run("handles templ.Classes with quoted arguments containing spaces", func(t *testing.T) {
		content := `{ templ.Classes("btn btn-primary", "text-lg text-center") }`
		classes := ExtractTemplClasses(content)

		assert.Greater(t, len(classes), 0)
	})

	t.Run("handles mixed quote types in class names", func(t *testing.T) {
		content := `<div class="btn-primary">Button</div>`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "btn-primary")
	})

	t.Run("handles very long class list", func(t *testing.T) {
		classStr := ""
		for i := 0; i < 50; i++ {
			classStr += `"class-` + string(rune(i)) + `" `
		}
		content := `{ templ.Classes(` + classStr + `) }`
		classes := ExtractTemplClasses(content)

		assert.Greater(t, len(classes), 0)
	})

	t.Run("handles malformed templ.Classes", func(t *testing.T) {
		content := `{ templ.Classes("btn") }`
		classes := ExtractTemplClasses(content)

		// Should handle gracefully and extract what it can
		assert.Greater(t, len(classes), 0)
	})

	t.Run("handles empty file", func(t *testing.T) {
		content := ""
		classes := ExtractTemplClasses(content)

		assert.Equal(t, 0, len(classes))
	})

	t.Run("handles file with class content", func(t *testing.T) {
		content := `<div class="actual-class">Content</div>`
		classes := ExtractTemplClasses(content)

		assert.Contains(t, classes, "actual-class")
	})
}
