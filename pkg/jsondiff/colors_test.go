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
	"encoding/json"
	"testing"
)

func TestDefaultStylesIncludesIgnored(t *testing.T) {
	styles := DefaultStyles()

	// The IgnoredLine style should be configured
	// Note: lipgloss doesn't apply colors in non-TTY environments (like tests)
	// so we can't test the actual rendering, but we can verify the style exists

	// Test that we can call Render without panic
	_ = styles.IgnoredLine.Render("test")

	// Test all other styles are also present
	_ = styles.AddedLine.Render("test")
	_ = styles.RemovedLine.Render("test")
	_ = styles.Normal.Render("test")
}

func TestStylesFromConfigWithIgnored(t *testing.T) {
	configJSON := `{
		"version": 1,
		"colors": {
			"add": {
				"foreground": {
					"line": {"hex": "#00ff00", "ansi256": 10, "ansi": 10},
					"inline": {"hex": "#008000", "ansi256": 2, "ansi": 2}
				},
				"background": {}
			},
			"remove": {
				"foreground": {
					"line": {"hex": "#ff0000", "ansi256": 9, "ansi": 9},
					"inline": {"hex": "#800000", "ansi256": 1, "ansi": 1}
				},
				"background": {}
			},
			"ignored": {
				"foreground": {"hex": "#0000ff", "ansi256": 12, "ansi": 12},
				"background": {}
			}
		}
	}`

	var config ColorConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	styles := StylesFromConfig(&config)

	// Test that the style can be rendered without panic
	// Note: lipgloss doesn't apply colors in non-TTY environments
	_ = styles.IgnoredLine.Render("test")

	// Verify all styles work
	_ = styles.AddedLine.Render("test")
	_ = styles.RemovedLine.Render("test")
}

func TestStylesFromConfigWithoutIgnored(t *testing.T) {
	// Config without ignored field configuration
	configJSON := `{
		"version": 1,
		"colors": {
			"add": {
				"foreground": {
					"line": {"hex": "#00ff00", "ansi256": 10, "ansi": 10},
					"inline": {"hex": "#008000", "ansi256": 2, "ansi": 2}
				},
				"background": {}
			},
			"remove": {
				"foreground": {
					"line": {"hex": "#ff0000", "ansi256": 9, "ansi": 9},
					"inline": {"hex": "#800000", "ansi256": 1, "ansi": 1}
				},
				"background": {}
			}
		}
	}`

	var config ColorConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	styles := StylesFromConfig(&config)

	// Should still have default ignored style
	// Test that we can render without panic
	_ = styles.IgnoredLine.Render("test")
}

func TestColorConfigStructure(t *testing.T) {
	// Test that the ColorConfig struct properly unmarshals with ignored field
	configJSON := `{
		"version": 1,
		"colors": {
			"ignored": {
				"foreground": {"hex": "#0000ff", "ansi256": 12, "ansi": 12},
				"background": {"hex": "#ffff00", "ansi256": 11, "ansi": 11}
			}
		}
	}`

	var config ColorConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatalf("Failed to unmarshal config with ignored field: %v", err)
	}

	if config.Colors.Ignored.Foreground.Hex != "#0000ff" {
		t.Errorf("Expected ignored foreground hex #0000ff, got %s", config.Colors.Ignored.Foreground.Hex)
	}

	if config.Colors.Ignored.Background.Hex != "#ffff00" {
		t.Errorf("Expected ignored background hex #ffff00, got %s", config.Colors.Ignored.Background.Hex)
	}
}
