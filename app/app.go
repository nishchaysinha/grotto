package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fsnotify/fsnotify"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/owomeister/grotto/config"
	"github.com/owomeister/grotto/editor"
	"github.com/owomeister/grotto/gitstatus"
	"github.com/owomeister/grotto/terminal"
	"github.com/owomeister/grotto/ui"
)

// diffDebounceMsg fires after the debounce delay for a pending diff request.
type diffDebounceMsg struct {
	seq  int
	path string
}

// diffResultMsg carries async diff results back to the update loop.
type diffResultMsg struct {
	path    string
	changes map[int]gitstatus.LineChange
}

// gitExternalChangeMsg fires when .git/HEAD or .git/index changes externally.
type gitExternalChangeMsg struct{}

type Config struct {
	Path       string
	Line       int
	AIProvider string
	NoAI       bool
}

type FocusedPanel int

const (
	FocusSidebar FocusedPanel = iota
	FocusEditor
	FocusTerminal
	FocusAI
)

// CloseTabMsg is forwarded from editor when last tab closes.
type CloseTabMsg = editor.CloseTabMsg

type Model struct {
	cfg        Config
	gitRoot    string
	diffSeq    int
	gitWatchCh chan struct{}
	width      int
	height     int
	focus      FocusedPanel

	sidebarVisible  bool
	terminalVisible bool
	aiPanelVisible  bool

	// Dynamic panel sizes (resizable by dragging)
	sidebarW      int
	aiPanelW      int
	terminalRatio float64

	// Drag state
	dragging  DragTarget
	hoverEdge DragTarget

	sidebar  ui.Model
	panes    editor.PaneManager
	terminal terminal.Model
	aiPanel  terminal.Model
	overlay  Overlay

	// View cache — skip re-rendering unchanged panels
	cachedSidebar  string
	cachedTerminal string
	cachedAI       string
	dirtyPanels    uint8 // bitmask: 1=sidebar 2=terminal 4=ai 8=all
}

type DragTarget int

const (
	DragNone DragTarget = iota
	DragSidebarEdge
	DragAIEdge
	DragTerminalEdge
)

const (
	defaultSidebarW      = 22
	defaultAIPanelW      = 35
	chromeH              = 2
	defaultTerminalRatio = 0.3
	minPanelW            = 20
	minTermH             = 4

	dirtySidebar  uint8 = 1
	dirtyTerminal uint8 = 2
	dirtyAI       uint8 = 4
	dirtyAll      uint8 = 0xFF
)

var (
	titleStyle     = lipgloss.NewStyle()
	btnStyle       = lipgloss.NewStyle()
	btnActiveStyle = lipgloss.NewStyle()
	statusStyle    = lipgloss.NewStyle()
	borderDim      = lipgloss.NewStyle()
	borderHover    = lipgloss.NewStyle()
	borderActive   = lipgloss.NewStyle()
)

func applyTheme(t config.Theme) {
	c := lipgloss.Color
	titleStyle = lipgloss.NewStyle().
		Background(c(t.TitleBg)).Foreground(c(t.TitleFg)).Bold(true).Padding(0, 1)
	btnStyle = lipgloss.NewStyle().
		Background(c(t.BtnBg)).Foreground(c(t.BtnFg)).Padding(0, 1)
	btnActiveStyle = lipgloss.NewStyle().
		Background(c(t.BtnActiveBg)).Foreground(c(t.BtnActiveFg)).Bold(true).Padding(0, 1)
	statusStyle = lipgloss.NewStyle().
		Background(c(t.StatusBg)).Foreground(c(t.StatusFg)).Padding(0, 1)
	borderDim = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(c(t.BorderDim))
	borderHover = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(c(t.BorderHover))
	borderActive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(c(t.BorderActive))
}

