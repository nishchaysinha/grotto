package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Test 7: "Quit" in the command palette actually returns tea.Quit.
func TestExecCommandQuit(t *testing.T) {
	m := New(Config{Path: "/home/nishchay/grotto", NoAI: true})
	cmd := m.execCommand("Quit")
	if cmd == nil {
		t.Fatal("execCommand(\"Quit\") returned nil, want tea.Quit cmd")
	}
	// Execute the cmd to get the message it produces.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("execCommand(\"Quit\") produced %T, want tea.QuitMsg", msg)
	}
}
