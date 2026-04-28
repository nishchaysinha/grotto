package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// OverlayMode indicates which overlay is active.
type OverlayMode int

const (
	OverlayNone OverlayMode = iota
	OverlayFileFinder
	OverlayCommandPalette
	OverlayShortcuts
)

// Overlay handles Ctrl+P file finder, Ctrl+Shift+P command palette, and F5 shortcuts help.
type Overlay struct {
	mode     OverlayMode
	query    string
	cursor   int // cursor in query
	items    []string
	filtered []string
	selected int
	scroll   int // scroll offset for shortcuts view
}

type Command struct {
	Name   string
	Action func(m *Model)
}

var (
	overlayBoxStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#282A36")).
			Foreground(lipgloss.Color("#F8F8F2")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)
	overlayInputStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#44475A")).
				Foreground(lipgloss.Color("#F8F8F2")).
				Padding(0, 1)
	overlayItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#AAAAAA"))
	overlaySelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#44475A")).
				Foreground(lipgloss.Color("#F8F8F2"))
)

func (o *Overlay) Active() bool { return o.mode != OverlayNone }

func (o *Overlay) OpenFileFinder(rootPath string) {
	o.mode = OverlayFileFinder
	o.query = ""
	o.cursor = 0
	o.selected = 0
	o.items = walkFiles(rootPath, 500)
	o.filtered = o.items
}

func (o *Overlay) OpenCommandPalette(commands []string) {
	o.mode = OverlayCommandPalette
	o.query = ""
	o.cursor = 0
	o.selected = 0
	o.items = commands
	o.filtered = commands
}

func (o *Overlay) OpenShortcuts() {
	o.mode = OverlayShortcuts
	o.scroll = 0
}

func (o *Overlay) Close() {
	o.mode = OverlayNone
}

// Update handles input. Returns (consumed, selectedItem) where selectedItem is non-empty on Enter.
func (o *Overlay) Update(msg tea.Msg) (bool, string) {
	if o.mode == OverlayNone {
		return false, ""
	}
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false, ""
	}

	switch km.String() {
	case "esc":
		o.Close()
		return true, ""
	case "enter":
		if o.mode == OverlayShortcuts {
			o.Close()
			return true, ""
		}
		result := ""
		if len(o.filtered) > 0 && o.selected < len(o.filtered) {
			result = o.filtered[o.selected]
		}
		o.Close()
		return true, result
	case "up", "ctrl+p":
		if o.mode == OverlayShortcuts {
			if o.scroll > 0 {
				o.scroll--
			}
			return true, ""
		}
		if o.selected > 0 {
			o.selected--
		}
		return true, ""
	case "down", "ctrl+n":
		if o.mode == OverlayShortcuts {
			o.scroll++
			return true, ""
		}
		if o.selected < len(o.filtered)-1 {
			o.selected++
		}
		return true, ""
	case "backspace":
		if o.mode == OverlayShortcuts {
			return true, ""
		}
		if o.cursor > 0 {
			o.query = o.query[:o.cursor-1] + o.query[o.cursor:]
			o.cursor--
			o.filter()
		}
		return true, ""
	default:
		if o.mode == OverlayShortcuts {
			return true, ""
		}
		if k := km.Key(); k.Text != "" {
			o.query = o.query[:o.cursor] + k.Text + o.query[o.cursor:]
			o.cursor += len(k.Text)
			o.filter()
			return true, ""
		}
	}
	return true, ""
}

func (o *Overlay) filter() {
	if o.query == "" {
		o.filtered = o.items
		o.selected = 0
		return
	}
	q := strings.ToLower(o.query)
	o.filtered = nil
	for _, item := range o.items {
		if fuzzyMatch(strings.ToLower(item), q) {
			o.filtered = append(o.filtered, item)
		}
	}
	o.selected = 0
}

