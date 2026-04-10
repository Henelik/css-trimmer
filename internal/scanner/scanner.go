package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/Henelik/css-trimmer/internal/config"
)

var (
	classNameRegex = regexp.MustCompile(`className="([^"]*)"`)
	classRegex     = regexp.MustCompile(`class="([^"]*)"`)
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
	if _, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return s.classes, s.filesScanned, nil
		}
		return nil, 0, err
	}

	var files []string

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if info.IsDir() {
			return nil
		}

		if !s.config.IsExtensionIncluded(path) {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errCh := make(chan error, len(files))

	for _, path := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			classes, err := s.extractFileClasses(path)
			if err != nil {
				errCh <- err
				return
			}

			mu.Lock()
			s.filesScanned++
			for _, className := range classes {
				if className != "" && !s.classSet[className] {
					s.classes = append(s.classes, className)
					s.classSet[className] = true
				}
			}
			mu.Unlock()
		}(path)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return nil, 0, err
		}
	}

	return s.classes, s.filesScanned, nil
}

func (s *Scanner) extractFileClasses(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		htmlClasses, err := ExtractHTMLClasses(strings.NewReader(string(content)))
		if err != nil {
			return nil, nil
		}
		return htmlClasses, nil
	case ".templ":
		return ExtractTemplClasses(string(content)), nil
	case ".jsx", ".tsx":
		return extractJSXClasses(string(content)), nil
	default:
		return nil, nil
	}
}

// extractJSXClasses extracts classes from JSX/TSX content.
func extractJSXClasses(content string) []string {
	var classes []string
	classSet := make(map[string]bool)

	// Pattern: className="foo bar"
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
