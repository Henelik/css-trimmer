package report

import (
	"encoding/json"
	"testing"

	"github.com/Henelik/css-trimmer/internal/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReporterTextReport(t *testing.T) {
	t.Run("generates text report with classes summary", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:        []string{"btn", "header"},
			Unused:      []string{"unused-1", "unused-2"},
			Whitelisted: []string{},
			Blacklisted: []string{},
			ToRemove:    []string{"unused-1", "unused-2"},
		}
		reporter := NewReporter(result, 10, "/output.css", "/output.css.bak")
		report := reporter.TextReport()

		require.NotNil(t, report)
		assert.Contains(t, report, "css-trimmer")
		assert.Contains(t, report, "10 files scanned")
		assert.Contains(t, report, "4 classes defined")
		assert.Contains(t, report, "2 used")
		assert.Contains(t, report, "Removing 2 classes")
	})

	t.Run("includes output file information", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:   []string{"class"},
			Unused: []string{},
		}
		reporter := NewReporter(result, 1, "/output.css", "/output.css.bak")
		report := reporter.TextReport()

		assert.Contains(t, report, "Wrote: /output.css")
		assert.Contains(t, report, "backup: /output.css.bak")
	})

	t.Run("shows no backup when backup is empty", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:   []string{"class"},
			Unused: []string{},
		}
		reporter := NewReporter(result, 1, "/output.css", "")
		report := reporter.TextReport()

		assert.Contains(t, report, "backup: none")
	})

	t.Run("omits output information when file is empty", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:   []string{"class"},
			Unused: []string{},
		}
		reporter := NewReporter(result, 1, "", "")
		report := reporter.TextReport()

		assert.NotContains(t, report, "Wrote:")
	})

	t.Run("handles zero removals", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:     []string{"btn", "header"},
			Unused:   []string{},
			ToRemove: []string{},
		}
		reporter := NewReporter(result, 5, "", "")
		report := reporter.TextReport()

		assert.NotContains(t, report, "Removing")
		assert.Contains(t, report, "2 classes defined")
		assert.Contains(t, report, "2 used")
	})
}

func TestReporterJSONReport(t *testing.T) {
	t.Run("generates valid JSON report", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:        []string{"btn", "header"},
			Unused:      []string{"unused"},
			Whitelisted: []string{"safe"},
			Blacklisted: []string{"deprecated"},
			ToRemove:    []string{"unused", "deprecated"},
		}
		reporter := NewReporter(result, 10, "/output.css", "")
		jsonReport := reporter.JSONReport()

		var data map[string]any
		err := json.Unmarshal([]byte(jsonReport), &data)
		require.NoError(t, err)
	})

	t.Run("includes all required fields", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:        []string{"btn"},
			Unused:      []string{"unused"},
			Whitelisted: []string{"safe"},
			Blacklisted: []string{"deprecated"},
			ToRemove:    []string{"unused", "deprecated"},
		}
		reporter := NewReporter(result, 5, "/output.css", "")
		jsonReport := reporter.JSONReport()

		var data map[string]any
		json.Unmarshal([]byte(jsonReport), &data)

		assert.Equal(t, float64(5), data["scanned_files"])
		assert.Equal(t, float64(2), data["defined"])
		assert.Equal(t, float64(1), data["used"])
		assert.Equal(t, "/output.css", data["output_file"])
	})

	t.Run("includes to_remove array", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:     []string{},
			Unused:   []string{},
			ToRemove: []string{"class1", "class2"},
		}
		reporter := NewReporter(result, 1, "", "")
		jsonReport := reporter.JSONReport()

		var data map[string]any
		json.Unmarshal([]byte(jsonReport), &data)

		toRemove := data["to_remove"].([]any)
		assert.Equal(t, 2, len(toRemove))
	})

	t.Run("includes whitelisted array", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:        []string{},
			Unused:      []string{},
			Whitelisted: []string{"safe-1", "safe-2"},
			ToRemove:    []string{},
		}
		reporter := NewReporter(result, 1, "", "")
		jsonReport := reporter.JSONReport()

		var data map[string]any
		json.Unmarshal([]byte(jsonReport), &data)

		whitelisted := data["whitelisted"].([]any)
		assert.Equal(t, 2, len(whitelisted))
	})

	t.Run("includes blacklisted array", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:        []string{},
			Unused:      []string{},
			Blacklisted: []string{"deprecated"},
			ToRemove:    []string{"deprecated"},
		}
		reporter := NewReporter(result, 1, "", "")
		jsonReport := reporter.JSONReport()

		var data map[string]any
		json.Unmarshal([]byte(jsonReport), &data)

		blacklisted := data["blacklisted"].([]any)
		assert.Equal(t, 1, len(blacklisted))
	})

	t.Run("omits output_file when empty", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:   []string{},
			Unused: []string{},
		}
		reporter := NewReporter(result, 1, "", "")
		jsonReport := reporter.JSONReport()

		var data map[string]any
		json.Unmarshal([]byte(jsonReport), &data)

		assert.NotContains(t, data, "output_file")
	})

	t.Run("handles empty arrays in JSON", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:        []string{},
			Unused:      []string{},
			Whitelisted: []string{},
			Blacklisted: []string{},
			ToRemove:    []string{},
		}
		reporter := NewReporter(result, 0, "", "")
		jsonReport := reporter.JSONReport()

		var data map[string]any
		err := json.Unmarshal([]byte(jsonReport), &data)
		require.NoError(t, err)

		assert.Equal(t, []any{}, data["to_remove"])
		assert.Equal(t, []any{}, data["whitelisted"])
	})
}

