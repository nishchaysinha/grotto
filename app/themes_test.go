package app

import (
	"strings"
	"testing"

	"github.com/owomeister/grotto/config"
)

// Command palette must list every built-in theme plus Quit / AI commands.
func TestCommandNames_IncludesAllBuiltInThemes(t *testing.T) {
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	names := m.commandNames()

	seen := map[string]bool{}
	for _, n := range names {
		seen[n] = true
	}

	for _, tname := range config.BuiltInThemeNames() {
		entry := "Theme: " + tname
		if !seen[entry] {
			t.Errorf("command palette missing %q", entry)
		}
	}
	if !seen["Quit"] {
		t.Error("command palette missing \"Quit\"")
	}
}

// Theme entries follow the "Theme: <name>" format that execCommand parses.
func TestCommandNames_ThemeEntryFormat(t *testing.T) {
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	for _, n := range m.commandNames() {
		if !strings.HasPrefix(n, "Theme: ") {
			continue
		}
		tname := strings.TrimPrefix(n, "Theme: ")
		if _, ok := config.ThemeByName(tname); !ok {
			t.Errorf("command %q references unknown theme %q", n, tname)
		}
	}
}

// Selecting a theme via the command palette routes through applyNamedTheme.
// We can't easily assert the lipgloss styles changed without reflection,
// but we can verify execCommand accepts "Theme: nord" without error and
// returns nil (no cmd is needed — theme switch is purely side-effecting).
func TestExecCommand_ThemeSwitch(t *testing.T) {
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	// Mark panels clean so we can detect that the switch dirties them.
	m.dirtyPanels = 0
	m.cachedSidebar = "cached-sidebar"
	m.cachedTerminal = "cached-terminal"
	m.cachedAI = "cached-ai"

	cmd := m.execCommand("Theme: nord")
	if cmd != nil {
		t.Errorf("execCommand(\"Theme: nord\") returned non-nil cmd %T", cmd())
	}
	if m.dirtyPanels != dirtyAll {
		t.Errorf("dirtyPanels: got %#x, want dirtyAll (%#x)", m.dirtyPanels, dirtyAll)
	}
	if m.cachedSidebar != "" || m.cachedTerminal != "" || m.cachedAI != "" {
		t.Error("theme switch did not invalidate cached panel renders")
	}
}

// Unknown themes are ignored silently — they shouldn't crash or mark panels dirty.
func TestExecCommand_ThemeSwitch_Unknown(t *testing.T) {
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	m.dirtyPanels = 0
	m.cachedSidebar = "preserved"

	cmd := m.execCommand("Theme: not-real")
	if cmd != nil {
		t.Error("unknown theme returned non-nil cmd")
	}
	if m.dirtyPanels != 0 {
		t.Error("unknown theme marked panels dirty")
	}
	if m.cachedSidebar != "preserved" {
		t.Error("unknown theme cleared cached sidebar")
	}
}
