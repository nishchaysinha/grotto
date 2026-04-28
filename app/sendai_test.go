package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/owomeister/grotto/editor"
)

func TestFormatSendToAI_SelectionWithGoPath(t *testing.T) {
	snap := editor.SelectionSnapshot{
		FilePath:     "/home/me/proj/foo.go",
		Text:         "func bar() int { return 1 }",
		StartLine:    10,
		EndLine:      10,
		HasSelection: true,
	}
	out := formatSendToAI(snap)

	// Header with basename and line range.
	if !strings.Contains(out, "# foo.go:L10-L10") {
		t.Errorf("missing L10-L10 header, got:\n%s", out)
	}
	// Fenced block with go lang tag.
	if !strings.Contains(out, "```go\n") {
		t.Errorf("missing ```go fence, got:\n%s", out)
	}
	if !strings.Contains(out, "func bar() int { return 1 }") {
		t.Errorf("missing selection body, got:\n%s", out)
	}
	if !strings.Contains(out, "\n```\n") {
		t.Errorf("missing closing fence, got:\n%s", out)
	}
}

func TestFormatSendToAI_WholeFileHeader(t *testing.T) {
	snap := editor.SelectionSnapshot{
		FilePath:     "/tmp/main.py",
		Text:         "print('hi')",
		StartLine:    1,
		EndLine:      1,
		HasSelection: false,
	}
	out := formatSendToAI(snap)
	if !strings.Contains(out, "# main.py (full file, 1 lines)") {
		t.Errorf("expected full-file header, got:\n%s", out)
	}
	if !strings.Contains(out, "```python\n") {
		t.Errorf("expected python fence, got:\n%s", out)
	}
}

// Untitled buffers (no FilePath) skip the header and pick an empty lang tag.
func TestFormatSendToAI_UntitledBuffer(t *testing.T) {
	snap := editor.SelectionSnapshot{
		Text:         "hello",
		StartLine:    1,
		EndLine:      1,
		HasSelection: true,
	}
	out := formatSendToAI(snap)
	if strings.HasPrefix(out, "# ") {
		t.Errorf("untitled buffer should not emit a header, got:\n%s", out)
	}
	// Opening fence should have no language.
	if !strings.HasPrefix(out, "```\n") {
		t.Errorf("untitled buffer should use bare ``` fence, got:\n%s", out)
	}
}

func TestLangFor(t *testing.T) {
	cases := map[string]string{
		"/x/foo.go":         "go",
		"/x/foo.ts":         "typescript",
		"/x/foo.tsx":        "typescript",
		"/x/foo.py":         "python",
		"/x/foo.rs":         "rust",
		"/x/Cargo.toml":     "toml",
		"/x/README.md":      "markdown",
		"/x/script.sh":      "bash",
		"/x/no-extension":   "",
		"/x/foo.unknownext": "",
		"":                  "",
	}
	for path, want := range cases {
		if got := langFor(path); got != want {
			t.Errorf("langFor(%q) = %q, want %q", path, got, want)
		}
	}
}

// Command palette lists the new entry.
func TestCommandNames_IncludesSendToAI(t *testing.T) {
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	found := false
	for _, n := range m.commandNames() {
		if n == "Send Selection to AI" {
			found = true
			break
		}
	}
	if !found {
		t.Error("command palette missing \"Send Selection to AI\"")
	}
}

// sendSelectionToAI returns nil when there's no buffer to snapshot.
func TestSendSelectionToAI_NoBufferReturnsNil(t *testing.T) {
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	cmd := m.sendSelectionToAI()
	if cmd != nil {
		t.Errorf("expected nil cmd when no file open, got %T", cmd())
	}
}

// Once a buffer is open, sendSelectionToAI should return a non-nil cmd.
// We don't attempt to walk the batched Cmds end-to-end (tea's BatchMsg
// shape is an implementation detail) — the payload construction itself is
// covered separately by TestFormatSendToAI_*.
func TestSendSelectionToAI_ReturnsCmdWhenBufferOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.go")
	if err := os.WriteFile(path, []byte("package demo\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	if err := m.panes.OpenFile(path); err != nil {
		t.Fatalf("OpenFile: %v", err)
	}

	cmd := m.sendSelectionToAI()
	if cmd == nil {
		t.Fatal("sendSelectionToAI returned nil cmd when a buffer is open")
	}
}

// The full pipeline: a buffer-backed Snapshot + formatSendToAI must contain
// both the file header and the content.
func TestFormatSendToAI_EndToEndFromBuffer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.go")
	if err := os.WriteFile(path, []byte("package demo\n\nfunc X() {}\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	if err := m.panes.OpenFile(path); err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	snap, ok := m.panes.Snapshot()
	if !ok {
		t.Fatal("Snapshot returned ok=false")
	}

	blob := formatSendToAI(snap)
	if !strings.Contains(blob, "# demo.go") {
		t.Errorf("missing file header in blob:\n%s", blob)
	}
	if !strings.Contains(blob, "package demo") {
		t.Errorf("missing buffer content in blob:\n%s", blob)
	}
	if !strings.Contains(blob, "```go\n") {
		t.Errorf("missing go fence in blob:\n%s", blob)
	}
}
