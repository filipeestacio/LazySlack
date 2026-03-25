package input

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTypeAndSend(t *testing.T) {
	m := New("C1", "", "#general")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	if m.Value() != "hi" {
		t.Errorf("Value() = %q, want %q", m.Value(), "hi")
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected send command on Enter")
	}
}

func TestEscDismisses(t *testing.T) {
	m := New("C1", "", "#general")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected dismiss command on Esc")
	}
}

func TestEmptyEnterDoesNotSend(t *testing.T) {
	m := New("C1", "", "#general")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command on empty Enter")
	}
}
