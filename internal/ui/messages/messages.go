package messages

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/slack"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type OpenThreadMsg struct {
	ChannelID string
	ThreadTS  string
	Message   slack.Message
}

type OpenComposeMsg struct {
	ChannelID   string
	ChannelName string
}

type RequestPaginationMsg struct{}

type CopyMsg struct{ Text string }

type Model struct {
	messages    []slack.Message
	cursor      int
	channelID   string
	channelName string
	width       int
	height      int
	renderer    *slack.Renderer
}

func New(renderer *slack.Renderer) Model {
	return Model{renderer: renderer}
}

func (m *Model) SetMessages(msgs []slack.Message) {
	m.messages = msgs
	m.cursor = 0
}

func (m *Model) AppendMessages(msgs []slack.Message) {
	m.messages = append(msgs, m.messages...)
}

func (m *Model) SetChannel(id, name string) {
	m.channelID = id
	m.channelName = name
	m.messages = nil
	m.cursor = 0
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m Model) SelectedMessage() *slack.Message {
	if len(m.messages) == 0 || m.cursor >= len(m.messages) {
		return nil
	}
	msg := m.messages[m.cursor]
	return &msg
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.messages)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		} else {
			return m, func() tea.Msg { return RequestPaginationMsg{} }
		}
	case "g":
		m.cursor = 0
	case "G":
		if len(m.messages) > 0 {
			m.cursor = len(m.messages) - 1
		}
	case "t":
		sel := m.SelectedMessage()
		if sel != nil && sel.ThreadTS != "" {
			ts := sel.ThreadTS
			id := m.channelID
			s := *sel
			return m, func() tea.Msg {
				return OpenThreadMsg{ChannelID: id, ThreadTS: ts, Message: s}
			}
		}
	case "c":
		id := m.channelID
		name := m.channelName
		return m, func() tea.Msg {
			return OpenComposeMsg{ChannelID: id, ChannelName: name}
		}
	case "y":
		if sel := m.SelectedMessage(); sel != nil {
			text := sel.Text
			return m, func() tea.Msg { return CopyMsg{Text: text} }
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.channelID == "" {
		return styles.MessagesStyle.Render("Select a channel")
	}

	var b strings.Builder
	header := styles.SidebarSection.Render("#" + m.channelName)
	b.WriteString(header + "\n\n")

	for i, msg := range m.messages {
		b.WriteString(m.renderMessage(msg, i == m.cursor))
		b.WriteString("\n")
	}

	return styles.MessagesStyle.Render(b.String())
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

	if msg.ReplyCount > 0 {
		line += "\n" + styles.MessageReaction.Render(fmt.Sprintf("💬 %d replies", msg.ReplyCount))
	}

	for _, r := range msg.Reactions {
		line += " " + styles.MessageReaction.Render(fmt.Sprintf(":%s: %d", r.Name, r.Count))
	}

	if selected {
		return styles.SidebarSelected.Render(line)
	}
	return line
}
