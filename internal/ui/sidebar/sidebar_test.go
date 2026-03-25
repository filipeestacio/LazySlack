package sidebar

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/slack"
)

func TestNavigateDown(t *testing.T) {
	m := New()
	m.SetChannels([]slack.Channel{
		{ID: "C1", Name: "general"},
		{ID: "C2", Name: "random"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.CursorIndex() != 1 {
		t.Errorf("cursor = %d, want 1", m.CursorIndex())
	}
}

func TestNavigateUp(t *testing.T) {
	m := New()
	m.SetChannels([]slack.Channel{
		{ID: "C1", Name: "general"},
		{ID: "C2", Name: "random"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.CursorIndex() != 0 {
		t.Errorf("cursor = %d, want 0", m.CursorIndex())
	}
}

func TestSelectChannel(t *testing.T) {
	m := New()
	m.SetChannels([]slack.Channel{
		{ID: "C1", Name: "general"},
		{ID: "C2", Name: "random"},
	})

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on Enter")
	}
}

func TestFilter(t *testing.T) {
	m := New()
	m.SetChannels([]slack.Channel{
		{ID: "C1", Name: "general"},
		{ID: "C2", Name: "random"},
		{ID: "C3", Name: "engineering"},
	})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	visible := m.VisibleItems()
	if len(visible) != 2 {
		t.Errorf("expected 2 matches for 'gen', got %d", len(visible))
	}
}
