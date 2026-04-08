package report

import (
	"encoding/json"
	"fmt"
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
		fmt.Fprintf(&out, "  Removing %d classes", len(r.result.ToRemove))
		out.WriteString("\n")
	}

	if len(r.result.Whitelisted) > 0 {
		fmt.Fprintf(&out, "  Keeping %d (whitelisted)", len(r.result.Whitelisted))
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
