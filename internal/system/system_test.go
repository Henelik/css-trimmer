package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Henelik/css-trimmer/internal/config"
	"github.com/Henelik/css-trimmer/internal/css"
	"github.com/Henelik/css-trimmer/internal/diff"
	"github.com/Henelik/css-trimmer/internal/report"
	"github.com/Henelik/css-trimmer/internal/scanner"
)

// testdataDir returns the absolute path to the system testdata directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("testdata")
	require.NoError(t, err)
	return dir
}

// runPipeline runs the full css-trimmer pipeline (scan -> parse -> diff)
// against the given source directory and CSS file, returning the diff result.
func runPipeline(t *testing.T, srcDir, cssFile string) *diff.DiffResult {
	t.Helper()

	cfg := config.DefaultConfig()

	scan := scanner.NewScanner(cfg)
	usedClasses, _, err := scan.Scan(srcDir)
	require.NoError(t, err)

	content, err := os.ReadFile(cssFile)
	require.NoError(t, err)

	inventory, err := css.ParseCSS(string(content))
	require.NoError(t, err)

	result, err := diff.Compute(inventory.AllClasses(), usedClasses, cfg)
	require.NoError(t, err)

	return result
}

// TestSystem_EndToEnd runs css-trimmer against the full testdata tree and
// verifies that referenced classes are kept and unreferenced classes are
// removed across all source types (HTML, templ, JSX).
func TestSystem_EndToEnd(t *testing.T) {
	td := testdataDir(t)
	cssFile := filepath.Join(td, "input.css")

	// Scan all three source folders together.
	result := runPipeline(t, td, cssFile)

	// Classes referenced in the source folders must be marked as used.
	used := make(map[string]bool)
	for _, c := range result.Used {
		used[c] = true
	}

	// HTML-referenced classes.
	for _, c := range []string{"container", "header", "nav", "nav-item", "main-content",
		"card", "card-title", "btn", "link", "list", "item", "list-footer", "sidebar",
		"footer", "input", "desktop-only", "grid-layout", "mobile-only"} {
		assert.True(t, used[c], "expected %q to be used (referenced in HTML)", c)
	}

	// templ-referenced classes.
	for _, c := range []string{"btn-primary"} {
		assert.True(t, used[c], "expected %q to be used (referenced in templ)", c)
	}

	// JSX-referenced classes.
	for _, c := range []string{"container", "header", "nav", "nav-item", "main-content",
		"card", "card-title", "btn", "footer", "input"} {
		assert.True(t, used[c], "expected %q to be used (referenced in JSX)", c)
	}

	// Unreferenced classes must be marked for removal.
	toRemove := make(map[string]bool)
	for _, c := range result.ToRemove {
		toRemove[c] = true
	}
	for _, c := range []string{"unused-class", "legacy-style", "obsolete-widget",
		"deprecated-theme", "never-used"} {
		assert.True(t, toRemove[c], "expected %q to be removed (unreferenced)", c)
	}
}

// TestSystem_HTMLOnly verifies the HTML scanner in isolation.
func TestSystem_HTMLOnly(t *testing.T) {
	td := testdataDir(t)
	htmlDir := filepath.Join(td, "html")
	cssFile := filepath.Join(td, "input.css")

	result := runPipeline(t, htmlDir, cssFile)

	used := make(map[string]bool)
	for _, c := range result.Used {
		used[c] = true
	}

	// HTML references these.
	for _, c := range []string{"container", "header", "nav", "nav-item", "main-content",
		"card", "card-title", "btn", "link", "list", "item", "list-footer", "sidebar",
		"footer", "input"} {
		assert.True(t, used[c], "expected %q to be used (HTML)", c)
	}

	// templ-only and JSX-only classes should NOT be used when only HTML is scanned.
	assert.False(t, used["btn-primary"], "btn-primary is templ-only, should not be used in HTML-only scan")
}

// TestSystem_TemplOnly verifies the templ scanner in isolation.
func TestSystem_TemplOnly(t *testing.T) {
	td := testdataDir(t)
	templDir := filepath.Join(td, "templ")
	cssFile := filepath.Join(td, "input.css")

	result := runPipeline(t, templDir, cssFile)

	used := make(map[string]bool)
	for _, c := range result.Used {
		used[c] = true
	}

	// templ references these.
	for _, c := range []string{"container", "header", "nav", "nav-item", "main-content",
		"card", "card-title", "btn", "link", "footer", "btn-primary"} {
		assert.True(t, used[c], "expected %q to be used (templ)", c)
	}

	// HTML-only and JSX-only classes should NOT be used.
	assert.False(t, used["sidebar"], "sidebar is HTML-only, should not be used in templ-only scan")
	assert.False(t, used["input"], "input is HTML/JSX-only, should not be used in templ-only scan")
}

