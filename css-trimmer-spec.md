# `css-trimmer` — specification

## Overview

`css-trimmer` is a static analysis CLI tool that removes unused CSS class rules from a CSS file by scanning a directory of source files (HTML, `.templ`, etc.) for class references. It is deterministic and configurable via a YAML file. Changes are applied immediately by default; use `--dry-run` to preview.

---

## Command signature

```
css-trimmer [flags] <src-dir> <css-file>
```

| Argument | Description |
|---|---|
| `<src-dir>` | Directory tree to scan for class references |
| `<css-file>` | CSS file to analyse and rewrite |

### Flags

| Flag | Default | Description |
|---|---|---|
| `--dry-run` | `false` | Print what would be removed; do not write |
| `--config` | `css-trimmer.yaml` | Path to config file |
| `--output` | (in-place) | Write result to a different file instead |
| `--format` | `text` | Report format: `text`, `json` |
| `--verbose` | `false` | Print every class found and its decision |
| `--no-backup` | `false` | Skip creating a `.bak` file before writing |

---

## Configuration file (`css-trimmer.yaml`)

```yaml
# Classes never removed, regardless of whether they appear in source files.
whitelist:
  - js-*          # glob patterns supported
  - is-active
  - hidden

# Classes always removed, even if they appear in source files.
blacklist:
  - debug-*
  - legacy-red

# File extensions to scan inside src-dir (default shown).
extensions:
  - .html
  - .templ
  - .jsx
  - .tsx

# Regex patterns treated as dynamic class usage.
# Any CSS class matching one of these regexes is considered "used"
# even if not found literally in source files.
dynamic_class_patterns:
  - "^theme-"
  - "^color-[a-z]+"

# If true, css-trimmer exits with code 1 when any classes would be removed.
# Useful for CI gates.
fail_on_removal: false
```

Glob patterns in `whitelist` / `blacklist` are resolved with [`path.Match`](https://pkg.go.dev/path#Match) semantics. `*` matches any sequence of non-separator characters. Blacklist takes precedence over whitelist.

---

## Package structure

```
css-trimmer/
├── cmd/
│   └── css-trimmer/
│       └── main.go          # CLI wiring (cobra or flag stdlib)
├── internal/
│   ├── config/
│   │   └── config.go        # YAML loader, validation
│   ├── scanner/
│   │   ├── scanner.go       # Walk src-dir, dispatch to extractors
│   │   ├── html.go          # Parse class="..." attributes (golang.org/x/net/html)
│   │   └── templ.go         # Regex-based class extraction from .templ files
│   ├── css/
│   │   ├── parser.go        # Build inventory of defined classes
│   │   └── writer.go        # Remove rules and write output
│   ├── diff/
│   │   └── diff.go          # Set logic: defined − (used ∪ whitelist) ∪ blacklist
│   └── report/
│       └── report.go        # Text and JSON report formatters
├── css-trimmer.yaml          # Example config
└── go.mod
```

---

## Core data types

```go
// The complete set of CSS class names defined in the CSS file.
type ClassInventory map[string][]CSSRule

// One CSS rule block associated with a class selector.
type CSSRule struct {
    Selector  string
    Body      string
    StartLine int
    EndLine   int
}

// Result of the diff stage.
type DiffResult struct {
    Used        []string  // found in source files
    Unused      []string  // defined in CSS but not used
    Whitelisted []string
    Blacklisted []string
    ToRemove    []string  // final set: (Unused − Whitelist) ∪ Blacklist
}
```

---

## Scanner behaviour

The scanner walks `<src-dir>` recursively, filtering by configured extensions.

**HTML extractor** — uses `golang.org/x/net/html` to tokenise the file. For every start token it reads the `class` attribute and splits on whitespace. This is not regex-based, so it handles multi-line attributes and HTML entities correctly.

**Templ extractor** — `.templ` files are not valid HTML, but class strings appear in two patterns:

```
class="foo bar baz"
templ.Classes("foo", "bar")
```

Two regex passes cover both. A third fallback pass scans for any bare `"word"` string that looks like a CSS identifier and marks it as dynamically possible (reported as uncertain, never auto-removed unless on the blacklist).

**Dynamic class guards** — if `dynamic_class_patterns` is set in config, any defined CSS class matching one of those patterns is added to the "used" set unconditionally. This prevents removing classes assembled at runtime like `color-${value}`.

---

## CSS parser behaviour

The CSS parser uses a line-oriented state machine sufficient for single-file BEM/utility stylesheets. It:

- extracts `.classname` selectors, including compound selectors (`.foo.bar`, `.foo > .bar`)
- records all selectors that include a given class name so the full rule can be preserved or removed
- handles `@media`, `@keyframes`, and `@supports` blocks (the at-rule wrapper is preserved if any inner rule survives, removed if all inner rules are removed)
- treats `/* css-trimmer-ignore */` inline comments as a per-rule whitelist override

**Selector decomposition** — given `.btn.is-active:hover`, the parser extracts both `btn` and `is-active` as class names. A rule is only removed when *every* class in its selector would be removed.

---

## Diff engine logic

```
to_remove = (defined − used − whitelist_matches) ∪ blacklist_matches
```

Precedence:

1. Blacklist always wins — the class is removed even if found in source.
2. Whitelist next — the class is kept even if not found in source.
3. Dynamic patterns add to the "used" set before set subtraction.
4. Remaining defined classes not in "used" are removed.

---

## Writer behaviour

- Before writing, creates `<css-file>.bak` unless `--no-backup` is set.
- Removes complete rule blocks (selector + `{…}` body).
- Preserves all comments not directly inside a removed rule.
- Preserves original whitespace and line endings in all retained content.
- If `--output` is given, writes to that path; the original is untouched (no `.bak` created).
- Empty `@media` / `@supports` wrappers left by removed inner rules are also removed.

---

## Report output

### Text (default)

```
css-trimmer — 3 files scanned, 142 classes defined, 97 used

  Removing 45 classes:
    .legacy-card         (not referenced)
    .debug-outline       (blacklisted)
    .btn-deprecated      (not referenced)
    ...

  Keeping 5 (whitelisted):
    .js-modal-open
    .is-active
    ...

  Wrote: styles/main.css  (backup: styles/main.css.bak)
```

### JSON (`--format json`)

```json
{
  "scanned_files": 3,
  "defined": 142,
  "used": 97,
  "to_remove": ["legacy-card", "debug-outline"],
  "whitelisted": ["js-modal-open", "is-active"],
  "blacklisted": ["debug-outline"],
  "output_file": "styles/main.css"
}
```

---

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success, no changes needed or changes applied |
| `1` | Changes were made and `fail_on_removal: true` |
| `2` | Configuration or argument error |
| `3` | File I/O error |

---

## Dependencies

| Package | Purpose |
|---|---|
| `golang.org/x/net/html` | HTML tokeniser |
| `gopkg.in/yaml.v3` | Config file parsing |
| `github.com/spf13/cobra` | CLI flags and subcommands |
| stdlib `path/filepath`, `regexp`, `os` | Everything else |

---

## Future extension points

- **Multi-CSS mode** — accept a glob for `<css-file>` to clean several files in one pass
- **Watch mode** — `--watch` re-runs on file change, useful during development
- **Plugin API** — register custom extractors for other template formats (`.svelte`, `.vue`) via a simple `Extractor` interface
- **Source maps** — emit a map of which source file justified keeping each class, for auditability
