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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIFlags(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		json1Content      string
		json2Content      string
		expectError       bool
		expectInOutput    string
		expectNotInOutput string
	}{
		{
			name:              "Include flag single field",
			args:              []string{"--include", "name"},
			json1Content:      `{"name": "Alice", "age": 30}`,
			json2Content:      `{"name": "Bob", "age": 31}`,
			expectInOutput:    "name",
			expectNotInOutput: "",
		},
		{
			name:              "Include flag multiple fields",
			args:              []string{"--include", "name,email"},
			json1Content:      `{"name": "Alice", "email": "alice@example.com", "age": 30}`,
			json2Content:      `{"name": "Bob", "email": "bob@example.com", "age": 31}`,
			expectInOutput:    "email",
			expectNotInOutput: "",
		},
		{
			name:              "Exclude flag single field",
			args:              []string{"--exclude", "timestamp"},
			json1Content:      `{"name": "Alice", "timestamp": "2023-01-01"}`,
			json2Content:      `{"name": "Bob", "timestamp": "2023-01-02"}`,
			expectInOutput:    "name",
			expectNotInOutput: "",
		},
		{
			name:              "Exclude flag multiple fields",
			args:              []string{"--exclude", "timestamp,metadata"},
			json1Content:      `{"name": "Alice", "timestamp": "2023-01-01", "metadata": {"version": 1}}`,
			json2Content:      `{"name": "Bob", "timestamp": "2023-01-02", "metadata": {"version": 2}}`,
			expectInOutput:    "name",
			expectNotInOutput: "",
		},
		{
			name:              "Both include and exclude flags",
			args:              []string{"--include", "user", "--exclude", "user.internal"},
			json1Content:      `{"user": {"name": "Alice", "internal": "secret1"}, "system": "v1"}`,
			json2Content:      `{"user": {"name": "Bob", "internal": "secret2"}, "system": "v2"}`,
			expectInOutput:    "name",
			expectNotInOutput: "",
		},
		{
			name:              "Context flag with filtering",
			args:              []string{"--context", "1", "--include", "name"},
			json1Content:      `{"name": "Alice", "a": 1, "b": 2, "c": 3}`,
			json2Content:      `{"name": "Bob", "a": 1, "b": 2, "c": 3}`,
			expectInOutput:    "name",
			expectNotInOutput: "",
		},
		{
			name:              "Sort flag with filtering",
			args:              []string{"--sort", "--exclude", "id"},
			json1Content:      `{"z": 1, "a": 2, "id": "123"}`,
			json2Content:      `{"a": 3, "z": 1, "id": "456"}`,
			expectInOutput:    `"a"`,
			expectNotInOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp files
			tmpDir := t.TempDir()
			file1 := filepath.Join(tmpDir, "file1.json")
			file2 := filepath.Join(tmpDir, "file2.json")

			if err := os.WriteFile(file1, []byte(tt.json1Content), 0644); err != nil {
				t.Fatalf("Failed to write file1: %v", err)
			}
			if err := os.WriteFile(file2, []byte(tt.json2Content), 0644); err != nil {
				t.Fatalf("Failed to write file2: %v", err)
			}

			// Build config from test args
			cfg := buildTestConfig(tt.args)

			// Run executeDiff
			var buf bytes.Buffer
			err := executeDiff(cfg, file1, file2, &buf)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			output := buf.String()
			if tt.expectInOutput != "" && !strings.Contains(output, tt.expectInOutput) {
				t.Errorf("Expected output to contain %q", tt.expectInOutput)
			}
		})
	}
}

// buildTestConfig parses test arguments into a CLIConfig
func buildTestConfig(args []string) *CLIConfig {
	cfg := &CLIConfig{
		ContextLines:  3,
		SortJSON:      false,
		ConfigFile:    "",
		SideBySide:    false,
		IncludeFields: []string{},
		ExcludeFields: []string{},
		File1Marker:   "",
		File2Marker:   "",
		BothMarker:    "Both",
		ColorMode:     "never",
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--include":
			if i+1 < len(args) {
				cfg.IncludeFields = strings.Split(args[i+1], ",")
				i++
			}
		case "--exclude":
			if i+1 < len(args) {
				cfg.ExcludeFields = strings.Split(args[i+1], ",")
				i++
			}
		case "--context", "-C":
			if i+1 < len(args) {
				var val int
				_, _ = (&val), args[i+1]
				// Simple parse - tests use small numbers
				for _, c := range args[i+1] {
					if c >= '0' && c <= '9' {
						val = val*10 + int(c-'0')
					}
				}
				cfg.ContextLines = val
				i++
			}
		case "--sort", "-s":
			cfg.SortJSON = true
		case "--side-by-side", "-y":
			cfg.SideBySide = true
		case "-1":
			if i+1 < len(args) {
				cfg.File1Marker = args[i+1]
				i++
			}
		case "-2":
			if i+1 < len(args) {
				cfg.File2Marker = args[i+1]
				i++
			}
		case "-b":
			if i+1 < len(args) {
				cfg.BothMarker = args[i+1]
				i++
			}
		}
	}

	return cfg
}

