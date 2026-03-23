package terminal

import (
	"fmt"
	"image/color"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ActiveState/vt10x"
	pty "github.com/creack/pty/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// TickMsg is the render tick, prefix-scoped.
type TickMsg struct{ Prefix string }

func tickCmd(prefix string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(50 * time.Millisecond)
		return TickMsg{Prefix: prefix}
	}
}

type screenLine struct {
	chars []rune
	fg    []vt10x.Color
	bg    []vt10x.Color
}

type term struct {
	ptyFile *os.File
	cmd     *exec.Cmd
	vt      *vt10x.VT
	state   *vt10x.State
	done    atomic.Bool
	name    string
	history []screenLine // scrollback ring buffer
	prevTop string       // previous top row content, to detect scroll
}

type Model struct {
	tabs    []*term
	active  int
	width   int
	height  int
	focused bool
	prefix  string
	ticking bool
	scrollY int // lines into history (0 = live)
}

func New() Model              { return Model{prefix: "term"} }
func NewAI() Model            { return Model{prefix: "ai"} }
func (m Model) Init() tea.Cmd { return nil }

func (m Model) termH() int {
	if len(m.tabs) > 0 {
		return max(m.height-1, 1)
	}
	return m.height
}

func (m *Model) closeTerm(i int) {
	if i < 0 || i >= len(m.tabs) {
		return
	}
	t := m.tabs[i]
	if t.ptyFile != nil {
		_ = t.ptyFile.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
	m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
	if m.active >= len(m.tabs) {
		m.active = max(len(m.tabs)-1, 0)
	}
	m.scrollY = 0
}

func (m Model) TabCount() int                       { return len(m.tabs) }
func (m *Model) AddTerm() tea.Cmd                   { return m.addTermCmd("", "") }
func (m *Model) AddTermWithCmd(n, c string) tea.Cmd { return m.addTermCmd(n, c) }

func (m *Model) addTermCmd(name, command string) tea.Cmd {
	if m.width <= 0 || m.termH() <= 0 {
		return nil
	}
	var cmd *exec.Cmd
	if command != "" {
		parts := strings.Fields(command)
		cmd = exec.Command(parts[0], parts[1:]...)
	} else {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmd = exec.Command(shell)
	}
	if name == "" {
		name = fmt.Sprintf("sh %d", len(m.tabs)+1)
	}
	h := m.termH()
	cmd.Env = filterEnv(os.Environ(), "TERM", "COLORTERM")
	cmd.Env = append(cmd.Env, "TERM=xterm-256color", "COLORTERM=", fmt.Sprintf("COLUMNS=%d", m.width), fmt.Sprintf("LINES=%d", h))

	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(h), Cols: uint16(m.width)})
	if err != nil {
		return nil
	}

	st := &vt10x.State{}
	vt, _ := vt10x.New(st, f, f)
	vt.Resize(m.width, h)

	t := &term{ptyFile: f, cmd: cmd, vt: vt, state: st, name: name}
	m.tabs = append(m.tabs, t)
	m.active = len(m.tabs) - 1
	m.scrollY = 0

	go func() {
		for {
			if err := vt.Parse(); err != nil {
				break
			}
		}
		t.done.Store(true)
	}()

	if !m.ticking {
		m.ticking = true
		return tickCmd(m.prefix)
	}
	return nil
}

func (m *Model) Start() tea.Cmd {
	if len(m.tabs) > 0 {
		return nil
	}
	return m.addTermCmd("", "")
}

func (m *Model) SetSize(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	m.width = w
	m.height = h
	th := m.termH()
	for _, t := range m.tabs {
		if t.ptyFile != nil {
			_ = pty.Setsize(t.ptyFile, &pty.Winsize{Rows: uint16(th), Cols: uint16(w)})
			t.vt.Resize(w, th)
		}
	}
}

func (m *Model) SetFocused(f bool) { m.focused = f }
func (m Model) Width() int         { return m.width }
func (m Model) Height() int        { return m.height }

