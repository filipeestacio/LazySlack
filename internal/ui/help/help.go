package help

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type CloseMsg struct{}

type Model struct {
	width  int
	height int
}

func New() Model { return Model{} }

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.Type == tea.KeyEsc || msg.String() == "?" || msg.String() == "q" {
			return m, func() tea.Msg { return CloseMsg{} }
		}
	}
	return m, nil
}

func (m Model) View() string {
	bindings := []struct{ key, action string }{
		{"j/k", "Navigate up/down"},
		{"h/l", "Focus sidebar/messages"},
		{"Enter", "Select channel"},
		{"c", "Compose message"},
		{"t", "Open thread"},
		{"r", "React with emoji"},
		{"y", "Copy message"},
		{"/", "Filter channels"},
		{"g/G", "Jump to top/bottom"},
		{"Tab", "Toggle sidebar section"},
		{"Esc", "Close overlay"},
		{"?", "Toggle this help"},
		{"q", "Quit"},
	}

	var b strings.Builder
	b.WriteString(styles.SidebarSection.Render("Keybindings"))
	b.WriteString("\n\n")

	for _, bind := range bindings {
		b.WriteString(styles.StatusBarKey.Render(" "+bind.key+" "))
		b.WriteString("  ")
		b.WriteString(bind.action)
		b.WriteString("\n")
	}

	return styles.OverlayStyle.Width(m.width).Render(b.String())
}