// fuzzyMatch checks if all chars of pattern appear in s in order.
func fuzzyMatch(s, pattern string) bool {
	pi := 0
	for i := 0; i < len(s) && pi < len(pattern); i++ {
		if s[i] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}

// shortcutSection holds a named group of key bindings for the shortcuts overlay.
type shortcutSection struct {
	title string
	rows  [][2]string
}

var shortcutSections = []shortcutSection{
	{
		title: "Global",
		rows: [][2]string{
			{"Ctrl+Q", "Quit"},
			{"Ctrl+B", "Toggle sidebar"},
			{"Ctrl+` / F3", "Toggle terminal"},
			{"Ctrl+Shift+A / F4", "Toggle AI panel"},
			{"Ctrl+P / F1", "Fuzzy file finder"},
			{"Ctrl+Shift+P / F2", "Command palette"},
			{"F5", "Show keyboard shortcuts"},
			{"Ctrl+1/2/3/4", "Focus pane 1-4"},
			{"Esc", "Focus sidebar (from editor)"},
		},
	},
	{
		title: "Editor — Navigation",
		rows: [][2]string{
			{"↑ ↓ ← →", "Move cursor"},
			{"Home / End", "Start / end of line"},
			{"Ctrl+← / Ctrl+→", "Word left / right"},
			{"PgUp / PgDn", "Page up / down"},
			{"Ctrl+G", "Go to line"},
		},
	},
	{
		title: "Editor — Selection",
		rows: [][2]string{
			{"Shift+↑↓←→", "Extend selection"},
			{"Shift+Home / End", "Select to start/end of line"},
			{"Ctrl+Shift+←/→", "Select word left/right"},
			{"Shift+PgUp / PgDn", "Select page up/down"},
			{"Ctrl+A", "Select all"},
			{"Double-click", "Select word"},
			{"Triple-click", "Select line"},
		},
	},
	{
		title: "Editor — Editing",
		rows: [][2]string{
			{"Enter", "New line (auto-indent)"},
			{"Tab", "Insert 4 spaces / indent selection"},
			{"Shift+Tab", "Dedent line/selection"},
			{"Ctrl+D", "Duplicate line"},
			{"Ctrl+Z", "Undo"},
			{"Ctrl+Y", "Redo"},
			{"Ctrl+S", "Save"},
		},
	},
	{
		title: "Editor — Clipboard",
		rows: [][2]string{
			{"Ctrl+C", "Copy selection"},
			{"Ctrl+X", "Cut selection"},
			{"Ctrl+V", "Paste"},
		},
	},
	{
		title: "Editor — Search",
		rows: [][2]string{
			{"Ctrl+F", "Find"},
			{"Ctrl+H", "Find & replace"},
			{"Enter / ↓ / Ctrl+N", "Next match"},
			{"↑ / Ctrl+P", "Previous match"},
			{"Tab", "Switch find ↔ replace field"},
			{"Enter (in replace)", "Replace one"},
			{"Ctrl+Shift+Enter", "Replace all"},
			{"Esc", "Close search"},
		},
	},
	{
		title: "Tabs",
		rows: [][2]string{
			{"Ctrl+Tab", "Next tab"},
			{"Ctrl+Shift+Tab", "Previous tab"},
			{"Ctrl+W", "Close tab"},
			{"Middle-click tab", "Close tab"},
		},
	},
	{
		title: "Split Panes",
		rows: [][2]string{
			{"Ctrl+\\", "Split right"},
			{"Ctrl+Shift+\\", "Split down"},
			{"Ctrl+Shift+W", "Close pane"},
			{"Ctrl+1/2/3/4", "Focus pane by number"},
		},
	},
}

func (o *Overlay) View(width, height int) string {
	if o.mode == OverlayNone {
		return ""
	}

	if o.mode == OverlayShortcuts {
		return o.viewShortcuts(width, height)
	}

	boxW := min(width-4, 60)
	maxItems := min(len(o.filtered), 12)

	label := "> "
	if o.mode == OverlayCommandPalette {
		label = "> "
	}
	input := overlayInputStyle.Width(boxW - 4).Render(label + o.query + "▏")

	var lines []string
	lines = append(lines, input)

	start := 0
	if o.selected >= maxItems {
		start = o.selected - maxItems + 1
	}
	end := min(start+maxItems, len(o.filtered))

	for i := start; i < end; i++ {
		s := overlayItemStyle
		if i == o.selected {
			s = overlaySelectedStyle
		}
		lines = append(lines, s.Width(boxW-4).Render(o.filtered[i]))
	}

	content := strings.Join(lines, "\n")
	box := overlayBoxStyle.Width(boxW).Render(content)

	// Center horizontally
	pad := max((width-boxW-2)/2, 0)
	return strings.Repeat(" ", pad) + box
}

// viewShortcuts renders the keyboard shortcuts reference overlay.
func (o *Overlay) viewShortcuts(width, height int) string {
	boxW := min(width-4, 66)
	innerW := boxW - 4 // account for border + padding

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9")).Bold(true)
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))

	// Build all lines upfront so we can scroll them.
	var allLines []string
	allLines = append(allLines, overlayInputStyle.Width(innerW).Render("  Keyboard Shortcuts  —  ↑↓ scroll  •  Esc close"))
	allLines = append(allLines, "")

	for _, sec := range shortcutSections {
		allLines = append(allLines, sectionStyle.Render(sec.title))
		colW := 22
		for _, row := range sec.rows {
			key := keyStyle.Render(row[0])
			desc := row[1]
			// Pad key column
			padding := max(colW-len(row[0]), 1)
			line := key + strings.Repeat(" ", padding) + desc
			allLines = append(allLines, line)
		}
		allLines = append(allLines, "")
	}

	// Determine visible window
	maxVisible := min(height-6, 20) // leave room for box chrome
	totalLines := len(allLines)
	maxScroll := max(totalLines-maxVisible, 0)
	if o.scroll > maxScroll {
		o.scroll = maxScroll
	}
	end := min(o.scroll+maxVisible, totalLines)
	visible := allLines[o.scroll:end]

	// Scroll indicator
	if totalLines > maxVisible {
		pct := 0
		if maxScroll > 0 {
			pct = (o.scroll * 100) / maxScroll
		}
		indicator := dimStyle.Render(fmt.Sprintf("── %d%% ──", pct))
		visible = append(visible, indicator)
	}

	content := strings.Join(visible, "\n")
	box := overlayBoxStyle.Width(boxW).Render(content)

	pad := max((width-boxW-2)/2, 0)
	return strings.Repeat(" ", pad) + box
}

// walkFiles collects relative file paths under root, up to limit.
func walkFiles(root string, limit int) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		// Skip hidden dirs and common noise
		if d.IsDir() && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__") {
			return filepath.SkipDir
		}
		if !d.IsDir() && !strings.HasPrefix(name, ".") {
			rel, _ := filepath.Rel(root, path)
			files = append(files, rel)
			if len(files) >= limit {
				return filepath.SkipAll
			}
		}
		return nil
	})
	sort.Strings(files)
	return files
}