func New(cfg Config) Model {
	zone.NewGlobal()
	absPath, _ := filepath.Abs(cfg.Path)
	cfg.Path = absPath

	// Load theme and apply to all packages before constructing any components.
	grottoCfg := config.Load()
	applyTheme(grottoCfg.Theme)
	editor.ApplyTheme(grottoCfg.Theme)
	ui.ApplyTheme(grottoCfg.Theme)

	m := Model{
		cfg:            cfg,
		gitRoot:        gitstatus.FindRoot(absPath),
		focus:          FocusSidebar,
		sidebarVisible: true,
		aiPanelVisible: !cfg.NoAI,
		sidebarW:       defaultSidebarW,
		aiPanelW:       defaultAIPanelW,
		terminalRatio:  defaultTerminalRatio,
		sidebar:        ui.New(cfg.Path),
		panes:          editor.NewPaneManager(),
		terminal:       terminal.New(),
		aiPanel:        terminal.NewAI(),
	}
	m.updateFocus()
	if m.gitRoot != "" {
		m.sidebar.Refresh(m.gitRoot)
		m.gitWatchCh = startGitWatcher(m.gitRoot)
	}
	return m
}

// startGitWatcher watches .git/HEAD and .git/index and sends to the returned
// channel whenever either file changes.
func startGitWatcher(gitRoot string) chan struct{} {
	ch := make(chan struct{}, 1)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return ch
	}
	_ = watcher.Add(filepath.Join(gitRoot, ".git", "HEAD"))
	_ = watcher.Add(filepath.Join(gitRoot, ".git", "index"))
	go func() {
		defer func() { _ = watcher.Close() }()
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Drain burst of events before notifying
				for len(watcher.Events) > 0 {
					<-watcher.Events
				}
				select {
				case ch <- struct{}{}:
				default:
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return ch
}

func (m Model) Init() tea.Cmd {
	if m.gitWatchCh != nil {
		return m.waitForGitChange()
	}
	return nil
}

// waitForGitChange blocks until the git watcher signals a change.
func (m Model) waitForGitChange() tea.Cmd {
	ch := m.gitWatchCh
	return func() tea.Msg {
		<-ch
		return gitExternalChangeMsg{}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()
		return m, nil

	case CloseTabMsg:
		// Last tab closed in active pane — auto-close pane
		m.panes.ClosePane()
		m.recalcLayout()
		return m, nil

	case sendAIPayloadMsg:
		// If the AI panel was just opened, the PTY may not be ready yet.
		// Poll a few times before giving up rather than dropping the payload.
		if m.aiPanel.TabCount() == 0 {
			if msg.attempts >= 20 {
				return m, nil // give up after ~1s
			}
			next := msg
			next.attempts++
			return m, func() tea.Msg {
				time.Sleep(50 * time.Millisecond)
				return next
			}
		}
		_, _ = m.aiPanel.SendText(msg.payload)
		return m, nil

	case editor.FileSavedMsg:
		if m.gitRoot != "" {
			m.sidebar.Refresh(m.gitRoot)
			m.panes.RefreshAllDiffs(m.gitRoot)
		}
		return m, nil

	case editor.BufferChangedMsg:
		if m.gitRoot == "" {
			return m, nil
		}
		m.diffSeq++
		seq := m.diffSeq
		path := msg.Path
		return m, func() tea.Msg {
			time.Sleep(150 * time.Millisecond)
			return diffDebounceMsg{seq: seq, path: path}
		}

	case diffDebounceMsg:
		if msg.seq != m.diffSeq || m.gitRoot == "" {
			return m, nil
		}
		path := msg.path
		gitRoot := m.gitRoot
		return m, func() tea.Msg {
			return diffResultMsg{path: path, changes: gitstatus.GetLineDiff(gitRoot, path)}
		}

	case diffResultMsg:
		m.panes.UpdateLineDiff(msg.path, msg.changes)
		return m, nil

	case gitExternalChangeMsg:
		if m.gitRoot != "" {
			m.sidebar.Refresh(m.gitRoot)
			m.panes.RefreshAllDiffs(m.gitRoot)
		}
		return m, m.waitForGitChange()

	case tea.KeyPressMsg:
		// Overlay intercepts all input when active
		if m.overlay.Active() {
			mode := m.overlay.mode
			consumed, result := m.overlay.Update(msg)
			if consumed {
				if result != "" {
					switch mode {
					case OverlayFileFinder:
						_ = m.panes.OpenFile(filepath.Join(m.cfg.Path, result))
						if m.gitRoot != "" {
							m.panes.RefreshAllDiffs(m.gitRoot)
						}
						m.focus = FocusEditor
						m.updateFocus()
					case OverlayCommandPalette:
						cmd := m.execCommand(result)
						if cmd != nil {
							return m, cmd
						}
					}
				}
				return m, nil
			}
		}

		// If editor has active search, let it handle keys first
		if m.focus == FocusEditor && m.panes.HasSearchActive() {
			var cmd tea.Cmd
			m.panes, cmd = m.panes.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+q":
			return m, tea.Quit
		case "f5":
			m.overlay.OpenShortcuts()
			return m, nil
		case "ctrl+p", "f1":
			if m.focus != FocusTerminal && m.focus != FocusAI {
				m.overlay.OpenFileFinder(m.cfg.Path)
				return m, nil
			}
		case "ctrl+shift+p", "f2":
			if m.focus != FocusTerminal && m.focus != FocusAI {
				m.overlay.OpenCommandPalette(m.commandNames())
				return m, nil
			}
		case "ctrl+b":
			m.sidebarVisible = !m.sidebarVisible
			if !m.sidebarVisible && m.focus == FocusSidebar {
				m.focus = FocusEditor
			}
			m.recalcLayout()
			m.updateFocus()
			return m, nil
		case "ctrl+`", "f3":
			return m, m.toggleTerminal()
		case "ctrl+shift+a", "f4":
			return m, m.toggleAI()
		case "ctrl+shift+l":
			return m, m.sendSelectionToAI()
		// Split panes
		case "ctrl+\\":
			m.panes.Split(editor.SplitRight)
			m.recalcLayout()
			return m, nil
		case "ctrl+shift+\\":
			m.panes.Split(editor.SplitDown)
			m.recalcLayout()
			return m, nil
		case "ctrl+shift+w":
			m.panes.ClosePane()
			m.recalcLayout()
			return m, nil
		// Focus switching
		case "ctrl+1":
			if m.panes.PaneCount() > 1 {
				m.focus = FocusEditor
				m.panes.FocusPane(0)
				m.updateFocus()
			} else if m.sidebarVisible {
				m.focus = FocusSidebar
				m.updateFocus()
			}
			return m, nil
		case "ctrl+2":
			if m.panes.PaneCount() > 1 {
				m.focus = FocusEditor
				m.panes.FocusPane(1)
				m.updateFocus()
			} else {
				m.focus = FocusEditor
				m.updateFocus()
			}
			return m, nil
		case "ctrl+3":
			if m.panes.PaneCount() > 2 {
				m.focus = FocusEditor
				m.panes.FocusPane(2)
				m.updateFocus()
			} else if m.terminalVisible {
				m.focus = FocusTerminal
				m.updateFocus()
			}
			return m, nil
		case "ctrl+4":
			if m.panes.PaneCount() > 3 {
				m.focus = FocusEditor
				m.panes.FocusPane(3)
				m.updateFocus()
			} else if m.aiPanelVisible {
				m.focus = FocusAI
				m.updateFocus()
			}
			return m, nil
		case "esc":
			if m.focus == FocusEditor && m.sidebarVisible {
				m.focus = FocusSidebar
				m.updateFocus()
				return m, nil
			}
		}

	case tea.MouseClickMsg:
		// Right-click starts resize drag
		if msg.Button == tea.MouseRight {
			m.startDrag(msg.X, msg.Y)
			return m, nil
		}
		// Left-click: title bar buttons
		if msg.Y == 0 {
			if zone.Get("btn-files").InBounds(msg) {
				m.sidebarVisible = !m.sidebarVisible
				if !m.sidebarVisible && m.focus == FocusSidebar {
					m.focus = FocusEditor
				}
				m.recalcLayout()
				m.updateFocus()
				return m, nil
			}
			if zone.Get("btn-terminal").InBounds(msg) {
				if m.terminalVisible {
					cmd := m.terminal.AddTerm()
					return m, cmd
				}
				return m, m.toggleTerminal()
			}
			if zone.Get("btn-cmd").InBounds(msg) {
				m.overlay.OpenCommandPalette(m.commandNames())
				return m, nil
			}
			if zone.Get("btn-find").InBounds(msg) {
				m.overlay.OpenFileFinder(m.cfg.Path)
				return m, nil
			}
			if zone.Get("btn-split").InBounds(msg) {
				m.panes.Split(editor.SplitRight)
				m.recalcLayout()
				return m, nil
			}
			if zone.Get("btn-ai").InBounds(msg) {
				if m.aiPanelVisible {
					cmd := m.aiPanel.AddTermWithCmd(m.aiProvider(), m.aiCommand())
					return m, cmd
				}
				return m, m.toggleAI()
			}
			if zone.Get("btn-help").InBounds(msg) {
				m.overlay.OpenShortcuts()
				return m, nil
			}
		}
		m.handleMouseFocus(msg.X, msg.Y)

	case tea.MouseMotionMsg:
		if m.dragging != DragNone {
			m.handleDrag(msg.X, msg.Y)
			return m, nil
		}
		// Detect hover over resizable border edges (visual highlight)
		m.hoverEdge = m.detectEdgeHover(msg.X, msg.Y)
		if m.hoverEdge != DragNone {
			return m, nil
		}

	case tea.MouseWheelMsg:
		m.handleMouseFocus(msg.X, msg.Y)

	case tea.MouseReleaseMsg:
		m.dragging = DragNone

	case ui.OpenFileMsg:
		_ = m.panes.OpenFile(msg.Path)
		if m.gitRoot != "" {
			m.panes.RefreshAllDiffs(m.gitRoot)
		}
		m.focus = FocusEditor
		m.updateFocus()
		return m, nil

	case ui.DirLoadedMsg:
		// Always route to sidebar regardless of focus.
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		m.dirtyPanels |= dirtySidebar
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	// Dispatch to focused child
	var cmd tea.Cmd
	switch m.focus {
	case FocusSidebar:
		m.sidebar, cmd = m.sidebar.Update(msg)
		m.dirtyPanels |= dirtySidebar
		cmds = append(cmds, cmd)
	case FocusEditor:
		m.panes, cmd = m.panes.Update(msg)
		cmds = append(cmds, cmd)
	case FocusTerminal:
		m.terminal, cmd = m.terminal.Update(msg)
		m.dirtyPanels |= dirtyTerminal
		cmds = append(cmds, cmd)
	case FocusAI:
		m.aiPanel, cmd = m.aiPanel.Update(msg)
		m.dirtyPanels |= dirtyAI
		cmds = append(cmds, cmd)
	}

	// Forward tick messages to terminal/AI for rendering (even when not focused)
	if _, ok := msg.(terminal.TickMsg); ok {
		if m.focus != FocusTerminal && m.terminalVisible {
			m.terminal, cmd = m.terminal.Update(msg)
			m.dirtyPanels |= dirtyTerminal
			cmds = append(cmds, cmd)
		}
		if m.focus != FocusAI && m.aiPanelVisible {
			m.aiPanel, cmd = m.aiPanel.Update(msg)
			m.dirtyPanels |= dirtyAI
			cmds = append(cmds, cmd)
		}
	}

	// Auto-hide panels when all tabs closed
	if m.terminalVisible && m.terminal.TabCount() == 0 {
		m.terminalVisible = false
		if m.focus == FocusTerminal {
			m.focus = FocusEditor
		}
		m.recalcLayout()
		m.updateFocus()
	}
	if m.aiPanelVisible && m.aiPanel.TabCount() == 0 {
		m.aiPanelVisible = false
		if m.focus == FocusAI {
			m.focus = FocusEditor
		}
		m.recalcLayout()
		m.updateFocus()
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) toggleTerminal() tea.Cmd {
	m.terminalVisible = !m.terminalVisible
	if !m.terminalVisible && m.focus == FocusTerminal {
		m.focus = FocusEditor
	}
	if m.terminalVisible {
		m.focus = FocusTerminal
	}
	m.recalcLayout()
	m.updateFocus()
	return m.terminal.Start()
}

func (m *Model) toggleAI() tea.Cmd {
	m.aiPanelVisible = !m.aiPanelVisible
	if !m.aiPanelVisible && m.focus == FocusAI {
		m.focus = FocusEditor
	}
	if m.aiPanelVisible {
		m.focus = FocusAI
		// If no provider set, show picker
		if m.cfg.AIProvider == "" && m.aiPanel.TabCount() == 0 {
			m.overlay.OpenCommandPalette([]string{
				"AI: kiro-cli",
				"AI: claude",
				"AI: codex",
				"AI: shell (plain terminal)",
			})
			m.recalcLayout()
			m.updateFocus()
			return nil
		}
	}
	m.recalcLayout()
	m.updateFocus()
	if m.aiPanelVisible && m.aiPanel.TabCount() == 0 {
		return m.aiPanel.AddTermWithCmd(m.aiProvider(), m.aiCommand())
	}
	return nil
}

func (m Model) aiProvider() string {
	if m.cfg.AIProvider != "" {
		return m.cfg.AIProvider
	}
	return "kiro-cli"
}

func (m Model) aiCommand() string {
	switch m.aiProvider() {
	case "kiro-cli":
		return "kiro-cli chat"
	case "claude":
		return "claude"
	case "codex":
		return "codex"
	case "shell":
		return "" // empty = default shell
	default:
		return m.cfg.AIProvider
	}
}

func (m *Model) handleMouseFocus(x, y int) {
	sw := 0
	if m.sidebarVisible {
		sw = m.sidebarW
	}
	aw := 0
	if m.aiPanelVisible {
		aw = m.aiPanelW
	}

	oldFocus := m.focus
	if m.sidebarVisible && x < sw {
		m.focus = FocusSidebar
	} else if m.aiPanelVisible && x >= m.width-aw {
		m.focus = FocusAI
	} else if m.terminalVisible {
		contentH := max(m.height-chromeH, 1)
		tH := max(int(float64(contentH)*m.terminalRatio), 5)
		termTop := 1 + contentH - tH
		if y >= termTop {
			m.focus = FocusTerminal
		} else {
			m.focus = FocusEditor
		}
	} else {
		m.focus = FocusEditor
	}
	if m.focus != oldFocus {
		m.updateFocus()
	}
}

func (m *Model) updateFocus() {
	m.panes.SetFocused(m.focus == FocusEditor)
	m.terminal.SetFocused(m.focus == FocusTerminal)
	m.aiPanel.SetFocused(m.focus == FocusAI)
	m.dirtyPanels = dirtyAll // border color changes on focus
}

// startDrag detects which panel edge to resize based on right-click position.
func (m *Model) startDrag(x, y int) {
	sw := 0
	if m.sidebarVisible {
		sw = m.sidebarW
	}
	aw := 0
	if m.aiPanelVisible {
		aw = m.aiPanelW
	}
	aiEdge := m.width - aw

	// Find closest vertical divider
	distSidebar := abs(x - sw)
	distAI := abs(x - aiEdge)

	// Horizontal: terminal divider
	if m.terminalVisible {
		contentH := max(m.height-chromeH, 1)
		tH := max(int(float64(contentH)*m.terminalRatio), 5)
		termTop := 1 + contentH - tH
		distTerm := abs(y - termTop)
		// If closer to horizontal divider than any vertical
		if distTerm < distSidebar && distTerm < distAI {
			m.dragging = DragTerminalEdge
			return
		}
	}

	if m.sidebarVisible && (distSidebar <= distAI || !m.aiPanelVisible) {
		m.dragging = DragSidebarEdge
	} else if m.aiPanelVisible {
		m.dragging = DragAIEdge
	}
}

// detectEdgeHover returns which border edge the mouse is hovering over (within 1 cell).
func (m *Model) detectEdgeHover(x, y int) DragTarget {
	const threshold = 1

	sw := 0
	if m.sidebarVisible {
		sw = m.sidebarW
	}
	aw := 0
	if m.aiPanelVisible {
		aw = m.aiPanelW
	}
	aiEdge := m.width - aw

	// Check horizontal terminal divider
	if m.terminalVisible {
		contentH := max(m.height-chromeH, 1)
		tH := max(int(float64(contentH)*m.terminalRatio), 5)
		termTop := 1 + contentH - tH
		if abs(y-termTop) <= threshold && y > 0 && y < m.height-1 {
			return DragTerminalEdge
		}
	}

	// Check vertical dividers (only in the content area, not title/status)
	if y > 0 && y < m.height-1 {
		if m.sidebarVisible && abs(x-sw) <= threshold {
			return DragSidebarEdge
		}
		if m.aiPanelVisible && abs(x-aiEdge) <= threshold {
			return DragAIEdge
		}
	}

	return DragNone
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (m *Model) handleDrag(x, y int) {
	switch m.dragging {
	case DragSidebarEdge:
		m.sidebarW = max(min(x, m.width/2), minPanelW)
		m.recalcLayout()
	case DragAIEdge:
		m.aiPanelW = max(min(m.width-x, m.width/2), minPanelW)
		m.recalcLayout()
	case DragTerminalEdge:
		contentH := max(m.height-chromeH, 1)
		// y is the new divider position from top
		termH := max(m.height-1-y, minTermH) // -1 for status bar
		m.terminalRatio = float64(termH) / float64(contentH)
		m.terminalRatio = max(min(m.terminalRatio, 0.8), 0.1)
		m.recalcLayout()
	}
}

func (m *Model) recalcLayout() {
	m.dirtyPanels = dirtyAll
	contentH := max(m.height-chromeH, 1)

	sw := 0
	if m.sidebarVisible {
		sw = m.sidebarW
	}
	aw := 0
	if m.aiPanelVisible {
		aw = m.aiPanelW
	}
	cw := max(m.width-sw-aw, 10)

	si := max(sw-2, 0)
	ai := max(aw-2, 0)
	ch := max(contentH-2, 0)

	edH := ch
	tH := 0
	if m.terminalVisible {
		tH = max(int(float64(contentH)*m.terminalRatio), 5)
		edH = max(contentH-tH-2, 1)
		tH = max(tH-2, 1)
	}

	m.sidebar.SetSize(si, ch)
	m.sidebar.SetScreenY(2) // title bar (1) + top border (1)
	m.terminal.SetSize(max(cw-2, 0), tH)
	m.aiPanel.SetSize(ai, ch)

	if m.panes.PaneCount() == 1 {
		m.panes.SetSize(max(cw-2, 1), edH)
	} else {
		m.panes.SetSize(cw, edH+2)
	}

	editorOffsetX := sw + 1
	editorOffsetY := 1 + 1
	if m.panes.PaneCount() > 1 {
		editorOffsetX = sw
		editorOffsetY = 1
	}
	m.panes.SetOffset(editorOffsetX, editorOffsetY)
}

func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	if m.width == 0 {
		v.Content = "loading..."
		return v
	}

	contentH := max(m.height-chromeH, 1)

	// Compute column widths (mirrors recalcLayout)
	sw := 0
	if m.sidebarVisible {
		sw = m.sidebarW
	}
	aw := 0
	if m.aiPanelVisible {
		aw = m.aiPanelW
	}
	cw := max(m.width-sw-aw, 10)

	// Title bar with clickable buttons
	dir := filepath.Base(m.cfg.Path)
	titleLeft := fmt.Sprintf(" ● grotto — %s", dir)
	if m.panes.HasFile() && m.panes.Buf() != nil {
		fname := filepath.Base(m.panes.Buf().FilePath)
		dirty := ""
		if m.panes.Buf().Dirty {
			dirty = " ●"
		}
		titleLeft = fmt.Sprintf(" ● grotto — %s — %s%s", dir, fname, dirty)
	}

	// Buttons
	bFiles := m.btn("btn-files", "📁 Files", m.sidebarVisible)
	bFind := m.btn("btn-find", "🔍 Find", false)
	bCmd := m.btn("btn-cmd", "⌘ Cmd", false)
	bSplit := m.btn("btn-split", "◫ Split", false)
	bTerm := m.btn("btn-terminal", "▶ Term", m.terminalVisible)
	bAI := m.btn("btn-ai", "✦ AI", m.aiPanelVisible)
	bHelp := m.btn("btn-help", "? Keys", false)

	buttons := bFiles + " " + bFind + " " + bCmd + " " + bSplit + " " + bTerm + " " + bAI + " " + bHelp
	gap := max(m.width-lipgloss.Width(titleLeft)-lipgloss.Width(buttons)-2, 1)
	titleContent := titleLeft + strings.Repeat(" ", gap) + buttons
	title := titleStyle.Width(m.width).Render(titleContent)

	// Status bar
	statusText := " READY"
	if m.hoverEdge != DragNone {
		statusText = " ↔ Right-click + drag to resize"
	} else if m.panes.HasFile() && m.panes.Buf() != nil {
		b := m.panes.Buf()
		statusText = fmt.Sprintf(" Ln %d, Col %d │ %s",
			b.Cursor.Line+1, b.Cursor.Col+1,
			filepath.Base(b.FilePath))
		if b.Dirty {
			statusText += " [modified]"
		}
	}
	status := statusStyle.Width(m.width).Render(statusText)

	// Editor area
	var center string
	if m.panes.PaneCount() == 1 {
		eb := m.bdr(FocusEditor)
		center = eb.Width(cw - 2).Height(m.panes.Height()).Render(m.panes.View())
	} else {
		center = m.panes.View()
	}

	if m.terminalVisible {
		if m.dirtyPanels&dirtyTerminal != 0 || m.cachedTerminal == "" {
			tb := m.bdr(FocusTerminal)
			m.cachedTerminal = tb.Width(cw - 2).Height(m.terminal.Height()).Render(m.terminal.View())
		}
		center = lipgloss.JoinVertical(lipgloss.Left, center, m.cachedTerminal)
	}

	// Clamp center column to exactly contentH lines
	centerLines := strings.Split(center, "\n")
	if len(centerLines) > contentH {
		centerLines = centerLines[:contentH]
		center = strings.Join(centerLines, "\n")
	}

	var cols []string
	if m.sidebarVisible {
		if m.dirtyPanels&dirtySidebar != 0 || m.cachedSidebar == "" {
			sb := m.bdr(FocusSidebar)
			m.cachedSidebar = sb.Height(contentH - 2).Render(m.sidebar.View())
		}
		cols = append(cols, m.cachedSidebar)
	}
	cols = append(cols, center)
	if m.aiPanelVisible {
		if m.dirtyPanels&dirtyAI != 0 || m.cachedAI == "" {
			ab := m.bdr(FocusAI)
			aiBox := ab.Width(aw - 2).Height(contentH - 2).Render(m.aiPanel.View())
			// Clamp to exactly contentH lines, preserving bottom border
			aiBoxLines := strings.Split(aiBox, "\n")
			if len(aiBoxLines) > contentH {
				top := aiBoxLines[:contentH-1]
				bottom := aiBoxLines[len(aiBoxLines)-1]
				aiBoxLines = append(top, bottom)
			}
			m.cachedAI = strings.Join(aiBoxLines, "\n")
		}
		cols = append(cols, m.cachedAI)
	}
	m.dirtyPanels = 0

	content := lipgloss.JoinHorizontal(lipgloss.Top, cols...)

	// Only zone.Scan the title bar (where zone markers live) — not the full output
	title = zone.Scan(title)

	full := lipgloss.JoinVertical(lipgloss.Left, title, content, status)

	// Overlay floats on top
	if m.overlay.Active() {
		overlayView := m.overlay.View(m.width, m.height)
		lines := strings.Split(full, "\n")
		overlayLines := strings.Split(overlayView, "\n")
		insertAt := 2
		for i, ol := range overlayLines {
			pos := insertAt + i
			if pos < len(lines) {
				lines[pos] = ol
			}
		}
		full = strings.Join(lines, "\n")
	}

	v.Content = full
	return v
}

func (m Model) btn(id, label string, active bool) string {
	s := btnStyle
	if active {
		s = btnActiveStyle
	}
	return zone.Mark(id, s.Render(label))
}

func (m Model) bdr(p FocusedPanel) lipgloss.Style {
	// Hover over resize edge takes priority — signals "draggable"
	if m.hoverEdge != DragNone {
		switch p {
		case FocusSidebar:
			if m.hoverEdge == DragSidebarEdge {
				return borderHover
			}
		case FocusAI:
			if m.hoverEdge == DragAIEdge {
				return borderHover
			}
		case FocusEditor:
			if m.hoverEdge == DragSidebarEdge || m.hoverEdge == DragAIEdge || m.hoverEdge == DragTerminalEdge {
				return borderHover
			}
		case FocusTerminal:
			if m.hoverEdge == DragTerminalEdge {
				return borderHover
			}
		}
	}
	if m.focus == p {
		return borderActive
	}
	return borderDim
}

func (m Model) commandNames() []string {
	return []string{
		"Toggle Sidebar",
		"Toggle Terminal",
		"Toggle AI Panel",
		"Send Selection to AI",
		"AI: kiro-cli",
		"AI: claude",
		"AI: codex",
		"AI: shell (plain terminal)",
		"Split Right",
		"Split Down",
		"Close Pane",
		"Quit",
	}
}

func (m *Model) execCommand(name string) tea.Cmd {
	switch name {
	case "Toggle Sidebar":
		m.sidebarVisible = !m.sidebarVisible
		m.recalcLayout()
	case "Toggle Terminal":
		return m.toggleTerminal()
	case "Toggle AI Panel":
		return m.toggleAI()
	case "Send Selection to AI":
		return m.sendSelectionToAI()
	case "Split Right":
		m.panes.Split(editor.SplitRight)
		m.recalcLayout()
	case "Split Down":
		m.panes.Split(editor.SplitDown)
		m.recalcLayout()
	case "Close Pane":
		m.panes.ClosePane()
		m.recalcLayout()
	case "Quit":
		return tea.Quit
	case "AI: kiro-cli", "AI: claude", "AI: codex", "AI: shell (plain terminal)":
		provider := strings.TrimPrefix(name, "AI: ")
		if provider == "shell (plain terminal)" {
			provider = "shell"
		}
		m.cfg.AIProvider = provider
		m.aiPanelVisible = true
		m.focus = FocusAI
		m.recalcLayout()
		m.updateFocus()
		return m.aiPanel.AddTermWithCmd(provider, m.aiCommand())
	}
	return nil
}
