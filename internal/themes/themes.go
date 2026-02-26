// Package themes provides visual theming and styling functionality for the
// sequencer application. It manages color schemes, visual styles, accent colors,
// and UI art elements, offering multiple built-in themes and the ability to
// switch between them for customizing the application's appearance.
package themes

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/chriserin/sq/internal/grid"
)

type Theme struct {
	colors           map[string]string
	accentColors     []string
	accentIcons      []rune
	lineActionColors map[grid.Action]string
}

var defaultColors = map[string]string{
	"AppTitleColor":                "#fafafa",
	"AppDescriptorColor":           "#fafafa",
	"AltSeqBackgroundColor":        "#2c2c2c",
	"SeqBackgroundColor":           "#000000",
	"SeqCursorColor":               "#444444",
	"SeqVisualColor":               "#aaaaaa",
	"SeqOverlayColor":              "#333388",
	"SeqMiddleOverlayColor":        "#405810",
	"SelectedAttributeColor":       "#5cdffb",
	"NumberColor":                  "#fcbd15",
	"Black":                        "#000000",
	"White":                        "#ffffff",
	"ActiveRatchetColor":           "#abfaa9",
	"MutedRatchetColor":            "#f34213",
	"CurrentPlayingColor":          "#abfaa9",
	"ActivePlayingColor":           "#f34213",
	"ArrangementHeaderColor":       "FAFAFA",
	"ArrangementGroupColor":        "#F25D94",
	"ArrangementIndentColor":       "#4b4261",
	"ArrangementSelectedLineColor": "#3b4261",
	"PatternModeColor":             "#ed3902",
}

var defaultTheme = Theme{
	colors: defaultColors,
	accentColors: []string{
		"#000000",
		"#ed3902",
		"#f564a9",
		"#f8730e",
		"#fcc05c",
		"#5cdffb",
		"#1e89ef",
		"#164de5",
		"#0246a7",
	},
	accentIcons: []rune{
		' ',
		'✤',
		'⎈',
		'⚙',
		'⊚',
		'✦',
		'❖',
		'✥',
		'❄',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#000000",
		grid.ActionLineReset:    "#cf142b",
		grid.ActionLineReverse:  "#f8730e",
		grid.ActionLineSkipBeat: "#a9e5bb",
		grid.ActionLineResetAll: "#fcf6b1",
		grid.ActionLineBounce:   "#fcf6b1",
		grid.ActionLineDelay:    "#cc4bc2",
	},
}

var seafoamColors = map[string]string{
	"AppTitleColor":                "#a6d0c7",
	"AppDescriptorColor":           "#cce7e0",
	"AltSeqBackgroundColor":        "#013112",
	"SeqBackgroundColor":           "#0a2622",
	"SeqCursorColor":               "#3d8a7d",
	"SeqVisualColor":               "#88c3b8",
	"SeqOverlayColor":              "#336699",
	"SeqMiddleOverlayColor":        "#4d7a68",
	"SelectedAttributeColor":       "#ff9f45",
	"NumberColor":                  "#c5d86d",
	"Black":                        "#0a2622",
	"White":                        "#e6f2ef",
	"ActiveRatchetColor":           "#90e0c9",
	"MutedRatchetColor":            "#f76c5e",
	"CurrentPlayingColor":          "#90e0c9",
	"ActivePlayingColor":           "#f76c5e",
	"ArrangementHeaderColor":       "#e6f2ef",
	"ArrangementGroupColor":        "#f76c5e",
	"ArrangementIndentColor":       "#4d7a68",
	"ArrangementSelectedLineColor": "#2c5d54",
	"PatternModeColor":             "#f39c6b",
}

var seafoamTheme = Theme{
	colors: seafoamColors,
	accentColors: []string{
		"#0a2622",
		"#f76c5e",
		"#f39c6b",
		"#ffd166",
		"#c5d86d",
		"#4fd1bf",
		"#2ec4b6",
		"#1a8fe3",
		"#3c73a8",
	},
	accentIcons: []rune{
		' ',
		'✤',
		'⎈',
		'⚙',
		'⊚',
		'✦',
		'❖',
		'✥',
		'❄',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#0a2622",
		grid.ActionLineReset:    "#f76c5e",
		grid.ActionLineReverse:  "#f39c6b",
		grid.ActionLineSkipBeat: "#90e0c9",
		grid.ActionLineResetAll: "#c5d86d",
		grid.ActionLineBounce:   "#c5d86d",
		grid.ActionLineDelay:    "#a682ff",
	},
}

