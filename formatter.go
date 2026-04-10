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
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	defaultTerminalWidth = 120
	minColumnWidth       = 30
	defaultFile1Marker   = "1"
	defaultFile2Marker   = "2"
	defaultBothMarker    = "Both"
)

// FormatterOptions configures a Formatter.
type FormatterOptions struct {
	Styles      *Styles
	File1Marker string
	File2Marker string
	BothMarker  string
}

type Formatter struct {
	styles       *Styles
	file1Marker  string
	file2Marker  string
	bothMarker   string
	maxMarkerLen int
}

// NewFormatter creates a Formatter with the given styles.
// For more control over markers, use NewFormatterWithOptions instead.
func NewFormatter(styles *Styles) *Formatter {
	return NewFormatterWithOptions(FormatterOptions{Styles: styles})
}

// NewFormatterWithOptions creates a Formatter with the given options.
func NewFormatterWithOptions(opts FormatterOptions) *Formatter {
	if opts.Styles == nil {
		opts.Styles = DefaultStyles()
	}
	if opts.File1Marker == "" {
		opts.File1Marker = defaultFile1Marker
	}
	if opts.File2Marker == "" {
		opts.File2Marker = defaultFile2Marker
	}
	if opts.BothMarker == "" {
		opts.BothMarker = defaultBothMarker
	}

	f := &Formatter{
		styles:      opts.Styles,
		file1Marker: opts.File1Marker,
		file2Marker: opts.File2Marker,
		bothMarker:  opts.BothMarker,
	}
	f.updateMaxMarkerLen()
	return f
}

// SetMarkers sets custom markers for file1, file2, and both.
// Deprecated: Use NewFormatterWithOptions instead.
func (f *Formatter) SetMarkers(file1Marker, file2Marker, bothMarker string) {
	if file1Marker != "" {
		f.file1Marker = file1Marker
	}
	if file2Marker != "" {
		f.file2Marker = file2Marker
	}
	if bothMarker != "" {
		f.bothMarker = bothMarker
	}
	f.updateMaxMarkerLen()
}

// updateMaxMarkerLen calculates the maximum marker length for padding
func (f *Formatter) updateMaxMarkerLen() {
	f.maxMarkerLen = max(len(f.file1Marker), len(f.file2Marker), len(f.bothMarker))
}

func (f *Formatter) Format(diffs []DiffLine) string {
	// Group diffs by key for better comparison
	grouped := f.groupDiffsByKey(diffs)

	var output strings.Builder
	for _, group := range grouped {
		for _, diff := range group {
			f.formatDiffLine(&output, diff)
		}
	}

	return output.String()
}

// formatDiffLine formats a single diff line with custom markers
func (f *Formatter) formatDiffLine(output *strings.Builder, diff DiffLine) {
	prefix := f.getFileMarker(diff.Type)
	paddedPrefix := f.padMarker(prefix)

	// Handle ignored fields - show with blue color but don't modify content
	if diff.IsIgnored {
		output.WriteString(f.styles.LineNumber.Render(paddedPrefix))
		output.WriteString(" ")
		output.WriteString(f.styles.IgnoredLine.Render("~ " + diff.Content))
		output.WriteString("\n")
		return
	}

	switch diff.Type {
	case DiffTypeEqual:
		if diff.Content == "..." {
			output.WriteString("...\n")
		} else {
			output.WriteString(f.styles.LineNumber.Render(paddedPrefix))
			output.WriteString(" ")
			// Add ~ prefix for equal lines to show they're unchanged
			output.WriteString(f.styles.Normal.Render("~ " + diff.Content))
			output.WriteString("\n")
		}
	case DiffTypeRemoved:
		output.WriteString(f.styles.LineNumber.Render(paddedPrefix))
		output.WriteString(" ")

		if diff.InlineStart >= 0 && diff.InlineEnd > diff.InlineStart {
			output.WriteString(f.formatInlineRemoved(diff.Content, diff.InlineStart, diff.InlineEnd))
		} else {
			output.WriteString(f.styles.RemovedLine.Render("- " + diff.Content))
		}
		output.WriteString("\n")
	case DiffTypeAdded:
		output.WriteString(f.styles.LineNumber.Render(paddedPrefix))
		output.WriteString(" ")

		if diff.InlineStart >= 0 && diff.InlineEnd > diff.InlineStart {
			output.WriteString(f.formatInlineAdded(diff.Content, diff.InlineStart, diff.InlineEnd))
		} else {
			output.WriteString(f.styles.AddedLine.Render("+ " + diff.Content))
		}
		output.WriteString("\n")
	}
}

