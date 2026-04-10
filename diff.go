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

// Package jsondiff provides functionality for comparing JSON files
// and displaying differences with syntax highlighting.
//
// Features:
//   - Line-by-line and inline difference highlighting
//   - Customizable context lines around changes
//   - JSON key sorting before comparison
//   - Field inclusion/exclusion filtering
//   - Side-by-side diff formatting
//   - Configurable color schemes via lipgloss
//
// Example usage:
//
//	json1 := []byte(`{"name": "Alice", "age": 30}`)
//	json2 := []byte(`{"name": "Bob", "age": 30}`)
//
//	opts := jsondiff.DiffOptions{
//	    ContextLines: 3,
//	    SortJSON:     true,
//	}
//
//	diffs, err := jsondiff.Diff(json1, json2, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	formatter := jsondiff.NewFormatter(jsondiff.DefaultStyles())
//	fmt.Print(formatter.Format(diffs))
package jsondiff

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strings"
)

type DiffType int

const (
	DiffTypeEqual DiffType = iota
	DiffTypeAdded
	DiffTypeRemoved
)

const (
	defaultContextLines = 3
)

// Differ compares two JSON documents and returns differences.
type Differ interface {
	Diff(json1, json2 []byte, opts DiffOptions) ([]DiffLine, error)
}

// DiffFormatter formats diff output for display.
type DiffFormatter interface {
	Format(diffs []DiffLine) string
	FormatSideBySide(diffs []DiffLine, marker1, marker2 string) string
}

type DiffLine struct {
	Type        DiffType
	LineNum1    int
	LineNum2    int
	Content     string
	InlineStart int
	InlineEnd   int
	IsIgnored   bool // not included in comparison
}

type DiffOptions struct {
	ContextLines  int
	SortJSON      bool
	IncludeFields []string // empty = all fields
	ExcludeFields []string
}

// Diff compares two JSON documents and returns their differences.
// This is a convenience wrapper around DiffWithContext using context.Background().
func Diff(json1, json2 []byte, opts DiffOptions) ([]DiffLine, error) {
	return DiffWithContext(context.Background(), json1, json2, opts)
}

// DiffWithContext compares two JSON documents with cancellation support.
// The context is checked before expensive operations to allow early termination.
func DiffWithContext(ctx context.Context, json1, json2 []byte, opts DiffOptions) ([]DiffLine, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// If filtering is enabled, process the JSON objects
	if len(opts.IncludeFields) > 0 || len(opts.ExcludeFields) > 0 {
		return diffWithFilteringContext(ctx, json1, json2, opts)
	}

	// Original behavior for no filtering
	var lines1, lines2 []string
	var err error

	if opts.SortJSON {
		lines1, err = formatJSONLines(json1, true)
		if err != nil {
			return nil, fmt.Errorf("error formatting json1: %w", err)
		}
		lines2, err = formatJSONLines(json2, true)
		if err != nil {
			return nil, fmt.Errorf("error formatting json2: %w", err)
		}
	} else {
		lines1, err = formatJSONLines(json1, false)
		if err != nil {
			return nil, fmt.Errorf("error formatting json1: %w", err)
		}
		lines2, err = formatJSONLines(json2, false)
		if err != nil {
			return nil, fmt.Errorf("error formatting json2: %w", err)
		}
	}

	// Check context before LCS computation (potentially expensive)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	lcs := longestCommonSubsequence(lines1, lines2)
	diffs := buildDiff(lines1, lines2, lcs)

	return addContext(diffs, opts.ContextLines), nil
}