var dynamiteColors = map[string]string{
	"AppTitleColor":                "#ee4242",
	"AppDescriptorColor":           "#ffe6d1",
	"AltSeqBackgroundColor":        "#3d0d0d",
	"SeqBackgroundColor":           "#1c0404",
	"SeqCursorColor":               "#7a1414",
	"SeqVisualColor":               "#ff5252",
	"SeqOverlayColor":              "#9e2424",
	"SeqMiddleOverlayColor":        "#c91a1a",
	"SelectedAttributeColor":       "#00e5ff",
	"NumberColor":                  "#ff914d",
	"Black":                        "#1c0404",
	"White":                        "#fff8f0",
	"ActiveRatchetColor":           "#ffeb3b",
	"MutedRatchetColor":            "#f44336",
	"CurrentPlayingColor":          "#ffeb3b",
	"ActivePlayingColor":           "#f44336",
	"ArrangementHeaderColor":       "#fff8f0",
	"ArrangementGroupColor":        "#e91e63",
	"ArrangementIndentColor":       "#740505",
	"ArrangementSelectedLineColor": "#580f0f",
	"PatternModeColor":             "#ff914d",
}

var dynamiteTheme = Theme{
	colors: dynamiteColors,
	accentColors: []string{
		"#1c0404",
		"#f44336",
		"#e91e63",
		"#ff914d",
		"#ffeb3b",
		"#ffd700",
		"#ff5722",
		"#d81b60",
		"#b71c1c",
	},
	accentIcons: []rune{
		' ',
		'✸',
		'✯',
		'☄',
		'✧',
		'⊚',
		'☢',
		'⚔',
		'❄',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#1c0404",
		grid.ActionLineReset:    "#f44336",
		grid.ActionLineReverse:  "#ff914d",
		grid.ActionLineSkipBeat: "#ffa000",
		grid.ActionLineResetAll: "#ffeb3b",
		grid.ActionLineBounce:   "#ffd700",
		grid.ActionLineDelay:    "#e91e63",
	},
}

var springtimeColors = map[string]string{
	"AppTitleColor":                "#ffdb58",
	"AppDescriptorColor":           "#ff9eb3",
	"AltSeqBackgroundColor":        "#154018",
	"SeqBackgroundColor":           "#090d05",
	"SeqCursorColor":               "#27735e",
	"SeqVisualColor":               "#8eb656",
	"SeqOverlayColor":              "#434a4f",
	"SeqMiddleOverlayColor":        "#a6d173",
	"SelectedAttributeColor":       "#ff5e8a",
	"NumberColor":                  "#ff9eb3",
	"Black":                        "#070215",
	"White":                        "#f9fdf5",
	"ActiveRatchetColor":           "#c3f584",
	"MutedRatchetColor":            "#fc6c85",
	"CurrentPlayingColor":          "#c3f584",
	"ActivePlayingColor":           "#fc6c85",
	"ArrangementHeaderColor":       "#f9fdf5",
	"ArrangementGroupColor":        "#ff9eb3",
	"ArrangementIndentColor":       "#a6d173",
	"ArrangementSelectedLineColor": "#d6e9c6",
	"PatternModeColor":             "#ffdb58",
}

var springtimeTheme = Theme{
	colors: springtimeColors,
	accentColors: []string{
		"#3c4f2f",
		"#fc6c85",
		"#ff9eb3",
		"#ffdb58",
		"#c3f584",
		"#9ed36a",
		"#7dcfb6",
		"#b3daff",
		"#93c4e6",
	},
	accentIcons: []rune{
		' ',
		'❀',
		'✿',
		'❁',
		'✾',
		'❃',
		'✤',
		'✽',
		'✻',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#3c4f2f",
		grid.ActionLineReset:    "#fc6c85",
		grid.ActionLineReverse:  "#ffdb58",
		grid.ActionLineSkipBeat: "#7dcfb6",
		grid.ActionLineResetAll: "#c3f584",
		grid.ActionLineBounce:   "#9ed36a",
		grid.ActionLineDelay:    "#ff9eb3",
	},
}

