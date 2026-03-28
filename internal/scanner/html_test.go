package scanner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractHTMLClasses(t *testing.T) {
	t.Run("extracts single class", func(t *testing.T) {
		html := `<div class="btn">Button</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "btn")
	})

	t.Run("extracts multiple classes from single element", func(t *testing.T) {
		html := `<div class="btn btn-primary large">Button</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "btn-primary")
		assert.Contains(t, classes, "large")
	})

	t.Run("extracts classes from multiple elements", func(t *testing.T) {
		html := `
		<header class="header">Header</header>
		<main class="main">Main</main>
		<footer class="footer">Footer</footer>
		`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "header")
		assert.Contains(t, classes, "main")
		assert.Contains(t, classes, "footer")
	})

	t.Run("avoids duplicate classes", func(t *testing.T) {
		html := `
		<div class="btn btn">Button 1</div>
		<button class="btn">Button 2</button>
		`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		// Count occurrences of "btn"
		count := 0
		for _, class := range classes {
			if class == "btn" {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("handles empty class attribute", func(t *testing.T) {
		html := `<div class="">Empty</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Equal(t, 0, len(classes))
	})

	t.Run("handles self-closing tags", func(t *testing.T) {
		html := `<img class="icon" src="icon.png" />`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "icon")
	})

	t.Run("ignores elements without class attribute", func(t *testing.T) {
		html := `
		<div id="main">Content</div>
		<span data-value="test">Span</span>
		<p class="paragraph">Paragraph</p>
		`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "paragraph")
		assert.Equal(t, 1, len(classes))
	})

	t.Run("handles whitespace in class attribute", func(t *testing.T) {
		html := `<div class="btn    primary    large">Button</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "primary")
		assert.Contains(t, classes, "large")
	})

	t.Run("handles newlines in class attribute", func(t *testing.T) {
		html := `<div class="btn
primary
large">Button</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Greater(t, len(classes), 0)
	})

	t.Run("handles classes with hyphens and underscores", func(t *testing.T) {
		html := `<div class="btn-primary form_control text-2xl">Content</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "btn-primary")
		assert.Contains(t, classes, "form_control")
		assert.Contains(t, classes, "text-2xl")
	})

	t.Run("processes complete HTML document", func(t *testing.T) {
		html := `<!DOCTYPE html>
		<html>
		<head>
			<title>Test</title>
		</head>
		<body class="app">
			<header class="header sticky">
				<nav class="navbar">
					<a href="#" class="nav-link active">Home</a>
				</nav>
			</header>
			<main class="container">
				<section class="hero">
					<button class="btn btn-primary">Start</button>
				</section>
			</main>
		</body>
		</html>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "app")
		assert.Contains(t, classes, "header")
		assert.Contains(t, classes, "sticky")
		assert.Contains(t, classes, "navbar")
		assert.Contains(t, classes, "nav-link")
		assert.Contains(t, classes, "active")
		assert.Contains(t, classes, "container")
		assert.Contains(t, classes, "hero")
		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "btn-primary")
	})

	t.Run("handles invalid HTML gracefully", func(t *testing.T) {
		html := `<div class="test"><not-closed>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		// Should handle gracefully even with invalid HTML
		require.NoError(t, err)
		assert.Greater(t, len(classes), 0)
	})

	t.Run("handles escaped quotes in attributes", func(t *testing.T) {
		html := `<div class="btn primary">Button</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "primary")
	})

	t.Run("returns empty array for empty HTML", func(t *testing.T) {
		html := ``
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Equal(t, 0, len(classes))
	})

	t.Run("handles case-sensitive class names", func(t *testing.T) {
		html := `<div class="Button PRIMARY btn">Content</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "Button")
		assert.Contains(t, classes, "PRIMARY")
		assert.Contains(t, classes, "btn")
	})
}

func TestExtractHTMLClasses_EdgeCases(t *testing.T) {
	t.Run("handles attributes in different order", func(t *testing.T) {
		html := `<div data-test="value" id="main" class="test-class" title="Test">Content</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "test-class")
	})

	t.Run("handles multiple whitespace types", func(t *testing.T) {
		html := `<div class="btn	primary  large">Button</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "primary")
		assert.Contains(t, classes, "large")
	})

	t.Run("handles long class list", func(t *testing.T) {
		classStr := ""
		for i := 0; i < 15; i++ {
			classStr += "class" + string(rune('0'+byte(i%10))) + " "
		}
		html := `<div class="` + classStr + `">Content</div>`

		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Greater(t, len(classes), 5)
	})

	t.Run("handles nested elements", func(t *testing.T) {
		html := `
		<div class="outer">
			<div class="inner">
				<span class="text">Content</span>
			</div>
		</div>
		`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "outer")
		assert.Contains(t, classes, "inner")
		assert.Contains(t, classes, "text")
	})

	t.Run("ignores class attribute in comments", func(t *testing.T) {
		html := `<!-- <div class="commented">This is commented</div> -->
		<div class="active">Active</div>`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "active")
	})
}

func TestExtractHTMLClasses_RealWorldScenarios(t *testing.T) {
	t.Run("extracts from Bootstrap HTML", func(t *testing.T) {
		html := `
		<nav class="navbar navbar-expand-lg navbar-light bg-light">
			<div class="container-fluid">
				<button class="navbar-toggler" type="button">
					<span class="navbar-toggler-icon"></span>
				</button>
				<div class="collapse navbar-collapse">
					<ul class="navbar-nav ms-auto">
						<li class="nav-item">
							<a class="nav-link active" href="#">Home</a>
						</li>
					</ul>
				</div>
			</div>
		</nav>
		`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "navbar")
		assert.Contains(t, classes, "navbar-expand-lg")
		assert.Contains(t, classes, "container-fluid")
		assert.Contains(t, classes, "nav-item")
	})

	t.Run("extracts from Tailwind HTML", func(t *testing.T) {
		html := `
		<div class="flex flex-col items-center justify-center min-h-screen bg-gradient-to-r from-blue-500 to-purple-600">
			<h1 class="text-4xl font-bold text-white mb-4">Welcome</h1>
			<button class="px-6 py-3 bg-white text-blue-500 font-semibold rounded-lg hover:bg-gray-100">
				Get Started
			</button>
		</div>
		`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "flex")
		assert.Contains(t, classes, "text-4xl")
		assert.Contains(t, classes, "font-bold")
		assert.Contains(t, classes, "px-6")
	})

	t.Run("extracts from complex form", func(t *testing.T) {
		html := `
		<form class="form-container">
			<div class="form-group">
				<label class="form-label">Email</label>
				<input class="form-control form-control-lg" type="email" />
			</div>
			<div class="form-group">
				<label class="form-label">Password</label>
				<input class="form-control" type="password" />
			</div>
			<button class="btn btn-primary btn-block">Submit</button>
		</form>
		`
		classes, err := ExtractHTMLClasses(strings.NewReader(html))

		require.NoError(t, err)
		assert.Contains(t, classes, "form-container")
		assert.Contains(t, classes, "form-group")
		assert.Contains(t, classes, "form-control")
		assert.Contains(t, classes, "btn-primary")
	})
}
