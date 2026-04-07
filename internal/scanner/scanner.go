package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Henelik/css-trimmer/internal/config"
)

// Scanner walks a directory and extracts CSS class references.
type Scanner struct {
	config       *config.Config
	classSet     map[string]bool
	classes      []string
	filesScanned int
}

// NewScanner creates a new directory scanner.
func NewScanner(cfg *config.Config) *Scanner {
	return &Scanner{
		config:   cfg,
		classSet: make(map[string]bool),
		classes:  []string{},
	}
}

// Scan walks the src directory and collects all class references.
func (s *Scanner) Scan(srcDir string) ([]string, int, error) {
	if err := filepath.Walk(srcDir, s.visitFile); err != nil {
		return nil, 0, err
	}

	return s.classes, s.filesScanned, nil
}

// visitFile is called for each file in the directory walk.
func (s *Scanner) visitFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		return nil // Skip files with errors
	}

	if info.IsDir() {
		return nil
	}

	// Check if file extension should be scanned
	if !s.config.IsExtensionIncluded(path) {
		return nil
	}

	s.filesScanned++

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil // Skip unreadable files
	}

	ext := filepath.Ext(path)
	var classes []string

	switch ext {
	case ".html":
		htmlClasses, err := ExtractHTMLClasses(strings.NewReader(string(content)))
		if err != nil {
			return nil
		}
		classes = htmlClasses

	case ".templ":
		classes = ExtractTemplClasses(string(content))

	case ".jsx", ".tsx":
		// For JSX/TSX, use HTML-like extraction (class attribute patterns)
		// But also support className strings
		text := string(content)
		classes = extractJSXClasses(text)

	default:
		return nil
	}

	// Add classes to set, avoiding duplicates
	for _, className := range classes {
		if className != "" && !s.classSet[className] {
			s.classes = append(s.classes, className)
			s.classSet[className] = true
		}
	}

	return nil
}

// extractJSXClasses extracts classes from JSX/TSX content.
func extractJSXClasses(content string) []string {
	var classes []string
	classSet := make(map[string]bool)

	// Pattern: className="foo bar"
	classNameRegex := regexp.MustCompile(`className="([^"]*)"`)
	for _, match := range classNameRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			for part := range strings.FieldsSeq(match[1]) {
				if part != "" && !classSet[part] {
					classes = append(classes, part)
					classSet[part] = true
				}
			}
		}
	}

	// Pattern: class="foo bar" (sometimes JSX uses class too)
	classRegex := regexp.MustCompile(`class="([^"]*)"`)
	for _, match := range classRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			for part := range strings.FieldsSeq(match[1]) {
				if part != "" && !classSet[part] {
					classes = append(classes, part)
					classSet[part] = true
				}
			}
		}
	}

	return classes
}
