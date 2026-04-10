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

package jsondiff

import (
	"strings"
	"testing"
)

func TestFormatterWithIgnoredFields(t *testing.T) {
	formatter := NewFormatter(DefaultStyles())

	tests := []struct {
		name         string
		diffs        []DiffLine
		expectPrefix string
		expectColor  bool
	}{
		{
			name: "Regular diff line",
			diffs: []DiffLine{
				{
					Type:     DiffTypeEqual,
					LineNum1: 1,
					LineNum2: 1,
					Content:  `"name": "Alice"`,
				},
			},
			expectPrefix: `"name": "Alice"`,
			expectColor:  false,
		},
		{
			name: "Ignored field with tilde",
			diffs: []DiffLine{
				{
					Type:      DiffTypeEqual,
					LineNum1:  2,
					LineNum2:  2,
					Content:   `"age": 30`,
					IsIgnored: true,
				},
			},
			expectPrefix: "~",
			expectColor:  true,
		},
		{
			name: "Ignored field with clean content",
			diffs: []DiffLine{
				{
					Type:      DiffTypeEqual,
					LineNum1:  3,
					LineNum2:  3,
					Content:   `"timestamp": "2023-01-01"`,
					IsIgnored: true,
				},
			},
			expectPrefix: "~",
			expectColor:  true,
		},
		{
			name: "Mixed regular and ignored",
			diffs: []DiffLine{
				{
					Type:     DiffTypeAdded,
					LineNum1: -1,
					LineNum2: 1,
					Content:  `"name": "Bob"`,
				},
				{
					Type:      DiffTypeEqual,
					LineNum1:  2,
					LineNum2:  2,
					Content:   `"id": "123"`,
					IsIgnored: true,
				},
			},
			expectPrefix: "+", // Testing first line
			expectColor:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatter.Format(tt.diffs)

			// Check for tilde prefix on ignored fields
			if tt.diffs[0].IsIgnored {
				if !strings.Contains(output, "~") {
					t.Error("Ignored field should have ~ prefix")
				}
			}

			// Check that output contains expected content
			if tt.expectPrefix != "" && !strings.Contains(output, tt.expectPrefix) {
				t.Errorf("Expected output to contain %q, got: %s", tt.expectPrefix, output)
			}

			// Note: lipgloss doesn't apply colors in non-TTY environments (like tests)
			// We can't reliably test for ANSI codes, but we can verify the output structure
		})
	}
}

func TestFormatSideBySideWithIgnored(t *testing.T) {
	formatter := NewFormatter(DefaultStyles())

	diffs := []DiffLine{
		{
			Type:     DiffTypeEqual,
			LineNum1: 1,
			LineNum2: 1,
			Content:  "{",
		},
		{
			Type:      DiffTypeEqual,
			LineNum1:  2,
			LineNum2:  2,
			Content:   `  "timestamp": "2023-01-01"`,
			IsIgnored: true,
		},
		{
			Type:     DiffTypeRemoved,
			LineNum1: 3,
			LineNum2: -1,
			Content:  `  "name": "Alice"`,
		},
		{
			Type:     DiffTypeAdded,
			LineNum1: -1,
			LineNum2: 3,
			Content:  `  "name": "Bob"`,
		},
		{
			Type:     DiffTypeEqual,
			LineNum1: 4,
			LineNum2: 4,
			Content:  "}",
		},
	}

	output := formatter.FormatSideBySide(diffs, "test1.json", "test2.json")

	// Check that ignored field appears with tilde
	if !strings.Contains(output, "~") {
		t.Error("Side-by-side output should contain ~ for ignored fields")
	}

	// Check that the output contains both sides
	if !strings.Contains(output, "|") {
		t.Error("Side-by-side output should contain | separator")
	}
}

func TestFormatterHandlesNilStyles(t *testing.T) {
	// Test that formatter works with nil styles (should use defaults)
	formatter := NewFormatter(nil)

	diffs := []DiffLine{
		{
			Type:      DiffTypeEqual,
			LineNum1:  1,
			LineNum2:  1,
			Content:   `"field": "value"`,
			IsIgnored: true,
		},
	}

	output := formatter.Format(diffs)

	if !strings.Contains(output, "~") {
		t.Error("Formatter with nil styles should still handle ignored fields")
	}
}

