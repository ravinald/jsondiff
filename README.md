# jsondiff

[![Go Reference](https://pkg.go.dev/badge/github.com/ravinald/jsondiff.svg)](https://pkg.go.dev/github.com/ravinald/jsondiff)
[![CI](https://github.com/ravinald/jsondiff/actions/workflows/ci.yml/badge.svg)](https://github.com/ravinald/jsondiff/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ravinald/jsondiff)](https://goreportcard.com/report/github.com/ravinald/jsondiff)
[![Release](https://img.shields.io/github/v/release/ravinald/jsondiff)](https://github.com/ravinald/jsondiff/releases)
[![License](https://img.shields.io/github/license/ravinald/jsondiff)](LICENSE)

A human-friendly JSON diff tool for the terminal. Compare JSON files with colored output, inline change highlighting, field filtering, and side-by-side views.

## Why jsondiff?

This tool was inspired by [josephburnett/jd](https://github.com/josephburnett/jd), an excellent JSON diff and patch utility. However, as an operator frequently comparing configuration files, I needed something where differences would **stand out more visually** - particularly when reviewing changes with teammates or tracking down configuration drift.

The key requirements that led to creating this tool:

1. **Side-by-side comparison** – Seeing the old and new values next to each other provides immediate context, especially for large config files
2. **Field filtering** – When comparing configs, you often want to ignore timestamps, metadata, or other noisy fields while focusing on what matters
3. **Familiar diff output** - Line-by-line output similar to traditional `diff` or IDE diff views, rather than structural patch formats

### How It Compares

| Feature                 | [jd](https://github.com/josephburnett/jd)    | [wI2L/jsondiff](https://github.com/wI2L/jsondiff)   | jsondiff                         |
|-------------------------|----------------------------------------------|-----------------------------------------------------|----------------------------------|
| **Output style**        | Compact structural                           | JSON Patch (RFC 6902)                               | Line-by-line diff                |
| **Use case**            | Diffing & patching                           | Automated systems, webhooks                         | Visual comparison for operators  |
| **Side-by-side**        | No                                           | No                                                  | Yes                              |
| **Inline highlighting** | No                                           | No                                                  | Yes (bold/faint)                 |
| **Field filtering**     | Path-targeted                                | JSON Pointer ignores                                | Dot notation (`address.city`)    |
| **Source markers**      | No                                           | No                                                  | Yes (shows which file)           |
| **CLI tool**            | Yes                                          | Library only                                        | Yes                              |

**jd** output (compact structural format):
```
@ ["name"]
- "Moo Cow"
+ "Moo D. Cow"
```

**jsondiff** output (line-by-line with context):
```diff
 config.json - "name": "Moo Cow"
 intent.json + "name": "Moo D. Cow"
        Both ~ "age": 30
```

### When to Use Each

**Use [jd](https://github.com/josephburnett/jd) when you need to:**
- Apply patches to JSON/YAML files
- Work with set/multiset semantics for arrays
- Generate patches in multiple formats (native, RFC 6902, RFC 7386)

**Use [wI2L/jsondiff](https://github.com/wI2L/jsondiff) when you need to:**
- Generate patches for Kubernetes admission controllers
- Build REST API PATCH endpoints
- Create audit logs with reversible operations

**Use this jsondiff when you need to:**
- Visually compare configuration files as an operator
- See changes in context with side-by-side view
- Filter out noisy fields to focus on meaningful differences
- Share diffs with teammates where readability matters

## Features

- **Visual Diff Output**: Line-by-line comparison with source markers
- **Multiple Display Modes**: Standard unified diff or side-by-side comparison
- **Smart Highlighting**: Color-coded output with inline change highlighting (bold for changes, faint for unchanged portions)
- **Field Filtering**: Include or exclude specific fields from comparison
- **Nested Field Support**: Filter nested fields using dot notation (`address.city`, `user.profile.name`)
- **Context Control**: Configurable context lines around changes (like `diff -C`)
- **JSON Normalization**: Optional key sorting before comparison to reduce false positives
- **Customizable Colors**: Full color customization via JSON configuration files
- **Ignored Field Visualization**: Excluded fields shown with `~` prefix in blue
- **Terminal-Aware**: Adapts side-by-side width to terminal size

## Installation

### Using Go Install

```bash
go install github.com/ravinald/jsondiff/cmd/jsondiff@latest
```

### Build from Source

```bash
git clone https://github.com/ravinald/jsondiff.git
cd jsondiff
go build -o jsondiff ./cmd/jsondiff
```

### Using Makefile

```bash
# Show all available targets
make help

# Build and test
make all

# Install to GOPATH/bin
make install

# Run tests with coverage
make test-coverage
```

## Quick Start

```bash
# Basic comparison
jsondiff old.json new.json

# Sort keys before comparing (reduces false positives from key reordering)
jsondiff -s old.json new.json

# Side-by-side view
jsondiff -y old.json new.json

# Only compare specific fields
jsondiff --include name,email user1.json user2.json

# Exclude noisy fields like timestamps
jsondiff --exclude timestamp,metadata,_id data1.json data2.json
```

## CLI Usage

### Command-Line Options

```
jsondiff [flags] file1.json file2.json

Flags:
  -C, --context int      Number of context lines to show (default 3)
  -s, --sort             Sort JSON keys before comparing
  -y, --side-by-side     Display side-by-side diff
      --color string     Color output: always, never, auto (default "never")
      --include strings  Fields to include in comparison (comma-separated)
      --exclude strings  Fields to exclude from comparison (comma-separated)
      --config string    Path to color configuration file
  -1 string              Marker for lines from first file (default: filename)
  -2 string              Marker for lines from second file (default: filename)
  -b string              Marker for lines in both files (default "Both")
  -h, --help             Help for jsondiff
```

### Field Filtering

#### Include Specific Fields

Only compare `name` and `email` fields:

```bash
jsondiff --include name,email user1.json user2.json
```

Output:
```diff
user1.json - "email": "moo@cow.org"
user2.json + "email": "moo@pina.org"
user1.json - "name": "Moo Cow"
user2.json + "name": "Moo D. Cow"
      Both ~ "age": 30
      Both ~ "address": {...}
```

Fields not in the include list are shown with `~` prefix in blue, indicating they were excluded from the comparison.

#### Exclude Specific Fields

Compare everything except noisy fields:

```bash
jsondiff --exclude timestamp,metadata,_id data1.json data2.json
```

#### Nested Field Filtering

Filter using dot notation for nested fields:

```bash
# Include only the city within address
jsondiff --include address.city user1.json user2.json

# Exclude sensitive nested data
jsondiff --exclude user.password,user.token auth1.json auth2.json

# Combine include and exclude
jsondiff --include user --exclude user.internal data1.json data2.json
```

### Display Modes

#### Standard Unified Diff (Default)

```bash
jsondiff config.json intent.json
```

Output:
```diff
config.json {
config.json -   "name": "Moo Cow"
 intent.json +   "name": "Moo D. Cow"
        Both ~   "age": 30
config.json }
```

#### Side-by-Side Comparison

```bash
jsondiff -y config.json intent.json
```

Output:
```
config.json                              | intent.json
-----------------------------------------|-----------------------------------------
~ {                                      | ~ {
-   "name": "Moo Cow"                    | +   "name": "Moo D. Cow"
~   "age": 30                            | ~   "age": 30
~ }                                      | ~ }
```

#### Custom Source Markers

```bash
# Use custom labels instead of filenames
jsondiff -1 "Before" -2 "After" -b "=" old.json new.json

# Short markers
jsondiff -1 A -2 B file1.json file2.json
```

Output:
```diff
Before - "name": "Moo Cow"
 After + "name": "Moo D. Cow"
     = ~ "age": 30
```

### Context Lines

```bash
# Show 5 lines of context around changes
jsondiff -C 5 file1.json file2.json

# Show only changes (no context)
jsondiff -C 0 file1.json file2.json
```

## Library Usage

### Basic Example

```go
package main

import (
    "fmt"
    "log"
    "github.com/ravinald/jsondiff"
)

func main() {
    json1 := []byte(`{"name": "Moo Cow", "age": 30}`)
    json2 := []byte(`{"name": "Moo D. Cow", "age": 31}`)

    opts := jsondiff.DiffOptions{
        ContextLines: 3,
        SortJSON:     false,
    }

    diffs, err := jsondiff.Diff(json1, json2, opts)
    if err != nil {
        log.Fatal(err)
    }

    // Enhance with inline change highlighting
    diffs = jsondiff.EnhanceDiffsWithInlineChanges(diffs)

    // Format and display
    formatter := jsondiff.NewFormatter(jsondiff.DefaultStyles())
    fmt.Print(formatter.Format(diffs))
}
```

### Field Filtering Example

```go
opts := jsondiff.DiffOptions{
    ContextLines:  3,
    SortJSON:      true,
    IncludeFields: []string{"name", "email", "address.city"},
    ExcludeFields: []string{"timestamp", "internal"},
}

diffs, err := jsondiff.Diff(json1, json2, opts)
```

### Custom Markers Example

```go
formatter := jsondiff.NewFormatterWithOptions(jsondiff.FormatterOptions{
    Styles:      jsondiff.DefaultStyles(),
    File1Marker: "API Response",
    File2Marker: "Expected",
    BothMarker:  "Match",
})
output := formatter.Format(diffs)
```

### Side-by-Side Output

```go
formatter := jsondiff.NewFormatter(jsondiff.DefaultStyles())
output := formatter.FormatSideBySide(diffs, "before.json", "after.json")
```

### Custom Color Configuration

```go
import "encoding/json"

configJSON := `{
    "version": 1,
    "colors": {
        "add": {
            "foreground": { "line": {"hex": "#00ff00", "ansi256": 10, "ansi": 10} }
        },
        "remove": {
            "foreground": { "line": {"hex": "#ff0000", "ansi256": 9, "ansi": 9} }
        },
        "ignored": {
            "foreground": {"hex": "#0080ff", "ansi256": 12, "ansi": 12}
        }
    }
}`

var config jsondiff.ColorConfig
json.Unmarshal([]byte(configJSON), &config)

styles := jsondiff.StylesFromConfig(&config)
formatter := jsondiff.NewFormatter(styles)
```

## Configuration File

jsondiff looks for a configuration file in the following order:

1. Path specified via `--config` flag (required to exist)
2. `~/.config/jsondiff/config.json` (optional, warns if invalid)

If the default config file doesn't exist, default colors are used. If it exists but is invalid, a warning is printed and defaults are used.

### Example Configuration

Create `~/.config/jsondiff/config.json` for custom colors:

```json
{
  "version": 1,
  "colors": {
    "add": {
      "foreground": {
        "line": { "hex": "#00ff00", "ansi256": 10, "ansi": 10 }
      },
      "background": {}
    },
    "remove": {
      "foreground": {
        "line": { "hex": "#ff0000", "ansi256": 9, "ansi": 9 }
      },
      "background": {}
    },
    "ignored": {
      "foreground": { "hex": "#0080ff", "ansi256": 12, "ansi": 12 },
      "background": {}
    }
  }
}
```

Or specify a custom path:

```bash
jsondiff --config /path/to/colors.json --color=always file1.json file2.json
```

### Color Values

Color values support multiple formats for terminal compatibility:
- `hex`: True color (24-bit) for modern terminals
- `ansi256`: 256-color palette for broader compatibility
- `ansi`: 16-color ANSI for maximum compatibility

## API Reference

### Types

```go
// DiffOptions configures the diff behavior
type DiffOptions struct {
    ContextLines  int      // Lines of context around changes (default: 3)
    SortJSON      bool     // Sort keys before comparison
    IncludeFields []string // Fields to include (empty = all)
    ExcludeFields []string // Fields to exclude
}

// DiffLine represents a single line in the diff output
type DiffLine struct {
    Type        DiffType // Equal, Added, or Removed
    LineNum1    int      // Line number in first file
    LineNum2    int      // Line number in second file
    Content     string   // Line content
    InlineStart int      // Start position of inline change (-1 if none)
    InlineEnd   int      // End position of inline change
    IsIgnored   bool     // True if field was filtered out
}

// DiffType indicates the type of difference
type DiffType int

const (
    DiffTypeEqual   DiffType = iota  // Line exists in both files
    DiffTypeAdded                     // Line only in second file
    DiffTypeRemoved                   // Line only in first file
)
```

### Functions

```go
// Diff compares two JSON byte slices and returns the differences
func Diff(json1, json2 []byte, opts DiffOptions) ([]DiffLine, error)

// DiffWithContext compares JSON with cancellation support
func DiffWithContext(ctx context.Context, json1, json2 []byte, opts DiffOptions) ([]DiffLine, error)

// EnhanceDiffsWithInlineChanges adds character-level change markers
// to paired added/removed lines with matching JSON keys
func EnhanceDiffsWithInlineChanges(diffs []DiffLine) []DiffLine

// NewFormatter creates a new formatter with the given styles
func NewFormatter(styles *Styles) *Formatter

// NewFormatterWithOptions creates a formatter with full configuration
func NewFormatterWithOptions(opts FormatterOptions) *Formatter

// FormatterOptions configures a Formatter
type FormatterOptions struct {
    Styles      *Styles
    File1Marker string
    File2Marker string
    BothMarker  string
}

// SetMarkers configures custom labels (Deprecated: use NewFormatterWithOptions)
func (f *Formatter) SetMarkers(file1Marker, file2Marker, bothMarker string)

// Format generates unified diff output
func (f *Formatter) Format(diffs []DiffLine) string

// FormatSideBySide generates two-column diff output
func (f *Formatter) FormatSideBySide(diffs []DiffLine, leftHeader, rightHeader string) string

// DefaultStyles returns the default color configuration
func DefaultStyles() *Styles

// StylesFromConfig creates styles from a ColorConfig
func StylesFromConfig(config *ColorConfig) *Styles
```

## How It Works

### Algorithm

jsondiff uses a text-based Longest Common Subsequence (LCS) algorithm:

1. **JSON Normalization**: Both inputs are parsed and reformatted with consistent indentation
2. **Optional Sorting**: If `-s` is specified, object keys are sorted alphabetically
3. **Field Filtering**: If include/exclude filters are set, fields are marked for filtering
4. **LCS Computation**: Dynamic programming finds the longest common subsequence of lines
5. **Diff Generation**: Backtracking through the LCS matrix produces the diff
6. **Context Filtering**: Only lines within the context window are kept
7. **Inline Enhancement**: Paired add/remove lines with matching keys get character-level highlighting

### Inline Change Detection

When a removed line and added line share the same JSON key and meet similarity thresholds:
- At least 30% character overlap
- Length difference no more than 50%

The tool computes the common prefix and suffix to identify the exact changed portion:

```diff
- "name": "Moo Cow"       # "Cow" is bold, rest is faint
+ "name": "Moo D. Cow"    # "D. Cow" is bold, rest is faint
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for Makefile targets)

### Building

```bash
# Build binary
go build -o jsondiff ./cmd/jsondiff

# Or using make
make build
```

### Testing

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...

# With race detector
go test -race ./...

# Using make
make test
make test-coverage
make test-race
```

### Code Quality

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Lint (requires golangci-lint)
golangci-lint run

# Using make
make fmt
make vet
make lint
```

## Dependencies

- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling with adaptive colors
- [spf13/pflag](https://github.com/spf13/pflag) - POSIX/GNU-style flag parsing
- [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) - Terminal size detection

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Reporting Issues

When reporting issues, please include:
- Your Go version (`go version`)
- Your OS and terminal
- Sample JSON files that reproduce the issue
- Expected vs actual output

## License

Apache 2.0 - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [josephburnett/jd](https://github.com/josephburnett/jd) - an excellent JSON diff and patch tool
- Visual diff style influenced by modern code editors and AI assistants
- Built with excellent Go libraries from the [Charm](https://charm.sh/) and [Spf13](https://github.com/spf13) ecosystems