// getFileMarker returns the appropriate marker based on diff type
func (f *Formatter) getFileMarker(diffType DiffType) string {
	switch diffType {
	case DiffTypeRemoved:
		return f.file1Marker
	case DiffTypeAdded:
		return f.file2Marker
	case DiffTypeEqual:
		return f.bothMarker
	default:
		return ""
	}
}

// padMarker right-justifies the marker with spaces to match maxMarkerLen
func (f *Formatter) padMarker(marker string) string {
	padding := f.maxMarkerLen - len(marker)
	if padding > 0 {
		return strings.Repeat(" ", padding) + marker
	}
	return marker
}

// groupDiffsByKey groups diff lines by their JSON key for better comparison
func (f *Formatter) groupDiffsByKey(diffs []DiffLine) [][]DiffLine {
	var groups [][]DiffLine
	var currentGroup []DiffLine
	processedIndices := make(map[int]bool)

	for i, diff := range diffs {
		if processedIndices[i] {
			continue
		}

		// Start a new group with this diff
		currentGroup = []DiffLine{diff}
		processedIndices[i] = true

		// Extract the key from this line
		key := extractJSONKey(diff.Content)

		// If this is a line with a key, look for matching line of opposite type
		if key != "" {
			// For removed lines, look for matching added line
			if diff.Type == DiffTypeRemoved {
				for j := i + 1; j < len(diffs); j++ {
					if processedIndices[j] {
						continue
					}

					if diffs[j].Type == DiffTypeAdded {
						addedKey := extractJSONKey(diffs[j].Content)
						if key == addedKey {
							// Found matching key - add to group right after the removed line
							currentGroup = append(currentGroup, diffs[j])
							processedIndices[j] = true
							break
						}
					}
				}
			}
			// For equal lines (both ignored and non-ignored), look for duplicates to consolidate
			if diff.Type == DiffTypeEqual {
				for j := i + 1; j < len(diffs) && !processedIndices[j]; j++ {
					if diffs[j].Type == DiffTypeEqual {
						otherKey := extractJSONKey(diffs[j].Content)
						// Only consolidate if both have the same non-empty key
						if key == otherKey {
							processedIndices[j] = true
							break
						}
					}
				}
			}
		}

		groups = append(groups, currentGroup)
	}

	return groups
}

// extractJSONKey extracts the JSON key from a line (e.g., "name" from '"name": "value"')
func extractJSONKey(line string) string {
	trimmed := strings.TrimSpace(line)

	if !strings.HasPrefix(trimmed, "\"") {
		return ""
	}
	trimmed = trimmed[1:] // skip opening quote

	// Find the closing quote and colon
	colonIndex := strings.Index(trimmed, "\":")
	if colonIndex > 0 {
		return trimmed[:colonIndex]
	}

	return ""
}

func (f *Formatter) formatInlineRemoved(content string, start, end int) string {
	var result strings.Builder
	result.WriteString(f.styles.RemovedLine.Render("- "))

	// Find where the value starts (after the colon)
	colonPos := strings.Index(content, ":")
	valueStart := colonPos + 1
	if colonPos >= 0 && valueStart < len(content) {
		// Skip leading spaces after colon
		for valueStart < len(content) && content[valueStart] == ' ' {
			valueStart++
		}
	}

	// Render the key part (up to and including colon and spaces) without Faint
	if colonPos >= 0 && valueStart > 0 {
		result.WriteString(f.styles.RemovedLine.Render(content[:valueStart]))

		// Apply styling only to the value part
		if start >= valueStart && end > start {
			// Unchanged part of value before change
			if start > valueStart {
				result.WriteString(f.styles.RemovedLine.Faint(true).Render(content[valueStart:start]))
			}
			// Changed part - apply Bold(true)
			result.WriteString(f.styles.RemovedLine.Bold(true).Render(content[start:end]))
			// Unchanged part of value after change
			if end < len(content) {
				result.WriteString(f.styles.RemovedLine.Faint(true).Render(content[end:]))
			}
		} else {
			// No inline changes in value, render normally
			result.WriteString(f.styles.RemovedLine.Render(content[valueStart:]))
		}
	} else {
		// No key-value structure, apply styling to entire content
		if start > 0 {
			result.WriteString(f.styles.RemovedLine.Faint(true).Render(content[:start]))
		}
		if end > start {
			result.WriteString(f.styles.RemovedLine.Bold(true).Render(content[start:end]))
		}
		if end < len(content) {
			result.WriteString(f.styles.RemovedLine.Faint(true).Render(content[end:]))
		}
	}

	return result.String()
}

