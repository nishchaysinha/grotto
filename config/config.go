package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Theme holds every color used by the editor, keyed by TOML field name.
// All values are hex color strings (e.g. "#FF0000").
type Theme struct {
	// ── App chrome ───────────────────────────────────────────────────────────
	TitleBg      string `toml:"title_bg"`
	TitleFg      string `toml:"title_fg"`
	BtnBg        string `toml:"btn_bg"`
	BtnFg        string `toml:"btn_fg"`
	BtnActiveBg  string `toml:"btn_active_bg"`
	BtnActiveFg  string `toml:"btn_active_fg"`
	StatusBg     string `toml:"status_bg"`
	StatusFg     string `toml:"status_fg"`
	BorderDim    string `toml:"border_dim"`
	BorderHover  string `toml:"border_hover"`
	BorderActive string `toml:"border_active"`

	// ── Editor ───────────────────────────────────────────────────────────────
	GutterFg    string `toml:"gutter_fg"`
	CurLineBg   string `toml:"cur_line_bg"`
	TabBarBg    string `toml:"tab_bar_bg"`
	TabFg       string `toml:"tab_fg"`
	TabActiveFg string `toml:"tab_active_fg"`
	TabActiveBg string `toml:"tab_active_bg"`
	SelectionBg string `toml:"selection_bg"`
	BracketHLBg string `toml:"bracket_hl_bg"`

	// ── Search overlay ───────────────────────────────────────────────────────
	SearchOverlayBg     string `toml:"search_overlay_bg"`
	SearchOverlayFg     string `toml:"search_overlay_fg"`
	SearchInputBg       string `toml:"search_input_bg"`
	SearchInputFg       string `toml:"search_input_fg"`
	SearchMatchBg       string `toml:"search_match_bg"`
	SearchMatchFg       string `toml:"search_match_fg"`
	SearchActiveMatchBg string `toml:"search_active_match_bg"`
	SearchActiveMatchFg string `toml:"search_active_match_fg"`
	SearchLabelFg       string `toml:"search_label_fg"`

	// ── Sidebar ──────────────────────────────────────────────────────────────
	SidebarSelectedBg string `toml:"sidebar_selected_bg"`
	SidebarSelectedFg string `toml:"sidebar_selected_fg"`
	SidebarDirIcon    string `toml:"sidebar_dir_icon"`

	// ── Git status ───────────────────────────────────────────────────────────
	GitAdded     string `toml:"git_added"`
	GitModified  string `toml:"git_modified"`
	GitDeleted   string `toml:"git_deleted"`
	GitRenamed   string `toml:"git_renamed"`
	GitUntracked string `toml:"git_untracked"`
}

// Config is the top-level configuration structure.
type Config struct {
	// ThemeName selects a built-in theme by its identifier (e.g. "nord",
	// "solarized-dark"). If empty, defaults to "dracula". Per-field overrides
	// in the [theme] table are applied on top of the selected base.
	ThemeName string `toml:"theme_name"`
	Theme     Theme  `toml:"theme"`
}

// DefaultTheme returns the built-in Dracula theme — grotto's default look.
// Callers wanting a specific named theme should use ThemeByName instead.
func DefaultTheme() Theme {
	return DraculaTheme()
}

// Load reads ~/.config/grotto/config.toml if it exists, fills any missing
// fields with defaults, and returns the result.
//
// Resolution order for theme colors:
//  1. Start with Dracula as the baseline.
//  2. If `theme_name = "..."` is set and names a built-in theme, use that as
//     the baseline instead.
//  3. Apply any per-field overrides from the [theme] table on top.
func Load() Config {
	cfg := Config{Theme: DefaultTheme()}

	path := configPath()
	if path == "" {
		return cfg
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg // file doesn't exist — use defaults silently
	}

	var partial Config
	if _, err := toml.Decode(string(data), &partial); err != nil {
		return cfg // malformed TOML — use defaults silently
	}

	if partial.ThemeName != "" {
		if base, ok := ThemeByName(partial.ThemeName); ok {
			cfg.Theme = base
			cfg.ThemeName = partial.ThemeName
		}
	}

	// Per-field overrides layer on top of the selected base.
	mergeTheme(&cfg.Theme, partial.Theme)
	return cfg
}