var orangegroveColors = map[string]string{
	"AppTitleColor":                "#fff4e6",
	"AppDescriptorColor":           "#ffe8cc",
	"AltSeqBackgroundColor":        "#3a2e1e",
	"SeqBackgroundColor":           "#1a1410",
	"SeqCursorColor":               "#6a5433",
	"SeqVisualColor":               "#cc9966",
	"SeqOverlayColor":              "#770012",
	"SeqMiddleOverlayColor":        "#dd5533",
	"SelectedAttributeColor":       "#00c8ff",
	"NumberColor":                  "#ffcb69",
	"Black":                        "#1a1410",
	"White":                        "#fff4e6",
	"ActiveRatchetColor":           "#f8c537",
	"MutedRatchetColor":            "#f45d48",
	"CurrentPlayingColor":          "#f8c537",
	"ActivePlayingColor":           "#f45d48",
	"ArrangementHeaderColor":       "#fff4e6",
	"ArrangementGroupColor":        "#f45d48",
	"ArrangementIndentColor":       "#8c6d46",
	"ArrangementSelectedLineColor": "#4f3f29",
	"PatternModeColor":             "#ff8c42",
}

var orangegroveTheme = Theme{
	colors: orangegroveColors,
	accentColors: []string{
		"#1a1410",
		"#f45d48",
		"#ff8c42",
		"#ffb347",
		"#ffcb69",
		"#f8c537",
		"#e8871e",
		"#c05746",
		"#95533c",
	},
	accentIcons: []rune{
		' ',
		'☀',
		'♠',
		'♣',
		'❂',
		'✺',
		'♦',
		'⊙',
		'⊚',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#1a1410",
		grid.ActionLineReset:    "#f45d48",
		grid.ActionLineReverse:  "#ff8c42",
		grid.ActionLineSkipBeat: "#ffb347",
		grid.ActionLineResetAll: "#f8c537",
		grid.ActionLineBounce:   "#ffcb69",
		grid.ActionLineDelay:    "#c05746",
	},
}

var cyberpunkColors = map[string]string{
	"AppTitleColor":                "#4da55a",
	"AppDescriptorColor":           "#c0f0f0",
	"AltSeqBackgroundColor":        "#300330",
	"SeqBackgroundColor":           "#070215",
	"SeqCursorColor":               "#2d555a",
	"SeqVisualColor":               "#8e44ad",
	"SeqOverlayColor":              "#4e1f7d",
	"SeqMiddleOverlayColor":        "#2e0f50",
	"SelectedAttributeColor":       "#fcee21",
	"NumberColor":                  "#ff00ff",
	"Black":                        "#070215",
	"White":                        "#edfdfd",
	"ActiveRatchetColor":           "#00ff9f",
	"MutedRatchetColor":            "#ff003c",
	"CurrentPlayingColor":          "#00ff9f",
	"ActivePlayingColor":           "#ff003c",
	"ArrangementHeaderColor":       "#edfdfd",
	"ArrangementGroupColor":        "#ff00ff",
	"ArrangementIndentColor":       "#3e1f6d",
	"ArrangementSelectedLineColor": "#1a0933",
	"PatternModeColor":             "#ff00ff",
}

var cyberpunkTheme = Theme{
	colors: cyberpunkColors,
	accentColors: []string{
		"#070215",
		"#ff003c",
		"#ff00ff",
		"#f706cf",
		"#c100fd",
		"#00ff9f",
		"#05eebb",
		"#00f0ff",
		"#05cce6",
	},
	accentIcons: []rune{
		' ',
		'◉',
		'◎',
		'◍',
		'○',
		'Ω',
		'λ',
		'⚠',
		'☣',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#070215",
		grid.ActionLineReset:    "#ff003c",
		grid.ActionLineReverse:  "#f706cf",
		grid.ActionLineSkipBeat: "#00f0ff",
		grid.ActionLineResetAll: "#00ff9f",
		grid.ActionLineBounce:   "#c100fd",
		grid.ActionLineDelay:    "#ff00ff",
	},
}

