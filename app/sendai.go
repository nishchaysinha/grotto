package app

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/owomeister/grotto/editor"
)

// formatSendToAI builds the text blob injected into the AI panel's stdin.
// Structure:
//
//	<optional "# file.go:L10-L25" header>
//	```<lang>
//	<text>
//	```
//	<trailing newline so the AI REPL consumes the paste as a submitted message>
//
// It does NOT add a trailing "\r" on its own — the caller decides whether to
// submit or leave the blob waiting in the prompt.
func formatSendToAI(s editor.SelectionSnapshot) string {
	var b strings.Builder

	// Header with file + line range when we know the path.
	if s.FilePath != "" {
		name := filepath.Base(s.FilePath)
		switch {
		case s.HasSelection:
			fmt.Fprintf(&b, "# %s:L%d-L%d\n", name, s.StartLine, s.EndLine)
		default:
			fmt.Fprintf(&b, "# %s (full file, %d lines)\n", name, s.EndLine)
		}
	}

	lang := langFor(s.FilePath)
	fmt.Fprintf(&b, "```%s\n%s\n```\n", lang, s.Text)
	return b.String()
}

// langFor returns a simple markdown fence hint for the given file extension.
// Falls back to empty string when unknown — the AI CLI renders it as a plain
// fenced block.
func langFor(path string) string {
	if path == "" {
		return ""
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".java":
		return "java"
	case ".kt":
		return "kotlin"
	case ".swift":
		return "swift"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".hpp":
		return "cpp"
	case ".sh", ".bash", ".zsh":
		return "bash"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".sql":
		return "sql"
	}
	return ""
}

// sendSelectionToAI pipes the active pane's selection (or the whole buffer
// when nothing is selected) into the AI panel's stdin, opening the panel
// first if it isn't visible. Returns a tea.Cmd for any side effects the
// caller should run (e.g. launching the AI provider on first open).
func (m *Model) sendSelectionToAI() tea.Cmd {
	snap, ok := m.panes.Snapshot()
	if !ok {
		return nil
	}

	var cmds []tea.Cmd

	// Open the AI panel if hidden. Reuse toggleAI so provider selection and
	// tab spawn logic stay in one place; it also focuses the panel.
	if !m.aiPanelVisible {
		if c := m.toggleAI(); c != nil {
			cmds = append(cmds, c)
		}
	} else {
		m.focus = FocusAI
		m.updateFocus()
	}

	// If no AI CLI is running yet the send would be silently dropped, so
	// defer the write until the tab exists. toggleAI's returned Cmd spawns
	// the PTY asynchronously; we use a one-shot tea.Msg to re-enter Update
	// once the spawn is in-flight.
	blob := formatSendToAI(snap)
	cmds = append(cmds, func() tea.Msg { return sendAIPayloadMsg{payload: blob} })

	return tea.Batch(cmds...)
}

// sendAIPayloadMsg carries the pre-formatted text into the Update loop so
// we can write to the AI panel's PTY after any pending spawn has started.
// `attempts` increments each time we re-queue the message while waiting for
// the PTY to come up; the handler gives up after a bounded number of retries.
type sendAIPayloadMsg struct {
	payload  string
	attempts int
}
