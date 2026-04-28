package editor

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/owomeister/grotto/gitstatus"
)

// SplitDir is the direction of a split.
type SplitDir int

const (
	SplitRight SplitDir = iota
	SplitDown
)

// Layout describes how panes are arranged.
// With 1 pane: single. With 2: either side-by-side or stacked.
// With 3: left + right column split vertically. With 4: 2x2 grid.
type Layout int

const (
	LayoutSingle     Layout = iota
	LayoutColumns           // 2 side by side
	LayoutRows              // 2 stacked
	LayoutLeftRight2        // 1 left, 2 right stacked
	LayoutGrid              // 2x2
)

var (
	paneBorderDim    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#555555"))
	paneBorderActive = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7D56F4"))
)

// PaneManager manages 1-4 editor panes.
type PaneManager struct {
	panes   []Model
	active  int
	layout  Layout
	width   int
	height  int
	focused bool // whether the editor area has focus (vs sidebar/terminal/AI)

	// Screen offset from parent
	offsetX int
	offsetY int

	// Shared buffers: path → *Buffer for cross-pane sharing
	buffers map[string]*Buffer
}

func NewPaneManager() PaneManager {
	return PaneManager{
		panes:   []Model{NewModel()},
		layout:  LayoutSingle,
		buffers: make(map[string]*Buffer),
	}
}

func (pm *PaneManager) ActivePane() *Model {
	return &pm.panes[pm.active]
}

func (pm PaneManager) HasFile() bool {
	return pm.panes[pm.active].HasFile()
}

func (pm PaneManager) Buf() *Buffer {
	return pm.panes[pm.active].Buf()
}

func (pm PaneManager) Width() int  { return pm.width }
func (pm PaneManager) Height() int { return pm.height }

func (pm *PaneManager) SetFocused(f bool) {
	pm.focused = f
	for i := range pm.panes {
		pm.panes[i].SetFocused(f && i == pm.active)
	}
}

func (pm *PaneManager) SetOffset(x, y int) {
	pm.offsetX = x
	pm.offsetY = y
	pm.recalcPanes()
}

func (pm *PaneManager) SetSize(w, h int) {
	pm.width = w
	pm.height = h
	pm.recalcPanes()
}

// OpenFile opens a file in the active pane, using shared buffer if already open elsewhere.
func (pm *PaneManager) OpenFile(path string) error {
	// Check if buffer already exists in shared pool
	if buf, ok := pm.buffers[path]; ok {
		p := pm.ActivePane()
		// Check if already open in this pane's tabs
		for i, t := range p.tabs {
			if t.buf.FilePath == path {
				p.active = i
				p.updateGutter()
				return nil
			}
		}
		// Add as new tab sharing the buffer
		p.tabs = append(p.tabs, Tab{buf: buf, hl: NewHighlighter(path)})
		p.active = len(p.tabs) - 1
		p.updateGutter()
		return nil
	}
	// New buffer
	err := pm.ActivePane().OpenFile(path)
	if err == nil {
		pm.buffers[path] = pm.ActivePane().Buf()
	}
	return err
}

// Split creates a new pane in the given direction. Max 4 panes.
func (pm *PaneManager) Split(dir SplitDir) {
	if len(pm.panes) >= 4 {
		return
	}
	newPane := NewModel()
	// Copy current file to new pane (shared buffer)
	if buf := pm.Buf(); buf != nil {
		newPane.tabs = append(newPane.tabs, Tab{buf: buf, hl: NewHighlighter(buf.FilePath)})
		newPane.active = 0
		newPane.updateGutter()
	}
	pm.panes = append(pm.panes, newPane)
	pm.active = len(pm.panes) - 1
	pm.updateLayout(dir)
	pm.recalcPanes()
	pm.SetFocused(pm.focused)
}

func (pm *PaneManager) updateLayout(lastDir SplitDir) {
	switch len(pm.panes) {
	case 1:
		pm.layout = LayoutSingle
	case 2:
		if lastDir == SplitDown {
			pm.layout = LayoutRows
		} else {
			pm.layout = LayoutColumns
		}
	case 3:
		pm.layout = LayoutLeftRight2
	case 4:
		pm.layout = LayoutGrid
	}
}

