package emoji

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type SelectMsg struct {
	ChannelID string
	MessageTS string
	Emoji     string
}

type CloseMsg struct{}

type EmojiEntry struct {
	Name   string
	Glyph  string
}

type Model struct {
	channelID string
	messageTS string
	query     string
	cursor    int
	visible   []EmojiEntry
}

func New(channelID, messageTS string) Model {
	m := Model{
		channelID: channelID,
		messageTS: messageTS,
	}
	m.visible = commonEmojis()
	return m
}

func (m Model) VisibleEmojis() []EmojiEntry {
	return m.visible
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return m, func() tea.Msg { return CloseMsg{} }
		case tea.KeyEnter:
			if len(m.visible) > 0 {
				e := m.visible[m.cursor]
				id := m.channelID
				ts := m.messageTS
				name := e.Name
				return m, func() tea.Msg {
					return SelectMsg{ChannelID: id, MessageTS: ts, Emoji: name}
				}
			}
			return m, func() tea.Msg { return CloseMsg{} }
		case tea.KeyBackspace:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.applyFilter()
			}
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.visible)-1 {
				m.cursor++
			}
		case tea.KeyRunes:
			m.query += string(msg.Runes)
			m.applyFilter()
		}
	}
	return m, nil
}

func (m *Model) applyFilter() {
	m.cursor = 0
	if m.query == "" {
		m.visible = commonEmojis()
		return
	}
	q := strings.ToLower(m.query)
	all := commonEmojis()
	m.visible = nil
	for _, e := range all {
		if strings.Contains(e.Name, q) {
			m.visible = append(m.visible, e)
		}
	}
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(styles.SidebarSection.Render("Emoji: " + m.query + "█"))
	b.WriteString("\n")

	for i, e := range m.visible {
		line := e.Glyph + " " + e.Name
		if i == m.cursor {
			b.WriteString(styles.SidebarSelected.Render(line))
		} else {
			b.WriteString(styles.SidebarNormal.Render(line))
		}
		b.WriteString("\n")
	}

	return styles.OverlayStyle.Render(b.String())
}

func commonEmojis() []EmojiEntry {
	return []EmojiEntry{
		{Name: "+1", Glyph: "👍"},
		{Name: "-1", Glyph: "👎"},
		{Name: "thumbsup", Glyph: "👍"},
		{Name: "thumbsdown", Glyph: "👎"},
		{Name: "heart", Glyph: "❤️"},
		{Name: "tada", Glyph: "🎉"},
		{Name: "laugh", Glyph: "😄"},
		{Name: "joy", Glyph: "😂"},
		{Name: "smile", Glyph: "😊"},
		{Name: "grinning", Glyph: "😀"},
		{Name: "thinking", Glyph: "🤔"},
		{Name: "eyes", Glyph: "👀"},
		{Name: "fire", Glyph: "🔥"},
		{Name: "rocket", Glyph: "🚀"},
		{Name: "wave", Glyph: "👋"},
		{Name: "clap", Glyph: "👏"},
		{Name: "pray", Glyph: "🙏"},
		{Name: "muscle", Glyph: "💪"},
		{Name: "check", Glyph: "✅"},
		{Name: "x", Glyph: "❌"},
		{Name: "warning", Glyph: "⚠️"},
		{Name: "white_check_mark", Glyph: "✅"},
		{Name: "raised_hands", Glyph: "🙌"},
		{Name: "100", Glyph: "💯"},
		{Name: "star", Glyph: "⭐"},
		{Name: "sparkles", Glyph: "✨"},
		{Name: "zap", Glyph: "⚡"},
		{Name: "bulb", Glyph: "💡"},
		{Name: "memo", Glyph: "📝"},
		{Name: "computer", Glyph: "💻"},
	}
}