// configPath returns the path to the user's config file, or "" if the home
// directory cannot be determined.
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "grotto", "config.toml")
}

// mergeTheme copies non-empty fields from src into dst.
func mergeTheme(dst *Theme, src Theme) {
	if src.TitleBg != "" {
		dst.TitleBg = src.TitleBg
	}
	if src.TitleFg != "" {
		dst.TitleFg = src.TitleFg
	}
	if src.BtnBg != "" {
		dst.BtnBg = src.BtnBg
	}
	if src.BtnFg != "" {
		dst.BtnFg = src.BtnFg
	}
	if src.BtnActiveBg != "" {
		dst.BtnActiveBg = src.BtnActiveBg
	}
	if src.BtnActiveFg != "" {
		dst.BtnActiveFg = src.BtnActiveFg
	}
	if src.StatusBg != "" {
		dst.StatusBg = src.StatusBg
	}
	if src.StatusFg != "" {
		dst.StatusFg = src.StatusFg
	}
	if src.BorderDim != "" {
		dst.BorderDim = src.BorderDim
	}
	if src.BorderHover != "" {
		dst.BorderHover = src.BorderHover
	}
	if src.BorderActive != "" {
		dst.BorderActive = src.BorderActive
	}
	if src.GutterFg != "" {
		dst.GutterFg = src.GutterFg
	}
	if src.CurLineBg != "" {
		dst.CurLineBg = src.CurLineBg
	}
	if src.TabBarBg != "" {
		dst.TabBarBg = src.TabBarBg
	}
	if src.TabFg != "" {
		dst.TabFg = src.TabFg
	}
	if src.TabActiveFg != "" {
		dst.TabActiveFg = src.TabActiveFg
	}
	if src.TabActiveBg != "" {
		dst.TabActiveBg = src.TabActiveBg
	}
	if src.SelectionBg != "" {
		dst.SelectionBg = src.SelectionBg
	}
	if src.BracketHLBg != "" {
		dst.BracketHLBg = src.BracketHLBg
	}
	if src.SearchOverlayBg != "" {
		dst.SearchOverlayBg = src.SearchOverlayBg
	}
	if src.SearchOverlayFg != "" {
		dst.SearchOverlayFg = src.SearchOverlayFg
	}
	if src.SearchInputBg != "" {
		dst.SearchInputBg = src.SearchInputBg
	}
	if src.SearchInputFg != "" {
		dst.SearchInputFg = src.SearchInputFg
	}
	if src.SearchMatchBg != "" {
		dst.SearchMatchBg = src.SearchMatchBg
	}
	if src.SearchMatchFg != "" {
		dst.SearchMatchFg = src.SearchMatchFg
	}
	if src.SearchActiveMatchBg != "" {
		dst.SearchActiveMatchBg = src.SearchActiveMatchBg
	}
	if src.SearchActiveMatchFg != "" {
		dst.SearchActiveMatchFg = src.SearchActiveMatchFg
	}
	if src.SearchLabelFg != "" {
		dst.SearchLabelFg = src.SearchLabelFg
	}
	if src.SidebarSelectedBg != "" {
		dst.SidebarSelectedBg = src.SidebarSelectedBg
	}
	if src.SidebarSelectedFg != "" {
		dst.SidebarSelectedFg = src.SidebarSelectedFg
	}
	if src.SidebarDirIcon != "" {
		dst.SidebarDirIcon = src.SidebarDirIcon
	}
	if src.GitAdded != "" {
		dst.GitAdded = src.GitAdded
	}
	if src.GitModified != "" {
		dst.GitModified = src.GitModified
	}
	if src.GitDeleted != "" {
		dst.GitDeleted = src.GitDeleted
	}
	if src.GitRenamed != "" {
		dst.GitRenamed = src.GitRenamed
	}
	if src.GitUntracked != "" {
		dst.GitUntracked = src.GitUntracked
	}
}