var brainiacColors = map[string]string{
	"AppTitleColor":                "#4fb99f",
	"AppDescriptorColor":           "#c4dfe6",
	"AltSeqBackgroundColor":        "#142233",
	"SeqBackgroundColor":           "#002b00",
	"SeqCursorColor":               "#006b82",
	"SeqVisualColor":               "#07889b",
	"SeqOverlayColor":              "#005111",
	"SeqMiddleOverlayColor":        "#006677",
	"SelectedAttributeColor":       "#f8b24b",
	"NumberColor":                  "#c4dfe6",
	"Black":                        "#002b00",
	"White":                        "#e8f1f2",
	"ActiveRatchetColor":           "#66b3ba",
	"MutedRatchetColor":            "#4fb99f",
	"CurrentPlayingColor":          "#66b3ba",
	"ActivePlayingColor":           "#4fb99f",
	"ArrangementHeaderColor":       "#e8f1f2",
	"ArrangementGroupColor":        "#66b3ba",
	"ArrangementIndentColor":       "#007888",
	"ArrangementSelectedLineColor": "#004d5d",
	"PatternModeColor":             "#4fb99f",
}

var brainiacTheme = Theme{
	colors: brainiacColors,
	accentColors: []string{
		"#002b33",
		"#4fb99f",
		"#66b3ba",
		"#07889b",
		"#c4dfe6",
		"#8bd7d2",
		"#1b98a2",
		"#883394",
		"#aa77cc",
	},
	accentIcons: []rune{
		' ',
		'⌘',
		'⌥',
		'∂',
		'Σ',
		'Π',
		'≡',
		'≈',
		'∞',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#002b33",
		grid.ActionLineReset:    "#4fb99f",
		grid.ActionLineReverse:  "#07889b",
		grid.ActionLineSkipBeat: "#1b98a2",
		grid.ActionLineResetAll: "#66b3ba",
		grid.ActionLineBounce:   "#8bd7d2",
		grid.ActionLineDelay:    "#c4dfe6",
	},
}

var spaceodysseyColors = map[string]string{
	"AppTitleColor":                "#d070f0",
	"AppDescriptorColor":           "#c0c0f0",
	"AltSeqBackgroundColor":        "#14142a",
	"SeqBackgroundColor":           "#020210",
	"SeqCursorColor":               "#2a6655",
	"SeqVisualColor":               "#3a3a75",
	"SeqOverlayColor":              "#2d2d50",
	"SeqMiddleOverlayColor":        "#40406f",
	"SelectedAttributeColor":       "#ffcf00",
	"NumberColor":                  "#f0c0c0",
	"Black":                        "#020210",
	"White":                        "#e6e6ff",
	"ActiveRatchetColor":           "#c0c0f0",
	"MutedRatchetColor":            "#c050c0",
	"CurrentPlayingColor":          "#c0c0f0",
	"ActivePlayingColor":           "#c050c0",
	"ArrangementHeaderColor":       "#e6e6ff",
	"ArrangementGroupColor":        "#c050c0",
	"ArrangementIndentColor":       "#40406f",
	"ArrangementSelectedLineColor": "#191936",
	"PatternModeColor":             "#9090e0",
}

var spaceodysseyTheme = Theme{
	colors: spaceodysseyColors,
	accentColors: []string{
		"#020210",
		"#c030f0",
		"#c050c0",
		"#f090df",
		"#dd80c0",
		"#f0c0c0",
		"#1fcfec",
		"#10c0dd",
		"#10a0ff",
	},
	accentIcons: []rune{
		' ',
		'☄',
		'★',
		'☽',
		'☼',
		'⋆',
		'✧',
		'✪',
		'⊛',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#020210",
		grid.ActionLineReset:    "#c050c0",
		grid.ActionLineReverse:  "#9090e0",
		grid.ActionLineSkipBeat: "#8080c0",
		grid.ActionLineResetAll: "#c0c0f0",
		grid.ActionLineBounce:   "#a0a0ff",
		grid.ActionLineDelay:    "#f0c0c0",
	},
}

var nineteenfiftyeightColors = map[string]string{
	"AppTitleColor":                "#ecdfce",
	"AppDescriptorColor":           "#d9c8af",
	"AltSeqBackgroundColor":        "#2e2418",
	"SeqBackgroundColor":           "#1a1610",
	"SeqCursorColor":               "#5a4a3c",
	"SeqVisualColor":               "#917a64",
	"SeqOverlayColor":              "#6b5744",
	"SeqMiddleOverlayColor":        "#4a3c30",
	"SelectedAttributeColor":       "#4587be",
	"NumberColor":                  "#d09554",
	"Black":                        "#1a1610",
	"White":                        "#ecdfce",
	"ActiveRatchetColor":           "#b1b85a",
	"MutedRatchetColor":            "#c94a35",
	"CurrentPlayingColor":          "#b1b85a",
	"ActivePlayingColor":           "#c94a35",
	"ArrangementHeaderColor":       "#ecdfce",
	"ArrangementGroupColor":        "#c94a35",
	"ArrangementIndentColor":       "#6b5744",
	"ArrangementSelectedLineColor": "#2e2418",
	"PatternModeColor":             "#d09554",
}

