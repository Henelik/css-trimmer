# css-trimmer

[![Test, Build & Release](https://github.com/Henelik/css-trimmer/actions/workflows/release.yaml/badge.svg)](https://github.com/Henelik/css-trimmer/actions/workflows/release.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Henelik/css-trimmer.svg)](https://pkg.go.dev/github.com/Henelik/css-trimmer)
[![Go Report Card](https://goreportcard.com/badge/github.com/Henelik/css-trimmer)](https://goreportcard.com/report/github.com/Henelik/css-trimmer)
[![License: MIT NON-AI](https://img.shields.io/badge/License-MIT%20NON--AI-blue.svg)](LICENSE)

CSS trimmer is a CLI tool that uses static analysis to remove unused classes from CSS files.

## Installation

Download the latest release from [GitHub Releases](https://github.com/Henelik/css-trimmer/releases) and extract it.

Or install from source:
```bash
go install github.com/Henelik/css-trimmer/cmd/css-trimmer@latest
```

## Usage

```bash
css-trimmer <src-dir> <css-file> [flags]
```

### Examples

```bash
# Analyze and remove unused CSS classes
css-trimmer ./src ./styles.css

# Preview changes without writing to file
css-trimmer ./src ./styles.css --dry-run

# Write output to a different file
css-trimmer ./src ./styles.css --output ./styles.trimmed.css
```

### Flags

- `--dry-run` - Show what would be removed without making changes
- `--config <path>` - Config file (default: css-trimmer.yaml)
- `--output <path>` - Write result to a different file
- `--format <format>` - Report format: text or json (default: text)
- `--verbose` - Print every class found and its decision
- `--no-backup` - Skip creating a .bak file before writing
