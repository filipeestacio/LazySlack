package thread

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/slack"
)

func TestSetReplies(t *testing.T) {
	m := New(nil)
	m.SetReplies([]slack.Message{
		{Text: "parent", Timestamp: "1"},
		{Text: "reply1", Timestamp: "2"},
		{Text: "reply2", Timestamp: "3"},
	})
	if len(m.messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(m.messages))
	}
}

func TestNavigate(t *testing.T) {
	m := New(nil)
	m.SetReplies([]slack.Message{
		{Text: "parent", Timestamp: "1"},
		{Text: "reply", Timestamp: "2"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
}

func TestEscCloses(t *testing.T) {
	m := New(nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected close command on Esc")
	}
}