// TestSystem_JSXOnly verifies the JSX scanner in isolation.
func TestSystem_JSXOnly(t *testing.T) {
	td := testdataDir(t)
	jsxDir := filepath.Join(td, "jsx")
	cssFile := filepath.Join(td, "input.css")

	result := runPipeline(t, jsxDir, cssFile)

	used := make(map[string]bool)
	for _, c := range result.Used {
		used[c] = true
	}

	// JSX references these.
	for _, c := range []string{"container", "header", "nav", "nav-item", "main-content",
		"card", "card-title", "btn", "footer", "input"} {
		assert.True(t, used[c], "expected %q to be used (JSX)", c)
	}

	// HTML-only and templ-only classes should NOT be used.
	assert.False(t, used["sidebar"], "sidebar is HTML-only, should not be used in JSX-only scan")
	assert.False(t, used["btn-primary"], "btn-primary is templ-only, should not be used in JSX-only scan")
}

// TestSystem_WriteRemovesUnused verifies that running the writer against the
// diff result actually removes the unused class rules from the CSS.
func TestSystem_WriteRemovesUnused(t *testing.T) {
	td := testdataDir(t)
	cssFile := filepath.Join(td, "input.css")

	result := runPipeline(t, td, cssFile)

	content, err := os.ReadFile(cssFile)
	require.NoError(t, err)

	// Write to a temp output file (no backup).
	tmpOut := filepath.Join(t.TempDir(), "out.css")
	err = css.Write(string(content), result.ToRemove, tmpOut, false)
	require.NoError(t, err)

	out, err := os.ReadFile(tmpOut)
	require.NoError(t, err)
	outStr := string(out)

	// Unused classes must be gone from the output.
	for _, c := range []string{"unused-class", "legacy-style", "obsolete-widget",
		"deprecated-theme", "never-used"} {
		assert.NotContains(t, outStr, "."+c, "expected %q rule to be removed from output", c)
	}

	// Used classes must remain.
	for _, c := range []string{"container", "header", "btn", "card", "footer", "input"} {
		assert.Contains(t, outStr, "."+c, "expected %q rule to remain in output", c)
	}
}

// TestSystem_AtRulesPreserved verifies that at-rule blocks (media, supports,
// keyframes) and their nested classes survive the write step.
func TestSystem_AtRulesPreserved(t *testing.T) {
	td := testdataDir(t)
	cssFile := filepath.Join(td, "input.css")

	result := runPipeline(t, td, cssFile)

	content, err := os.ReadFile(cssFile)
	require.NoError(t, err)

	tmpOut := filepath.Join(t.TempDir(), "out.css")
	err = css.Write(string(content), result.ToRemove, tmpOut, false)
	require.NoError(t, err)

	out, err := os.ReadFile(tmpOut)
	require.NoError(t, err)
	outStr := string(out)

	// At-rule blocks should be preserved.
	assert.Contains(t, outStr, "@media")
	assert.Contains(t, outStr, "@supports")
	assert.Contains(t, outStr, "@keyframes")

	// Nested classes inside at-rules that are used should remain.
	assert.Contains(t, outStr, ".desktop-only")
	assert.Contains(t, outStr, ".grid-layout")
	assert.Contains(t, outStr, ".mobile-only")
	assert.Contains(t, outStr, "fade-in")
}

// TestSystem_NoSourceDir verifies graceful handling when the source directory
// does not exist (nothing is scanned, so everything is considered unused).
func TestSystem_NoSourceDir(t *testing.T) {
	td := testdataDir(t)
	cssFile := filepath.Join(td, "input.css")

	missing := filepath.Join(td, "does-not-exist")
	result := runPipeline(t, missing, cssFile)

	// With no sources, every defined class is unused and marked for removal.
	assert.NotEmpty(t, result.ToRemove)
	assert.Contains(t, result.ToRemove, "container")
	assert.Contains(t, result.ToRemove, "btn")
}

// TestSystem_JSONReport verifies the JSON report output is valid and complete.
func TestSystem_JSONReport(t *testing.T) {
	td := testdataDir(t)
	cssFile := filepath.Join(td, "input.css")

	cfg := config.DefaultConfig()
	scan := scanner.NewScanner(cfg)
	usedClasses, filesScanned, err := scan.Scan(td)
	require.NoError(t, err)

	content, err := os.ReadFile(cssFile)
	require.NoError(t, err)
	inventory, err := css.ParseCSS(string(content))
	require.NoError(t, err)
	result, err := diff.Compute(inventory.AllClasses(), usedClasses, cfg)
	require.NoError(t, err)

	// Build a reporter and check the JSON output.
	rep := report.NewReporter(result, filesScanned, cssFile, "")
	jsonOut, err := rep.JSONReport()
	require.NoError(t, err)
	assert.Contains(t, jsonOut, `"scanned_files"`)
	assert.Contains(t, jsonOut, `"to_remove"`)
	assert.Contains(t, jsonOut, `"used"`)
	assert.Contains(t, jsonOut, `"defined"`)
	assert.True(t, strings.Contains(jsonOut, "unused-class"), "unused-class should appear in to_remove")
}
