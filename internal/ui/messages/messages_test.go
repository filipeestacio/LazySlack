package messages

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/slack"
)

func TestSetMessages(t *testing.T) {
	m := New(nil, nil)
	m.SetMessages([]slack.Message{
		{Text: "hello", Timestamp: "1706000001.000000", UserID: "U1"},
		{Text: "world", Timestamp: "1706000002.000000", UserID: "U2"},
	})
	if len(m.messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(m.messages))
	}
}

func TestNavigateMessages(t *testing.T) {
	m := New(nil, nil)
	m.SetMessages([]slack.Message{
		{Text: "third", Timestamp: "1706000003.000000"},
		{Text: "second", Timestamp: "1706000002.000000"},
		{Text: "first", Timestamp: "1706000001.000000"},
	})

	if m.cursor != 2 {
		t.Errorf("initial cursor = %d, want 2 (newest message)", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.cursor != 1 {
		t.Errorf("cursor after k = %d, want 1", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 2 {
		t.Errorf("cursor after j = %d, want 2", m.cursor)
	}
}

func TestOpenThread(t *testing.T) {
	m := New(nil, nil)
	m.SetMessages([]slack.Message{
		{Text: "has thread", Timestamp: "1706000001.000000", ThreadTS: "1706000001.000000", ReplyCount: 3},
	})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Fatal("expected command for thread open")
	}
}

func TestJumpToBottom(t *testing.T) {
	m := New(nil, nil)
	m.SetMessages([]slack.Message{
		{Text: "a", Timestamp: "1"},
		{Text: "b", Timestamp: "2"},
		{Text: "c", Timestamp: "3"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2", m.cursor)
	}
}
