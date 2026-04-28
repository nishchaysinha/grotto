package editor

import (
	"os"
	"path/filepath"
	"testing"
)

// DirtyBuffers returns only dirty buffers, de-duplicated across panes.
func TestPaneManager_DirtyBuffers(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "alpha\n")
	b := writeFile(t, dir, "b.txt", "bravo\n")

	pm := NewPaneManager()
	// Open two files in pane 0.
	mustOpen(t, &pm, a)
	mustOpen(t, &pm, b)

	if got := pm.DirtyBuffers(); len(got) != 0 {
		t.Errorf("fresh open: DirtyBuffers = %d, want 0", len(got))
	}

	// Mutate only `a`.
	findBuf(t, &pm, a).InsertChar('x')

	dirty := pm.DirtyBuffers()
	if len(dirty) != 1 {
		t.Fatalf("after mutating a: DirtyBuffers = %d, want 1", len(dirty))
	}
	if dirty[0].FilePath != a {
		t.Errorf("dirty buffer path: got %q, want %q", dirty[0].FilePath, a)
	}

	// Split — shared buffer for `b` appears in two panes. Mutate from either;
	// it must appear only once in DirtyBuffers.
	pm.Split(SplitRight)
	findBuf(t, &pm, b).InsertChar('y')

	dirty = pm.DirtyBuffers()
	if len(dirty) != 2 {
		t.Fatalf("after mutating b in split: DirtyBuffers = %d, want 2", len(dirty))
	}
	paths := map[string]bool{dirty[0].FilePath: true, dirty[1].FilePath: true}
	if !paths[a] || !paths[b] {
		t.Errorf("dirty paths: got %v, want both a.txt and b.txt", paths)
	}
}

// SaveAllDirty writes every dirty buffer to disk and clears the dirty flag.
func TestPaneManager_SaveAllDirty(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "alpha\n")
	b := writeFile(t, dir, "b.txt", "bravo\n")

	pm := NewPaneManager()
	mustOpen(t, &pm, a)
	mustOpen(t, &pm, b)
	findBuf(t, &pm, a).InsertChar('X')
	findBuf(t, &pm, b).InsertChar('Y')

	if err := pm.SaveAllDirty(); err != nil {
		t.Fatalf("SaveAllDirty: %v", err)
	}

	if got := pm.DirtyBuffers(); len(got) != 0 {
		t.Errorf("after SaveAllDirty: DirtyBuffers = %d, want 0", len(got))
	}

	// Verify on disk.
	if data, _ := os.ReadFile(a); string(data) != "Xalpha\n" {
		t.Errorf("file a: got %q, want %q", string(data), "Xalpha\n")
	}
	if data, _ := os.ReadFile(b); string(data) != "Ybravo\n" {
		t.Errorf("file b: got %q, want %q", string(data), "Ybravo\n")
	}
}

// ---- helpers ----

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return p
}

func mustOpen(t *testing.T, pm *PaneManager, path string) {
	t.Helper()
	if err := pm.OpenFile(path); err != nil {
		t.Fatalf("OpenFile(%s): %v", path, err)
	}
}

func findBuf(t *testing.T, pm *PaneManager, path string) *Buffer {
	t.Helper()
	for i := range pm.panes {
		for _, tab := range pm.panes[i].tabs {
			if tab.buf != nil && tab.buf.FilePath == path {
				return tab.buf
			}
		}
	}
	t.Fatalf("no buffer for path %q", path)
	return nil
}
