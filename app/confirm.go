package app

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ConfirmAction is the user's choice on a confirm dialog.
type ConfirmAction int

const (
	ConfirmPending ConfirmAction = iota
	ConfirmSave
	ConfirmDiscard
	ConfirmCancel
)

// ConfirmReason identifies which caller raised the dialog so the app can
// route the post-confirmation action correctly.
type ConfirmReason int

const (
	ConfirmReasonNone ConfirmReason = iota
	ConfirmReasonQuit
	ConfirmReasonCloseTab
)

// ConfirmDialog is a modal asking the user what to do about unsaved files
// before a destructive action (quit, close tab).
type ConfirmDialog struct {
	active bool
	reason ConfirmReason
	files  []string // display names of dirty files
}

func (d *ConfirmDialog) Active() bool          { return d.active }
func (d *ConfirmDialog) Reason() ConfirmReason { return d.reason }

// Open shows the dialog with the list of dirty file paths.
func (d *ConfirmDialog) Open(reason ConfirmReason, dirtyPaths []string) {
	d.active = true
	d.reason = reason
	d.files = make([]string, len(dirtyPaths))
	for i, p := range dirtyPaths {
		if p == "" {
			d.files[i] = "(untitled)"
		} else {
			d.files[i] = filepath.Base(p)
		}
	}
}

func (d *ConfirmDialog) Close() {
	d.active = false
	d.reason = ConfirmReasonNone
	d.files = nil
}

// Update processes a key press. Returns (consumed, action). `action` is
// ConfirmPending while the dialog is waiting for input.
func (d *ConfirmDialog) Update(msg tea.Msg) (bool, ConfirmAction) {
	if !d.active {
		return false, ConfirmPending
	}
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false, ConfirmPending
	}
	switch km.String() {
	case "s", "S":
		return true, ConfirmSave
	case "d", "D":
		return true, ConfirmDiscard
	case "c", "C", "esc":
		return true, ConfirmCancel
	case "enter":
		// Enter defaults to Save — the least destructive option.
		return true, ConfirmSave
	}
	return true, ConfirmPending // consume other keys while modal is up
}

// View renders the dialog centered horizontally at the top of the content area.
func (d *ConfirmDialog) View(width, height int) string {
	if !d.active {
		return ""
	}

	title := "Unsaved changes"
	switch d.reason {
	case ConfirmReasonQuit:
		title = "Unsaved changes — quit?"
	case ConfirmReasonCloseTab:
		title = "Unsaved changes — close tab?"
	}

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true)
	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))

	var lines []string
	lines = append(lines, titleStyle.Render(title))
	lines = append(lines, "")
	for _, f := range d.files {
		lines = append(lines, "  • "+fileStyle.Render(f))
	}
	lines = append(lines, "")
	lines = append(lines,
		keyStyle.Render("[S]")+dimStyle.Render("ave  ")+
			keyStyle.Render("[D]")+dimStyle.Render("iscard  ")+
			keyStyle.Render("[C]")+dimStyle.Render("ancel / Esc"))

	content := strings.Join(lines, "\n")

	boxW := min(width-4, 50)
	box := overlayBoxStyle.Width(boxW).Render(content)

	pad := max((width-boxW-2)/2, 0)
	return strings.Repeat(" ", pad) + box
}