func (f *Formatter) formatInlineAdded(content string, start, end int) string {
	var result strings.Builder
	result.WriteString(f.styles.AddedLine.Render("+ "))

	// Find where the value starts (after the colon)
	colonPos := strings.Index(content, ":")
	valueStart := colonPos + 1
	if colonPos >= 0 && valueStart < len(content) {
		// Skip leading spaces after colon
		for valueStart < len(content) && content[valueStart] == ' ' {
			valueStart++
		}
	}

	// Render the key part (up to and including colon and spaces) without Faint
	if colonPos >= 0 && valueStart > 0 {
		result.WriteString(f.styles.AddedLine.Render(content[:valueStart]))

		// Apply styling only to the value part
		if start >= valueStart && end > start {
			// Unchanged part of value before change
			if start > valueStart {
				result.WriteString(f.styles.AddedLine.Faint(true).Render(content[valueStart:start]))
			}
			// Changed part - apply Bold(true)
			result.WriteString(f.styles.AddedLine.Bold(true).Render(content[start:end]))
			// Unchanged part of value after change
			if end < len(content) {
				result.WriteString(f.styles.AddedLine.Faint(true).Render(content[end:]))
			}
		} else {
			// No inline changes in value, render normally
			result.WriteString(f.styles.AddedLine.Render(content[valueStart:]))
		}
	} else {
		// No key-value structure, apply styling to entire content
		if start > 0 {
			result.WriteString(f.styles.AddedLine.Faint(true).Render(content[:start]))
		}
		if end > start {
			result.WriteString(f.styles.AddedLine.Bold(true).Render(content[start:end]))
		}
		if end < len(content) {
			result.WriteString(f.styles.AddedLine.Faint(true).Render(content[end:]))
		}
	}

	return result.String()
}

func (f *Formatter) FormatSideBySide(diffs []DiffLine, leftMarker, rightMarker string) string {
	var output strings.Builder

	// Get terminal width and calculate column widths
	terminalWidth := defaultTerminalWidth
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		terminalWidth = width
	}

	const separator = " | "
	columnWidth := max((terminalWidth-len(separator))/2, minColumnWidth)

	// Use markers as headers (they will be the filenames by default)
	leftHeader := leftMarker
	rightHeader := rightMarker

	// Format and add headers with underline
	leftHeaderFormatted := f.styles.Normal.Bold(true).Render(leftHeader)
	rightHeaderFormatted := f.styles.Normal.Bold(true).Render(rightHeader)

	output.WriteString(padRight(leftHeaderFormatted, columnWidth))
	output.WriteString(separator)
	output.WriteString(rightHeaderFormatted)
	output.WriteString("\n")

	// Add separator line
	output.WriteString(strings.Repeat("-", columnWidth))
	output.WriteString(separator)
	output.WriteString(strings.Repeat("-", columnWidth))
	output.WriteString("\n")

	// Group diffs by key for better alignment
	grouped := f.groupDiffsForSideBySide(diffs)

	for _, group := range grouped {
		f.formatSideBySideGroup(&output, group, columnWidth, separator)
	}

	return output.String()
}

// groupDiffsForSideBySide groups diffs for side-by-side display with proper key alignment
func (f *Formatter) groupDiffsForSideBySide(diffs []DiffLine) [][]DiffLine {
	var groups [][]DiffLine
	processedIndices := make(map[int]bool)

	for i, diff := range diffs {
		if processedIndices[i] {
			continue
		}

		group := []DiffLine{diff}
		processedIndices[i] = true

		// Extract key from this line
		key := extractJSONKey(diff.Content)

		// For lines with keys, look for matching keys
		if key != "" && diff.Type == DiffTypeRemoved {
			// Look for matching added line
			for j := i + 1; j < len(diffs); j++ {
				if processedIndices[j] {
					continue
				}

				if diffs[j].Type == DiffTypeAdded {
					addedKey := extractJSONKey(diffs[j].Content)
					if key == addedKey {
						// Found matching key - group them
						group = []DiffLine{diff, diffs[j]}
						processedIndices[j] = true
						break
					}
				}
			}
		}

		groups = append(groups, group)
	}

	return groups
}

