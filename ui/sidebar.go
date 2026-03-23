package ui

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// OpenFileMsg is emitted when a file is selected.
type OpenFileMsg struct {
	Path string
}

// TreeNode represents a file or directory in the tree.
type TreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Children []TreeNode
	Expanded bool
	Depth    int
}

// FlatNode is a visible node in the flattened tree.
type FlatNode struct {
	Node  *TreeNode
	Depth int
}

// Model is the sidebar file tree component.
type Model struct {
	root          TreeNode
	flat          []FlatNode
	cursor        int
	scroll        int
	width         int
	height        int
	ignores       []string
	selectedStyle lipgloss.Style
}

// New creates a sidebar model rooted at the given path.
func New(rootPath string) Model {
	abs, _ := filepath.Abs(rootPath)
	ignores := loadGitignore(abs)

	m := Model{
		ignores: ignores,
		selectedStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FAFAFA")),
	}

	m.root = TreeNode{
		Name:     filepath.Base(abs),
		Path:     abs,
		IsDir:    true,
		Expanded: true,
		Depth:    0,
	}
	m.root.Children = readDir(abs, 0, ignores)
	m.flatten()
	return m
}

// SetSize updates the sidebar dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) Width() int  { return m.width }
func (m Model) Height() int { return m.height }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case "down":
			if m.cursor < len(m.flat)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case "enter":
			return m.selectCurrent()
		}

	case tea.MouseClickMsg:
		for i, fn := range m.flat {
			if zone.Get("sidebar-" + fn.Node.Path).InBounds(msg) {
				m.cursor = i
				return m.selectCurrent()
			}
		}

	case tea.MouseWheelMsg:
		if msg.Y < 0 && m.scroll > 0 {
			m.scroll--
		} else if msg.Y > 0 && m.scroll < len(m.flat)-m.height {
			m.scroll++
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 || len(m.flat) == 0 {
		return ""
	}

	var b strings.Builder
	end := m.scroll + m.height
	if end > len(m.flat) {
		end = len(m.flat)
	}

	for i := m.scroll; i < end; i++ {
		fn := m.flat[i]
		indent := strings.Repeat("  ", fn.Depth)

		var icon string
		if fn.Node.IsDir {
			if fn.Node.Expanded {
				icon = "▼ 📁 "
			} else {
				icon = "▶ 📁 "
			}
		} else {
			icon = "  📄 "
		}

		line := indent + icon + fn.Node.Name

		// Truncate to width
		if m.width > 0 && len(line) > m.width {
			line = line[:m.width]
		}

		if i == m.cursor {
			line = m.selectedStyle.Render(line)
		}

		line = zone.Mark("sidebar-"+fn.Node.Path, line)

		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// selectCurrent toggles a directory or emits OpenFileMsg for a file.
func (m Model) selectCurrent() (Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.flat) {
		return m, nil
	}
	node := m.flat[m.cursor].Node
	if node.IsDir {
		node.Expanded = !node.Expanded
		if node.Expanded && len(node.Children) == 0 {
			node.Children = readDir(node.Path, node.Depth, m.ignores)
		}
		m.flatten()
		return m, nil
	}
	return m, func() tea.Msg { return OpenFileMsg{Path: node.Path} }
}

func (m *Model) ensureVisible() {
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	} else if m.height > 0 && m.cursor >= m.scroll+m.height {
		m.scroll = m.cursor - m.height + 1
	}
}

func (m *Model) flatten() {
	m.flat = m.flat[:0]
	flattenNode(&m.root, &m.flat)
}

func flattenNode(n *TreeNode, out *[]FlatNode) {
	*out = append(*out, FlatNode{Node: n, Depth: n.Depth})
	if n.IsDir && n.Expanded {
		for i := range n.Children {
			flattenNode(&n.Children[i], out)
		}
	}
}

func readDir(dir string, parentDepth int, ignores []string) []TreeNode {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var dirs, files []TreeNode
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || isIgnored(name, ignores) {
			continue
		}
		node := TreeNode{
			Name:  name,
			Path:  filepath.Join(dir, name),
			IsDir: e.IsDir(),
			Depth: parentDepth + 1,
		}
		if node.IsDir {
			dirs = append(dirs, node)
		} else {
			files = append(files, node)
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return append(dirs, files...)
}

func loadGitignore(root string) []string {
	f, err := os.Open(filepath.Join(root, ".gitignore"))
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var patterns []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSuffix(line, "/")
		patterns = append(patterns, line)
	}
	return patterns
}

func isIgnored(name string, patterns []string) bool {
	for _, p := range patterns {
		if name == p {
			return true
		}
		// Simple wildcard: *.ext
		if strings.HasPrefix(p, "*") && strings.HasSuffix(name, p[1:]) {
			return true
		}
	}
	return false
}
