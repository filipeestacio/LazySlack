package thread

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/slack"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type CloseMsg struct{}

type OpenThreadComposeMsg struct {
	ChannelID string
	ThreadTS  string
}

type Model struct {
	messages  []slack.Message
	cursor    int
	channelID string
	threadTS  string
	width     int
	height    int
	renderer  *slack.Renderer
}

func New(renderer *slack.Renderer) Model {
	return Model{renderer: renderer}
}

func (m *Model) SetThread(channelID, threadTS string) {
	m.channelID = channelID
	m.threadTS = threadTS
	m.messages = nil
	m.cursor = 0
}

func (m *Model) SetReplies(msgs []slack.Message) {
	m.messages = msgs
	m.cursor = 0
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) ChannelID() string { return m.channelID }
func (m Model) ThreadTS() string  { return m.threadTS }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.messages)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g":
			m.cursor = 0
		case "G":
			if len(m.messages) > 0 {
				m.cursor = len(m.messages) - 1
			}
		case "esc":
			return m, func() tea.Msg { return CloseMsg{} }
		case "c":
			id := m.channelID
			ts := m.threadTS
			return m, func() tea.Msg {
				return OpenThreadComposeMsg{ChannelID: id, ThreadTS: ts}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	header := styles.SidebarSection.Render("Thread")
	b.WriteString(header + "\n\n")

	for i, msg := range m.messages {
		b.WriteString(m.renderMessage(msg, i == m.cursor))
		b.WriteString("\n")
	}

	return styles.ThreadBorder.Render(b.String())
}

func (m Model) renderMessage(msg slack.Message, selected bool) string {
	text := msg.Text
	if m.renderer != nil {
		text = m.renderer.RenderPlain(text)
	}

	username := msg.Username
	if username == "" {
		username = msg.UserID
	}

	userStyle := styles.MessageUsername.Foreground(styles.ColorForUser(msg.UserID))
	tsStyle := styles.MessageTimestamp
	textStyle := styles.MessageText

	line := userStyle.Render(username) + " " +
		tsStyle.Render(msg.Time().Format("15:04")) + "\n" +
		textStyle.Render(text)

	for _, r := range msg.Reactions {
		line += " " + styles.MessageReaction.Render(fmt.Sprintf(":%s: %d", r.Name, r.Count))
	}

	if selected {
		return styles.SidebarSelected.Render(line)
	}
	return line
}