func TestReporter_RealWorldScenarios(t *testing.T) {
	t.Run("generates report for complex scenario", func(t *testing.T) {
		result := &diff.DiffResult{
			Used: []string{
				"btn", "btn-primary", "btn-secondary",
				"header", "nav", "nav-item",
				"container", "row", "col-12",
			},
			Unused: []string{
				"deprecated-btn", "old-style", "legacy-class",
				"unused-1", "unused-2", "unused-3",
			},
			Whitelisted: []string{
				"safe-class", "framework-class", "vendor-class",
			},
			Blacklisted: []string{
				"deprecated-btn", "old-style",
			},
			ToRemove: []string{
				"deprecated-btn", "old-style", "legacy-class", "unused-1", "unused-2", "unused-3",
			},
		}
		reporter := NewReporter(result, 50, "/dist/style.min.css", "/dist/style.min.css.bak")
		textReport := reporter.TextReport()
		jsonReport := reporter.JSONReport()

		// Verify text report
		assert.Contains(t, textReport, "50 files scanned")
		assert.Contains(t, textReport, "15 classes defined")
		assert.Contains(t, textReport, "9 used")
		assert.Contains(t, textReport, "Removing 6 classes")
		assert.Contains(t, textReport, "Keeping 3 (whitelisted)")

		// Verify JSON report
		var data map[string]any
		json.Unmarshal([]byte(jsonReport), &data)
		assert.Equal(t, float64(50), data["scanned_files"])
		assert.Equal(t, float64(15), data["defined"])
	})

	t.Run("handles zero usage scenario", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:        []string{},
			Unused:      []string{"class1", "class2", "class3"},
			Whitelisted: []string{},
			Blacklisted: []string{},
			ToRemove:    []string{"class1", "class2", "class3"},
		}
		reporter := NewReporter(result, 5, "", "")
		report := reporter.TextReport()

		assert.Contains(t, report, "Removing 3 classes")
		assert.Contains(t, report, "0 used")
	})

	t.Run("handles all whitelisted scenario", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:        []string{},
			Unused:      []string{},
			Whitelisted: []string{"class1", "class2", "class3"},
			Blacklisted: []string{},
			ToRemove:    []string{},
		}
		reporter := NewReporter(result, 1, "", "")
		report := reporter.TextReport()

		assert.Contains(t, report, "Keeping 3 (whitelisted)")
		assert.NotContains(t, report, "Removing")
	})
}

func TestReporter_EdgeCases(t *testing.T) {
	t.Run("handles single class scenario", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:   []string{"single"},
			Unused: []string{},
		}
		reporter := NewReporter(result, 1, "", "")
		report := reporter.TextReport()

		assert.Contains(t, report, "1 files scanned")
		assert.Contains(t, report, "1 classes defined")
	})

	t.Run("handles special characters in file paths", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:   []string{},
			Unused: []string{},
		}
		reporter := NewReporter(result, 1, "/path/with spaces/file.css", "/path/with spaces/file.css.bak")
		report := reporter.TextReport()

		assert.Contains(t, report, "/path/with spaces/file.css")
	})

	t.Run("JSON report is properly indented", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:   []string{"class"},
			Unused: []string{},
		}
		reporter := NewReporter(result, 1, "", "")
		jsonReport := reporter.JSONReport()

		// Should contain proper indentation
		assert.Contains(t, jsonReport, "\n  ")
	})

	t.Run("handles nil backup file gracefully", func(t *testing.T) {
		result := &diff.DiffResult{
			Used:   []string{},
			Unused: []string{},
		}
		reporter := NewReporter(result, 0, "/output.css", "")
		report := reporter.TextReport()

		assert.Contains(t, report, "backup: none")
	})
}
