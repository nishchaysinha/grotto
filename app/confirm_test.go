package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// ConfirmDialog unit tests — the dialog itself, independent of the app.

func TestConfirmDialog_OpensAndCloses(t *testing.T) {
	var d ConfirmDialog
	if d.Active() {
		t.Fatal("dialog active before Open()")
	}
	d.Open(ConfirmReasonQuit, []string{"a.go"})
	if !d.Active() {
		t.Fatal("dialog not active after Open()")
	}
	if d.Reason() != ConfirmReasonQuit {
		t.Errorf("reason: got %v, want ConfirmReasonQuit", d.Reason())
	}
	d.Close()
	if d.Active() {
		t.Fatal("dialog still active after Close()")
	}
}

func TestConfirmDialog_KeyActions(t *testing.T) {
	cases := []struct {
		key  string
		want ConfirmAction
	}{
		{"s", ConfirmSave},
		{"S", ConfirmSave},
		{"enter", ConfirmSave}, // Enter defaults to Save
		{"d", ConfirmDiscard},
		{"D", ConfirmDiscard},
		{"c", ConfirmCancel},
		{"C", ConfirmCancel},
		{"esc", ConfirmCancel},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			var d ConfirmDialog
			d.Open(ConfirmReasonQuit, []string{"a.go"})
			consumed, action := d.Update(keyFromString(tc.key))
			if !consumed {
				t.Fatalf("key %q was not consumed", tc.key)
			}
			if action != tc.want {
				t.Errorf("key %q: got action %v, want %v", tc.key, action, tc.want)
			}
		})
	}
}

func TestConfirmDialog_IgnoresInputWhenInactive(t *testing.T) {
	var d ConfirmDialog
	consumed, action := d.Update(keyFromString("s"))
	if consumed {
		t.Error("inactive dialog consumed input")
	}
	if action != ConfirmPending {
		t.Errorf("inactive dialog returned action %v, want ConfirmPending", action)
	}
}

func TestConfirmDialog_OtherKeysAreSwallowed(t *testing.T) {
	// While the modal is up, unrelated keys must be consumed (to prevent them
	// from reaching the editor) but must not resolve the dialog.
	var d ConfirmDialog
	d.Open(ConfirmReasonQuit, []string{"a.go"})
	consumed, action := d.Update(keyFromString("x"))
	if !consumed {
		t.Error("random key should be consumed while modal is active")
	}
	if action != ConfirmPending {
		t.Errorf("random key resolved dialog with action %v", action)
	}
	if !d.Active() {
		t.Error("dialog closed itself on unrelated key")
	}
}

// ---- App integration tests ----

// Ctrl+Q on a clean editor quits immediately (no dialog).
func TestCtrlQ_CleanEditor_Quits(t *testing.T) {
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	updated, cmd := m.Update(keyFromString("ctrl+q"))
	if _, ok := updated.(Model); !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	if cmd == nil {
		t.Fatal("Ctrl+Q with no dirty buffers should return tea.Quit, got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("Ctrl+Q with clean editor: cmd produced non-QuitMsg")
	}
}

// Ctrl+Q with a dirty buffer opens the dialog instead of quitting.
func TestCtrlQ_DirtyBuffer_OpensDialog(t *testing.T) {
	tmp := mustWriteTemp(t, "hello\n")
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	if err := m.panes.OpenFile(tmp); err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	// Mark the buffer dirty by mutating it.
	m.panes.Buf().InsertChar('x')

	updated, cmd := m.Update(keyFromString("ctrl+q"))
	mm, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	if cmd != nil {
		// Dialog opened → no Cmd should be returned (certainly not tea.Quit).
		if msg := cmd(); msg != nil {
			if _, isQuit := msg.(tea.QuitMsg); isQuit {
				t.Fatal("Ctrl+Q with dirty buffer quit instead of showing dialog")
			}
		}
	}
	if !mm.confirm.Active() {
		t.Error("Ctrl+Q with dirty buffer did not activate confirm dialog")
	}
	if mm.confirm.Reason() != ConfirmReasonQuit {
		t.Errorf("dialog reason: got %v, want ConfirmReasonQuit", mm.confirm.Reason())
	}
}

// Pressing Discard in the dialog issues a Quit.
func TestConfirmDialog_DiscardQuits(t *testing.T) {
	tmp := mustWriteTemp(t, "hello\n")
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	_ = m.panes.OpenFile(tmp)
	m.panes.Buf().InsertChar('x')

	// Open dialog via Ctrl+Q
	updated, _ := m.Update(keyFromString("ctrl+q"))
	m = updated.(Model)

	// Press 'd' to discard.
	updated, cmd := m.Update(keyFromString("d"))
	m = updated.(Model)
	if m.confirm.Active() {
		t.Error("dialog still active after Discard")
	}
	if cmd == nil {
		t.Fatal("Discard should return tea.Quit cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("Discard: cmd did not produce QuitMsg")
	}

	// The buffer should still be dirty — discard does NOT save.
	if !m.panes.Buf().Dirty {
		t.Error("Discard saved the buffer (should have left it dirty)")
	}
}

// Pressing Save in the dialog persists the file and then issues Quit.
func TestConfirmDialog_SaveThenQuits(t *testing.T) {
	tmp := mustWriteTemp(t, "hello\n")
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	_ = m.panes.OpenFile(tmp)
	m.panes.Buf().InsertChar('x')

	updated, _ := m.Update(keyFromString("ctrl+q"))
	m = updated.(Model)

	updated, cmd := m.Update(keyFromString("s"))
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("Save should return tea.Quit cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("Save: cmd did not produce QuitMsg")
	}
	if m.panes.Buf().Dirty {
		t.Error("Save did not clear dirty flag on buffer")
	}
	// And the file on disk should contain the edit.
	data, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "xhello\n" {
		t.Errorf("file content: got %q, want %q", string(data), "xhello\n")
	}
}

// Pressing Cancel closes the dialog and does NOT quit.
func TestConfirmDialog_CancelDoesNotQuit(t *testing.T) {
	tmp := mustWriteTemp(t, "hello\n")
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	_ = m.panes.OpenFile(tmp)
	m.panes.Buf().InsertChar('x')

	updated, _ := m.Update(keyFromString("ctrl+q"))
	m = updated.(Model)

	updated, cmd := m.Update(keyFromString("esc"))
	m = updated.(Model)
	if m.confirm.Active() {
		t.Error("dialog still active after Cancel")
	}
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Error("Cancel should not quit")
		}
	}
	if !m.panes.Buf().Dirty {
		t.Error("Cancel should leave the buffer dirty")
	}
}

// Command palette "Quit" with dirty buffers opens dialog, does not quit.
func TestExecCommandQuit_DirtyOpensDialog(t *testing.T) {
	tmp := mustWriteTemp(t, "hello\n")
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	_ = m.panes.OpenFile(tmp)
	m.panes.Buf().InsertChar('x')

	cmd := m.execCommand("Quit")
	if cmd != nil {
		t.Errorf("execCommand(\"Quit\") with dirty buffer should return nil cmd, got %T", cmd())
	}
	if !m.confirm.Active() {
		t.Error("execCommand(\"Quit\") did not open the dialog")
	}
}

// ---- helpers ----

func mustWriteTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "grotto-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	return filepath.Clean(f.Name())
}

// keyFromString builds a KeyPressMsg for a logical key string like "ctrl+q",
// "enter", "s", or "esc". Single-character inputs become plain rune keys.
func keyFromString(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "ctrl+q":
		return tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl}
	}
	if len(s) == 1 {
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
	// Fallback: single rune of first char.
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s[:1]}
}
