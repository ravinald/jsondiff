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

func TestFieldFiltering(t *testing.T) {
	tests := []struct {
		name          string
		json1         string
		json2         string
		includeFields []string
		excludeFields []string
		expectIgnored bool
	}{
		{
			name: "Include specific fields",
			json1: `{
				"name": "Alice",
				"age": 30,
				"city": "NYC"
			}`,
			json2: `{
				"name": "Bob",
				"age": 30,
				"city": "LA"
			}`,
			includeFields: []string{"name", "city"},
			expectIgnored: true, // age field will be ignored
		},
		{
			name: "Exclude specific fields",
			json1: `{
				"name": "Alice",
				"timestamp": "2023-01-01",
				"data": "value1"
			}`,
			json2: `{
				"name": "Bob",
				"timestamp": "2023-01-02",
				"data": "value2"
			}`,
			excludeFields: []string{"timestamp"},
			expectIgnored: true, // timestamp field will be ignored
		},
		{
			name: "Include nested fields",
			json1: `{
				"user": {
					"name": "Alice",
					"email": "alice@example.com"
				},
				"metadata": {
					"created": "2023-01-01"
				}
			}`,
			json2: `{
				"user": {
					"name": "Bob",
					"email": "bob@example.com"
				},
				"metadata": {
					"created": "2023-01-02"
				}
			}`,
			includeFields: []string{"user.name"},
			expectIgnored: true, // email and metadata fields will be ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DiffOptions{
				ContextLines:  3,
				IncludeFields: tt.includeFields,
				ExcludeFields: tt.excludeFields,
			}

			diffs, err := Diff([]byte(tt.json1), []byte(tt.json2), opts)
			if err != nil {
				t.Fatalf("Diff failed: %v", err)
			}

			// Check if we have any ignored fields
			hasIgnored := false
			for _, diff := range diffs {
				if diff.IsIgnored || strings.HasPrefix(diff.Content, "~") {
					hasIgnored = true
					break
				}
			}

			if hasIgnored != tt.expectIgnored {
				t.Errorf("Expected ignored fields: %v, got: %v", tt.expectIgnored, hasIgnored)
			}
		})
	}
}

