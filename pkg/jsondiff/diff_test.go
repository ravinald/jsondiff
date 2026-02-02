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
	"context"
	"encoding/json"
	"testing"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name     string
		json1    string
		json2    string
		opts     DiffOptions
		wantDiff bool
	}{
		{
			name:     "identical JSONs",
			json1:    `{"key": "value"}`,
			json2:    `{"key": "value"}`,
			opts:     DiffOptions{ContextLines: 3},
			wantDiff: false,
		},
		{
			name:     "different values",
			json1:    `{"key": "value1"}`,
			json2:    `{"key": "value2"}`,
			opts:     DiffOptions{ContextLines: 3},
			wantDiff: true,
		},
		{
			name:     "added key",
			json1:    `{"key1": "value1"}`,
			json2:    `{"key1": "value1", "key2": "value2"}`,
			opts:     DiffOptions{ContextLines: 3},
			wantDiff: true,
		},
		{
			name:     "removed key",
			json1:    `{"key1": "value1", "key2": "value2"}`,
			json2:    `{"key1": "value1"}`,
			opts:     DiffOptions{ContextLines: 3},
			wantDiff: true,
		},
		{
			name:     "sorted comparison",
			json1:    `{"b": 2, "a": 1}`,
			json2:    `{"a": 1, "b": 2}`,
			opts:     DiffOptions{ContextLines: 3, SortJSON: true},
			wantDiff: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs, err := Diff([]byte(tt.json1), []byte(tt.json2), tt.opts)
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}

			hasDiff := false
			for _, diff := range diffs {
				if diff.Type != DiffTypeEqual {
					hasDiff = true
					break
				}
			}

			if hasDiff != tt.wantDiff {
				t.Errorf("Diff() hasDiff = %v, want %v", hasDiff, tt.wantDiff)
			}
		})
	}
}

func TestFindInlineChanges(t *testing.T) {
	tests := []struct {
		name                 string
		line1, line2         string
		wantStart1, wantEnd1 int
		wantStart2, wantEnd2 int
	}{
		{
			name:       "identical lines",
			line1:      "same content",
			line2:      "same content",
			wantStart1: -1, wantEnd1: -1,
			wantStart2: -1, wantEnd2: -1,
		},
		{
			name:       "middle change",
			line1:      `"key": "value1"`,
			line2:      `"key": "value2"`,
			wantStart1: 13, wantEnd1: 14,
			wantStart2: 13, wantEnd2: 14,
		},
		{
			name:       "prefix change",
			line1:      `old_value`,
			line2:      `new_value`,
			wantStart1: 0, wantEnd1: 3,
			wantStart2: 0, wantEnd2: 3,
		},
		{
			name:       "suffix change",
			line1:      `value_old`,
			line2:      `value_new`,
			wantStart1: 6, wantEnd1: 9,
			wantStart2: 6, wantEnd2: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start1, end1, start2, end2 := FindInlineChanges(tt.line1, tt.line2)
			if start1 != tt.wantStart1 || end1 != tt.wantEnd1 ||
				start2 != tt.wantStart2 || end2 != tt.wantEnd2 {
				t.Errorf("FindInlineChanges() = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
					start1, end1, start2, end2,
					tt.wantStart1, tt.wantEnd1, tt.wantStart2, tt.wantEnd2)
			}
		})
	}
}

func TestSortJSONKeys(t *testing.T) {
	input := map[string]interface{}{
		"z": "last",
		"a": "first",
		"m": map[string]interface{}{
			"nested_z": 1,
			"nested_a": 2,
		},
	}

	result := sortJSONKeys(input)

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	expected := `{"a":"first","m":{"nested_a":2,"nested_z":1},"z":"last"}`
	if string(jsonBytes) != expected {
		t.Errorf("sortJSONKeys() = %s, want %s", string(jsonBytes), expected)
	}
}

func TestDiffWithContext(t *testing.T) {
	tests := []struct {
		name      string
		json1     string
		json2     string
		opts      DiffOptions
		cancelled bool
		wantErr   bool
	}{
		{
			name:      "normal context succeeds",
			json1:     `{"key": "value1"}`,
			json2:     `{"key": "value2"}`,
			opts:      DiffOptions{ContextLines: 3},
			cancelled: false,
			wantErr:   false,
		},
		{
			name:      "cancelled context returns error",
			json1:     `{"key": "value1"}`,
			json2:     `{"key": "value2"}`,
			opts:      DiffOptions{ContextLines: 3},
			cancelled: true,
			wantErr:   true,
		},
		{
			name:      "cancelled context with filtering returns error",
			json1:     `{"name": "Alice", "age": 30}`,
			json2:     `{"name": "Bob", "age": 31}`,
			opts:      DiffOptions{ContextLines: 3, IncludeFields: []string{"name"}},
			cancelled: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.cancelled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel() // Cancel immediately
			}

			_, err := DiffWithContext(ctx, []byte(tt.json1), []byte(tt.json2), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("DiffWithContext() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.cancelled && err != context.Canceled {
				t.Errorf("DiffWithContext() error = %v, want context.Canceled", err)
			}
		})
	}
}
