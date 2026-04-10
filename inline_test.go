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
	"testing"
)

func TestEnhanceDiffsWithInlineChanges(t *testing.T) {
	tests := []struct {
		name  string
		diffs []DiffLine
		want  []DiffLine
	}{
		{
			name: "adjacent matching lines",
			diffs: []DiffLine{
				{Type: DiffTypeRemoved, Content: `  "name": "Alice"`},
				{Type: DiffTypeAdded, Content: `  "name": "Bob"`},
			},
			want: []DiffLine{
				{Type: DiffTypeRemoved, Content: `  "name": "Alice"`, InlineStart: 11, InlineEnd: 16},
				{Type: DiffTypeAdded, Content: `  "name": "Bob"`, InlineStart: 11, InlineEnd: 14},
			},
		},
		{
			name: "non-adjacent matching lines with same key",
			diffs: []DiffLine{
				{Type: DiffTypeRemoved, Content: `  "name": "Alice"`},
				{Type: DiffTypeRemoved, Content: `  "age": 30`},
				{Type: DiffTypeAdded, Content: `  "name": "Bob"`},
			},
			want: []DiffLine{
				{Type: DiffTypeRemoved, Content: `  "name": "Alice"`, InlineStart: 11, InlineEnd: 16},
				{Type: DiffTypeAdded, Content: `  "name": "Bob"`, InlineStart: 11, InlineEnd: 14},
				{Type: DiffTypeRemoved, Content: `  "age": 30`},
			},
		},
		{
			name: "multiple key matches",
			diffs: []DiffLine{
				{Type: DiffTypeRemoved, Content: `  "name": "Alice"`},
				{Type: DiffTypeRemoved, Content: `  "city": "NYC"`},
				{Type: DiffTypeAdded, Content: `  "city": "LA"`},
				{Type: DiffTypeAdded, Content: `  "name": "Bob"`},
			},
			want: []DiffLine{
				{Type: DiffTypeRemoved, Content: `  "name": "Alice"`, InlineStart: 11, InlineEnd: 16},
				{Type: DiffTypeAdded, Content: `  "name": "Bob"`, InlineStart: 11, InlineEnd: 14},
				{Type: DiffTypeRemoved, Content: `  "city": "NYC"`, InlineStart: 11, InlineEnd: 14},
				{Type: DiffTypeAdded, Content: `  "city": "LA"`, InlineStart: 11, InlineEnd: 13},
			},
		},
		{
			name: "no key match",
			diffs: []DiffLine{
				{Type: DiffTypeRemoved, Content: `  "name": "Alice"`},
				{Type: DiffTypeAdded, Content: `  "age": 30`},
			},
			want: []DiffLine{
				{Type: DiffTypeRemoved, Content: `  "name": "Alice"`},
				{Type: DiffTypeAdded, Content: `  "age": 30`},
			},
		},
		{
			name: "lines too dissimilar",
			diffs: []DiffLine{
				{Type: DiffTypeRemoved, Content: `  "name": "A"`},
				{Type: DiffTypeAdded, Content: `  "name": "VeryLongNameThatExceedsThreshold"`},
			},
			want: []DiffLine{
				{Type: DiffTypeRemoved, Content: `  "name": "A"`},
				{Type: DiffTypeAdded, Content: `  "name": "VeryLongNameThatExceedsThreshold"`},
			},
		},
		{
			name: "equal lines unchanged",
			diffs: []DiffLine{
				{Type: DiffTypeEqual, Content: `  "unchanged": true`},
				{Type: DiffTypeRemoved, Content: `  "name": "Alice"`},
				{Type: DiffTypeAdded, Content: `  "name": "Bob"`},
			},
			want: []DiffLine{
				{Type: DiffTypeEqual, Content: `  "unchanged": true`},
				{Type: DiffTypeRemoved, Content: `  "name": "Alice"`, InlineStart: 11, InlineEnd: 16},
				{Type: DiffTypeAdded, Content: `  "name": "Bob"`, InlineStart: 11, InlineEnd: 14},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnhanceDiffsWithInlineChanges(tt.diffs)

			if len(got) != len(tt.want) {
				t.Errorf("EnhanceDiffsWithInlineChanges() returned %d lines, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].Type != tt.want[i].Type ||
					got[i].Content != tt.want[i].Content ||
					got[i].InlineStart != tt.want[i].InlineStart ||
					got[i].InlineEnd != tt.want[i].InlineEnd {
					t.Errorf("EnhanceDiffsWithInlineChanges() line %d =\n  got:  {Type:%v, Content:%q, InlineStart:%d, InlineEnd:%d}\n  want: {Type:%v, Content:%q, InlineStart:%d, InlineEnd:%d}",
						i,
						got[i].Type, got[i].Content, got[i].InlineStart, got[i].InlineEnd,
						tt.want[i].Type, tt.want[i].Content, tt.want[i].InlineStart, tt.want[i].InlineEnd)
				}
			}
		})
	}
}

func TestExtractJSONKeyForInline(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "simple key-value",
			line: `  "name": "value"`,
			want: "name",
		},
		{
			name: "key with spaces",
			line: `    "field": 123`,
			want: "field",
		},
		{
			name: "no quotes",
			line: `  field: value`,
			want: "",
		},
		{
			name: "no colon",
			line: `  "field"`,
			want: "",
		},
		{
			name: "empty line",
			line: ``,
			want: "",
		},
		{
			name: "just bracket",
			line: `  {`,
			want: "",
		},
		{
			name: "nested key",
			line: `    "user.name": "Alice"`,
			want: "user.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONKeyForInline(tt.line)
			if got != tt.want {
				t.Errorf("extractJSONKeyForInline(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}
