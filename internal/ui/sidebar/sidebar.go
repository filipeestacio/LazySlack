package sidebar

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/filipeestacio/lazyslack/internal/slack"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type ChannelSelectedMsg struct {
	ID   string
	Name string
}

type Item struct {
	ID        string
	Name      string
	IsSection bool
	IsPrivate bool
	Unread    bool
}

type Model struct {
	items        []Item
	filtered     []int
	cursor       int
	filtering    bool
	filterText   string
	channelsOpen bool
	dmsOpen      bool
	height       int
}

func New() Model {
	return Model{
		channelsOpen: true,
		dmsOpen:      true,
	}
}

func (m *Model) SetChannels(channels []slack.Channel) {
	m.items = nil
	m.items = append(m.items, Item{Name: "Channels", IsSection: true})
	for _, ch := range channels {
		prefix := "#"
		if ch.IsPrivate {
			prefix = "🔒"
		}
		m.items = append(m.items, Item{
			ID:        ch.ID,
			Name:      prefix + " " + ch.Name,
			IsPrivate: ch.IsPrivate,
		})
	}
	m.resetFilter()
}

func (m *Model) SetDMs(convs []slack.Conversation, resolve func(string) string) {
	m.items = append(m.items, Item{Name: "Direct Messages", IsSection: true})
	for _, c := range convs {
		name := resolve(c.UserID)
		m.items = append(m.items, Item{ID: c.ID, Name: name})
	}
	m.resetFilter()
}

func (m *Model) SetHeight(h int) { m.height = h }
func (m Model) CursorIndex() int { return m.cursor }

func (m Model) VisibleItems() []Item {
	var result []Item
	for _, idx := range m.filtered {
		result = append(result, m.items[idx])
	}
	return result
}

func (m Model) SelectedID() string {
	if m.cursor >= len(m.filtered) {
		return ""
	}
	return m.items[m.filtered[m.cursor]].ID
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			if m.items[m.filtered[m.cursor]].IsSection {
				if m.cursor < len(m.filtered)-1 {
					m.cursor++
				}
			}
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			if m.items[m.filtered[m.cursor]].IsSection {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		}
	case "g":
		m.cursor = 0
		if len(m.filtered) > 0 && m.items[m.filtered[0]].IsSection {
			if len(m.filtered) > 1 {
				m.cursor = 1
			}
		}
	case "G":
		m.cursor = len(m.filtered) - 1
	case "enter":
		for i := m.cursor; i < len(m.filtered); i++ {
			item := m.items[m.filtered[i]]
			if !item.IsSection {
				return m, func() tea.Msg {
					return ChannelSelectedMsg{ID: item.ID, Name: item.Name}
				}
			}
		}
	case "/":
		m.filtering = true
		m.filterText = ""
	case "tab":
		for i := m.cursor; i >= 0; i-- {
			if m.items[m.filtered[i]].IsSection {
				if m.items[m.filtered[i]].Name == "Channels" {
					m.channelsOpen = !m.channelsOpen
				} else {
					m.dmsOpen = !m.dmsOpen
				}
				m.rebuildFiltered()
				break
			}
		}
	}
	return m, nil
}

func (m Model) updateFilter(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filtering = false
		m.filterText = ""
		m.resetFilter()
	case tea.KeyEnter:
		m.filtering = false
	case tea.KeyBackspace:
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.applyFilter()
		}
	case tea.KeyRunes:
		m.filterText += string(msg.Runes)
		m.applyFilter()
	}
	return m, nil
}

func (m *Model) resetFilter() {
	m.filtered = make([]int, len(m.items))
	for i := range m.items {
		m.filtered[i] = i
	}
	m.cursor = 0
}

func (m *Model) applyFilter() {
	if m.filterText == "" {
		m.resetFilter()
		return
	}
	m.filtered = nil
	query := strings.ToLower(m.filterText)
	for i, item := range m.items {
		if item.IsSection || strings.Contains(strings.ToLower(item.Name), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor = 0
	for i, idx := range m.filtered {
		if !m.items[idx].IsSection {
			m.cursor = i
			break
		}
	}
}

func (m *Model) rebuildFiltered() {
	m.filtered = nil
	inChannels := false
	inDMs := false
	for i, item := range m.items {
		if item.IsSection {
			if item.Name == "Channels" {
				inChannels = true
				inDMs = false
			} else {
				inChannels = false
				inDMs = true
			}
			m.filtered = append(m.filtered, i)
			continue
		}
		if inChannels && !m.channelsOpen {
			continue
		}
		if inDMs && !m.dmsOpen {
			continue
		}
		m.filtered = append(m.filtered, i)
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) View() string {
	var b strings.Builder

	if m.filtering {
		b.WriteString(styles.SidebarSection.Render("/" + m.filterText + "█"))
		b.WriteString("\n")
	}

	for i, idx := range m.filtered {
		item := m.items[idx]
		if item.IsSection {
			b.WriteString(styles.SidebarSection.Render(item.Name))
		} else if i == m.cursor {
			b.WriteString(styles.SidebarSelected.Render(item.Name))
		} else {
			style := styles.SidebarNormal
			if item.Unread {
				style = style.Copy().Bold(true)
			}
			b.WriteString(style.Render(item.Name))
		}
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().Width(30).Render(b.String())
}
