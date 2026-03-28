package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/owomeister/grotto/config"
)

// ApplyTheme reinitializes all package-level style variables from t.
// Must be called before creating any sidebar components.
func ApplyTheme(t config.Theme) {
	c := lipgloss.Color

	gitAddedStyle = lipgloss.NewStyle().Foreground(c(t.GitAdded))
	gitModifiedStyle = lipgloss.NewStyle().Foreground(c(t.GitModified))
	gitDeletedStyle = lipgloss.NewStyle().Foreground(c(t.GitDeleted))
	gitRenamedStyle = lipgloss.NewStyle().Foreground(c(t.GitRenamed))
	gitUntrackedStyle = lipgloss.NewStyle().Foreground(c(t.GitUntracked))
	selectedStyle = lipgloss.NewStyle().
		Background(c(t.SidebarSelectedBg)).
		Foreground(c(t.SidebarSelectedFg))
	if t.SidebarDirIcon != "" {
		dirIconStyle = lipgloss.NewStyle().Foreground(c(t.SidebarDirIcon))
	}
}