func diffWithFilteringContext(ctx context.Context, json1, json2 []byte, opts DiffOptions) ([]DiffLine, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var obj1, obj2 any
	if err := json.Unmarshal(json1, &obj1); err != nil {
		return nil, fmt.Errorf("error parsing json1: %w", err)
	}
	if err := json.Unmarshal(json2, &obj2); err != nil {
		return nil, fmt.Errorf("error parsing json2: %w", err)
	}

	// Mark ignored fields internally (with ~ prefix for tracking)
	marked1, _ := markIgnoredFieldsInObjectSimple(obj1, opts.IncludeFields, opts.ExcludeFields, "")
	marked2, _ := markIgnoredFieldsInObjectSimple(obj2, opts.IncludeFields, opts.ExcludeFields, "")

	// Sort if needed
	if opts.SortJSON {
		marked1 = sortJSONKeys(marked1)
		marked2 = sortJSONKeys(marked2)
	}

	// Format to lines
	lines1, err := formatJSONObject(marked1)
	if err != nil {
		return nil, fmt.Errorf("error formatting json1: %w", err)
	}
	lines2, err := formatJSONObject(marked2)
	if err != nil {
		return nil, fmt.Errorf("error formatting json2: %w", err)
	}

	// Check context before building diff
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Build special diff that handles ignored lines
	diffs := buildFilteredDiff(lines1, lines2)

	return addContext(diffs, opts.ContextLines), nil
}

// markIgnoredFieldsInObjectSimple marks ignored fields with ~ prefix for internal tracking
func markIgnoredFieldsInObjectSimple(obj any, includeFields, excludeFields []string, currentPath string) (any, map[string]bool) {
	ignoredPaths := make(map[string]bool)

	// If no filters specified, return as-is
	if len(includeFields) == 0 && len(excludeFields) == 0 {
		return obj, ignoredPaths
	}

	switch val := obj.(type) {
	case map[string]any:
		result := make(map[string]any)

		for key, value := range val {
			fieldPath := key
			if currentPath != "" {
				fieldPath = currentPath + "." + key
			}

			if !shouldIncludeField(fieldPath, includeFields, excludeFields) {
				// Mark with ~ for internal tracking only
				ignoredPaths[fieldPath] = true
				result["~"+key] = value
			} else {
				// Recursively process nested objects
				nestedObj, nestedIgnored := markIgnoredFieldsInObjectSimple(value, includeFields, excludeFields, fieldPath)
				result[key] = nestedObj
				// Merge nested ignored paths
				maps.Copy(ignoredPaths, nestedIgnored)
			}
		}

		return result, ignoredPaths

	case []any:
		// Process array elements
		result := make([]any, len(val))
		for i, item := range val {
			arrayPath := fmt.Sprintf("%s[%d]", currentPath, i)
			nestedObj, nestedIgnored := markIgnoredFieldsInObjectSimple(item, includeFields, excludeFields, arrayPath)
			result[i] = nestedObj
			// Merge nested ignored paths
			maps.Copy(ignoredPaths, nestedIgnored)
		}
		return result, ignoredPaths

	default:
		// Primitive values remain unchanged
		return obj, ignoredPaths
	}
}