// formatSideBySideGroup formats a group of related diffs for side-by-side display
func (f *Formatter) formatSideBySideGroup(output *strings.Builder, group []DiffLine, columnWidth int, separator string) {
	if len(group) == 0 {
		return
	}

	// Handle single line or paired lines
	if len(group) == 1 {
		diff := group[0]

		switch diff.Type {
		case DiffTypeEqual:
			if diff.Content == "..." {
				output.WriteString("...\n")
				return
			}

			// For ignored equal lines, check which file they come from
			if diff.IsIgnored {
				// Ignored equal lines should only appear on the side where they exist
				// Based on LineNum1 and LineNum2, determine which side has the content
				leftText := ""
				rightText := ""

				if diff.LineNum1 > 0 {
					leftText = truncateString(diff.Content, columnWidth)
					leftText = "~ " + leftText
				}
				if diff.LineNum2 > 0 {
					rightText = truncateString(diff.Content, columnWidth)
					rightText = "~ " + rightText
				}

				// Only show on the side(s) where it exists
				if leftText != "" {
					leftFormatted := f.styles.IgnoredLine.Render(leftText)
					output.WriteString(padRight(leftFormatted, columnWidth))
				} else {
					output.WriteString(strings.Repeat(" ", columnWidth))
				}

				output.WriteString(separator)

				if rightText != "" {
					rightFormatted := f.styles.IgnoredLine.Render(rightText)
					output.WriteString(rightFormatted)
				}
				output.WriteString("\n")
			} else {
				// Non-ignored equal lines appear on both sides with ~ prefix
				leftText := truncateString(diff.Content, columnWidth)
				rightText := leftText
				// Prepend ~ to indicate these are equal/unchanged
				leftText = "~ " + leftText
				rightText = "~ " + rightText
				leftFormatted := f.styles.Normal.Render(leftText)
				rightFormatted := f.styles.Normal.Render(rightText)
				output.WriteString(padRight(leftFormatted, columnWidth))
				output.WriteString(separator)
				output.WriteString(rightFormatted)
				output.WriteString("\n")
			}

		case DiffTypeRemoved:
			// Only on left side
			if diff.IsIgnored {
				leftText := truncateString(diff.Content, columnWidth)
				leftText = "~ " + leftText
				leftFormatted := f.styles.IgnoredLine.Render(leftText)
				output.WriteString(padRight(leftFormatted, columnWidth))
			} else if diff.InlineStart >= 0 && diff.InlineEnd > diff.InlineStart {
				// Has inline changes - format with inline styling
				leftFormatted := f.formatSideBySideInlineRemoved(diff.Content, diff.InlineStart, diff.InlineEnd, columnWidth)
				output.WriteString(padRight(leftFormatted, columnWidth))
			} else {
				leftText := truncateString(diff.Content, columnWidth)
				leftFormatted := f.styles.RemovedLine.Render("- " + leftText)
				output.WriteString(padRight(leftFormatted, columnWidth))
			}
			output.WriteString(separator)
			output.WriteString("\n")

		case DiffTypeAdded:
			// Only on right side
			output.WriteString(strings.Repeat(" ", columnWidth))
			output.WriteString(separator)

			if diff.IsIgnored {
				rightText := truncateString(diff.Content, columnWidth)
				rightText = "~ " + rightText
				rightFormatted := f.styles.IgnoredLine.Render(rightText)
				output.WriteString(rightFormatted)
			} else if diff.InlineStart >= 0 && diff.InlineEnd > diff.InlineStart {
				// Has inline changes - format with inline styling
				rightFormatted := f.formatSideBySideInlineAdded(diff.Content, diff.InlineStart, diff.InlineEnd, columnWidth)
				output.WriteString(rightFormatted)
			} else {
				rightText := truncateString(diff.Content, columnWidth)
				rightFormatted := f.styles.AddedLine.Render("+ " + rightText)
				output.WriteString(rightFormatted)
			}
			output.WriteString("\n")
		}
	} else if len(group) == 2 {
		// Paired remove/add with matching keys
		leftDiff := group[0]
		rightDiff := group[1]

		if leftDiff.IsIgnored && rightDiff.IsIgnored {
			// Both ignored - show as equal ignored line
			leftText := truncateString(leftDiff.Content, columnWidth)
			rightText := truncateString(rightDiff.Content, columnWidth)
			leftFormatted := f.styles.IgnoredLine.Render("~ " + leftText)
			rightFormatted := f.styles.IgnoredLine.Render("~ " + rightText)
			output.WriteString(padRight(leftFormatted, columnWidth))
			output.WriteString(separator)
			output.WriteString(rightFormatted)
		} else {
			// Show as changed - check for inline changes
			if leftDiff.InlineStart >= 0 && leftDiff.InlineEnd > leftDiff.InlineStart {
				// Has inline changes - format with inline styling
				leftFormatted := f.formatSideBySideInlineRemoved(leftDiff.Content, leftDiff.InlineStart, leftDiff.InlineEnd, columnWidth)
				output.WriteString(padRight(leftFormatted, columnWidth))
			} else {
				leftText := truncateString(leftDiff.Content, columnWidth)
				leftFormatted := f.styles.RemovedLine.Render("- " + leftText)
				output.WriteString(padRight(leftFormatted, columnWidth))
			}

			output.WriteString(separator)

			if rightDiff.InlineStart >= 0 && rightDiff.InlineEnd > rightDiff.InlineStart {
				// Has inline changes - format with inline styling
				rightFormatted := f.formatSideBySideInlineAdded(rightDiff.Content, rightDiff.InlineStart, rightDiff.InlineEnd, columnWidth)
				output.WriteString(rightFormatted)
			} else {
				rightText := truncateString(rightDiff.Content, columnWidth)
				rightFormatted := f.styles.AddedLine.Render("+ " + rightText)
				output.WriteString(rightFormatted)
			}
		}
		output.WriteString("\n")
	}
}

