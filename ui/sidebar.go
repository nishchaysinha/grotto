package ui

import (
	"bufio"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/owomeister/grotto/gitstatus"
)

var (
	gitAddedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))
	gitModifiedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B"))
	gitDeletedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E06C75"))
	gitRenamedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#61AFEF"))
	gitUntrackedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#56B6C2"))
	selectedStyle     = lipgloss.NewStyle().
				Background(lipgloss.Color("#7D56F4")).
				Foreground(lipgloss.Color("#FAFAFA"))
	dirIconStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#89DDFF"))
	indentGuide   = lipgloss.NewStyle().Foreground(lipgloss.Color("#3E4452"))
	fileIconStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6D8086"))
)

// fileIconColor returns a color for the given filename based on its extension.
// All icons are single-width Unicode (1 cell) to avoid emoji width issues.
func fileIconColor(name string) color.Color {
	ext := strings.ToLower(filepath.Ext(name))
	base := strings.ToLower(name)

	switch base {
	case "makefile", "justfile":
		return lipgloss.Color("#6D8086")
	case "dockerfile", "containerfile":
		return lipgloss.Color("#2496ED")
	case "license", "licence":
		return lipgloss.Color("#D4AA00")
	case "package.json":
		return lipgloss.Color("#8BC34A")
	case "go.mod", "go.sum":
		return lipgloss.Color("#00ADD8")
	}

	switch ext {
	case ".go":
		return lipgloss.Color("#00ADD8")
	case ".rs":
		return lipgloss.Color("#DEA584")
	case ".py":
		return lipgloss.Color("#FFBC03")
	case ".js", ".mjs", ".cjs", ".jsx":
		return lipgloss.Color("#F1E05A")
	case ".ts", ".mts", ".cts", ".tsx":
		return lipgloss.Color("#3178C6")
	case ".html", ".htm":
		return lipgloss.Color("#E44D26")
	case ".css", ".scss", ".sass":
		return lipgloss.Color("#563D7C")
	case ".json":
		return lipgloss.Color("#CBCB41")
	case ".yaml", ".yml":
		return lipgloss.Color("#CB171E")
	case ".toml":
		return lipgloss.Color("#9C4221")
	case ".xml":
		return lipgloss.Color("#E37933")
	case ".sh", ".bash", ".zsh", ".fish":
		return lipgloss.Color("#89E051")
	case ".md", ".mdx":
		return lipgloss.Color("#42A5F5")
	case ".c", ".h":
		return lipgloss.Color("#599EFF")
	case ".cpp", ".cc", ".cxx", ".hpp":
		return lipgloss.Color("#F34B7D")
	case ".java":
		return lipgloss.Color("#CC3E44")
	case ".kt", ".kts":
		return lipgloss.Color("#7F52FF")
	case ".rb":
		return lipgloss.Color("#CC342D")
	case ".swift":
		return lipgloss.Color("#F05138")
	case ".lua":
		return lipgloss.Color("#51A0CF")
	case ".sql":
		return lipgloss.Color("#E38C00")
	case ".lock":
		return lipgloss.Color("#6D8086")
	}

	return lipgloss.Color("#6D8086")
}

// OpenFileMsg is emitted when a file is selected.
type OpenFileMsg struct {
	Path string
}

// DirLoadedMsg is returned when an async directory read completes.
type DirLoadedMsg struct {
	Path     string
	Children []TreeNode
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
	root      TreeNode
	flat      []FlatNode
	cursor    int
	scroll    int
	width     int
	height    int
	screenY   int // Y offset of first content row on screen
	ignores   []string
	rootPath  string
	gitStatus map[string]gitstatus.FileStatus
}