// shouldIncludeField determines if a field should be included in the comparison
func shouldIncludeField(fieldPath string, includeFields, excludeFields []string) bool {
	// If include fields are specified, only include if field matches or is nested under an included field
	if len(includeFields) > 0 {
		included := false
		for _, pattern := range includeFields {
			// Check if field matches the include pattern
			if matchesFieldPattern(fieldPath, pattern) {
				included = true
				break
			}
			// Also include parent fields when a nested field is specified
			// This ensures the JSON structure remains valid
			if strings.HasPrefix(pattern, fieldPath+".") {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	// Check exclusions - only exclude exact matches or nested fields
	for _, pattern := range excludeFields {
		if matchesFieldPattern(fieldPath, pattern) {
			return false
		}
	}

	return true
}

// matchesFieldPattern checks if a field path matches a pattern (supports nested fields)
func matchesFieldPattern(fieldPath, pattern string) bool {
	// Exact match
	if fieldPath == pattern {
		return true
	}

	// Check if field is nested under pattern (for recursive inclusion/exclusion)
	if strings.HasPrefix(fieldPath, pattern+".") {
		return true
	}

	// For include patterns only: Check if pattern is nested under field
	// This allows including parent fields when child is specified
	// But we don't want this for exclusions - excluding a child shouldn't exclude the parent
	// This check is done at the call site now

	return false
}

// formatJSONObject formats a JSON object into lines
func formatJSONObject(obj any) ([]string, error) {
	if obj == nil {
		return []string{}, nil
	}

	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(formatted)))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// buildFilteredDiff creates diff with special handling for ignored fields (marked with ~)
func buildFilteredDiff(lines1, lines2 []string) []DiffLine {
	// Identify which lines contain ignored fields
	ignored1 := identifyIgnoredLines(lines1)
	ignored2 := identifyIgnoredLines(lines2)

	// Clean the ~ markers for comparison
	cleaned1 := cleanIgnoredMarkers(lines1)
	cleaned2 := cleanIgnoredMarkers(lines2)

	// Build diff using cleaned lines
	lcs := longestCommonSubsequence(cleaned1, cleaned2)
	diffs := buildDiff(cleaned1, cleaned2, lcs)

	// Process diffs to mark ignored and restore clean content
	for i := range diffs {
		lineNum1 := diffs[i].LineNum1
		lineNum2 := diffs[i].LineNum2

		// Check if line is ignored
		isIgnored := false
		if lineNum1 > 0 && lineNum1 <= len(ignored1) {
			isIgnored = isIgnored || ignored1[lineNum1-1]
		}
		if lineNum2 > 0 && lineNum2 <= len(ignored2) {
			isIgnored = isIgnored || ignored2[lineNum2-1]
		}

		diffs[i].IsIgnored = isIgnored

		// Use the cleaned content (without ~ in keys) for display
		if diffs[i].Type == DiffTypeRemoved && lineNum1 > 0 {
			diffs[i].Content = cleaned1[lineNum1-1]
		} else if diffs[i].Type == DiffTypeAdded && lineNum2 > 0 {
			diffs[i].Content = cleaned2[lineNum2-1]
		} else if diffs[i].Type == DiffTypeEqual && lineNum1 > 0 {
			diffs[i].Content = cleaned1[lineNum1-1]
		} else if diffs[i].Type == DiffTypeEqual && lineNum2 > 0 {
			diffs[i].Content = cleaned2[lineNum2-1]
		}
	}

	return diffs
}

// identifyIgnoredLines returns a boolean slice indicating which lines contain ignored fields
func identifyIgnoredLines(lines []string) []bool {
	ignored := make([]bool, len(lines))
	insideIgnored := 0 // Track nesting level inside ignored objects/arrays

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this line starts an ignored field
		if containsIgnoredField(line) {
			ignored[i] = true
			// If this line opens an object or array, track nesting
			if strings.HasSuffix(trimmed, "{") || strings.HasSuffix(trimmed, "[") {
				insideIgnored++
			}
			continue
		}

		// If we're inside an ignored object/array, this line is also ignored
		if insideIgnored > 0 {
			ignored[i] = true

			// Track nesting changes
			if strings.HasSuffix(trimmed, "{") || strings.HasSuffix(trimmed, "[") {
				insideIgnored++
			} else if strings.HasPrefix(trimmed, "}") || strings.HasPrefix(trimmed, "]") {
				insideIgnored--
			}
		}
	}
	return ignored
}

// cleanIgnoredMarkers removes ~ markers from field names for comparison
func cleanIgnoredMarkers(lines []string) []string {
	cleaned := make([]string, len(lines))
	for i, line := range lines {
		// Remove ~ from field names like "~fieldname": or ~"fieldname":
		cleaned[i] = strings.ReplaceAll(line, "\"~", "\"")
		cleaned[i] = strings.ReplaceAll(cleaned[i], "~\"", "\"")
	}
	return cleaned
}

// containsIgnoredField checks if a line contains an ignored field marker
func containsIgnoredField(line string) bool {
	// Check for field names starting with ~
	// This matches patterns like:  "~fieldname": value
	trimmed := strings.TrimSpace(line)

	// Look for patterns that indicate a field name with ~
	// 1. "~fieldname": - field name starts with ~
	// 2. ~"fieldname": - alternative format
	if strings.HasPrefix(trimmed, "\"~") || strings.HasPrefix(trimmed, "~\"") {
		return true
	}

	// Check for "~field": pattern in the middle of the line (for inline fields)
	// But make sure it's followed by a colon to indicate it's a field name
	idx := strings.Index(trimmed, "\"~")
	if idx > 0 {
		// Found "~ pattern, check if it's followed by ":
		afterTilde := trimmed[idx+2:]
		colonIdx := strings.Index(afterTilde, "\":")
		// It's a field if we find ": after the field name
		return colonIdx > 0 && colonIdx < len(afterTilde)-2
	}

	return false
}

func formatJSONLines(data []byte, sortKeys bool) ([]string, error) {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	if sortKeys {
		obj = sortJSONKeys(obj)
	}

	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(formatted)))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