func TestIgnoredFieldPrefixHandling(t *testing.T) {
	formatter := NewFormatter(DefaultStyles())

	tests := []struct {
		name           string
		content        string
		isIgnored      bool
		expectedPrefix string
	}{
		{
			name:           "Add tilde to ignored field",
			content:        `"age": 30`,
			isIgnored:      true,
			expectedPrefix: "~",
		},
		{
			name:           "Don't duplicate tilde",
			content:        `"age": 30`, // Content should not have ~ in it
			isIgnored:      true,
			expectedPrefix: "~ \"age\"", // We add "~ " as line prefix
		},
		{
			name:           "No tilde for non-ignored",
			content:        `"name": "Alice"`,
			isIgnored:      false,
			expectedPrefix: `"name"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := []DiffLine{
				{
					Type:      DiffTypeEqual,
					LineNum1:  1,
					LineNum2:  1,
					Content:   tt.content,
					IsIgnored: tt.isIgnored,
				},
			}

			output := formatter.Format(diffs)

			if !strings.Contains(output, tt.expectedPrefix) {
				t.Errorf("Expected prefix %q in output, got: %s", tt.expectedPrefix, output)
			}

			// Check tilde is present for ignored fields
			if tt.isIgnored && !strings.Contains(output, "~ ") {
				t.Errorf("Expected '~ ' prefix for ignored field in output: %s", output)
			}
		})
	}
}

func TestCustomMarkers(t *testing.T) {
	tests := []struct {
		name           string
		file1Marker    string
		file2Marker    string
		bothMarker     string
		diffType       DiffType
		expectedPrefix string
	}{
		{
			name:           "Default markers from NewFormatter",
			file1Marker:    "",
			file2Marker:    "",
			bothMarker:     "",
			diffType:       DiffTypeRemoved,
			expectedPrefix: "1", // NewFormatter sets default to "1"
		},
		{
			name:           "Custom file1 marker",
			file1Marker:    "API",
			file2Marker:    "",
			bothMarker:     "",
			diffType:       DiffTypeRemoved,
			expectedPrefix: "API",
		},
		{
			name:           "Custom file2 marker",
			file1Marker:    "",
			file2Marker:    "Config",
			bothMarker:     "",
			diffType:       DiffTypeAdded,
			expectedPrefix: "Config",
		},
		{
			name:           "Default both marker from NewFormatter",
			file1Marker:    "",
			file2Marker:    "",
			bothMarker:     "",
			diffType:       DiffTypeEqual,
			expectedPrefix: "Both", // NewFormatter sets default to "Both"
		},
		{
			name:           "Custom both marker",
			file1Marker:    "",
			file2Marker:    "",
			bothMarker:     "BOTH",
			diffType:       DiffTypeEqual,
			expectedPrefix: "BOTH",
		},
		{
			name:           "All custom markers",
			file1Marker:    "SRC",
			file2Marker:    "DST",
			bothMarker:     "EQL",
			diffType:       DiffTypeEqual,
			expectedPrefix: "EQL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatterWithOptions(FormatterOptions{
				File1Marker: tt.file1Marker,
				File2Marker: tt.file2Marker,
				BothMarker:  tt.bothMarker,
			})

			marker := formatter.getFileMarker(tt.diffType)
			if marker != tt.expectedPrefix {
				t.Errorf("Expected marker %q, got %q", tt.expectedPrefix, marker)
			}
		})
	}
}

func TestMarkerPadding(t *testing.T) {
	tests := []struct {
		name        string
		file1Marker string
		file2Marker string
		bothMarker  string
		expectedLen int
	}{
		{
			name:        "Equal length markers",
			file1Marker: "AAA",
			file2Marker: "BBB",
			bothMarker:  "CCC",
			expectedLen: 3,
		},
		{
			name:        "Different length markers",
			file1Marker: "A",
			file2Marker: "LONGER",
			bothMarker:  "BB",
			expectedLen: 6,
		},
		{
			name:        "Both marker longest",
			file1Marker: "1",
			file2Marker: "2",
			bothMarker:  "EQUAL",
			expectedLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatterWithOptions(FormatterOptions{
				File1Marker: tt.file1Marker,
				File2Marker: tt.file2Marker,
				BothMarker:  tt.bothMarker,
			})

			// Test padding for each marker
			padded1 := formatter.padMarker(tt.file1Marker)
			padded2 := formatter.padMarker(tt.file2Marker)
			paddedB := formatter.padMarker(tt.bothMarker)

			if len(padded1) != tt.expectedLen {
				t.Errorf("Expected padded length %d for file1 marker, got %d", tt.expectedLen, len(padded1))
			}
			if len(padded2) != tt.expectedLen {
				t.Errorf("Expected padded length %d for file2 marker, got %d", tt.expectedLen, len(padded2))
			}
			if len(paddedB) != tt.expectedLen {
				t.Errorf("Expected padded length %d for both marker, got %d", tt.expectedLen, len(paddedB))
			}

			// Check right justification
			if !strings.HasSuffix(padded1, tt.file1Marker) {
				t.Errorf("Marker should be right-justified: %q", padded1)
			}
			if !strings.HasSuffix(padded2, tt.file2Marker) {
				t.Errorf("Marker should be right-justified: %q", padded2)
			}
			if !strings.HasSuffix(paddedB, tt.bothMarker) {
				t.Errorf("Marker should be right-justified: %q", paddedB)
			}
		})
	}
}

func TestFormatterWithCustomMarkers(t *testing.T) {
	formatter := NewFormatterWithOptions(FormatterOptions{
		File1Marker: "FILE1",
		File2Marker: "FILE2",
		BothMarker:  "*",
	})

	diffs := []DiffLine{
		{
			Type:     DiffTypeEqual,
			LineNum1: 1,
			LineNum2: 1,
			Content:  "{",
		},
		{
			Type:     DiffTypeRemoved,
			LineNum1: 2,
			LineNum2: -1,
			Content:  `"old": "value"`,
		},
		{
			Type:     DiffTypeAdded,
			LineNum1: -1,
			LineNum2: 2,
			Content:  `"new": "value"`,
		},
	}

	output := formatter.Format(diffs)

	// Check that custom markers appear in output
	if !strings.Contains(output, "FILE1") {
		t.Error("Expected FILE1 marker in output")
	}
	if !strings.Contains(output, "FILE2") {
		t.Error("Expected FILE2 marker in output")
	}
	if !strings.Contains(output, "*") {
		t.Error("Expected * marker in output")
	}

	// Check padding - FILE1 and FILE2 are both 5 chars, so * should be padded to 5
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" || line == "..." {
			continue
		}
		// First 5 chars should be the padded marker
		if len(line) >= 5 {
			markerPart := line[:5]
			// Should end with one of our markers
			if !strings.HasSuffix(markerPart, "*") &&
				!strings.HasSuffix(markerPart, "FILE1") &&
				!strings.HasSuffix(markerPart, "FILE2") {
				t.Logf("Line doesn't have expected marker format: %q", line)
			}
		}
	}
}

func TestNewFormatterWithOptions(t *testing.T) {
	tests := []struct {
		name           string
		opts           FormatterOptions
		diffType       DiffType
		expectedMarker string
	}{
		{
			name:           "Empty options uses defaults",
			opts:           FormatterOptions{},
			diffType:       DiffTypeRemoved,
			expectedMarker: "1",
		},
		{
			name: "Custom file1 marker",
			opts: FormatterOptions{
				File1Marker: "LEFT",
			},
			diffType:       DiffTypeRemoved,
			expectedMarker: "LEFT",
		},
		{
			name: "Custom file2 marker",
			opts: FormatterOptions{
				File2Marker: "RIGHT",
			},
			diffType:       DiffTypeAdded,
			expectedMarker: "RIGHT",
		},
		{
			name: "Custom both marker",
			opts: FormatterOptions{
				BothMarker: "SAME",
			},
			diffType:       DiffTypeEqual,
			expectedMarker: "SAME",
		},
		{
			name: "All custom markers",
			opts: FormatterOptions{
				File1Marker: "OLD",
				File2Marker: "NEW",
				BothMarker:  "EQ",
			},
			diffType:       DiffTypeEqual,
			expectedMarker: "EQ",
		},
		{
			name: "With styles",
			opts: FormatterOptions{
				Styles:      DefaultStyles(),
				File1Marker: "SRC",
			},
			diffType:       DiffTypeRemoved,
			expectedMarker: "SRC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatterWithOptions(tt.opts)

			marker := formatter.getFileMarker(tt.diffType)
			if marker != tt.expectedMarker {
				t.Errorf("Expected marker %q, got %q", tt.expectedMarker, marker)
			}
		})
	}
}

func TestNewFormatterWithOptionsOutput(t *testing.T) {
	formatter := NewFormatterWithOptions(FormatterOptions{
		File1Marker: "API",
		File2Marker: "LOCAL",
		BothMarker:  "MATCH",
	})

	diffs := []DiffLine{
		{
			Type:     DiffTypeEqual,
			LineNum1: 1,
			LineNum2: 1,
			Content:  "{",
		},
		{
			Type:     DiffTypeRemoved,
			LineNum1: 2,
			LineNum2: -1,
			Content:  `"version": "1.0"`,
		},
		{
			Type:     DiffTypeAdded,
			LineNum1: -1,
			LineNum2: 2,
			Content:  `"version": "2.0"`,
		},
	}

	output := formatter.Format(diffs)

	// Check that custom markers appear in output
	if !strings.Contains(output, "API") {
		t.Error("Expected API marker in output")
	}
	if !strings.Contains(output, "LOCAL") {
		t.Error("Expected LOCAL marker in output")
	}
	if !strings.Contains(output, "MATCH") {
		t.Error("Expected MATCH marker in output")
	}
}