// ClosePane closes the active pane. If it's the last one, does nothing.
func (pm *PaneManager) ClosePane() {
	if len(pm.panes) <= 1 {
		return
	}
	pm.panes = append(pm.panes[:pm.active], pm.panes[pm.active+1:]...)
	if pm.active >= len(pm.panes) {
		pm.active = len(pm.panes) - 1
	}
	// Recalc layout
	switch len(pm.panes) {
	case 1:
		pm.layout = LayoutSingle
	case 2:
		switch pm.layout {
		case LayoutGrid, LayoutLeftRight2:
			pm.layout = LayoutColumns
		}
	case 3:
		pm.layout = LayoutLeftRight2
	}
	pm.recalcPanes()
	pm.SetFocused(pm.focused)
}

// FocusPane sets focus to pane at index i (0-based).
func (pm *PaneManager) FocusPane(i int) {
	if i >= 0 && i < len(pm.panes) {
		pm.active = i
		pm.SetFocused(pm.focused)
	}
}

// paneRects returns (x, y, w, h) for each pane within the available space.
// These are content dimensions (inside borders).
func (pm *PaneManager) paneRects() []rect {
	n := len(pm.panes)
	w, h := pm.width, pm.height
	if n == 1 {
		return []rect{{0, 0, w, h}}
	}

	switch pm.layout {
	case LayoutColumns: // 2 side by side
		lw := w / 2
		rw := w - lw
		return []rect{{0, 0, lw, h}, {lw, 0, rw, h}}

	case LayoutRows: // 2 stacked
		th := h / 2
		bh := h - th
		return []rect{{0, 0, w, th}, {0, th, w, bh}}

	case LayoutLeftRight2: // 1 left, 2 right stacked
		lw := w / 2
		rw := w - lw
		rth := h / 2
		rbh := h - rth
		return []rect{
			{0, 0, lw, h},
			{lw, 0, rw, rth},
			{lw, rth, rw, rbh},
		}

	case LayoutGrid: // 2x2
		lw := w / 2
		rw := w - lw
		th := h / 2
		bh := h - th
		return []rect{
			{0, 0, lw, th},
			{lw, 0, rw, th},
			{0, th, lw, bh},
			{lw, th, rw, bh},
		}
	}
	return []rect{{0, 0, w, h}}
}

type rect struct{ x, y, w, h int }

func (pm *PaneManager) recalcPanes() {
	rects := pm.paneRects()
	for i := range pm.panes {
		if i >= len(rects) {
			break
		}
		r := rects[i]
		if len(pm.panes) == 1 {
			// Single pane: parent draws border, offset already accounts for it
			pm.panes[i].SetSize(pm.width, pm.height)
			pm.panes[i].SetOffset(pm.offsetX, pm.offsetY)
		} else {
			// Multi-pane: we draw borders, add +1 for border top/left
			pw := max(r.w-2, 1)
			ph := max(r.h-2, 1)
			pm.panes[i].SetSize(pw, ph)
			pm.panes[i].SetOffset(pm.offsetX+r.x+1, pm.offsetY+r.y+1)
		}
	}
}

// FocusPaneAtScreen determines which pane was clicked based on screen coords.
func (pm *PaneManager) FocusPaneAtScreen(sx, sy int) {
	rects := pm.paneRects()
	rx := sx - pm.offsetX
	ry := sy - pm.offsetY
	for i, r := range rects {
		if rx >= r.x && rx < r.x+r.w && ry >= r.y && ry < r.y+r.h {
			if i != pm.active {
				pm.active = i
				pm.SetFocused(pm.focused)
			}
			return
		}
	}
}

func (pm PaneManager) Init() tea.Cmd { return nil }

func (pm PaneManager) Update(msg tea.Msg) (PaneManager, tea.Cmd) {
	// Route mouse clicks to correct pane
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		pm.FocusPaneAtScreen(msg.X, msg.Y)
	case tea.MouseMotionMsg:
		// Keep routing to active pane (for drag)
	case tea.MouseWheelMsg:
		pm.FocusPaneAtScreen(msg.X, msg.Y)
	}

	// Update active pane
	var cmd tea.Cmd
	pm.panes[pm.active], cmd = pm.panes[pm.active].Update(msg)
	return pm, cmd
}

