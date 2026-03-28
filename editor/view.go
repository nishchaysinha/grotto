package editor

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/owomeister/grotto/gitstatus"
)

// FileSavedMsg is emitted after a successful save, carrying the saved file path.
type FileSavedMsg struct{ Path string }

// BufferChangedMsg is emitted when the buffer content is modified.
type BufferChangedMsg struct{ Path string }

// CloseTabMsg is emitted when the last tab in a pane is closed.
type CloseTabMsg struct{}

// Tab represents a single open file tab with its own buffer and view state.
type Tab struct {
	buf         *Buffer
	hl          *Highlighter
	scrollY     int
	scrollX     int
	lineChanges map[int]gitstatus.LineChange
}

// Model is the editor view component managing multiple tabs.
type Model struct {
	tabs    []Tab
	active  int
	width   int
	height  int
	gutterW int
	focused bool
	offsetX int
	offsetY int

	dragging      bool
	lastClickTime tea.MouseClickMsg
	lastClickLine int
	clickCount    int

	search SearchOverlay
}

const tabBarH = 1

var (
	noFileStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	tabStyle       = lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("#888888"))
	tabActiveStyle = lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#44475a"))
	tabBarBg       = lipgloss.NewStyle().Background(lipgloss.Color("#21222C"))

	// Pre-computed raw ANSI prefixes for hot render path
	ansiGutter      = BuildANSIPrefix("#555555", "", false, false, false)
	ansiGutterSep   = BuildANSIPrefix("#555555", "", false, false, false)
	ansiCurLine     = BuildANSIPrefix("", "#2A2A2A", false, false, false)
	ansiCursor      = "\x1b[7m" // reverse
	ansiBracketHL   = BuildANSIPrefix("", "#44475a", true, false, false)
	ansiSelectionBg = BuildANSIPrefix("", "#44475a", false, false, false)
	ansiGitAdded    = BuildANSIPrefix("#98C379", "", false, false, false)
	ansiGitMod      = BuildANSIPrefix("#E5C07B", "", false, false, false)
	ansiGitDel      = BuildANSIPrefix("#E06C75", "", false, false, false)
)

func copyToClipboard(text string) tea.Cmd { return clipCopy(text) }

func NewModel() Model { return Model{} }

func (m *Model) activeTab() *Tab {
	if len(m.tabs) == 0 {
		return nil
	}
	return &m.tabs[m.active]
}

// OpenFile opens a file in a new tab, or switches to it if already open.
func (m *Model) OpenFile(path string) error {
	// Check if already open
	for i, t := range m.tabs {
		if t.buf.FilePath == path {
			m.active = i
			m.updateGutter()
			return nil
		}
	}
	buf, err := NewBuffer(path)
	if err != nil {
		return err
	}
	m.tabs = append(m.tabs, Tab{buf: buf, hl: NewHighlighter(path)})
	m.active = len(m.tabs) - 1
	m.updateGutter()
	return nil
}

// CloseTab closes the tab at index i. Returns true if there are tabs remaining.
func (m *Model) CloseTab(i int) bool {
	if i < 0 || i >= len(m.tabs) {
		return len(m.tabs) > 0
	}
	m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
	if m.active >= len(m.tabs) {
		m.active = max(len(m.tabs)-1, 0)
	}
	m.updateGutter()
	return len(m.tabs) > 0
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.updateGutter()
}

func (m *Model) SetFocused(f bool) { m.focused = f }
func (m Model) HasFile() bool      { return len(m.tabs) > 0 }
func (m Model) Width() int         { return m.width }
func (m Model) Height() int        { return m.height }

func (m Model) Buf() *Buffer {
	if len(m.tabs) == 0 {
		return nil
	}
	return m.tabs[m.active].buf
}

func (m *Model) SetOffset(x, y int) {
	m.offsetX = x
	m.offsetY = y
}

