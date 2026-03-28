package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Henelik/css-trimmer/internal/config"
	"github.com/Henelik/css-trimmer/internal/css"
	"github.com/Henelik/css-trimmer/internal/diff"
	"github.com/Henelik/css-trimmer/internal/report"
	"github.com/Henelik/css-trimmer/internal/scanner"
)

var (
	dryRun     bool
	configPath string
	outputPath string
	format     string
	verbose    bool
	noBackup   bool
)

var rootCmd = &cobra.Command{
	Use:   "css-trimmer <src-dir> <css-file>",
	Short: "Remove unused CSS class rules from a CSS file",
	Long: `css-trimmer is a static analysis tool that removes unused CSS class rules
from a CSS file by scanning source files for class references.`,
	Args: cobra.ExactArgs(2),
	Run:  runCssTrimmer,
}

func init() {
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be removed; do not write")
	rootCmd.Flags().StringVar(&configPath, "config", "css-trimmer.yaml", "Path to config file")
	rootCmd.Flags().StringVar(&outputPath, "output", "", "Write result to a different file instead")
	rootCmd.Flags().StringVar(&format, "format", "text", "Report format: text, json")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Print every class found and its decision")
	rootCmd.Flags().BoolVar(&noBackup, "no-backup", false, "Skip creating a .bak file before writing")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
}

func runCssTrimmer(cmd *cobra.Command, args []string) {
	srcDir := args[0]
	cssFile := args[1]

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(2)
	}

	// Scan source directory
	scan := scanner.NewScanner(cfg)
	usedClasses, filesScanned, err := scan.Scan(srcDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Scan error: %v\n", err)
		os.Exit(3)
	}

	// Read CSS file
	cssContent, err := os.ReadFile(cssFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CSS read error: %v\n", err)
		os.Exit(3)
	}

	// Parse CSS
	parser := css.NewParser(string(cssContent))
	inventory, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "CSS parse error: %v\n", err)
		os.Exit(2)
	}

	// Compute diff
	diffResult := diff.Compute(inventory.AllClasses(), usedClasses, cfg)

	// Print verbose info if requested
	if verbose {
		fmt.Fprintf(os.Stderr, "Verbose output:\n")
		fmt.Fprintf(os.Stderr, "  Defined: %v\n", inventory.AllClasses())
		fmt.Fprintf(os.Stderr, "  Used: %v\n", usedClasses)
		fmt.Fprintf(os.Stderr, "  To remove: %v\n", diffResult.ToRemove)
	}

	// Determine output file
	outFile := outputPath
	if outFile == "" {
		outFile = cssFile
	}

	// Generate report
	backupFile := ""
	if !dryRun && !noBackup && outFile != "" {
		backupFile = outFile + ".bak"
	}

	rep := report.NewReporter(diffResult, filesScanned, outFile, backupFile)

	// Output report
	if format == "json" {
		fmt.Println(rep.JSONReport())
	} else {
		fmt.Print(rep.TextReport())
	}

	// Write if not dry run
	if !dryRun && len(diffResult.ToRemove) > 0 {
		writer := css.NewWriter(string(cssContent), diffResult.ToRemove)
		err := writer.Write(outFile, !noBackup)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
			os.Exit(3)
		}

		// Check fail_on_removal flag
		if cfg.FailOnRemoval && len(diffResult.ToRemove) > 0 {
			os.Exit(1)
		}
	}
}
