package input

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type SendMsg struct {
	ChannelID string
	ThreadTS  string
	Text      string
}

type DismissMsg struct{}

type Model struct {
	textarea    textarea.Model
	channelID   string
	threadTS    string
	channelName string
}

func New(channelID, threadTS, channelName string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetKeys("shift+enter", "alt+enter")

	return Model{
		textarea:    ta,
		channelID:   channelID,
		threadTS:    threadTS,
		channelName: channelName,
	}
}

func (m Model) Value() string {
	return m.textarea.Value()
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return m, func() tea.Msg { return DismissMsg{} }
		case tea.KeyEnter:
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				return m, nil
			}
			return m, func() tea.Msg {
				return SendMsg{
					ChannelID: m.channelID,
					ThreadTS:  m.threadTS,
					Text:      text,
				}
			}
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	header := styles.SidebarSection.Render("Compose → " + m.channelName)
	return styles.OverlayStyle.Render(
		header + "\n" + m.textarea.View(),
	)
}