var nineteenfiftyeightTheme = Theme{
	colors: nineteenfiftyeightColors,
	accentColors: []string{
		"#1a1610",
		"#c94a35",
		"#d09554",
		"#b1b85a",
		"#8ba353",
		"#37f4d6",
		"#5597ee",
		"#5779e6",
		"#42a5e6",
	},
	accentIcons: []rune{
		' ',
		'◆',
		'■',
		'●',
		'▲',
		'◉',
		'◍',
		'◇',
		'◠',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#1a1610",
		grid.ActionLineReset:    "#c94a35",
		grid.ActionLineReverse:  "#d09554",
		grid.ActionLineSkipBeat: "#4587be",
		grid.ActionLineResetAll: "#b1b85a",
		grid.ActionLineBounce:   "#8ba353",
		grid.ActionLineDelay:    "#6b5744",
	},
}

var appleiiplusColors = map[string]string{
	"AppTitleColor":                "#33ff33",
	"AppDescriptorColor":           "#33dd33",
	"AltSeqBackgroundColor":        "#002200",
	"SeqBackgroundColor":           "#000000",
	"SeqCursorColor":               "#007700",
	"SeqVisualColor":               "#00aa00",
	"SeqOverlayColor":              "#004400",
	"SeqMiddleOverlayColor":        "#224400",
	"SelectedAttributeColor":       "#9933cc",
	"NumberColor":                  "#cc9933",
	"Black":                        "#000000",
	"White":                        "#33ff33",
	"ActiveRatchetColor":           "#66ff66",
	"MutedRatchetColor":            "#cc3333",
	"CurrentPlayingColor":          "#66ff66",
	"ActivePlayingColor":           "#cc3333",
	"ArrangementHeaderColor":       "#33ff33",
	"ArrangementGroupColor":        "#cc3333",
	"ArrangementIndentColor":       "#007700",
	"ArrangementSelectedLineColor": "#003300",
	"PatternModeColor":             "#cc9933",
}

var appleiiplusTheme = Theme{
	colors: appleiiplusColors,
	accentColors: []string{
		"#000000",
		"#cc3333",
		"#cc9933",
		"#66ff66",
		"#33cc33",
		"#7733cc",
		"#cc33cc",
		"#b23300",
		"#992300",
	},
	accentIcons: []rune{
		' ',
		'○',
		'◎',
		'●',
		'□',
		'◆',
		'▲',
		'△',
		'☐',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#000000",
		grid.ActionLineReset:    "#cc3333",
		grid.ActionLineReverse:  "#cc9933",
		grid.ActionLineSkipBeat: "#3333cc",
		grid.ActionLineResetAll: "#66ff66",
		grid.ActionLineBounce:   "#33cc33",
		grid.ActionLineDelay:    "#9933cc",
	},
}

var matrixColors = map[string]string{
	"AppTitleColor":                "#00ff00",
	"AppDescriptorColor":           "#00ee00",
	"AltSeqBackgroundColor":        "#002200",
	"SeqBackgroundColor":           "#000000",
	"SeqCursorColor":               "#00aa00",
	"SeqVisualColor":               "#008800",
	"SeqOverlayColor":              "#005500",
	"SeqMiddleOverlayColor":        "#002200",
	"SelectedAttributeColor":       "#ffffff",
	"NumberColor":                  "#88ff88",
	"Black":                        "#000000",
	"White":                        "#00ff00",
	"ActiveRatchetColor":           "#00ff00",
	"MutedRatchetColor":            "#008800",
	"CurrentPlayingColor":          "#00ff00",
	"ActivePlayingColor":           "#008800",
	"ArrangementHeaderColor":       "#00ff00",
	"ArrangementGroupColor":        "#55ff55",
	"ArrangementIndentColor":       "#004400",
	"ArrangementSelectedLineColor": "#003300",
	"PatternModeColor":             "#88ff88",
}