// captureHistory snapshots the top row; if it changed, the old one scrolled off.
func (m *Model) captureHistory() {
	if len(m.tabs) == 0 || m.width <= 0 {
		return
	}
	t := m.tabs[m.active]
	if t.state == nil {
		return
	}
	t.state.Lock()
	defer t.state.Unlock()

	// Read current top row as string
	var top strings.Builder
	cols := m.width
	for x := range cols {
		ch, _, _ := t.state.Cell(x, 0)
		if ch == 0 {
			ch = ' '
		}
		top.WriteRune(ch)
	}
	cur := top.String()

	if t.prevTop != "" && t.prevTop != cur {
		// Previous top row scrolled off — save it
		sl := screenLine{
			chars: []rune(t.prevTop),
			fg:    make([]vt10x.Color, cols),
			bg:    make([]vt10x.Color, cols),
		}
		// Colors are lost for scrolled-off lines, use defaults
		for i := range sl.fg {
			sl.fg[i] = vt10x.DefaultFG
			sl.bg[i] = vt10x.DefaultBG
		}
		t.history = append(t.history, sl)
		if len(t.history) > 2000 {
			t.history = t.history[len(t.history)-2000:]
		}
	}
	t.prevTop = cur
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		if msg.Prefix != m.prefix {
			return m, nil
		}
		if len(m.tabs) > 0 && m.scrollY == 0 {
			m.captureHistory()
		}
		anyAlive := false
		for _, t := range m.tabs {
			if !t.done.Load() {
				anyAlive = true
				break
			}
		}
		if !anyAlive && len(m.tabs) == 0 {
			m.ticking = false
			return m, nil
		}
		return m, tickCmd(m.prefix)

	case tea.KeyPressMsg:
		if !m.focused || len(m.tabs) == 0 {
			return m, nil
		}
		m.scrollY = 0
		t := m.tabs[m.active]
		if t.ptyFile == nil {
			return m, nil
		}
		s := keyToSeq(msg)
		if s != "" {
			_, _ = io.WriteString(t.ptyFile, s)
		}
		return m, nil

	case tea.MouseWheelMsg:
		if len(m.tabs) == 0 {
			return m, nil
		}
		t := m.tabs[m.active]
		maxScroll := len(t.history)
		switch msg.Button {
		case tea.MouseWheelUp:
			m.scrollY = min(m.scrollY+3, maxScroll)
		case tea.MouseWheelDown:
			m.scrollY = max(m.scrollY-3, 0)
		}
		return m, nil

	case tea.MouseClickMsg:
		if len(m.tabs) == 0 {
			return m, nil
		}
		if zone.Get(fmt.Sprintf("%s-new", m.prefix)).InBounds(msg) {
			return m, m.addTermCmd("", "")
		}
		for i := range m.tabs {
			if zone.Get(fmt.Sprintf("%s-close-%d", m.prefix, i)).InBounds(msg) {
				m.closeTerm(i)
				return m, nil
			}
			if zone.Get(fmt.Sprintf("%s-tab-%d", m.prefix, i)).InBounds(msg) {
				m.active = i
				m.scrollY = 0
				return m, nil
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) Close() {
	for _, t := range m.tabs {
		if t.ptyFile != nil {
			_ = t.ptyFile.Close()
		}
		if t.cmd != nil && t.cmd.Process != nil {
			_ = t.cmd.Process.Kill()
		}
	}
}

func (m Model) View() string {
	if len(m.tabs) == 0 || m.width <= 0 || m.height <= 0 {
		return ""
	}

	var out strings.Builder

	// Tab bar
	if len(m.tabs) > 1 {
		tabDim := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		tabAct := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Underline(true)
		closeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CC0000"))
		for i, t := range m.tabs {
			s := tabDim
			if i == m.active {
				s = tabAct
			}
			out.WriteString(zone.Mark(fmt.Sprintf("%s-tab-%d", m.prefix, i), s.Render(" "+t.name+" ")))
			out.WriteString(zone.Mark(fmt.Sprintf("%s-close-%d", m.prefix, i), closeStyle.Render("✕ ")))
		}
		out.WriteString(zone.Mark(fmt.Sprintf("%s-new", m.prefix), lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(" [+] ")))
		out.WriteByte('\n')
	} else {
		closeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CC0000"))
		out.WriteString(zone.Mark(fmt.Sprintf("%s-close-0", m.prefix), closeStyle.Render(" ✕ close ")))
		out.WriteByte('\n')
	}

	t := m.tabs[m.active]
	if t.state == nil {
		return out.String()
	}

	t.state.Lock()
	defer t.state.Unlock()

	th := m.termH()

	if m.scrollY > 0 && len(t.history) > 0 {
		// Scrollback view
		hLen := len(t.history)
		// Show history lines, then fill with live screen lines
		histStart := max(hLen-m.scrollY, 0)
		histEnd := hLen
		histLines := histEnd - histStart
		liveLines := th - histLines

		for i := 0; i < th; i++ {
			if i > 0 {
				out.WriteByte('\n')
			}
			hi := histStart + i
			if i < histLines && hi < hLen {
				// History line (no color — just text)
				sl := &t.history[hi]
				dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
				for x := range m.width {
					if x < len(sl.chars) {
						out.WriteString(dimStyle.Render(string(sl.chars[x])))
					} else {
						out.WriteByte(' ')
					}
				}
			} else {
				// Live screen line
				ly := i - histLines
				if ly >= 0 && liveLines > 0 {
					for x := range m.width {
						ch, fg, bg := t.state.Cell(x, ly)
						if ch == 0 {
							ch = ' '
						}
						style := lipgloss.NewStyle()
						if c := vtColor(fg); c != nil {
							style = style.Foreground(c)
						}
						if c := vtColor(bg); c != nil {
							style = style.Background(c)
						}
						out.WriteString(style.Render(string(ch)))
					}
				}
			}
		}
		// Scroll indicator
		ind := fmt.Sprintf(" ↑ %d lines ", m.scrollY)
		indStyle := lipgloss.NewStyle().Background(lipgloss.Color("#C4A000")).Foreground(lipgloss.Color("#000000"))
		s := out.String()
		lines := strings.Split(s, "\n")
		if len(lines) > 0 {
			lines[len(lines)-1] = indStyle.Render(ind) + strings.Repeat(" ", max(m.width-len(ind), 0))
		}
		return strings.Join(lines, "\n")
	}

	// Live view
	cx, cy := t.state.Cursor()
	curVis := t.state.CursorVisible()

	for y := range th {
		if y > 0 {
			out.WriteByte('\n')
		}
		for x := range m.width {
			ch, fg, bg := t.state.Cell(x, y)
			if ch == 0 {
				ch = ' '
			}
			style := lipgloss.NewStyle()
			if c := vtColor(fg); c != nil {
				style = style.Foreground(c)
			}
			if c := vtColor(bg); c != nil {
				style = style.Background(c)
			}
			if curVis && m.focused && x == cx && y == cy {
				style = style.Reverse(true)
			}
			out.WriteString(style.Render(string(ch)))
		}
	}
	return out.String()
}

// filterEnv removes specified keys from an env slice.
func filterEnv(env []string, keys ...string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		skip := false
		for _, k := range keys {
			if strings.HasPrefix(e, k+"=") {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, e)
		}
	}
	return out
}

var ansiPalette = [16]string{
	"#000000", "#CC0000", "#4E9A06", "#C4A000",
	"#3465A4", "#75507B", "#06989A", "#D3D7CF",
	"#555753", "#EF2929", "#8AE234", "#FCE94F",
	"#729FCF", "#AD7FA8", "#34E2E2", "#EEEEEC",
}

func vtColor(c vt10x.Color) color.Color {
	if c == vt10x.DefaultFG || c == vt10x.DefaultBG {
		return nil
	}
	if c.ANSI() {
		return lipgloss.Color(ansiPalette[c])
	}
	if c < 256 {
		return lipgloss.Color(fmt.Sprintf("%d", c))
	}
	return nil
}

func keyToSeq(msg tea.KeyPressMsg) string {
	if msg.Text != "" {
		return msg.Text
	}
	switch msg.String() {
	case "enter":
		return "\r"
	case "backspace":
		return "\x7f"
	case "tab":
		return "\t"
	case "esc":
		return "\x1b"
	case "up":
		return "\x1b[A"
	case "down":
		return "\x1b[B"
	case "right":
		return "\x1b[C"
	case "left":
		return "\x1b[D"
	case "home":
		return "\x1b[H"
	case "end":
		return "\x1b[F"
	case "delete":
		return "\x1b[3~"
	case "pgup":
		return "\x1b[5~"
	case "pgdown":
		return "\x1b[6~"
	case "ctrl+c":
		return "\x03"
	case "ctrl+d":
		return "\x04"
	case "ctrl+z":
		return "\x1a"
	case "ctrl+l":
		return "\x0c"
	case "ctrl+a":
		return "\x01"
	case "ctrl+e":
		return "\x05"
	case "ctrl+k":
		return "\x0b"
	case "ctrl+u":
		return "\x15"
	case "ctrl+w":
		return "\x17"
	case "ctrl+r":
		return "\x12"
	case "ctrl+p":
		return "\x10"
	case "ctrl+n":
		return "\x0e"
	}
	if len(msg.String()) == 6 && msg.String()[:5] == "ctrl+" {
		c := msg.String()[5]
		if c >= 'a' && c <= 'z' {
			return string(rune(c - 'a' + 1))
		}
	}
	return ""
}
