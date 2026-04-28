package editor

import "testing"

// Test 1: Backspace on multi-byte UTF-8 rune deletes the whole rune, not just one byte.
func TestBackspaceUTF8(t *testing.T) {
	b := NewBufferFromString("héllo")
	// Move cursor to after é (byte col 3: h=1, é=2 bytes)
	b.Cursor.Col = 3
	b.Backspace()
	if b.Lines[0] != "hllo" {
		t.Errorf("Backspace on é: got %q, want %q", b.Lines[0], "hllo")
	}
	if b.Cursor.Col != 1 {
		t.Errorf("Cursor col after Backspace: got %d, want 1", b.Cursor.Col)
	}
}

// Test 2: CursorLeft on multi-byte rune steps back a full rune.
func TestCursorLeftUTF8(t *testing.T) {
	b := NewBufferFromString("héllo")
	b.Cursor.Col = 3 // after é
	b.CursorLeft()
	if b.Cursor.Col != 1 {
		t.Errorf("CursorLeft on é: col got %d, want 1", b.Cursor.Col)
	}
}

// Test 3: CursorRight on multi-byte rune steps forward a full rune.
func TestCursorRightUTF8(t *testing.T) {
	b := NewBufferFromString("héllo")
	b.Cursor.Col = 1 // on é
	b.CursorRight()
	if b.Cursor.Col != 3 {
		t.Errorf("CursorRight on é: col got %d, want 3", b.Cursor.Col)
	}
}

// Test 4: CursorWordLeft with multi-byte chars doesn't panic or split rune.
func TestCursorWordLeftUTF8(t *testing.T) {
	b := NewBufferFromString("héllo wörld")
	b.Cursor.Col = len("héllo wörld")
	b.CursorWordLeft() // should land at start of wörld
	// "héllo " is 8 bytes (h=1, é=2, l=1, l=1, o=1, space=1 → 7), wait: h(1)+é(2)+l+l+o+space = 7
	want := len("héllo ")
	if b.Cursor.Col != want {
		t.Errorf("CursorWordLeft: col got %d, want %d", b.Cursor.Col, want)
	}
}

// Test 5: indentSelection bumps EditVersion so BufferChangedMsg fires.
func TestIndentSelectionBumpsVersion(t *testing.T) {
	m := NewModel()
	_ = m.OpenFile("")
	// The model has an empty buffer. Set some content directly.
	m.tabs[0].buf.Lines = []string{"    hello", "    world"}
	m.tabs[0].buf.Selection = Selection{
		Anchor: Position{Line: 0, Col: 0},
		Head:   Position{Line: 1, Col: 0},
		Active: true,
	}
	before := m.tabs[0].buf.EditVersion
	m.indentSelection(-1) // dedent
	after := m.tabs[0].buf.EditVersion
	if after == before {
		t.Errorf("indentSelection did not bump EditVersion: before=%d after=%d", before, after)
	}
	if m.tabs[0].buf.Lines[0] != "hello" {
		t.Errorf("dedent line 0: got %q, want %q", m.tabs[0].buf.Lines[0], "hello")
	}
}