// editorH returns the height available for the editor content (minus tab bar).
func (m Model) editorH() int {
	if len(m.tabs) <= 1 {
		return m.height
	}
	return max(m.height-tabBarH, 1)
}

func (m Model) screenToBuffer(sx, sy int) (int, int) {
	t := m.tabs[m.active]
	rx := sx - m.offsetX
	ry := sy - m.offsetY
	// Account for tab bar
	if len(m.tabs) > 1 {
		ry -= tabBarH
	}
	gutterTotal := m.gutterW + 1
	if rx < gutterTotal || ry < 0 || ry >= m.editorH() {
		return -1, -1
	}
	line := t.scrollY + ry
	col := t.scrollX + (rx - gutterTotal)
	if line >= t.buf.LineCount() {
		line = t.buf.LineCount() - 1
	}
	if line < 0 {
		line = 0
	}
	if col > len(t.buf.Line(line)) {
		col = len(t.buf.Line(line))
	}
	if col < 0 {
		col = 0
	}
	return line, col
}

func (m *Model) updateGutter() {
	t := m.activeTab()
	if t == nil {
		m.gutterW = 4
		return
	}
	m.gutterW = len(fmt.Sprintf("%d", t.buf.LineCount())) + 1
	if m.gutterW < 3 {
		m.gutterW = 3
	}
}

