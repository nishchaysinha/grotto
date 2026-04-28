package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshot_NoBuffer(t *testing.T) {
	pm := NewPaneManager()
	if _, ok := pm.Snapshot(); ok {
		t.Error("Snapshot on empty pane returned ok=true")
	}
}

func TestSnapshot_WholeFileWhenNoSelection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.go")
	content := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	pm := NewPaneManager()
	if err := pm.OpenFile(path); err != nil {
		t.Fatalf("OpenFile: %v", err)
	}

	snap, ok := pm.Snapshot()
	if !ok {
		t.Fatal("Snapshot returned ok=false")
	}
	if snap.HasSelection {
		t.Error("HasSelection should be false when nothing is selected")
	}
	if snap.FilePath != path {
		t.Errorf("FilePath: got %q, want %q", snap.FilePath, path)
	}
	// NewBuffer strips the trailing newline; Snapshot joins lines back with \n.
	wantText := "package main\n\nfunc main() {}"
	if snap.Text != wantText {
		t.Errorf("Text: got %q, want %q", snap.Text, wantText)
	}
	if snap.StartLine != 1 {
		t.Errorf("StartLine: got %d, want 1", snap.StartLine)
	}
	if snap.EndLine != 3 {
		t.Errorf("EndLine: got %d, want 3", snap.EndLine)
	}
}

func TestSnapshot_ActiveSelection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.go")
	content := "line 1\nline 2\nline 3\nline 4\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	pm := NewPaneManager()
	if err := pm.OpenFile(path); err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	b := pm.Buf()

	// Select lines 2-3 inclusive.
	b.Selection = Selection{
		Anchor: Position{Line: 1, Col: 0},
		Head:   Position{Line: 2, Col: 6},
		Active: true,
	}

	snap, ok := pm.Snapshot()
	if !ok {
		t.Fatal("Snapshot returned ok=false")
	}
	if !snap.HasSelection {
		t.Error("HasSelection should be true")
	}
	if snap.Text != "line 2\nline 3" {
		t.Errorf("Text: got %q, want %q", snap.Text, "line 2\nline 3")
	}
	if snap.StartLine != 2 || snap.EndLine != 3 {
		t.Errorf("lines: got %d-%d, want 2-3", snap.StartLine, snap.EndLine)
	}
}

// An "active" selection where anchor==head (e.g. after a click) should fall
// back to whole-file mode rather than returning empty text.
func TestSnapshot_EmptyActiveSelectionFallsBackToWholeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.go")
	if err := os.WriteFile(path, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	pm := NewPaneManager()
	_ = pm.OpenFile(path)
	b := pm.Buf()
	b.Selection = Selection{
		Anchor: Position{Line: 0, Col: 2},
		Head:   Position{Line: 0, Col: 2},
		Active: true,
	}

	snap, ok := pm.Snapshot()
	if !ok {
		t.Fatal("Snapshot returned ok=false")
	}
	if snap.HasSelection {
		t.Error("zero-width selection should be treated as no selection")
	}
	if snap.Text != "hello" {
		t.Errorf("Text: got %q, want %q", snap.Text, "hello")
	}
}