var matrixTheme = Theme{
	colors: matrixColors,
	accentColors: []string{
		"#000000",
		"#88ff88",
		"#55ff55",
		"#00ff00",
		"#00cc00",
		"#22cc00",
		"#44aa00",
		"#668800",
		"#88cc00",
	},
	accentIcons: []rune{
		' ',
		'0',
		'1',
		'Φ',
		'Ψ',
		'Δ',
		'Ω',
		'∑',
		'π',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#000000",
		grid.ActionLineReset:    "#008800",
		grid.ActionLineReverse:  "#00aa00",
		grid.ActionLineSkipBeat: "#55ff55",
		grid.ActionLineResetAll: "#00ff00",
		grid.ActionLineBounce:   "#88ff88",
		grid.ActionLineDelay:    "#00cc00",
	},
}

var herbieColors = map[string]string{
	"AppTitleColor":                "#ffe01b",
	"AppDescriptorColor":           "#ffd000",
	"AltSeqBackgroundColor":        "#1c1c1c",
	"SeqBackgroundColor":           "#0c0c0c",
	"SeqCursorColor":               "#3c3c3c",
	"SeqVisualColor":               "#666666",
	"SeqOverlayColor":              "#2c2c2c",
	"SeqMiddleOverlayColor":        "#242424",
	"SelectedAttributeColor":       "#1b95e0",
	"NumberColor":                  "#ffe01b",
	"Black":                        "#0c0c0c",
	"White":                        "#ffe01b",
	"ActiveRatchetColor":           "#1b95e0",
	"MutedRatchetColor":            "#e84a5f",
	"CurrentPlayingColor":          "#1b95e0",
	"ActivePlayingColor":           "#e84a5f",
	"ArrangementHeaderColor":       "#ffe01b",
	"ArrangementGroupColor":        "#e84a5f",
	"ArrangementIndentColor":       "#3c3c3c",
	"ArrangementSelectedLineColor": "#1c1c1c",
	"PatternModeColor":             "#e84a5f",
}

var herbieTheme = Theme{
	colors: herbieColors,
	accentColors: []string{
		"#0c0c0c",
		"#e84a5f",
		"#ffe01b",
		"#feae5a",
		"#f9a03f",
		"#1b95e0",
		"#55b9f3",
		"#4c7f9e",
		"#2c3e50",
	},
	accentIcons: []rune{
		' ',
		'◉',
		'◈',
		'◇',
		'◎',
		'◔',
		'◑',
		'◕',
		'●',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#0c0c0c",
		grid.ActionLineReset:    "#e84a5f",
		grid.ActionLineReverse:  "#feae5a",
		grid.ActionLineSkipBeat: "#55b9f3",
		grid.ActionLineResetAll: "#ffe01b",
		grid.ActionLineBounce:   "#f9a03f",
		grid.ActionLineDelay:    "#4c7f9e",
	},
}

var milesColors = map[string]string{
	"AppTitleColor":                "#3498db",
	"AppDescriptorColor":           "#2980b9",
	"AltSeqBackgroundColor":        "#0c2233",
	"SeqBackgroundColor":           "#000e1a",
	"SeqCursorColor":               "#1f4662",
	"SeqVisualColor":               "#2a4d6a",
	"SeqOverlayColor":              "#17313d",
	"SeqMiddleOverlayColor":        "#102837",
	"SelectedAttributeColor":       "#3498db",
	"NumberColor":                  "#f39c12",
	"Black":                        "#000e1a",
	"White":                        "#ecf0f1",
	"ActiveRatchetColor":           "#2ecc71",
	"MutedRatchetColor":            "#e74c3c",
	"CurrentPlayingColor":          "#2ecc71",
	"ActivePlayingColor":           "#e74c3c",
	"ArrangementHeaderColor":       "#ecf0f1",
	"ArrangementGroupColor":        "#e74c3c",
	"ArrangementIndentColor":       "#17313d",
	"ArrangementSelectedLineColor": "#0c2233",
	"PatternModeColor":             "#8e44ad",
}

var milesTheme = Theme{
	colors: milesColors,
	accentColors: []string{
		"#000e1a",
		"#e74c3c",
		"#8e44ad",
		"#f39c12",
		"#2ecc71",
		"#1abc9c",
		"#3498db",
		"#2980b9",
		"#1970a9",
	},
	accentIcons: []rune{
		' ',
		'▣',
		'▢',
		'▤',
		'▥',
		'▧',
		'▨',
		'▩',
		'▪',
	},
	lineActionColors: map[grid.Action]string{
		grid.ActionNothing:      "#000e1a",
		grid.ActionLineReset:    "#e74c3c",
		grid.ActionLineReverse:  "#f39c12",
		grid.ActionLineSkipBeat: "#3498db",
		grid.ActionLineResetAll: "#2ecc71",
		grid.ActionLineBounce:   "#1abc9c",
		grid.ActionLineDelay:    "#8e44ad",
	},
}

