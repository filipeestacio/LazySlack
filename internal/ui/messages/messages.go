package messages

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	offset      int
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
	m.offset = 0
}

func (m *Model) AppendMessages(msgs []slack.Message) {
	m.messages = append(msgs, m.messages...)
}

func (m *Model) SetChannel(id, name string) {
	m.channelID = id
	m.channelName = name
	m.messages = nil
	m.cursor = 0
	m.offset = 0
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
			m.scrollToCursor()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.scrollToCursor()
		} else {
			return m, func() tea.Msg { return RequestPaginationMsg{} }
		}
	case "g":
		m.cursor = 0
		m.offset = 0
	case "G":
		if len(m.messages) > 0 {
			m.cursor = len(m.messages) - 1
			m.scrollToCursor()
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

func (m *Model) scrollToCursor() {
	if m.cursor < m.offset {
		m.offset = m.cursor
		return
	}

	viewH := m.height - 2
	if viewH < 1 {
		viewH = 20
	}

	for m.offset < m.cursor {
		lines := 0
		for i := m.offset; i <= m.cursor && i < len(m.messages); i++ {
			lines += m.messageLineCount(m.messages[i])
		}
		if lines <= viewH {
			break
		}
		m.offset++
	}
}

func (m Model) messageLineCount(msg slack.Message) int {
	text := msg.Text
	if m.renderer != nil {
		text = m.renderer.RenderPlain(text)
	}

	contentW := m.width - 4
	if contentW < 20 {
		contentW = 20
	}

	lines := 1
	for _, line := range strings.Split(text, "\n") {
		if len(line) == 0 {
			lines++
		} else {
			lines += (len(line) + contentW - 1) / contentW
		}
	}

	if msg.ReplyCount > 0 || len(msg.Reactions) > 0 {
		lines++
	}

	return lines
}

func (m Model) View() string {
	if m.channelID == "" {
		return styles.MessagesStyle.Render("Select a channel")
	}

	viewH := m.height - 2
	if viewH < 1 {
		viewH = 20
	}
	contentW := m.width - 2
	if contentW < 20 {
		contentW = 20
	}

	var b strings.Builder
	header := styles.SidebarSection.Copy().MarginTop(0).Render("#" + m.channelName)
	b.WriteString(header + "\n")

	lines := 0
	for i := m.offset; i < len(m.messages) && lines < viewH; i++ {
		rendered := m.renderMessage(m.messages[i], i == m.cursor, contentW)
		msgLines := strings.Count(rendered, "\n") + 1
		if lines+msgLines > viewH {
			break
		}
		b.WriteString(rendered)
		b.WriteString("\n")
		lines += msgLines
	}

	return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(
		strings.TrimRight(b.String(), "\n"))
}

func (m Model) renderMessage(msg slack.Message, selected bool, maxWidth int) string {
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
	textStyle := styles.MessageText.MaxWidth(maxWidth)

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
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("63")).
			MaxWidth(maxWidth).
			Render(line)
	}
	return lipgloss.NewStyle().PaddingLeft(2).MaxWidth(maxWidth).Render(line)
}
