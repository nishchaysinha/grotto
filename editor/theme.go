package editor

import (
	"charm.land/lipgloss/v2"

	"github.com/owomeister/grotto/config"
)

// ApplyTheme reinitializes all package-level style variables from t.
// Must be called before creating any editor components.
func ApplyTheme(t config.Theme) {
	c := lipgloss.Color

	// view.go styles
	gutterStyle = lipgloss.NewStyle().Foreground(c(t.GutterFg))
	curLineStyle = lipgloss.NewStyle().Background(c(t.CurLineBg))
	noFileStyle = lipgloss.NewStyle().Foreground(c(t.GutterFg))
	bracketHLStyle = lipgloss.NewStyle().Background(c(t.BracketHLBg)).Bold(true)
	tabStyle = lipgloss.NewStyle().Padding(0, 1).Foreground(c(t.TabFg))
	tabActiveStyle = lipgloss.NewStyle().Padding(0, 1).
		Foreground(c(t.TabActiveFg)).Background(c(t.TabActiveBg))
	tabBarBg = lipgloss.NewStyle().Background(c(t.TabBarBg))
	gitBarAdded = lipgloss.NewStyle().Foreground(c(t.GitAdded))
	gitBarModified = lipgloss.NewStyle().Foreground(c(t.GitModified))
	gitBarDeleted = lipgloss.NewStyle().Foreground(c(t.GitDeleted))
	gutterSep = lipgloss.NewStyle().Foreground(c(t.GutterFg))

	// search.go styles
	overlayStyle = lipgloss.NewStyle().
		Background(c(t.SearchOverlayBg)).
		Foreground(c(t.SearchOverlayFg)).
		Padding(0, 1)
	inputStyle = lipgloss.NewStyle().
		Background(c(t.SearchInputBg)).
		Foreground(c(t.SearchInputFg)).
		Padding(0, 1)
	matchHLStyle = lipgloss.NewStyle().
		Background(c(t.SearchMatchBg)).
		Foreground(c(t.SearchMatchFg))
	activeMatchStyle = lipgloss.NewStyle().
		Background(c(t.SearchActiveMatchBg)).
		Foreground(c(t.SearchActiveMatchFg))
	labelStyle = lipgloss.NewStyle().Foreground(c(t.SearchLabelFg))

	// panes.go styles
	paneBorderDim = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c(t.BorderDim))
	paneBorderActive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c(t.BorderActive))
}