var AccentColors = []string{
	"#000000",
	"#ed3902",
	"#f564a9",
	"#f8730e",
	"#fcc05c",
	"#5cdffb",
	"#1e89ef",
	"#164de5",
	"#0246a7",
}

var AccentIcons = []rune{
	' ',
	'✤',
	'⎈',
	'⚙',
	'⊚',
	'✦',
	'❖',
	'✥',
	'❄',
}

var ActionColors = map[grid.Action]string{
	grid.ActionNothing:      "#000000",
	grid.ActionLineReset:    "#cf142b",
	grid.ActionLineReverse:  "#f8730e",
	grid.ActionLineSkipBeat: "#a9e5bb",
	grid.ActionLineResetAll: "#fcf6b1",
	grid.ActionLineBounce:   "#fcf6b1",
	grid.ActionLineDelay:    "#cc4bc2",
}

// Colors
var AppTitleColor,
	AppDescriptorColor,
	AltSeqBackgroundColor,
	SeqBackgroundColor,
	SeqCursorColor,
	SeqVisualColor,
	SeqOverlayColor,
	SeqMiddleOverlayColor,
	PatternModeColor,
	SelectedAttributeColor,
	NumberColor,
	Black,
	White,
	ActiveRatchetColor,
	MutedRatchetColor,
	CurrentPlayingColor,
	ActivePlayingColor,
	ArrangementHeaderColor,
	ArrangementGroupColor,
	ArrangementIndentColor,
	ArrangementSelectedLineColor color.Color

// Styles
var AppTitleStyle,
	AppDescriptorStyle,
	ActiveStyle,
	MutedStyle,
	AltArtStyle,
	LineCursorStyle,
	SelectedStyle,
	NumberStyle,
	AccentModeStyle,
	BlackKeyStyle,
	WhiteKeyStyle,
	LineNumberStyle,
	GroupStyle,
	IndentStyle,
	NodeRowStyle,
	SeqBorderStyle lipgloss.Style

// Symbols
var CurrentlyPlayingSymbol,
	OverlayCurrentlyPlayingSymbol,
	OverlayCurrentlyLoopingSymbol,
	ActiveSymbol string

var Themes = []string{
	"default",
	"seafoam",
	"dynamite",
	"springtime",
	"orangegrove",
	"cyberpunk",
	"brainiac",
	"spaceodyssey",
	"1958",
	"appleiiplus",
	"matrix",
	"herbie",
	"miles",
}

func ChooseTheme(colorscheme string) {
	switch colorscheme {
	case "default":
		ApplyTheme(defaultTheme)
	case "seafoam":
		ApplyTheme(seafoamTheme)
	case "dynamite":
		ApplyTheme(dynamiteTheme)
	case "springtime":
		ApplyTheme(springtimeTheme)
	case "orangegrove":
		ApplyTheme(orangegroveTheme)
	case "cyberpunk":
		ApplyTheme(cyberpunkTheme)
	case "brainiac":
		ApplyTheme(brainiacTheme)
	case "spaceodyssey":
		ApplyTheme(spaceodysseyTheme)
	case "1958":
		ApplyTheme(nineteenfiftyeightTheme)
	case "appleiiplus":
		ApplyTheme(appleiiplusTheme)
	case "matrix":
		ApplyTheme(matrixTheme)
	case "herbie":
		ApplyTheme(herbieTheme)
	case "miles":
		ApplyTheme(milesTheme)
	default:
		ApplyTheme(milesTheme)
	}

	EvaluateStyles()
	EvaluateSymbols()
}

func ApplyTheme(theme Theme) {
	SetColors(theme.colors)
	SetAccentColors(theme.accentColors)
	SetAccentIcons(theme.accentIcons)
	SetActionColors(theme.lineActionColors)
}