// RefreshDiff updates the active tab's line-level git diff markers.
func (m *Model) RefreshDiff(gitRoot string) {
	t := m.activeTab()
	if t == nil || t.buf.FilePath == "" {
		return
	}
	t.lineChanges = gitstatus.GetLineDiff(gitRoot, t.buf.FilePath)
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	t := m.activeTab()

	// Search overlay consumes input when active
	if m.search.Active() && t != nil {
		if consumed, cmd := m.search.Update(msg, t.buf); consumed {
			m.ensureVisible()
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if t == nil {
			return m, nil
		}
		ks := msg.String()

		// Tab switching
		switch ks {
		case "ctrl+tab":
			if len(m.tabs) > 1 {
				m.active = (m.active + 1) % len(m.tabs)
				m.updateGutter()
			}
			return m, nil
		case "ctrl+shift+tab":
			if len(m.tabs) > 1 {
				m.active = (m.active - 1 + len(m.tabs)) % len(m.tabs)
				m.updateGutter()
			}
			return m, nil
		case "ctrl+w":
			// TODO: add unsaved prompt when t.buf.Dirty is true
			if !m.CloseTab(m.active) {
				return m, func() tea.Msg { return CloseTabMsg{} }
			}
			return m, nil
		}

		buf := t.buf
		hl := t.hl
		prevEditVersion := buf.EditVersion

		// Shift+arrow selection
		isShift := false
		switch ks {
		case "shift+up", "shift+down", "shift+left", "shift+right",
			"shift+home", "shift+end", "ctrl+shift+left", "ctrl+shift+right",
			"shift+pgup", "shift+pgdown":
			isShift = true
			if !buf.Selection.Active {
				buf.Selection.Anchor = buf.Cursor
				buf.Selection.Active = true
			}
		}

		if !isShift {
			switch ks {
			case "up", "down", "left", "right", "home", "end",
				"ctrl+left", "ctrl+right", "pgup", "pgdown":
				buf.Selection.Active = false
			}
		}

		// Delete selection on editing keys
		if buf.Selection.Active {
			switch ks {
			case "backspace", "delete", "enter", "tab":
				buf.DeleteSelection()
				if ks == "backspace" || ks == "delete" {
					m.ensureVisible()
					m.updateGutter()
					return m, func() tea.Msg { return BufferChangedMsg{Path: buf.FilePath} }
				}
			default:
				if msg.Key().Text != "" {
					buf.DeleteSelection()
				}
			}
		}

		eh := m.editorH()
		switch ks {
		case "up", "shift+up":
			buf.CursorUp()
		case "down", "shift+down":
			buf.CursorDown()
		case "left", "shift+left":
			buf.CursorLeft()
		case "right", "shift+right":
			buf.CursorRight()
		case "home", "shift+home":
			buf.CursorHome()
		case "end", "shift+end":
			buf.CursorEnd()
		case "ctrl+left", "ctrl+shift+left":
			buf.CursorWordLeft()
		case "ctrl+right", "ctrl+shift+right":
			buf.CursorWordRight()
		case "pgup", "shift+pgup":
			buf.PageUp(eh)
		case "pgdown", "shift+pgdown":
			buf.PageDown(eh)
		case "backspace":
			buf.Backspace()
		case "delete":
			buf.DeleteChar()
		case "enter":
			buf.NewLine()
		case "tab":
			if buf.Selection.Active {
				m.indentSelection(1)
			} else {
				for range 4 {
					buf.InsertChar(' ')
				}
			}
		case "shift+tab":
			m.indentSelection(-1)
		case "ctrl+s":
			if err := buf.Save(); err == nil {
				return m, func() tea.Msg { return FileSavedMsg{Path: buf.FilePath} }
			}
			return m, nil
		case "ctrl+z":
			buf.Undo()
		case "ctrl+y":
			buf.Redo()
		case "ctrl+a":
			buf.SelectAll()
		case "ctrl+c":
			if buf.Selection.Active {
				return m, copyToClipboard(buf.SelectedText())
			}
		case "ctrl+x":
			if buf.Selection.Active {
				text := buf.SelectedText()
				buf.DeleteSelection()
				return m, tea.Batch(copyToClipboard(text), func() tea.Msg { return BufferChangedMsg{Path: buf.FilePath} })
			}
		case "ctrl+v":
			return m, clipPaste()
		case "ctrl+d":
			line := buf.Line(buf.Cursor.Line)
			buf.Insert(Position{Line: buf.Cursor.Line, Col: len(line)}, "\n"+line)
			buf.CursorDown()
		case "ctrl+f":
			m.search.Open(SearchFind)
			return m, nil
		case "ctrl+h":
			m.search.Open(SearchReplace)
			return m, nil
		case "ctrl+g":
			m.search.Open(SearchGoToLine)
			return m, nil
		default:
			if k := msg.Key(); k.Text != "" {
				for _, r := range k.Text {
					buf.InsertChar(r)
				}
			}
		}

		if isShift {
			buf.Selection.Head = buf.Cursor
		}
		m.ensureVisible()
		m.updateGutter()
		if hl != nil {
			hl.InvalidateLine(buf.Cursor.Line)
			if buf.Cursor.Line > 0 {
				hl.InvalidateLine(buf.Cursor.Line - 1)
			}
			hl.InvalidateLine(buf.Cursor.Line + 1)
		}
		if buf.EditVersion != prevEditVersion && buf.FilePath != "" {
			return m, func() tea.Msg { return BufferChangedMsg{Path: buf.FilePath} }
		}

	case tea.ClipboardMsg:
		if t == nil {
			return m, nil
		}
		if text := msg.Content; text != "" {
			if t.buf.Selection.Active {
				t.buf.DeleteSelection()
			}
			t.buf.Insert(t.buf.Cursor, text)
			t.buf.Cursor = t.buf.advancePos(t.buf.Cursor, text)
			t.buf.clampCursor()
			m.ensureVisible()
			m.updateGutter()
			return m, func() tea.Msg { return BufferChangedMsg{Path: t.buf.FilePath} }
		}

	case tea.MouseClickMsg:
		switch msg.Button {
		case tea.MouseLeft:
			// Check if click is on tab bar
			if len(m.tabs) > 1 {
				ry := msg.Y - m.offsetY
				if ry == 0 {
					m.handleTabClick(msg.X - m.offsetX)
					return m, nil
				}
			}

			if t == nil {
				return m, nil
			}
			line, col := m.screenToBuffer(msg.X, msg.Y)
			if line >= 0 {
				sameSpot := m.lastClickLine == line
				if sameSpot && m.clickCount > 0 {
					m.clickCount++
				} else {
					m.clickCount = 1
				}
				m.lastClickLine = line
				m.lastClickTime = msg

				switch m.clickCount {
				case 2:
					s, e := WordAt(t.buf.Line(line), col)
					t.buf.Cursor = Position{Line: line, Col: e}
					t.buf.Selection = Selection{Anchor: Position{Line: line, Col: s}, Head: Position{Line: line, Col: e}, Active: s != e}
				case 3:
					t.buf.Selection = Selection{Anchor: Position{Line: line, Col: 0}, Head: Position{Line: line, Col: len(t.buf.Line(line))}, Active: true}
					t.buf.Cursor = Position{Line: line, Col: len(t.buf.Line(line))}
					m.clickCount = 0
				default:
					if msg.Mod.Contains(tea.ModShift) {
						if !t.buf.Selection.Active {
							t.buf.Selection.Anchor = t.buf.Cursor
							t.buf.Selection.Active = true
						}
						t.buf.Selection.Head = Position{Line: line, Col: col}
						t.buf.Cursor = Position{Line: line, Col: col}
					} else {
						t.buf.Cursor = Position{Line: line, Col: col}
						t.buf.Selection.Active = false
						m.dragging = true
						t.buf.Selection.Anchor = Position{Line: line, Col: col}
					}
				}
			}
		case tea.MouseMiddle:
			// Middle-click close tab
			ry := msg.Y - m.offsetY
			if ry == 0 && len(m.tabs) > 1 {
				idx := m.tabIndexAtX(msg.X - m.offsetX)
				if idx >= 0 {
					if !m.CloseTab(idx) {
						return m, func() tea.Msg { return CloseTabMsg{} }
					}
					return m, nil
				}
			}
		}

	case tea.MouseReleaseMsg:
		m.dragging = false

	case tea.MouseMotionMsg:
		if m.dragging && t != nil {
			line, col := m.screenToBuffer(msg.X, msg.Y)
			if line >= 0 {
				t.buf.Cursor = Position{Line: line, Col: col}
				t.buf.Selection.Head = Position{Line: line, Col: col}
				a := t.buf.Selection.Anchor
				if line != a.Line || col != a.Col {
					t.buf.Selection.Active = true
				}
				m.ensureVisible()
			}
		}

	case tea.MouseWheelMsg:
		if t == nil {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseWheelUp:
			t.scrollY = max(t.scrollY-3, 0)
		case tea.MouseWheelDown:
			t.scrollY += 3
			maxS := max(t.buf.LineCount()-m.editorH(), 0)
			if t.scrollY > maxS {
				t.scrollY = maxS
			}
		}
	}
	return m, nil
}

func (m *Model) handleTabClick(rx int) {
	x := 0
	for i, t := range m.tabs {
		name := filepath.Base(t.buf.FilePath)
		if t.buf.Dirty {
			name += " ●"
		}
		w := len(name) + 2 // padding
		if rx >= x && rx < x+w {
			m.active = i
			m.updateGutter()
			return
		}
		x += w
	}
}

func (m Model) tabIndexAtX(rx int) int {
	x := 0
	for i, t := range m.tabs {
		name := filepath.Base(t.buf.FilePath)
		if t.buf.Dirty {
			name += " ●"
		}
		w := len(name) + 2
		if rx >= x && rx < x+w {
			return i
		}
		x += w
	}
	return -1
}

func (m *Model) indentSelection(dir int) {
	t := m.activeTab()
	if t == nil {
		return
	}
	startL, endL := t.buf.Cursor.Line, t.buf.Cursor.Line
	if t.buf.Selection.Active {
		s, e := t.buf.selectionRange()
		startL, endL = s.Line, e.Line
	}
	for l := startL; l <= endL; l++ {
		line := t.buf.Lines[l]
		if dir > 0 {
			t.buf.Lines[l] = "    " + line
		} else {
			trimmed := strings.TrimPrefix(line, "    ")
			if trimmed == line {
				trimmed = strings.TrimPrefix(line, "\t")
			}
			t.buf.Lines[l] = trimmed
		}
		if t.hl != nil {
			t.hl.InvalidateLine(l)
		}
	}
	t.buf.Dirty = true
}

func (m *Model) ensureVisible() {
	t := m.activeTab()
	if t == nil {
		return
	}
	eh := m.editorH()
	cur := t.buf.Cursor.Line
	if cur < t.scrollY {
		t.scrollY = cur
	}
	if cur >= t.scrollY+eh {
		t.scrollY = cur - eh + 1
	}
}

func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	t := m.activeTab()
	if t == nil {
		var out strings.Builder
		for i := range m.height {
			if i > 0 {
				out.WriteByte('\n')
			}
			if i == m.height/2 {
				msg := "No file open"
				lpad := max((m.width-len(msg))/2, 0)
				rpad := max(m.width-lpad-len(msg), 0)
				out.WriteString(strings.Repeat(" ", lpad) + noFileStyle.Render(msg) + strings.Repeat(" ", rpad))
			} else {
				out.WriteString(strings.Repeat(" ", m.width))
			}
		}
		return out.String()
	}

	var out strings.Builder

	// Tab bar (only if multiple tabs)
	if len(m.tabs) > 1 {
		var bar strings.Builder
		for i, tab := range m.tabs {
			name := filepath.Base(tab.buf.FilePath)
			if tab.buf.Dirty {
				name += " ●"
			}
			s := tabStyle
			if i == m.active {
				s = tabActiveStyle
			}
			bar.WriteString(zone.Mark(fmt.Sprintf("tab-%d", i), s.Render(name)))
		}
		// Pad tab bar to full width
		rendered := bar.String()
		pad := max(m.width-lipgloss.Width(rendered), 0)
		out.WriteString(tabBarBg.Render(rendered + strings.Repeat(" ", pad)))
		out.WriteString("\n")
	}

	buf := t.buf
	hl := t.hl
	eh := m.editorH()
	contentW := max(m.width-m.gutterW-1, 1)

	// Bracket match
	matchLine, matchCol := -1, -1
	if m.focused {
		matchLine, matchCol = buf.MatchBracket()
	}

	// Search matches — build per-line lookup
	searchMatches := m.search.Matches()
	searchIdx := m.search.MatchIdx()
	matchesByLine := map[int][]int{} // line → index into searchMatches
	for i, sm := range searchMatches {
		matchesByLine[sm.Line] = append(matchesByLine[sm.Line], i)
	}

	for i := range eh {
		lineNum := t.scrollY + i
		if lineNum >= buf.LineCount() {
			out.WriteString(ansiGutter + strings.Repeat(" ", m.gutterW) + ansiReset)
			out.WriteString(ansiGutterSep + "│" + ansiReset)
			out.WriteString(strings.Repeat(" ", contentW))
			if i < eh-1 {
				out.WriteByte('\n')
			}
			continue
		}

		numStr := fmt.Sprintf("%*d ", m.gutterW-1, lineNum+1)
		out.WriteString(ansiGutter + numStr + ansiReset)

		// Git gutter marker
		sep := ansiGutterSep + "│" + ansiReset
		if t.lineChanges != nil {
			switch t.lineChanges[lineNum] {
			case gitstatus.LineAdded:
				sep = ansiGitAdded + "▌" + ansiReset
			case gitstatus.LineModified:
				sep = ansiGitMod + "▌" + ansiReset
			case gitstatus.LineDeleted:
				sep = ansiGitDel + "▾" + ansiReset
			}
		}
		out.WriteString(sep)

		rawLine := buf.Line(lineNum)
		isCurLine := lineNum == buf.Cursor.Line

		// Selection range for this line
		selStart, selEnd := -1, -1
		if buf.Selection.Active {
			s, e := buf.Selection.Anchor, buf.Selection.Head
			if s.Line > e.Line || (s.Line == e.Line && s.Col > e.Col) {
				s, e = e, s
			}
			if lineNum >= s.Line && lineNum <= e.Line {
				if lineNum == s.Line {
					selStart = s.Col
				} else {
					selStart = 0
				}
				if lineNum == e.Line {
					selEnd = e.Col
				} else {
					selEnd = len(rawLine)
				}
			}
		}

		// Get syntax spans
		var spans []StyledSpan
		if hl != nil {
			spans = hl.Highlight(lineNum, rawLine)
		}

		// Build per-byte ANSI prefix lookup
		charANSIs := make([]string, len(rawLine))
		idx := 0
		for _, sp := range spans {
			for range len(sp.Text) {
				if idx < len(charANSIs) {
					charANSIs[idx] = sp.ANSI
				}
				idx++
			}
		}

		// Determine if this line has any overlays
		hasOverlays := (selStart >= 0) ||
			len(matchesByLine[lineNum]) > 0 ||
			(lineNum == matchLine) ||
			(m.focused && isCurLine)

		vis := 0
		if !hasOverlays {
			// Fast path: render entire spans using raw ANSI
			bytePos := 0
			for _, sp := range spans {
				if vis >= contentW {
					break
				}
				var chunk strings.Builder
				for _, r := range sp.Text {
					rLen := utf8.RuneLen(r)
					if bytePos < t.scrollX {
						bytePos += rLen
						continue
					}
					if vis >= contentW {
						break
					}
					chunk.WriteRune(r)
					vis++
					bytePos += rLen
				}
				if chunk.Len() > 0 {
					out.WriteString(FastRender(chunk.String(), sp.ANSI))
				}
			}
		} else {
			// Overlay path: per-char but using raw ANSI
			ci := t.scrollX
			for ci < len(rawLine) && vis < contentW {
				r, size := utf8.DecodeRuneInString(rawLine[ci:])

				// Determine ANSI prefix
				baseANSI := ""
				if ci < len(charANSIs) {
					baseANSI = charANSIs[ci]
				}

				// Check overlays
				isSel := selStart >= 0 && ci >= selStart && ci < selEnd
				isBracket := lineNum == matchLine && ci == matchCol
				isCursor := m.focused && isCurLine && ci == buf.Cursor.Col
				searchHit := -1
				for _, mi := range matchesByLine[lineNum] {
					sm := searchMatches[mi]
					if ci >= sm.Col && ci < sm.Col+sm.Len {
						searchHit = mi
					}
				}

				ch := string(r)
				if isCursor {
					out.WriteString(ansiCursor + ch + ansiReset)
				} else if isBracket {
					out.WriteString(ansiBracketHL + ch + ansiReset)
				} else if searchHit >= 0 {
					if searchHit == searchIdx {
						out.WriteString(activeMatchStyle.Render(ch))
					} else {
						out.WriteString(matchHLStyle.Render(ch))
					}
				} else if isSel {
					out.WriteString(ansiSelectionBg + ch + ansiReset)
				} else {
					out.WriteString(FastRender(ch, baseANSI))
				}
				vis++
				ci += size
			}
		}

		if m.focused && isCurLine && buf.Cursor.Col >= len(rawLine) {
			out.WriteString(ansiCursor + " " + ansiReset)
			vis++
		}

		pad := max(contentW-vis, 0)
		if pad > 0 {
			if isCurLine {
				out.WriteString(ansiCurLine + strings.Repeat(" ", pad) + ansiReset)
			} else {
				out.WriteString(strings.Repeat(" ", pad))
			}
		}

		if i < eh-1 {
			out.WriteString("\n")
		}
	}
	result := out.String()

	// Overlay search bar at top if active
	if m.search.Active() {
		bar := m.search.View(m.width)
		lines := strings.SplitN(result, "\n", 2)
		if len(lines) > 1 {
			result = bar + "\n" + lines[1]
		} else {
			result = bar
		}
	}

	return result
}
