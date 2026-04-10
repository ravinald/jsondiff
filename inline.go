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

import "strings"

const (
	similarityLengthRatio = 0.5
	similarityCharRatio   = 0.3
)

func EnhanceDiffsWithInlineChanges(diffs []DiffLine) []DiffLine {
	result := make([]DiffLine, 0, len(diffs))
	processedIndices := make(map[int]bool)

	for i := range len(diffs) {
		if processedIndices[i] {
			continue
		}

		if diffs[i].Type == DiffTypeRemoved {
			// Extract the JSON key from the removed line
			removedKey := extractJSONKeyForInline(diffs[i].Content)

			// Look for a matching added line with the same key
			matchFound := false
			if removedKey != "" {
				for j := i + 1; j < len(diffs); j++ {
					if processedIndices[j] {
						continue
					}

					if diffs[j].Type == DiffTypeAdded {
						addedKey := extractJSONKeyForInline(diffs[j].Content)

						// If keys match and lines are similar, apply inline changes
						if removedKey == addedKey && similarLines(diffs[i].Content, diffs[j].Content) {
							start1, end1, start2, end2 := FindInlineChanges(diffs[i].Content, diffs[j].Content)

							removedDiff := diffs[i]
							removedDiff.InlineStart = start1
							removedDiff.InlineEnd = end1

							addedDiff := diffs[j]
							addedDiff.InlineStart = start2
							addedDiff.InlineEnd = end2

							result = append(result, removedDiff, addedDiff)
							processedIndices[i] = true
							processedIndices[j] = true
							matchFound = true
							break
						}
					}
				}
			}

			if !matchFound {
				result = append(result, diffs[i])
				processedIndices[i] = true
			}
		} else {
			result = append(result, diffs[i])
			processedIndices[i] = true
		}
	}

	return result
}

// extractJSONKeyForInline extracts the JSON key from a line for inline diff matching
func extractJSONKeyForInline(line string) string {
	trimmed := strings.TrimSpace(line)

	// Handle lines that start with quotes
	if !strings.HasPrefix(trimmed, "\"") {
		return ""
	}

	// Find the closing quote and colon
	colonIndex := strings.Index(trimmed, "\":")
	if colonIndex > 0 {
		// Extract key without the opening quote
		return trimmed[1:colonIndex]
	}

	return ""
}

func similarLines(line1, line2 string) bool {
	if len(line1) == 0 || len(line2) == 0 {
		return false
	}

	maxLen := max(len(line1), len(line2))
	minLen := min(len(line1), len(line2))

	if float64(minLen)/float64(maxLen) < similarityLengthRatio {
		return false
	}

	commonChars := 0
	for i := range minLen {
		if i < len(line1) && i < len(line2) && line1[i] == line2[i] {
			commonChars++
		}
	}

	return float64(commonChars)/float64(maxLen) > similarityCharRatio
}
