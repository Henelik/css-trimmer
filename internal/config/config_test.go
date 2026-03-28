package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigIsExtensionIncluded(t *testing.T) {
	tests := []struct {
		name       string
		extensions []string
		filePath   string
		expected   bool
	}{
		{
			name:       "matches .html extension",
			extensions: []string{".html", ".templ"},
			filePath:   "index.html",
			expected:   true,
		},
		{
			name:       "matches .jsx extension",
			extensions: []string{".html", ".jsx"},
			filePath:   "component.jsx",
			expected:   true,
		},
		{
			name:       "does not match non-included extension",
			extensions: []string{".html", ".jsx"},
			filePath:   "styles.css",
			expected:   false,
		},
		{
			name:       "matches with full path",
			extensions: []string{".html"},
			filePath:   "/path/to/index.html",
			expected:   true,
		},
		{
			name:       "handles nested paths",
			extensions: []string{".templ"},
			filePath:   "/src/components/button/Button.templ",
			expected:   true,
		},
		{
			name:       "case-sensitive extension matching",
			extensions: []string{".html"},
			filePath:   "index.HTML",
			expected:   false,
		},
		{
			name:       "empty file path with extensions configured",
			extensions: []string{".html"},
			filePath:   "",
			expected:   false,
		},
		{
			name:       "file with no extension",
			extensions: []string{".html"},
			filePath:   "Makefile",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Extensions: tt.extensions}
			result := cfg.IsExtensionIncluded(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
