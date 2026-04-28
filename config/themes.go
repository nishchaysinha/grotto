package config

import "strings"

// BuiltInThemes returns the palette of named themes shipped with grotto.
// The map key is the lowercase identifier used in `theme = "..."` config and
// in the command palette ("Theme: Nord" → "nord").
func BuiltInThemes() map[string]Theme {
	return map[string]Theme{
		"dracula":         DraculaTheme(),
		"nord":            NordTheme(),
		"solarized-dark":  SolarizedDarkTheme(),
		"solarized-light": SolarizedLightTheme(),
		"gruvbox-dark":    GruvboxDarkTheme(),
		"tokyonight":      TokyoNightTheme(),
	}
}

// BuiltInThemeNames returns the stable sorted list of theme identifiers
// for display in UI (command palette, docs, etc.).
func BuiltInThemeNames() []string {
	return []string{
		"dracula",
		"nord",
		"solarized-dark",
		"solarized-light",
		"gruvbox-dark",
		"tokyonight",
	}
}

// ThemeByName returns the theme with the given name and whether it was found.
// Matching is case-insensitive.
func ThemeByName(name string) (Theme, bool) {
	t, ok := BuiltInThemes()[strings.ToLower(name)]
	return t, ok
}

// DraculaTheme is grotto's default theme — dark, purple-forward.
func DraculaTheme() Theme {
	return Theme{
		TitleBg:      "#7D56F4",
		TitleFg:      "#FAFAFA",
		BtnBg:        "#5A3EC8",
		BtnFg:        "#FAFAFA",
		BtnActiveBg:  "#FAFAFA",
		BtnActiveFg:  "#5A3EC8",
		StatusBg:     "#3C3C3C",
		StatusFg:     "#AAAAAA",
		BorderDim:    "#555555",
		BorderHover:  "#A98AFF",
		BorderActive: "#7D56F4",

		GutterFg:    "#555555",
		CurLineBg:   "#2A2A2A",
		TabBarBg:    "#21222C",
		TabFg:       "#888888",
		TabActiveFg: "#FFFFFF",
		TabActiveBg: "#44475a",
		SelectionBg: "#44475a",
		BracketHLBg: "#44475a",

		SearchOverlayBg:     "#282A36",
		SearchOverlayFg:     "#F8F8F2",
		SearchInputBg:       "#44475A",
		SearchInputFg:       "#F8F8F2",
		SearchMatchBg:       "#FFB86C",
		SearchMatchFg:       "#282A36",
		SearchActiveMatchBg: "#FF79C6",
		SearchActiveMatchFg: "#282A36",
		SearchLabelFg:       "#6272A4",

		SidebarSelectedBg: "#7D56F4",
		SidebarSelectedFg: "#FAFAFA",
		SidebarDirIcon:    "#89DDFF",

		GitAdded:     "#50FA7B",
		GitModified:  "#E5C07B",
		GitDeleted:   "#E06C75",
		GitRenamed:   "#61AFEF",
		GitUntracked: "#A9DC76",
	}
}

// NordTheme is the Nord palette (arctic, muted blues).
// https://www.nordtheme.com
func NordTheme() Theme {
	return Theme{
		TitleBg:      "#5E81AC",
		TitleFg:      "#ECEFF4",
		BtnBg:        "#4C566A",
		BtnFg:        "#ECEFF4",
		BtnActiveBg:  "#88C0D0",
		BtnActiveFg:  "#2E3440",
		StatusBg:     "#3B4252",
		StatusFg:     "#D8DEE9",
		BorderDim:    "#4C566A",
		BorderHover:  "#81A1C1",
		BorderActive: "#88C0D0",

		GutterFg:    "#4C566A",
		CurLineBg:   "#3B4252",
		TabBarBg:    "#2E3440",
		TabFg:       "#81A1C1",
		TabActiveFg: "#ECEFF4",
		TabActiveBg: "#434C5E",
		SelectionBg: "#434C5E",
		BracketHLBg: "#4C566A",

		SearchOverlayBg:     "#2E3440",
		SearchOverlayFg:     "#ECEFF4",
		SearchInputBg:       "#3B4252",
		SearchInputFg:       "#ECEFF4",
		SearchMatchBg:       "#EBCB8B",
		SearchMatchFg:       "#2E3440",
		SearchActiveMatchBg: "#D08770",
		SearchActiveMatchFg: "#2E3440",
		SearchLabelFg:       "#81A1C1",

		SidebarSelectedBg: "#5E81AC",
		SidebarSelectedFg: "#ECEFF4",
		SidebarDirIcon:    "#88C0D0",

		GitAdded:     "#A3BE8C",
		GitModified:  "#EBCB8B",
		GitDeleted:   "#BF616A",
		GitRenamed:   "#81A1C1",
		GitUntracked: "#8FBCBB",
	}
}