func TestRunDiffWithFiltering(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test1.json")
	file2 := filepath.Join(tmpDir, "test2.json")

	json1 := `{
		"name": "Alice",
		"age": 30,
		"email": "alice@example.com"
	}`

	json2 := `{
		"name": "Bob",
		"age": 31,
		"email": "bob@example.com"
	}`

	if err := os.WriteFile(file1, []byte(json1), 0644); err != nil {
		t.Fatalf("Failed to write test file 1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(json2), 0644); err != nil {
		t.Fatalf("Failed to write test file 2: %v", err)
	}

	// Create config with include fields
	cfg := &CLIConfig{
		IncludeFields: []string{"name", "email"},
		ExcludeFields: []string{},
		ContextLines:  3,
		SortJSON:      false,
		SideBySide:    false,
		BothMarker:    "Both",
		ColorMode:     "never",
	}

	// Run the diff with captured output
	var buf bytes.Buffer
	err := executeDiff(cfg, file1, file2, &buf)
	if err != nil {
		t.Fatalf("executeDiff failed: %v", err)
	}
	output := buf.String()

	// Check that output contains expected fields
	if !strings.Contains(output, "name") {
		t.Error("Output should contain 'name' field")
	}

	if !strings.Contains(output, "email") {
		t.Error("Output should contain 'email' field")
	}

	// Check that age is marked as ignored (with ~)
	if !strings.Contains(output, "~") {
		t.Error("Output should contain ~ prefix for ignored fields")
	}
}

func TestConfigFileLoading(t *testing.T) {
	// Test that config file with ignored colors loads correctly
	configJSON := `{
		"version": 1,
		"colors": {
			"add": {
				"foreground": {
					"line": {"hex": "#00ff00", "ansi256": 10, "ansi": 10}
				}
			},
			"remove": {
				"foreground": {
					"line": {"hex": "#ff0000", "ansi256": 9, "ansi": 9}
				}
			},
			"ignored": {
				"foreground": {"hex": "#0000ff", "ansi256": 12, "ansi": 12}
			}
		}
	}`

	// Create temp config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(configFile, []byte(configJSON), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Colors.Ignored.Foreground.Hex != "#0000ff" {
		t.Errorf("Expected ignored color hex #0000ff, got %s", config.Colors.Ignored.Foreground.Hex)
	}
}

func TestCustomMarkerFlags(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *CLIConfig
		file1Path     string
		file2Path     string
		expectedFile1 string
		expectedFile2 string
		expectedBoth  string
	}{
		{
			name: "Default markers use filenames",
			cfg: &CLIConfig{
				File1Marker:  "",
				File2Marker:  "",
				BothMarker:   "Both",
				ContextLines: 3,
				ColorMode:    "never",
			},
			file1Path:     "test1.json",
			file2Path:     "test2.json",
			expectedFile1: "test1.json",
			expectedFile2: "test2.json",
			expectedBoth:  "Both",
		},
		{
			name: "Custom file1 marker",
			cfg: &CLIConfig{
				File1Marker:  "API",
				File2Marker:  "",
				BothMarker:   "Both",
				ContextLines: 3,
				ColorMode:    "never",
			},
			file1Path:     "file1.json",
			file2Path:     "file2.json",
			expectedFile1: "API",
			expectedFile2: "file2.json",
			expectedBoth:  "Both",
		},
		{
			name: "Custom file2 marker",
			cfg: &CLIConfig{
				File1Marker:  "",
				File2Marker:  "Config",
				BothMarker:   "Both",
				ContextLines: 3,
				ColorMode:    "never",
			},
			file1Path:     "file1.json",
			file2Path:     "file2.json",
			expectedFile1: "file1.json",
			expectedFile2: "Config",
			expectedBoth:  "Both",
		},
		{
			name: "Custom both marker",
			cfg: &CLIConfig{
				File1Marker:  "",
				File2Marker:  "",
				BothMarker:   "EQUAL",
				ContextLines: 3,
				ColorMode:    "never",
			},
			file1Path:     "data1.json",
			file2Path:     "data2.json",
			expectedFile1: "data1.json",
			expectedFile2: "data2.json",
			expectedBoth:  "EQUAL",
		},
		{
			name: "All custom markers",
			cfg: &CLIConfig{
				File1Marker:  "SRC",
				File2Marker:  "DST",
				BothMarker:   "*",
				ContextLines: 3,
				ColorMode:    "never",
			},
			file1Path:     "test.json",
			file2Path:     "test2.json",
			expectedFile1: "SRC",
			expectedFile2: "DST",
			expectedBoth:  "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp files
			tmpDir := t.TempDir()
			file1 := filepath.Join(tmpDir, tt.file1Path)
			file2 := filepath.Join(tmpDir, tt.file2Path)

			json1 := `{"test": "value1"}`
			json2 := `{"test": "value2"}`

			if err := os.WriteFile(file1, []byte(json1), 0644); err != nil {
				t.Fatalf("Failed to write test file 1: %v", err)
			}
			if err := os.WriteFile(file2, []byte(json2), 0644); err != nil {
				t.Fatalf("Failed to write test file 2: %v", err)
			}

			// Run executeDiff and capture output
			var buf bytes.Buffer
			err := executeDiff(tt.cfg, file1, file2, &buf)
			if err != nil {
				t.Fatalf("executeDiff failed: %v", err)
			}
			output := buf.String()

			// Check that expected markers appear in output
			if !strings.Contains(output, tt.expectedFile1) {
				t.Errorf("Expected output to contain file1 marker %q", tt.expectedFile1)
			}
			if !strings.Contains(output, tt.expectedFile2) {
				t.Errorf("Expected output to contain file2 marker %q", tt.expectedFile2)
			}
		})
	}
}

func TestShouldUseColor(t *testing.T) {
	tests := []struct {
		mode     string
		expected bool
	}{
		{"always", true},
		{"never", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			// Note: "auto" depends on terminal state, so we skip it
			if tt.mode == "auto" {
				return
			}
			result := shouldUseColor(tt.mode)
			if result != tt.expected {
				t.Errorf("shouldUseColor(%q) = %v, want %v", tt.mode, result, tt.expected)
			}
		})
	}
}
