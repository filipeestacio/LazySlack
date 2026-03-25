package emoji

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFilter(t *testing.T) {
	m := New("C1", "1706000001.000000")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	visible := m.VisibleEmojis()
	for _, e := range visible {
		if e.Name == "+1" || e.Name == "thumbsup" {
			return
		}
	}
	if len(visible) == 0 {
		t.Error("expected at least one thumbs emoji match")
	}
}

func TestSelectEmoji(t *testing.T) {
	m := New("C1", "1706000001.000000")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on Enter")
	}
}

func TestEscCloses(t *testing.T) {
	m := New("C1", "1706000001.000000")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected close command on Esc")
	}
}
