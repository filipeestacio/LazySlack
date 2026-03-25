package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/slack"
)

func TestFocusSwitching(t *testing.T) {
	m := newTestApp()

	if m.focus != focusSidebar {
		t.Errorf("initial focus = %d, want sidebar", m.focus)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.focus != focusMessages {
		t.Errorf("after 'l', focus = %d, want messages", m.focus)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.focus != focusSidebar {
		t.Errorf("after 'h', focus = %d, want sidebar", m.focus)
	}
}

func TestQuit(t *testing.T) {
	m := newTestApp()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestHelpToggle(t *testing.T) {
	m := newTestApp()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !m.showHelp {
		t.Error("expected help to be shown")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.showHelp {
		t.Error("expected help to be hidden")
	}
}

func TestNewMessagesUpdateView(t *testing.T) {
	m := newTestApp()
	m.messages.SetChannel("C1", "#test")

	newMsgs := []slack.Message{
		{Text: "new msg", Timestamp: "1706000010.000000", UserID: "U1"},
	}

	m, _ = m.Update(newMessagesMsg{messages: newMsgs})
	if sel := m.messages.SelectedMessage(); sel == nil || sel.Text != "new msg" {
		t.Error("expected new message to appear in messages view")
	}
}

func newTestApp() Model {
	return New(nil, "testcorp")
}
