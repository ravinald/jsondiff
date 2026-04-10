// Copyright 2024 Ravi Pina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	flag "github.com/spf13/pflag"
	"golang.org/x/term"

	"github.com/ravinald/jsondiff"
)

// CLIConfig holds all CLI configuration options.
type CLIConfig struct {
	ContextLines  int
	SortJSON      bool
	ConfigFile    string
	SideBySide    bool
	IncludeFields []string
	ExcludeFields []string
	File1Marker   string
	File2Marker   string
	BothMarker    string
	ColorMode     string
}

var (
	contextLines  = flag.IntP("context", "C", 3, "Number of context lines to show")
	sortJSON      = flag.BoolP("sort", "s", false, "Sort JSON keys before comparing")
	sideBySide    = flag.BoolP("side-by-side", "y", false, "Display side-by-side diff")
	configFile    = flag.String("config", "", "Path to configuration file")
	includeFields = flag.StringSlice("include", []string{}, "Fields to include in comparison (e.g., 'name,address.city')")
	excludeFields = flag.StringSlice("exclude", []string{}, "Fields to exclude from comparison (e.g., 'timestamp,metadata')")
	file1Marker   = flag.StringP("1", "1", "", "Marker for lines from first file (default: filename)")
	file2Marker   = flag.StringP("2", "2", "", "Marker for lines from second file (default: filename)")
	bothMarker    = flag.StringP("b", "b", "Both", "Marker for lines in both files")
	colorMode     = flag.String("color", "never", "Color output: always, never, auto")
	showHelp      = flag.BoolP("help", "h", false, "Show help message")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `jsondiff - Compare two JSON files and display differences

Usage:
  jsondiff [flags] <file1> <file2>

Flags:
`)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Error: exactly 2 arguments required, got %d\n\n", len(args))
		flag.Usage()
		os.Exit(1)
	}

	cfg := &CLIConfig{
		ContextLines:  *contextLines,
		SortJSON:      *sortJSON,
		ConfigFile:    *configFile,
		SideBySide:    *sideBySide,
		IncludeFields: *includeFields,
		ExcludeFields: *excludeFields,
		File1Marker:   *file1Marker,
		File2Marker:   *file2Marker,
		BothMarker:    *bothMarker,
		ColorMode:     *colorMode,
	}

	if err := executeDiff(cfg, args[0], args[1], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func executeDiff(cfg *CLIConfig, file1Path, file2Path string, out io.Writer) error {
	// Extract filenames for default markers
	file1Marker := cfg.File1Marker
	file2Marker := cfg.File2Marker
	if file1Marker == "" {
		file1Marker = filepath.Base(file1Path)
	}
	if file2Marker == "" {
		file2Marker = filepath.Base(file2Path)
	}

	json1, err := os.ReadFile(file1Path)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", file1Path, err)
	}

	json2, err := os.ReadFile(file2Path)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", file2Path, err)
	}

	// Set lipgloss color profile based on color mode
	switch cfg.ColorMode {
	case "always":
		lipgloss.SetColorProfile(termenv.TrueColor)
	case "never":
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	// For "auto", let lipgloss auto-detect

	var styles *jsondiff.Styles
	if shouldUseColor(cfg.ColorMode) {
		if cfg.ConfigFile != "" {
			// Explicitly specified config: fatal error if invalid
			config, err := loadConfig(cfg.ConfigFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			styles = jsondiff.StylesFromConfig(config)
		} else {
			// Try default config location
			defaultPath := defaultConfigPath()
			if defaultPath != "" {
				config, err := loadConfig(defaultPath)
				if err != nil {
					if !errors.Is(err, os.ErrNotExist) {
						fmt.Fprintf(os.Stderr, "Warning: invalid config at %s: %v\n", defaultPath, err)
					}
					styles = jsondiff.DefaultStyles()
				} else {
					styles = jsondiff.StylesFromConfig(config)
				}
			} else {
				styles = jsondiff.DefaultStyles()
			}
		}
	} else {
		styles = jsondiff.NoColorStyles()
	}

	opts := jsondiff.DiffOptions{
		ContextLines:  cfg.ContextLines,
		SortJSON:      cfg.SortJSON,
		IncludeFields: cfg.IncludeFields,
		ExcludeFields: cfg.ExcludeFields,
	}

	diffs, err := jsondiff.Diff(json1, json2, opts)
	if err != nil {
		return fmt.Errorf("comparing files: %w", err)
	}

	diffs = jsondiff.EnhanceDiffsWithInlineChanges(diffs)

	formatter := jsondiff.NewFormatterWithOptions(jsondiff.FormatterOptions{
		Styles:      styles,
		File1Marker: file1Marker,
		File2Marker: file2Marker,
		BothMarker:  cfg.BothMarker,
	})

	var output string
	if cfg.SideBySide {
		output = formatter.FormatSideBySide(diffs, file1Marker, file2Marker)
	} else {
		output = formatter.Format(diffs)
	}

	_, err = fmt.Fprint(out, output)
	return err
}

func shouldUseColor(mode string) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	case "auto":
		return term.IsTerminal(int(os.Stdout.Fd()))
	default:
		return false
	}
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "jsondiff", "config.json")
}

func loadConfig(path string) (*jsondiff.ColorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config jsondiff.ColorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}

	if config.Version != 1 {
		return nil, fmt.Errorf("unsupported config version: %d (expected 1)", config.Version)
	}

	return &config, nil
}
