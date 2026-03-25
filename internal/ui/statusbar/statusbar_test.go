package statusbar

import "testing"

func TestViewShowsWorkspace(t *testing.T) {
	m := New("testcorp")
	view := m.View(80)
	if len(view) == 0 {
		t.Fatal("expected non-empty view")
	}
}

func TestSetConnected(t *testing.T) {
	m := New("testcorp")
	m.SetConnected(false)
	if m.connected {
		t.Error("expected disconnected")
	}
	m.SetConnected(true)
	if !m.connected {
		t.Error("expected connected")
	}
}