func (f *Formatter) formatSideBySideInlineRemoved(content string, start, end int, columnWidth int) string {
	var result strings.Builder
	result.WriteString(f.styles.RemovedLine.Render("- "))

	// Truncate if needed but preserve inline change positions
	truncatedContent := content
	adjustedStart := start
	adjustedEnd := end

	// Account for the "- " prefix (2 chars)
	effectiveWidth := columnWidth - 2
	if len(content) > effectiveWidth {
		// If the content is too long, we need to decide what part to show
		// Prioritize showing the changed part
		if end <= effectiveWidth {
			// Changed part fits within the width
			truncatedContent = content[:effectiveWidth]
		} else if start >= len(content)-effectiveWidth {
			// Show the end including the change
			offset := len(content) - effectiveWidth
			truncatedContent = content[offset:]
			adjustedStart = start - offset
			adjustedEnd = end - offset
		} else {
			// Show a window around the change
			windowStart := max(0, start-10)
			windowEnd := min(len(content), windowStart+effectiveWidth)
			truncatedContent = content[windowStart:windowEnd]
			adjustedStart = start - windowStart
			adjustedEnd = min(end-windowStart, len(truncatedContent))
		}
	}

	// Find where the value starts (after the colon)
	colonPos := strings.Index(truncatedContent, ":")
	valueStart := colonPos + 1
	if colonPos >= 0 && valueStart < len(truncatedContent) {
		// Skip leading spaces after colon
		for valueStart < len(truncatedContent) && truncatedContent[valueStart] == ' ' {
			valueStart++
		}
	}

	// Render the key part without Faint
	if colonPos >= 0 && valueStart > 0 {
		result.WriteString(f.styles.RemovedLine.Render(truncatedContent[:valueStart]))

		// Apply styling only to the value part
		if adjustedStart >= valueStart && adjustedEnd > adjustedStart {
			// Unchanged part of value before change
			if adjustedStart > valueStart {
				result.WriteString(f.styles.RemovedLine.Faint(true).Render(truncatedContent[valueStart:adjustedStart]))
			}
			// Changed part - apply Bold(true)
			result.WriteString(f.styles.RemovedLine.Bold(true).Render(truncatedContent[adjustedStart:adjustedEnd]))
			// Unchanged part of value after change
			if adjustedEnd < len(truncatedContent) {
				result.WriteString(f.styles.RemovedLine.Faint(true).Render(truncatedContent[adjustedEnd:]))
			}
		} else {
			// No inline changes in value, render normally
			result.WriteString(f.styles.RemovedLine.Render(truncatedContent[valueStart:]))
		}
	} else {
		// No key-value structure, apply styling to entire content
		if adjustedStart > 0 {
			result.WriteString(f.styles.RemovedLine.Faint(true).Render(truncatedContent[:adjustedStart]))
		}
		if adjustedEnd > adjustedStart {
			result.WriteString(f.styles.RemovedLine.Bold(true).Render(truncatedContent[adjustedStart:adjustedEnd]))
		}
		if adjustedEnd < len(truncatedContent) {
			result.WriteString(f.styles.RemovedLine.Faint(true).Render(truncatedContent[adjustedEnd:]))
		}
	}

	return result.String()
}

