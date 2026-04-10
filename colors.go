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
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type ColorConfig struct {
	Version int `json:"version"`
	Colors  struct {
		Add struct {
			Foreground struct {
				Line ColorDef `json:"line"`
			} `json:"foreground"`
			Background ColorDef `json:"background"`
		} `json:"add"`
		Remove struct {
			Foreground struct {
				Line ColorDef `json:"line"`
			} `json:"foreground"`
			Background ColorDef `json:"background"`
		} `json:"remove"`
		Ignored struct {
			Foreground ColorDef `json:"foreground"`
			Background ColorDef `json:"background"`
		} `json:"ignored"`
	} `json:"colors"`
}

type ColorDef struct {
	Hex     string `json:"hex"`
	ANSI256 int    `json:"ansi256"`
	ANSI    int    `json:"ansi"`
}

type Styles struct {
	AddedLine   lipgloss.Style
	RemovedLine lipgloss.Style
	IgnoredLine lipgloss.Style
	LineNumber  lipgloss.Style
	Normal      lipgloss.Style
}

func DefaultStyles() *Styles {
	return &Styles{
		AddedLine: lipgloss.NewStyle().Foreground(lipgloss.CompleteAdaptiveColor{
			Light: lipgloss.CompleteColor{TrueColor: "#00ff00", ANSI256: "10", ANSI: "10"},
			Dark:  lipgloss.CompleteColor{TrueColor: "#00ff00", ANSI256: "10", ANSI: "10"},
		}),
		RemovedLine: lipgloss.NewStyle().Foreground(lipgloss.CompleteAdaptiveColor{
			Light: lipgloss.CompleteColor{TrueColor: "#ff0000", ANSI256: "9", ANSI: "9"},
			Dark:  lipgloss.CompleteColor{TrueColor: "#ff0000", ANSI256: "9", ANSI: "9"},
		}),
		IgnoredLine: lipgloss.NewStyle().Foreground(lipgloss.CompleteAdaptiveColor{
			Light: lipgloss.CompleteColor{TrueColor: "#0000ff", ANSI256: "12", ANSI: "12"},
			Dark:  lipgloss.CompleteColor{TrueColor: "#0080ff", ANSI256: "12", ANSI: "12"},
		}),
		LineNumber: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "8", Dark: "7"}),
		Normal:     lipgloss.NewStyle(),
	}
}

func NoColorStyles() *Styles {
	plain := lipgloss.NewStyle()
	return &Styles{
		AddedLine:   plain,
		RemovedLine: plain,
		IgnoredLine: plain,
		LineNumber:  plain,
		Normal:      plain,
	}
}

func StylesFromConfig(config *ColorConfig) *Styles {
	styles := DefaultStyles()

	if config == nil || config.Version != 1 {
		return styles
	}

	styles.AddedLine = createStyle(config.Colors.Add.Foreground.Line, config.Colors.Add.Background)
	styles.RemovedLine = createStyle(config.Colors.Remove.Foreground.Line, config.Colors.Remove.Background)

	if config.Colors.Ignored.Foreground.Hex != "" || config.Colors.Ignored.Foreground.ANSI256 != 0 || config.Colors.Ignored.Foreground.ANSI != 0 {
		styles.IgnoredLine = createStyle(config.Colors.Ignored.Foreground, config.Colors.Ignored.Background)
	}

	return styles
}

func createStyle(fg ColorDef, bg ColorDef) lipgloss.Style {
	style := lipgloss.NewStyle()

	if fg.Hex != "" || fg.ANSI256 != 0 || fg.ANSI != 0 {
		style = style.Foreground(createAdaptiveColor(fg))
	}

	if bg.Hex != "" || bg.ANSI256 != 0 || bg.ANSI != 0 {
		style = style.Background(createAdaptiveColor(bg))
	}

	return style
}

func createAdaptiveColor(color ColorDef) lipgloss.TerminalColor {
	ansi256 := ""
	if color.ANSI256 != 0 {
		ansi256 = fmt.Sprintf("%d", color.ANSI256)
	}

	ansi := ""
	if color.ANSI != 0 {
		ansi = fmt.Sprintf("%d", color.ANSI)
	}

	return lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{
			TrueColor: color.Hex,
			ANSI256:   ansi256,
			ANSI:      ansi,
		},
		Dark: lipgloss.CompleteColor{
			TrueColor: color.Hex,
			ANSI256:   ansi256,
			ANSI:      ansi,
		},
	}
}