// New creates a sidebar model rooted at the given path.
func New(rootPath string) Model {
	abs, _ := filepath.Abs(rootPath)
	ignores := loadGitignore(abs)

	m := Model{
		ignores:  ignores,
		rootPath: abs,
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

// SetScreenY sets the Y offset of the first content row on screen
// (used for coordinate-based mouse click detection).
func (m *Model) SetScreenY(y int) {
	m.screenY = y
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
		case "shift+right":
			m.expandAll(&m.root)
			m.flatten()
		case "shift+left":
			m.collapseAll(&m.root)
			m.flatten()
			m.cursor = 0
			m.scroll = 0
		case "right":
			if m.cursor >= 0 && m.cursor < len(m.flat) {
				node := m.flat[m.cursor].Node
				if node.IsDir && !node.Expanded {
					if len(node.Children) > 0 {
						node.Expanded = true
						m.flatten()
					} else {
						path := node.Path
						depth := node.Depth
						ignores := m.ignores
						return m, func() tea.Msg {
							return DirLoadedMsg{
								Path:     path,
								Children: readDir(path, depth, ignores),
							}
						}
					}
				}
			}
		case "left":
			if m.cursor >= 0 && m.cursor < len(m.flat) {
				node := m.flat[m.cursor].Node
				if node.IsDir && node.Expanded {
					node.Expanded = false
					m.flatten()
				}
			}
		}

	case DirLoadedMsg:
		if node := m.findNode(&m.root, msg.Path); node != nil {
			node.Children = msg.Children
			node.Expanded = true
			m.flatten()
		}

	case tea.MouseClickMsg:
		idx := m.scroll + (msg.Y - m.screenY)
		if idx >= 0 && idx < len(m.flat) {
			m.cursor = idx
			return m.selectCurrent()
		}

	case tea.MouseWheelMsg:
		maxScroll := len(m.flat) - m.height
		if maxScroll < 0 {
			maxScroll = 0
		}
		if msg.Y < 0 && m.scroll > 0 {
			m.scroll--
		} else if msg.Y > 0 && m.scroll < maxScroll {
			m.scroll++
		}
	}
	return m, nil
}