func SetColors(newColors map[string]string) {
	for key, value := range newColors {
		newColor := lipgloss.Color(value)
		switch key {
		case "AppTitleColor":
			AppTitleColor = newColor
		case "AppDescriptorColor":
			AppDescriptorColor = newColor
		case "AltSeqBackgroundColor":
			AltSeqBackgroundColor = newColor
		case "SeqBackgroundColor":
			SeqBackgroundColor = newColor
		case "SeqCursorColor":
			SeqCursorColor = newColor
		case "SeqVisualColor":
			SeqVisualColor = newColor
		case "SeqOverlayColor":
			SeqOverlayColor = newColor
		case "SeqMiddleOverlayColor":
			SeqMiddleOverlayColor = newColor
		case "SelectedAttributeColor":
			SelectedAttributeColor = newColor
		case "NumberColor":
			NumberColor = newColor
		case "Black":
			Black = newColor
		case "White":
			White = newColor
		case "ActiveRatchetColor":
			ActiveRatchetColor = newColor
		case "MutedRatchetColor":
			MutedRatchetColor = newColor
		case "CurrentPlayingColor":
			CurrentPlayingColor = newColor
		case "ActivePlayingColor":
			ActivePlayingColor = newColor
		case "ArrangementHeaderColor":
			ArrangementHeaderColor = newColor
		case "ArrangementGroupColor":
			ArrangementGroupColor = newColor
		case "ArrangementIndentColor":
			ArrangementIndentColor = newColor
		case "ArrangementSelectedLineColor":
			ArrangementSelectedLineColor = newColor
		case "PatternModeColor":
			PatternModeColor = newColor
		}
	}
}

func EvaluateSymbols() {
	CurrentlyPlayingSymbol = lipgloss.NewStyle().Foreground(CurrentPlayingColor).Render(" \u25CF ")
	OverlayCurrentlyPlayingSymbol = lipgloss.NewStyle().Background(SeqOverlayColor).Foreground(CurrentPlayingColor).Render(" \u25CF ")
	OverlayCurrentlyLoopingSymbol = lipgloss.NewStyle().Background(SeqOverlayColor).Foreground(CurrentPlayingColor).Render(" ∞ ")
	ActiveSymbol = lipgloss.NewStyle().Background(SeqOverlayColor).Foreground(ActivePlayingColor).Render(" \u25C9 ")
}

func EvaluateStyles() {

	ActiveStyle = lipgloss.NewStyle().Foreground(ActiveRatchetColor)
	MutedStyle = lipgloss.NewStyle().Foreground(MutedRatchetColor)
	AltArtStyle = lipgloss.NewStyle().Foreground(Darken(AppTitleColor, 20.0))
	LineCursorStyle = lipgloss.NewStyle().Foreground(Black).Background(Lighten(AltSeqBackgroundColor, 150.0))
	SelectedStyle = lipgloss.NewStyle().Background(SelectedAttributeColor).Foreground(Black)
	NumberStyle = lipgloss.NewStyle().Foreground(NumberColor)
	AccentModeStyle = lipgloss.NewStyle().Background(PatternModeColor).Foreground(Black)
	BlackKeyStyle = lipgloss.NewStyle().Background(Black).Foreground(White)
	WhiteKeyStyle = lipgloss.NewStyle().Background(White).Foreground(Black)
	LineNumberStyle = lipgloss.NewStyle().Foreground(Lighten(AppTitleColor, 10.0))

	GroupStyle = lipgloss.NewStyle().
		Foreground(ArrangementGroupColor).
		MarginRight(1).
		Bold(true)

	IndentStyle = lipgloss.NewStyle().
		Foreground(ArrangementIndentColor)

	NodeRowStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		MarginBottom(0)

	SeqBorderStyle = lipgloss.NewStyle().Foreground(Lighten(AltSeqBackgroundColor, 100.0))
	AppTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(AppTitleColor)
	AppDescriptorStyle = lipgloss.NewStyle().Foreground(Darken(AppTitleColor, 20.0))
}

func SetAccentColors(accentColors []string) {
	for i := range len(accentColors) {
		if i == 0 {
			continue
		}
		AccentColors[i] = accentColors[i]
	}
}

func SetAccentIcons(accentIcons []rune) {
	for i := range len(accentIcons) {
		if i == 0 {
			continue
		}
		AccentIcons[i] = accentIcons[i]
	}
}

func SetActionColors(actionColors map[grid.Action]string) {
	for k := range actionColors {
		if k == grid.ActionNothing {
			continue
		}
		ActionColors[k] = actionColors[k]
	}
}