func sortJSONKeys(v any) any {
	switch val := v.(type) {
	case map[string]any:
		sorted := make(map[string]any)
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sorted[k] = sortJSONKeys(val[k])
		}
		return sorted
	case []any:
		for i, item := range val {
			val[i] = sortJSONKeys(item)
		}
		return val
	default:
		return v
	}
}

func longestCommonSubsequence(lines1, lines2 []string) [][]int {
	m, n := len(lines1), len(lines2)
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if lines1[i-1] == lines2[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else {
				lcs[i][j] = max(lcs[i-1][j], lcs[i][j-1])
			}
		}
	}

	return lcs
}

func buildDiff(lines1, lines2 []string, lcs [][]int) []DiffLine {
	var diffs []DiffLine
	i, j := len(lines1), len(lines2)

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && lines1[i-1] == lines2[j-1] {
			diffs = append(diffs, DiffLine{
				Type:     DiffTypeEqual,
				LineNum1: i,
				LineNum2: j,
				Content:  lines1[i-1],
			})
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			diffs = append(diffs, DiffLine{
				Type:     DiffTypeAdded,
				LineNum1: -1,
				LineNum2: j,
				Content:  lines2[j-1],
			})
			j--
		} else {
			diffs = append(diffs, DiffLine{
				Type:     DiffTypeRemoved,
				LineNum1: i,
				LineNum2: -1,
				Content:  lines1[i-1],
			})
			i--
		}
	}

	for i, j := 0, len(diffs)-1; i < j; i, j = i+1, j-1 {
		diffs[i], diffs[j] = diffs[j], diffs[i]
	}

	return diffs
}

func addContext(diffs []DiffLine, contextLines int) []DiffLine {
	if contextLines < 0 {
		contextLines = defaultContextLines
	}

	var result []DiffLine
	changeIndices := []int{}

	for i, diff := range diffs {
		if diff.Type != DiffTypeEqual {
			changeIndices = append(changeIndices, i)
		}
	}

	if len(changeIndices) == 0 {
		return []DiffLine{}
	}

	included := make(map[int]bool)
	for _, idx := range changeIndices {
		for j := max(0, idx-contextLines); j <= min(len(diffs)-1, idx+contextLines); j++ {
			included[j] = true
		}
	}

	lastIncluded := -1
	for i, diff := range diffs {
		if included[i] {
			if lastIncluded != -1 && i-lastIncluded > 1 {
				result = append(result, DiffLine{
					Type:    DiffTypeEqual,
					Content: "...",
				})
			}
			result = append(result, diff)
			lastIncluded = i
		}
	}

	return result
}

func FindInlineChanges(line1, line2 string) (start1, end1, start2, end2 int) {
	if line1 == line2 {
		return -1, -1, -1, -1
	}

	commonPrefix := 0
	for commonPrefix < len(line1) && commonPrefix < len(line2) && line1[commonPrefix] == line2[commonPrefix] {
		commonPrefix++
	}

	commonSuffix := 0
	i, j := len(line1)-1, len(line2)-1
	for i >= commonPrefix && j >= commonPrefix && line1[i] == line2[j] {
		commonSuffix++
		i--
		j--
	}

	start1, end1 = commonPrefix, len(line1)-commonSuffix
	start2, end2 = commonPrefix, len(line2)-commonSuffix

	if start1 >= end1 {
		start1, end1 = -1, -1
	}
	if start2 >= end2 {
		start2, end2 = -1, -1
	}

	return start1, end1, start2, end2
}