func (pm PaneManager) View() string {
	if len(pm.panes) == 1 {
		// Single pane — no extra borders, parent handles it
		return pm.panes[0].View()
	}

	// Multi-pane: render each in a bordered box and compose
	rects := pm.paneRects()
	// Build a canvas approach: render each pane, place in grid
	// Use lipgloss join based on layout

	switch pm.layout {
	case LayoutColumns:
		l := pm.renderPane(0, rects[0])
		r := pm.renderPane(1, rects[1])
		return lipgloss.JoinHorizontal(lipgloss.Top, l, r)

	case LayoutRows:
		t := pm.renderPane(0, rects[0])
		b := pm.renderPane(1, rects[1])
		return lipgloss.JoinVertical(lipgloss.Left, t, b)

	case LayoutLeftRight2:
		l := pm.renderPane(0, rects[0])
		rt := pm.renderPane(1, rects[1])
		rb := pm.renderPane(2, rects[2])
		right := lipgloss.JoinVertical(lipgloss.Left, rt, rb)
		return lipgloss.JoinHorizontal(lipgloss.Top, l, right)

	case LayoutGrid:
		tl := pm.renderPane(0, rects[0])
		tr := pm.renderPane(1, rects[1])
		bl := pm.renderPane(2, rects[2])
		br := pm.renderPane(3, rects[3])
		top := lipgloss.JoinHorizontal(lipgloss.Top, tl, tr)
		bot := lipgloss.JoinHorizontal(lipgloss.Top, bl, br)
		return lipgloss.JoinVertical(lipgloss.Left, top, bot)
	}

	return pm.panes[0].View()
}

func (pm PaneManager) renderPane(i int, r rect) string {
	bdr := paneBorderDim
	if pm.focused && i == pm.active {
		bdr = paneBorderActive
	}
	content := pm.panes[i].View()
	pw := max(r.w-2, 1)
	ph := max(r.h-2, 1)
	return bdr.Width(pw).Height(ph).Render(content)
}

// PaneCount returns the number of panes.
func (pm PaneManager) PaneCount() int { return len(pm.panes) }

// RefreshAllDiffs updates git line-change markers for the active tab in every pane.
func (pm *PaneManager) RefreshAllDiffs(gitRoot string) {
	for i := range pm.panes {
		pm.panes[i].RefreshDiff(gitRoot)
	}
}

// UpdateLineDiff applies pre-computed diff results to every tab showing the given path.
func (pm *PaneManager) UpdateLineDiff(path string, changes map[int]gitstatus.LineChange) {
	for i := range pm.panes {
		for j := range pm.panes[i].tabs {
			if pm.panes[i].tabs[j].buf.FilePath == path {
				pm.panes[i].tabs[j].lineChanges = changes
			}
		}
	}
}

// HasSearchActive returns true if the active pane has a search overlay open.
func (pm PaneManager) HasSearchActive() bool {
	return pm.panes[pm.active].search.Active()
}

// DirtyBuffers returns the unique set of buffers with unsaved changes across
// all panes (buffers may be shared between panes — they're de-duped here).
func (pm PaneManager) DirtyBuffers() []*Buffer {
	seen := make(map[*Buffer]struct{})
	var out []*Buffer
	for i := range pm.panes {
		for _, t := range pm.panes[i].tabs {
			if t.buf == nil || !t.buf.Dirty {
				continue
			}
			if _, ok := seen[t.buf]; ok {
				continue
			}
			seen[t.buf] = struct{}{}
			out = append(out, t.buf)
		}
	}
	return out
}

// SaveAllDirty persists every dirty buffer. Returns the first error encountered,
// or nil if all saves succeeded.
func (pm PaneManager) SaveAllDirty() error {
	for _, b := range pm.DirtyBuffers() {
		if err := b.Save(); err != nil {
			return err
		}
	}
	return nil
}

// TabInfo returns a summary string for the status bar.
func (pm PaneManager) TabInfo() string {
	if len(pm.panes) <= 1 {
		return ""
	}
	return strings.Repeat("▪", len(pm.panes))
}
