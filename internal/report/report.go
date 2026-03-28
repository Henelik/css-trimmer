package report

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Henelik/css-trimmer/internal/diff"
)

// Reporter generates text and JSON reports of the trimming results.
type Reporter struct {
	result       *diff.DiffResult
	scannedFiles int
	outputFile   string
	backupFile   string
	verbose      bool
}

// NewReporter creates a new reporter.
func NewReporter(result *diff.DiffResult, scannedFiles int, outputFile, backupFile string) *Reporter {
	return &Reporter{
		result:       result,
		scannedFiles: scannedFiles,
		outputFile:   outputFile,
		backupFile:   backupFile,
	}
}

// TextReport generates a human-readable report.
func (r *Reporter) TextReport() string {
	var out strings.Builder

	defined := len(r.result.Used) + len(r.result.Unused)
	fmt.Fprintf(&out, "css-trimmer — %d files scanned, %d classes defined, %d used\n\n",
		r.scannedFiles, defined, len(r.result.Used))

	if len(r.result.ToRemove) > 0 {
		fmt.Fprintf(&out, "  Removing %d classes:\n", len(r.result.ToRemove))

		for _, className := range r.result.ToRemove {
			reason := r.getRemovalReason(className)
			fmt.Fprintf(&out, "    .%-24s (%s)\n", className, reason)
		}

		out.WriteString("\n")
	}

	if len(r.result.Whitelisted) > 0 {
		fmt.Fprintf(&out, "  Keeping %d (whitelisted):\n", len(r.result.Whitelisted))

		for _, className := range r.result.Whitelisted {
			fmt.Fprintf(&out, "    .%s\n", className)
		}

		out.WriteString("\n")
	}

	if r.outputFile != "" {
		backup := r.backupFile
		if backup == "" {
			backup = "none"
		}
		fmt.Fprintf(&out, "  Wrote: %s  (backup: %s)\n", r.outputFile, backup)
	}

	return out.String()
}

// JSONReport returns a JSON representation of the results.
func (r *Reporter) JSONReport() string {
	data := map[string]any{
		"scanned_files": r.scannedFiles,
		"defined":       len(r.result.Used) + len(r.result.Unused),
		"used":          len(r.result.Used),
		"to_remove":     r.result.ToRemove,
		"whitelisted":   r.result.Whitelisted,
		"blacklisted":   r.result.Blacklisted,
	}

	if r.outputFile != "" {
		data["output_file"] = r.outputFile
	}

	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonBytes)
}

// getRemovalReason returns the reason why a class is being removed.
func (r *Reporter) getRemovalReason(className string) string {
	if slices.Contains(r.result.Blacklisted, className) {
		return "blacklisted"
	}

	return "not referenced"
}