// SolarizedDarkTheme — Ethan Schoonover's Solarized (dark variant).
// https://ethanschoonover.com/solarized
func SolarizedDarkTheme() Theme {
	return Theme{
		TitleBg:      "#268BD2",
		TitleFg:      "#FDF6E3",
		BtnBg:        "#073642",
		BtnFg:        "#93A1A1",
		BtnActiveBg:  "#93A1A1",
		BtnActiveFg:  "#002B36",
		StatusBg:     "#073642",
		StatusFg:     "#93A1A1",
		BorderDim:    "#586E75",
		BorderHover:  "#2AA198",
		BorderActive: "#268BD2",

		GutterFg:    "#586E75",
		CurLineBg:   "#073642",
		TabBarBg:    "#002B36",
		TabFg:       "#586E75",
		TabActiveFg: "#FDF6E3",
		TabActiveBg: "#073642",
		SelectionBg: "#073642",
		BracketHLBg: "#073642",

		SearchOverlayBg:     "#002B36",
		SearchOverlayFg:     "#FDF6E3",
		SearchInputBg:       "#073642",
		SearchInputFg:       "#FDF6E3",
		SearchMatchBg:       "#B58900",
		SearchMatchFg:       "#002B36",
		SearchActiveMatchBg: "#CB4B16",
		SearchActiveMatchFg: "#FDF6E3",
		SearchLabelFg:       "#586E75",

		SidebarSelectedBg: "#268BD2",
		SidebarSelectedFg: "#FDF6E3",
		SidebarDirIcon:    "#2AA198",

		GitAdded:     "#859900",
		GitModified:  "#B58900",
		GitDeleted:   "#DC322F",
		GitRenamed:   "#6C71C4",
		GitUntracked: "#2AA198",
	}
}

// SolarizedLightTheme — light variant of Solarized.
func SolarizedLightTheme() Theme {
	return Theme{
		TitleBg:      "#268BD2",
		TitleFg:      "#FDF6E3",
		BtnBg:        "#EEE8D5",
		BtnFg:        "#586E75",
		BtnActiveBg:  "#586E75",
		BtnActiveFg:  "#FDF6E3",
		StatusBg:     "#EEE8D5",
		StatusFg:     "#586E75",
		BorderDim:    "#93A1A1",
		BorderHover:  "#2AA198",
		BorderActive: "#268BD2",

		GutterFg:    "#93A1A1",
		CurLineBg:   "#EEE8D5",
		TabBarBg:    "#FDF6E3",
		TabFg:       "#93A1A1",
		TabActiveFg: "#002B36",
		TabActiveBg: "#EEE8D5",
		SelectionBg: "#EEE8D5",
		BracketHLBg: "#EEE8D5",

		SearchOverlayBg:     "#FDF6E3",
		SearchOverlayFg:     "#002B36",
		SearchInputBg:       "#EEE8D5",
		SearchInputFg:       "#002B36",
		SearchMatchBg:       "#B58900",
		SearchMatchFg:       "#FDF6E3",
		SearchActiveMatchBg: "#CB4B16",
		SearchActiveMatchFg: "#FDF6E3",
		SearchLabelFg:       "#93A1A1",

		SidebarSelectedBg: "#268BD2",
		SidebarSelectedFg: "#FDF6E3",
		SidebarDirIcon:    "#2AA198",

		GitAdded:     "#859900",
		GitModified:  "#B58900",
		GitDeleted:   "#DC322F",
		GitRenamed:   "#6C71C4",
		GitUntracked: "#2AA198",
	}
}