func (f *Formatter) formatSideBySideInlineAdded(content string, start, end int, columnWidth int) string {
	var result strings.Builder
	result.WriteString(f.styles.AddedLine.Render("+ "))

	// Truncate if needed but preserve inline change positions
	truncatedContent := content
	adjustedStart := start
	adjustedEnd := end

	// Account for the "+ " prefix (2 chars)
	effectiveWidth := columnWidth - 2
	if len(content) > effectiveWidth {
		// If the content is too long, we need to decide what part to show
		// Prioritize showing the changed part
		if end <= effectiveWidth {
			// Changed part fits within the width
			truncatedContent = content[:effectiveWidth]
		} else if start >= len(content)-effectiveWidth {
			// Show the end including the change
			offset := len(content) - effectiveWidth
			truncatedContent = content[offset:]
			adjustedStart = start - offset
			adjustedEnd = end - offset
		} else {
			// Show a window around the change
			windowStart := max(0, start-10)
			windowEnd := min(len(content), windowStart+effectiveWidth)
			truncatedContent = content[windowStart:windowEnd]
			adjustedStart = start - windowStart
			adjustedEnd = min(end-windowStart, len(truncatedContent))
		}
	}

	// Find where the value starts (after the colon)
	colonPos := strings.Index(truncatedContent, ":")
	valueStart := colonPos + 1
	if colonPos >= 0 && valueStart < len(truncatedContent) {
		// Skip leading spaces after colon
		for valueStart < len(truncatedContent) && truncatedContent[valueStart] == ' ' {
			valueStart++
		}
	}

	// Render the key part without Faint
	if colonPos >= 0 && valueStart > 0 {
		result.WriteString(f.styles.AddedLine.Render(truncatedContent[:valueStart]))

		// Apply styling only to the value part
		if adjustedStart >= valueStart && adjustedEnd > adjustedStart {
			// Unchanged part of value before change
			if adjustedStart > valueStart {
				result.WriteString(f.styles.AddedLine.Faint(true).Render(truncatedContent[valueStart:adjustedStart]))
			}
			// Changed part - apply Bold(true)
			result.WriteString(f.styles.AddedLine.Bold(true).Render(truncatedContent[adjustedStart:adjustedEnd]))
			// Unchanged part of value after change
			if adjustedEnd < len(truncatedContent) {
				result.WriteString(f.styles.AddedLine.Faint(true).Render(truncatedContent[adjustedEnd:]))
			}
		} else {
			// No inline changes in value, render normally
			result.WriteString(f.styles.AddedLine.Render(truncatedContent[valueStart:]))
		}
	} else {
		// No key-value structure, apply styling to entire content
		if adjustedStart > 0 {
			result.WriteString(f.styles.AddedLine.Faint(true).Render(truncatedContent[:adjustedStart]))
		}
		if adjustedEnd > adjustedStart {
			result.WriteString(f.styles.AddedLine.Bold(true).Render(truncatedContent[adjustedStart:adjustedEnd]))
		}
		if adjustedEnd < len(truncatedContent) {
			result.WriteString(f.styles.AddedLine.Faint(true).Render(truncatedContent[adjustedEnd:]))
		}
	}

	return result.String()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func padRight(s string, length int) string {
	// Count visible characters (excluding ANSI escape sequences)
	visibleLen := len(stripANSI(s))
	if visibleLen >= length {
		return s
	}
	return s + strings.Repeat(" ", length-visibleLen)
}

func stripANSI(s string) string {
	var result strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			i += 2 // skip ESC [
			// Skip parameter bytes until the final byte (0x40-0x7E)
			for i < len(runes) && (runes[i] < 0x40 || runes[i] > 0x7E) {
				i++
			}
			// i now points at the final byte; the loop increment skips it
		} else if runes[i] == '\x1b' {
			// Non-CSI escape: skip ESC and the next byte
			i++
		} else {
			result.WriteRune(runes[i])
		}
	}
	return result.String()
}