// Refresh reloads git status for all files and propagates status up to parent directories.
func (m *Model) Refresh(gitRoot string) {
	fileStatus := gitstatus.GetFileStatuses(gitRoot)
	if fileStatus == nil {
		m.gitStatus = nil
		return
	}
	// Copy file statuses then propagate to parent directories.
	result := make(map[string]gitstatus.FileStatus, len(fileStatus)*2)
	for path, status := range fileStatus {
		result[path] = status
		dir := filepath.Dir(path)
		for dir != m.rootPath && len(dir) > len(m.rootPath) {
			if gitStatusPriority(status) > gitStatusPriority(result[dir]) {
				result[dir] = status
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
		if gitStatusPriority(status) > gitStatusPriority(result[m.rootPath]) {
			result[m.rootPath] = status
		}
	}
	m.gitStatus = result
}

// gitStatusPriority returns a sort weight so the "most important" status wins
// when multiple statuses compete for the same directory.
func gitStatusPriority(s gitstatus.FileStatus) int {
	switch s {
	case gitstatus.StatusModified:
		return 4
	case gitstatus.StatusDeleted:
		return 3
	case gitstatus.StatusAdded:
		return 2
	case gitstatus.StatusUntracked, gitstatus.StatusRenamed:
		return 1
	default:
		return 0
	}
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
		name := fn.Node.Name

		// Calculate how much width the prefix (indent + icon) consumes,
		// then truncate the name to fit the remainder. This guarantees
		// the visible width is correct regardless of ANSI or zone markers.
		var iconW int
		if fn.Node.IsDir {
			iconW = 3 // folder emoji (2) + space (1)
		} else {
			iconW = 0 // no file icon
		}
		prefixW := fn.Depth*2 + iconW
		nameW := m.width - prefixW
		if nameW < 2 {
			nameW = 2
		}

		// Truncate the raw name before any ANSI styling.
		displayName := name
		if len(displayName) > nameW {
			displayName = displayName[:nameW-1] + "…"
		}

		// Build indent with guide lines (│) at each depth level.
		var plainIndent, styledIndent string
		for d := 0; d < fn.Depth; d++ {
			plainIndent += "│ "
			styledIndent += indentGuide.Render("│") + " "
		}

		var icon, plainIcon string
		if fn.Node.IsDir {
			if fn.Node.Expanded {
				plainIcon = "📂 "
				icon = "📂 "
			} else {
				plainIcon = "📁 "
				icon = "📁 "
			}
		} else {
			plainIcon = ""
			icon = ""
		}

		var line string
		if i == m.cursor {
			// Build plain content, truncate, pad, then style so the
			// purple background covers the full line width.
			plainContent := plainIndent + plainIcon + displayName
			plainContent = ansi.Truncate(plainContent, m.width, "…")
			if vis := ansi.StringWidth(plainContent); vis < m.width {
				plainContent += strings.Repeat(" ", m.width-vis)
			}
			line = selectedStyle.Render(plainContent)
		} else {
			// Apply git status color; inherit untracked status from parent dir
			// because git reports "?? dir/" not individual files inside it.
			styledName := displayName
			if m.gitStatus != nil {
				status := m.gitStatus[fn.Node.Path]
				if status == gitstatus.StatusUnchanged && !fn.Node.IsDir {
					dir := filepath.Dir(fn.Node.Path)
					for len(dir) > len(m.rootPath) {
						if s := m.gitStatus[dir]; s != gitstatus.StatusUnchanged {
							status = s
							break
						}
						parent := filepath.Dir(dir)
						if parent == dir {
							break
						}
						dir = parent
					}
				}
				switch status {
				case gitstatus.StatusAdded:
					styledName = gitAddedStyle.Render(displayName)
				case gitstatus.StatusModified:
					styledName = gitModifiedStyle.Render(displayName)
				case gitstatus.StatusDeleted:
					styledName = gitDeletedStyle.Render(displayName)
				case gitstatus.StatusRenamed:
					styledName = gitRenamedStyle.Render(displayName)
				case gitstatus.StatusUntracked:
					styledName = gitUntrackedStyle.Render(displayName)
				}
			}
			line = styledIndent + icon + styledName

			// Hard-truncate the styled line to m.width, then pad.
			line = ansi.Truncate(line, m.width, "…")
			if vis := ansi.StringWidth(line); vis < m.width {
				line += strings.Repeat(" ", m.width-vis)
			}
		}

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
		if node.Expanded {
			node.Expanded = false
			m.flatten()
			return m, nil
		}
		if len(node.Children) > 0 {
			node.Expanded = true
			m.flatten()
			return m, nil
		}
		// Load children asynchronously.
		path := node.Path
		depth := node.Depth
		ignores := m.ignores
		return m, func() tea.Msg {
			return DirLoadedMsg{
				Path:     path,
				Children: readDir(path, depth, ignores),
			}
		}
	}
	return m, func() tea.Msg { return OpenFileMsg{Path: node.Path} }
}

// findNode locates a TreeNode by path in the tree.
func (m *Model) findNode(n *TreeNode, path string) *TreeNode {
	if n.Path == path {
		return n
	}
	for i := range n.Children {
		if found := m.findNode(&n.Children[i], path); found != nil {
			return found
		}
	}
	return nil
}

func (m *Model) expandAll(n *TreeNode) {
	if n.IsDir {
		n.Expanded = true
		if len(n.Children) == 0 {
			n.Children = readDir(n.Path, n.Depth, m.ignores)
		}
		for i := range n.Children {
			m.expandAll(&n.Children[i])
		}
	}
}

func (m *Model) collapseAll(n *TreeNode) {
	if n.IsDir {
		// Keep root expanded so the tree isn't empty.
		if n.Depth > 0 {
			n.Expanded = false
		}
		for i := range n.Children {
			m.collapseAll(&n.Children[i])
		}
	}
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

	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
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