// GruvboxDarkTheme — warm retro palette.
// https://github.com/morhetz/gruvbox
func GruvboxDarkTheme() Theme {
	return Theme{
		TitleBg:      "#D79921",
		TitleFg:      "#282828",
		BtnBg:        "#3C3836",
		BtnFg:        "#EBDBB2",
		BtnActiveBg:  "#FABD2F",
		BtnActiveFg:  "#282828",
		StatusBg:     "#3C3836",
		StatusFg:     "#A89984",
		BorderDim:    "#504945",
		BorderHover:  "#FABD2F",
		BorderActive: "#D79921",

		GutterFg:    "#504945",
		CurLineBg:   "#3C3836",
		TabBarBg:    "#1D2021",
		TabFg:       "#928374",
		TabActiveFg: "#FBF1C7",
		TabActiveBg: "#3C3836",
		SelectionBg: "#504945",
		BracketHLBg: "#504945",

		SearchOverlayBg:     "#282828",
		SearchOverlayFg:     "#EBDBB2",
		SearchInputBg:       "#3C3836",
		SearchInputFg:       "#EBDBB2",
		SearchMatchBg:       "#FABD2F",
		SearchMatchFg:       "#282828",
		SearchActiveMatchBg: "#FE8019",
		SearchActiveMatchFg: "#282828",
		SearchLabelFg:       "#928374",

		SidebarSelectedBg: "#D79921",
		SidebarSelectedFg: "#282828",
		SidebarDirIcon:    "#83A598",

		GitAdded:     "#B8BB26",
		GitModified:  "#FABD2F",
		GitDeleted:   "#FB4934",
		GitRenamed:   "#83A598",
		GitUntracked: "#8EC07C",
	}
}

// TokyoNightTheme — Tokyo Night (dark) palette.
// https://github.com/enkia/tokyo-night-vscode-theme
func TokyoNightTheme() Theme {
	return Theme{
		TitleBg:      "#7AA2F7",
		TitleFg:      "#1A1B26",
		BtnBg:        "#24283B",
		BtnFg:        "#C0CAF5",
		BtnActiveBg:  "#BB9AF7",
		BtnActiveFg:  "#1A1B26",
		StatusBg:     "#16161E",
		StatusFg:     "#A9B1D6",
		BorderDim:    "#414868",
		BorderHover:  "#7AA2F7",
		BorderActive: "#BB9AF7",

		GutterFg:    "#3B4261",
		CurLineBg:   "#24283B",
		TabBarBg:    "#16161E",
		TabFg:       "#565F89",
		TabActiveFg: "#C0CAF5",
		TabActiveBg: "#1A1B26",
		SelectionBg: "#283457",
		BracketHLBg: "#364A82",

		SearchOverlayBg:     "#1A1B26",
		SearchOverlayFg:     "#C0CAF5",
		SearchInputBg:       "#24283B",
		SearchInputFg:       "#C0CAF5",
		SearchMatchBg:       "#E0AF68",
		SearchMatchFg:       "#1A1B26",
		SearchActiveMatchBg: "#F7768E",
		SearchActiveMatchFg: "#1A1B26",
		SearchLabelFg:       "#565F89",

		SidebarSelectedBg: "#7AA2F7",
		SidebarSelectedFg: "#1A1B26",
		SidebarDirIcon:    "#7DCFFF",

		GitAdded:     "#9ECE6A",
		GitModified:  "#E0AF68",
		GitDeleted:   "#F7768E",
		GitRenamed:   "#7AA2F7",
		GitUntracked: "#73DACA",
	}
}