func TestFieldPattern(t *testing.T) {
	tests := []struct {
		fieldPath string
		pattern   string
		expected  bool
	}{
		{"name", "name", true},
		{"user.name", "user.name", true},
		{"user.name", "user", true},
		{"user", "user.name", false}, // Parent doesn't match child pattern
		{"user.email", "user", true},
		{"address.city", "address", true},
		{"name", "age", false},
		{"user.name", "address", false},
	}

	for _, tt := range tests {
		t.Run(tt.fieldPath+"_"+tt.pattern, func(t *testing.T) {
			result := matchesFieldPattern(tt.fieldPath, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesFieldPattern(%q, %q) = %v, want %v",
					tt.fieldPath, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestShouldIncludeField(t *testing.T) {
	tests := []struct {
		name          string
		fieldPath     string
		includeFields []string
		excludeFields []string
		expected      bool
	}{
		{
			name:      "No filters - include all",
			fieldPath: "any.field",
			expected:  true,
		},
		{
			name:          "Include list - field in list",
			fieldPath:     "user.name",
			includeFields: []string{"user.name", "user.email"},
			expected:      true,
		},
		{
			name:          "Include list - field not in list",
			fieldPath:     "user.age",
			includeFields: []string{"user.name", "user.email"},
			expected:      false,
		},
		{
			name:          "Exclude list - field in list",
			fieldPath:     "timestamp",
			excludeFields: []string{"timestamp", "metadata"},
			expected:      false,
		},
		{
			name:          "Exclude list - field not in list",
			fieldPath:     "name",
			excludeFields: []string{"timestamp", "metadata"},
			expected:      true,
		},
		{
			name:          "Both lists - included but excluded",
			fieldPath:     "user.internal",
			includeFields: []string{"user"},
			excludeFields: []string{"user.internal"},
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIncludeField(tt.fieldPath, tt.includeFields, tt.excludeFields)
			if result != tt.expected {
				t.Errorf("shouldIncludeField(%q) = %v, want %v",
					tt.fieldPath, result, tt.expected)
			}
		})
	}
}

func TestDiffWithComplexFiltering(t *testing.T) {
	tests := []struct {
		name          string
		json1         string
		json2         string
		includeFields []string
		excludeFields []string
		description   string
	}{
		{
			name: "Deeply nested fields",
			json1: `{
				"level1": {
					"level2": {
						"level3": {
							"target": "value1",
							"ignore": "data1"
						}
					}
				}
			}`,
			json2: `{
				"level1": {
					"level2": {
						"level3": {
							"target": "value2",
							"ignore": "data2"
						}
					}
				}
			}`,
			includeFields: []string{"level1.level2.level3.target"},
			description:   "Should only compare deeply nested target field",
		},
		{
			name: "Array handling",
			json1: `{
				"items": [
					{"id": 1, "name": "item1"},
					{"id": 2, "name": "item2"}
				],
				"metadata": "v1"
			}`,
			json2: `{
				"items": [
					{"id": 1, "name": "item1-changed"},
					{"id": 2, "name": "item2"}
				],
				"metadata": "v2"
			}`,
			includeFields: []string{"items"},
			description:   "Should handle arrays correctly (metadata excluded)",
		},
		{
			name:          "Empty include list",
			json1:         `{"a": 1, "b": 2}`,
			json2:         `{"a": 2, "b": 3}`,
			includeFields: []string{},
			excludeFields: []string{},
			description:   "Should include all fields when no filters",
		},
		{
			name: "Wildcard-like parent inclusion",
			json1: `{
				"user": {
					"profile": {"name": "Alice", "age": 30},
					"settings": {"theme": "dark"}
				}
			}`,
			json2: `{
				"user": {
					"profile": {"name": "Bob", "age": 31},
					"settings": {"theme": "light"}
				}
			}`,
			includeFields: []string{"user.profile"},
			description:   "Should include all fields under user.profile",
		},
		{
			name: "Mixed types",
			json1: `{
				"string": "value",
				"number": 42,
				"boolean": true,
				"null": null,
				"array": [1, 2, 3],
				"object": {"nested": "value"}
			}`,
			json2: `{
				"string": "changed",
				"number": 43,
				"boolean": false,
				"null": null,
				"array": [1, 2, 4],
				"object": {"nested": "changed"}
			}`,
			excludeFields: []string{"boolean", "null"},
			description:   "Should handle different JSON types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DiffOptions{
				ContextLines:  3,
				IncludeFields: tt.includeFields,
				ExcludeFields: tt.excludeFields,
			}

			diffs, err := Diff([]byte(tt.json1), []byte(tt.json2), opts)
			if err != nil {
				t.Fatalf("Diff failed: %v", err)
			}

			// Basic validation that diff was created
			// For filtered diffs, we may have no output if only ignored fields differ
			if len(diffs) == 0 && tt.json1 != tt.json2 && len(tt.includeFields) == 0 && len(tt.excludeFields) == 0 {
				t.Error("Expected diff output for different JSONs when no filters applied")
			}

			// Log the test description for clarity
			t.Logf("Test: %s", tt.description)
		})
	}
}

func TestBuildFilteredDiff(t *testing.T) {
	tests := []struct {
		name      string
		lines1    []string
		lines2    []string
		expectLen int
	}{
		{
			name: "All regular lines",
			lines1: []string{
				`"name": "Alice"`,
				`"age": 30`,
			},
			lines2: []string{
				`"name": "Bob"`,
				`"age": 31`,
			},
			expectLen: 4, // 2 removals + 2 additions
		},
		{
			name: "Mix of regular and ignored",
			lines1: []string{
				`"name": "Alice"`,
				`"~timestamp": "2023-01-01"`,
			},
			lines2: []string{
				`"name": "Bob"`,
				`"~timestamp": "2023-01-02"`,
			},
			expectLen: 3, // 1 removal + 1 addition + 1 ignored
		},
		{
			name: "All ignored lines",
			lines1: []string{
				`"~field1": "value1"`,
				`"~field2": "value2"`,
			},
			lines2: []string{
				`"~field1": "value1"`,
				`"~field2": "value2"`,
			},
			expectLen: 2, // 2 ignored lines shown as equal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := buildFilteredDiff(tt.lines1, tt.lines2)

			if len(diffs) < tt.expectLen {
				t.Errorf("Expected at least %d diff lines, got %d", tt.expectLen, len(diffs))
			}

			// Check that ignored lines are properly marked
			for _, diff := range diffs {
				if strings.Contains(diff.Content, "~") && !diff.IsIgnored {
					t.Error("Line with ~ should be marked as ignored")
				}
			}
		})
	}
}

func TestContainsIgnoredField(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{`"~field": "value"`, true},
		{`  "~nested": {`, true},
		{`~"field": "value"`, true},
		{`"field": "value"`, false},
		{`"field": "~value"`, false}, // ~ in value, not field name
		{`  "normalField": "value"`, false},
		{`}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := containsIgnoredField(tt.line)
			if result != tt.expected {
				t.Errorf("containsIgnoredField(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}
