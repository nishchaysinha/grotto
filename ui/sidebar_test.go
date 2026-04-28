package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Test 6: Sidebar mouse wheel scroll uses msg.Button, not msg.Y.
// Before the fix, msg.Y < 0 was the condition — but Y is a screen coordinate
// so it's always >= 0 and scroll never worked.
func TestSidebarMouseWheelScroll(t *testing.T) {
	m := New("/home/nishchay/grotto")
	// Force a long enough tree and a non-zero height so scrolling is possible.
	m.height = 3 // small window so even a few items overflow

	// Scroll down
	before := m.scroll
	m, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.scroll <= before && len(m.flat) > m.height {
		t.Errorf("scroll down: scroll stayed at %d (was %d), tree has %d items, height %d",
			m.scroll, before, len(m.flat), m.height)
	}

	// Scroll back up
	m, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if m.scroll != before {
		t.Errorf("scroll up: scroll got %d, want %d", m.scroll, before)
	}
}
