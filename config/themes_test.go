package config

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var hexColor = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Every field of every built-in theme must be a valid 6-char hex color.
// Missing fields would surface as empty strings that crash the lipgloss
// renderer; this catches typos and forgotten fields at compile-time.
func TestBuiltInThemes_AllFieldsPopulated(t *testing.T) {
	themes := BuiltInThemes()
	if len(themes) == 0 {
		t.Fatal("BuiltInThemes returned empty map")
	}
	for name, theme := range themes {
		t.Run(name, func(t *testing.T) {
			checkField(t, "TitleBg", theme.TitleBg)
			checkField(t, "TitleFg", theme.TitleFg)
			checkField(t, "BtnBg", theme.BtnBg)
			checkField(t, "BtnFg", theme.BtnFg)
			checkField(t, "BtnActiveBg", theme.BtnActiveBg)
			checkField(t, "BtnActiveFg", theme.BtnActiveFg)
			checkField(t, "StatusBg", theme.StatusBg)
			checkField(t, "StatusFg", theme.StatusFg)
			checkField(t, "BorderDim", theme.BorderDim)
			checkField(t, "BorderHover", theme.BorderHover)
			checkField(t, "BorderActive", theme.BorderActive)
			checkField(t, "GutterFg", theme.GutterFg)
			checkField(t, "CurLineBg", theme.CurLineBg)
			checkField(t, "TabBarBg", theme.TabBarBg)
			checkField(t, "TabFg", theme.TabFg)
			checkField(t, "TabActiveFg", theme.TabActiveFg)
			checkField(t, "TabActiveBg", theme.TabActiveBg)
			checkField(t, "SelectionBg", theme.SelectionBg)
			checkField(t, "BracketHLBg", theme.BracketHLBg)
			checkField(t, "SearchOverlayBg", theme.SearchOverlayBg)
			checkField(t, "SearchOverlayFg", theme.SearchOverlayFg)
			checkField(t, "SearchInputBg", theme.SearchInputBg)
			checkField(t, "SearchInputFg", theme.SearchInputFg)
			checkField(t, "SearchMatchBg", theme.SearchMatchBg)
			checkField(t, "SearchMatchFg", theme.SearchMatchFg)
			checkField(t, "SearchActiveMatchBg", theme.SearchActiveMatchBg)
			checkField(t, "SearchActiveMatchFg", theme.SearchActiveMatchFg)
			checkField(t, "SearchLabelFg", theme.SearchLabelFg)
			checkField(t, "SidebarSelectedBg", theme.SidebarSelectedBg)
			checkField(t, "SidebarSelectedFg", theme.SidebarSelectedFg)
			checkField(t, "SidebarDirIcon", theme.SidebarDirIcon)
			checkField(t, "GitAdded", theme.GitAdded)
			checkField(t, "GitModified", theme.GitModified)
			checkField(t, "GitDeleted", theme.GitDeleted)
			checkField(t, "GitRenamed", theme.GitRenamed)
			checkField(t, "GitUntracked", theme.GitUntracked)
		})
	}
}

// BuiltInThemeNames must return exactly the keys of BuiltInThemes so the
// command palette and the resolver stay in sync.
func TestBuiltInThemeNames_MatchesMap(t *testing.T) {
	names := BuiltInThemeNames()
	themes := BuiltInThemes()
	if len(names) != len(themes) {
		t.Errorf("names=%d themes=%d — out of sync", len(names), len(themes))
	}
	for _, n := range names {
		if _, ok := themes[n]; !ok {
			t.Errorf("BuiltInThemeNames lists %q but BuiltInThemes has no such key", n)
		}
	}
}

func TestThemeByName_CaseInsensitive(t *testing.T) {
	cases := []string{"nord", "NORD", "Nord", "nOrD"}
	for _, c := range cases {
		if _, ok := ThemeByName(c); !ok {
			t.Errorf("ThemeByName(%q): not found", c)
		}
	}
}

func TestThemeByName_Unknown(t *testing.T) {
	if _, ok := ThemeByName("does-not-exist"); ok {
		t.Error("ThemeByName returned ok=true for unknown theme")
	}
}

// Dracula remains the default — catches accidental swaps.
func TestDefaultTheme_IsDracula(t *testing.T) {
	if DefaultTheme() != DraculaTheme() {
		t.Error("DefaultTheme() no longer returns DraculaTheme()")
	}
}

// Load honors theme_name, applies built-in, lets [theme] override on top.
func TestLoad_ThemeNameSelectsBuiltin(t *testing.T) {
	dir := t.TempDir()
	// Point HOME at a temp dir so configPath resolves there.
	t.Setenv("HOME", dir)
	cfgDir := filepath.Join(dir, ".config", "grotto")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	tomlBody := `theme_name = "nord"
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlBody), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := Load()
	if cfg.ThemeName != "nord" {
		t.Errorf("ThemeName: got %q, want %q", cfg.ThemeName, "nord")
	}
	if cfg.Theme.TitleBg != NordTheme().TitleBg {
		t.Errorf("TitleBg: got %q, want Nord's %q", cfg.Theme.TitleBg, NordTheme().TitleBg)
	}
}

func TestLoad_PerFieldOverrideOnTopOfNamedTheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfgDir := filepath.Join(dir, ".config", "grotto")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Pick Nord as base, but recolor only the title bar.
	tomlBody := `theme_name = "nord"

[theme]
title_bg = "#FF00FF"
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlBody), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := Load()
	if cfg.Theme.TitleBg != "#FF00FF" {
		t.Errorf("title_bg override ignored: got %q, want %q", cfg.Theme.TitleBg, "#FF00FF")
	}
	// Other Nord fields should remain intact.
	if cfg.Theme.TitleFg != NordTheme().TitleFg {
		t.Errorf("title_fg: got %q, want Nord's %q (override should only affect title_bg)",
			cfg.Theme.TitleFg, NordTheme().TitleFg)
	}
}

func TestLoad_UnknownThemeNameFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	cfgDir := filepath.Join(dir, ".config", "grotto")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	tomlBody := `theme_name = "not-a-real-theme"
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlBody), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := Load()
	// Falls back to Dracula silently. ThemeName stays empty to signal the
	// config value was not applied.
	if cfg.ThemeName != "" {
		t.Errorf("ThemeName: got %q, want empty on unknown", cfg.ThemeName)
	}
	if cfg.Theme != DraculaTheme() {
		t.Error("expected Dracula fallback when theme_name is unknown")
	}
}

func TestLoad_NoConfigUsesDracula(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// No config file written.
	cfg := Load()
	if cfg.Theme != DraculaTheme() {
		t.Error("no config file should yield Dracula theme")
	}
}

func checkField(t *testing.T, name, value string) {
	t.Helper()
	if value == "" {
		t.Errorf("%s is empty", name)
		return
	}
	if !hexColor.MatchString(value) {
		t.Errorf("%s = %q is not a 6-char hex color like #RRGGBB", name, value)
	}
}
