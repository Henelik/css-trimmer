package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Henelik/css-trimmer/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScannerScan(t *testing.T) {
	t.Run("scans directory and returns classes", func(t *testing.T) {
		tmpdir := t.TempDir()
		htmlFile := filepath.Join(tmpdir, "test.html")
		err := os.WriteFile(htmlFile, []byte(`<div class="btn button">Text</div>`), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Extensions:           []string{".html"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		assert.Equal(t, 1, filesScanned)
		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "button")
	})

	t.Run("handles multiple files", func(t *testing.T) {
		tmpdir := t.TempDir()

		// Create HTML file
		htmlFile := filepath.Join(tmpdir, "test.html")
		err := os.WriteFile(htmlFile, []byte(`<div class="html-class">Content</div>`), 0644)
		require.NoError(t, err)

		// Create templ file
		templFile := filepath.Join(tmpdir, "test.templ")
		err = os.WriteFile(templFile, []byte(`<div class="templ-class">Content</div>`), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Extensions:           []string{".html", ".templ"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		assert.Equal(t, 2, filesScanned)
		assert.Contains(t, classes, "html-class")
		assert.Contains(t, classes, "templ-class")
	})

	t.Run("skips non-matching extensions", func(t *testing.T) {
		tmpdir := t.TempDir()

		// Create CSS file (should be skipped)
		cssFile := filepath.Join(tmpdir, "style.css")
		err := os.WriteFile(cssFile, []byte(`.css-class { color: red; }`), 0644)
		require.NoError(t, err)

		// Create HTML file (should be scanned)
		htmlFile := filepath.Join(tmpdir, "index.html")
		err = os.WriteFile(htmlFile, []byte(`<div class="html-class">Content</div>`), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Extensions:           []string{".html"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		assert.Equal(t, 1, filesScanned)
		assert.Contains(t, classes, "html-class")
		assert.NotContains(t, classes, "css-class")
	})

	t.Run("avoids duplicate classes", func(t *testing.T) {
		tmpdir := t.TempDir()
		htmlFile := filepath.Join(tmpdir, "test.html")
		err := os.WriteFile(htmlFile, []byte(`<div class="btn btn btn">Text</div>`), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Extensions:           []string{".html"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, _, err := scanner.Scan(tmpdir)

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

	t.Run("handles directory with subdirectories", func(t *testing.T) {
		tmpdir := t.TempDir()
		subdir := filepath.Join(tmpdir, "subdir")
		os.Mkdir(subdir, 0755)

		// Create file in root
		rootFile := filepath.Join(tmpdir, "root.html")
		os.WriteFile(rootFile, []byte(`<div class="root-class">Content</div>`), 0644)

		// Create file in subdirectory
		subFile := filepath.Join(subdir, "sub.html")
		os.WriteFile(subFile, []byte(`<div class="sub-class">Content</div>`), 0644)

		cfg := &config.Config{
			Extensions:           []string{".html"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		assert.Equal(t, 2, filesScanned)
		assert.Contains(t, classes, "root-class")
		assert.Contains(t, classes, "sub-class")
	})

	t.Run("handles nonexistent directory gracefully", func(t *testing.T) {
		cfg := config.DefaultConfig()
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan("/nonexistent/path")

		// filepath.Walk returns nil for nonexistent paths, no error
		require.NoError(t, err)
		assert.Equal(t, 0, len(classes))
		assert.Equal(t, 0, filesScanned)
	})

	t.Run("processes multiple files", func(t *testing.T) {
		tmpdir := t.TempDir()
		readableFile := filepath.Join(tmpdir, "readable.html")
		os.WriteFile(readableFile, []byte(`<div class="readable">Content</div>`), 0644)

		cfg := &config.Config{
			Extensions:           []string{".html"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		// Should process the file
		assert.GreaterOrEqual(t, filesScanned, 1)
		assert.Contains(t, classes, "readable")
	})
}

func TestScannerScan_JSXFiles(t *testing.T) {
	t.Run("scans JSX files", func(t *testing.T) {
		tmpdir := t.TempDir()
		jsxFile := filepath.Join(tmpdir, "Component.jsx")
		err := os.WriteFile(jsxFile, []byte(`export default function Component() {
  return <div className="btn btn-primary">Click me</div>
}`), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Extensions:           []string{".jsx"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		assert.Equal(t, 1, filesScanned)
		assert.Contains(t, classes, "btn")
		assert.Contains(t, classes, "btn-primary")
	})

	t.Run("scans TSX files", func(t *testing.T) {
		tmpdir := t.TempDir()
		tsxFile := filepath.Join(tmpdir, "Component.tsx")
		err := os.WriteFile(tsxFile, []byte(`interface Props {}
export default function Component(props: Props) {
  return <div className="text-lg font-bold">Title</div>
}`), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Extensions:           []string{".tsx"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		assert.Equal(t, 1, filesScanned)
		assert.Contains(t, classes, "text-lg")
		assert.Contains(t, classes, "font-bold")
	})
}

func TestScannerScan_EdgeCases(t *testing.T) {
	t.Run("handles empty directory", func(t *testing.T) {
		tmpdir := t.TempDir()

		cfg := config.DefaultConfig()
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		assert.Equal(t, 0, filesScanned)
		assert.Equal(t, 0, len(classes))
	})

	t.Run("handles files with special characters in class names", func(t *testing.T) {
		tmpdir := t.TempDir()
		htmlFile := filepath.Join(tmpdir, "test.html")
		err := os.WriteFile(htmlFile, []byte(`<div class="btn-primary-lg text-2xl">Content</div>`), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Extensions:           []string{".html"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, _, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		assert.Contains(t, classes, "btn-primary-lg")
		assert.Contains(t, classes, "text-2xl")
	})

	t.Run("handles very large class lists", func(t *testing.T) {
		tmpdir := t.TempDir()
		htmlFile := filepath.Join(tmpdir, "test.html")

		// Create a file with many classes
		var classStr strings.Builder
		classStr.WriteString(`<div class="`)

		for i := range 100 {
			classStr.WriteString("class")
			classStr.WriteRune(rune(i))
			classStr.WriteString(" ")
		}

		classStr.WriteString(`">Content</div>`)

		err := os.WriteFile(htmlFile, []byte(classStr.String()), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Extensions:           []string{".html"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		scannedClasses, filesScanned, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		assert.Equal(t, 1, filesScanned)
		assert.Greater(t, len(scannedClasses), 0)
	})
}

func TestScannerScan_AggregatesErrors(t *testing.T) {
	t.Run("returns joined error when multiple files fail", func(t *testing.T) {
		tmpdir := t.TempDir()

		// Use broken symlinks instead of chmod 0000 so os.ReadFile fails even
		// when the test is run as root (root bypasses file permissions).
		target := filepath.Join(tmpdir, "nonexistent-target")
		for i := range 3 {
			link := filepath.Join(tmpdir, fmt.Sprintf("fail%d.html", i))
			require.NoError(t, os.Symlink(target, link))
		}

		cfg := &config.Config{
			Extensions:           []string{".html"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan(tmpdir)

		require.Error(t, err)
		assert.Nil(t, classes)
		assert.Equal(t, 0, filesScanned)

		// All three error messages must be present in the joined error.
		errMsg := err.Error()
		for i := range 3 {
			assert.Contains(t, errMsg, fmt.Sprintf("fail%d.html", i))
		}
	})
}

func TestScannerScan_ConcurrencyLimit(t *testing.T) {
	t.Run("still scans many files correctly with concurrency limit", func(t *testing.T) {
		tmpdir := t.TempDir()
		numFiles := 50

		for i := range numFiles {
			f := filepath.Join(tmpdir, fmt.Sprintf("file%d.html", i))
			require.NoError(t, os.WriteFile(f, []byte(fmt.Sprintf(`<div class="class%d">Content</div>`, i)), 0644))
		}

		cfg := &config.Config{
			Extensions:           []string{".html"},
			Whitelist:            []string{},
			Blacklist:            []string{},
			DynamicClassPatterns: []string{},
		}
		scanner := NewScanner(cfg)
		classes, filesScanned, err := scanner.Scan(tmpdir)

		require.NoError(t, err)
		assert.Equal(t, numFiles, filesScanned)
		assert.Equal(t, numFiles, len(classes))
		for i := range numFiles {
			assert.Contains(t, classes, fmt.Sprintf("class%d", i))
		}
	})
}
