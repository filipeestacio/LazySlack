package app

import (
	"log"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/filipeestacio/lazyslack/internal/slack"
	"github.com/filipeestacio/lazyslack/internal/ui/emoji"
	"github.com/filipeestacio/lazyslack/internal/ui/help"
	"github.com/filipeestacio/lazyslack/internal/ui/input"
	"github.com/filipeestacio/lazyslack/internal/ui/messages"
	"github.com/filipeestacio/lazyslack/internal/ui/sidebar"
	"github.com/filipeestacio/lazyslack/internal/ui/statusbar"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
	"github.com/filipeestacio/lazyslack/internal/ui/thread"
)

type SlackClient interface {
	slack.SlackClient
}

type channelsLoadedMsg struct {
	channels []slack.Channel
	starred  map[string]bool
}
type dmsLoadedMsg struct {
	convs []slack.Conversation
}
type historyLoadedMsg struct{ result *slack.HistoryResult }
type threadLoadedMsg struct{ messages []slack.Message }
type messageSentMsg struct{}
type reactionAddedMsg struct{}
type errMsg struct{ err error }
type newMessagesMsg struct{ messages []slack.Message }

type Model struct {
	client    SlackClient
	cache     *slack.UserCache
	workspace string
	width     int
	height    int
	focus     focusArea

	showHelp   bool
	composing  bool
	showEmoji  bool
	showThread bool

	currentChannelID string

	sidebar   sidebar.Model
	messages  messages.Model
	thread    thread.Model
	input     input.Model
	emoji     emoji.Model
	help      help.Model
	statusbar statusbar.Model
	renderer  *slack.Renderer

	users []slack.User
}

func New(client SlackClient, cache *slack.UserCache, workspace string) Model {
	renderer := slack.NewRenderer(cache)
	return Model{
		client:    client,
		cache:     cache,
		workspace: workspace,
		sidebar:   sidebar.New(),
		messages:  messages.New(renderer, cache),
		thread:    thread.New(renderer),
		input:     input.New("", "", ""),
		emoji:     emoji.New("", ""),
		help:      help.New(),
		statusbar: statusbar.New(workspace),
		renderer:  renderer,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadChannels()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentH := msg.Height - 3
		m.sidebar.SetHeight(contentH)
		m.messages.SetSize(msg.Width-30-4, contentH)
		m.thread.SetSize(40, contentH)
		m.help.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case channelsLoadedMsg:
		m.sidebar.SetChannels(msg.channels, msg.starred)
		return m, m.loadDMs()

	case dmsLoadedMsg:
		resolve := func(id string) string {
			if m.cache != nil {
				return m.cache.ResolveUser(id)
			}
			return id
		}
		m.sidebar.SetDMs(msg.convs, resolve)
		return m, nil

	case historyLoadedMsg:
		m.messages.SetMessages(msg.result.Messages)
		return m, nil

	case threadLoadedMsg:
		m.thread.SetReplies(msg.messages)
		return m, nil

	case newMessagesMsg:
		m.messages.SetMessages(msg.messages)
		return m, nil

	case sidebar.ChannelSelectedMsg:
		m.messages.SetChannel(msg.ID, msg.Name)
		m.currentChannelID = msg.ID
		m.showThread = false
		return m, m.fetchHistory(msg.ID)

	case messages.OpenThreadMsg:
		m.showThread = true
		m.thread.SetThread(msg.ChannelID, msg.ThreadTS)
		m.focus = focusThread
		return m, m.fetchThread(msg.ChannelID, msg.ThreadTS)

	case messages.OpenComposeMsg:
		m.composing = true
		m.input = input.New(msg.ChannelID, "", msg.ChannelName)
		return m, nil

	case thread.CloseMsg:
		m.showThread = false
		m.focus = focusMessages
		return m, nil

	case thread.OpenThreadComposeMsg:
		m.composing = true
		m.input = input.New(msg.ChannelID, msg.ThreadTS, "thread")
		return m, nil

	case input.SendMsg:
		m.composing = false
		return m, m.sendMessage(msg.ChannelID, msg.ThreadTS, msg.Text)

	case input.DismissMsg:
		m.composing = false
		return m, nil

	case emoji.SelectMsg:
		m.showEmoji = false
		return m, m.addReaction(msg.ChannelID, msg.MessageTS, msg.Emoji)

	case emoji.CloseMsg:
		m.showEmoji = false
		return m, nil

	case help.CloseMsg:
		m.showHelp = false
		return m, nil

	case messages.CopyMsg:
		return m, func() tea.Msg {
			cmd := exec.Command("pbcopy")
			cmd.Stdin = strings.NewReader(msg.Text)
			cmd.Run()
			return nil
		}

	case messageSentMsg:
		return m, nil

	case reactionAddedMsg:
		return m, nil

	case errMsg:
		log.Printf("app error: %v", msg.err)
		m.statusbar.SetConnected(false)
		m.statusbar.SetError(msg.err.Error())
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.composing {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	if m.showEmoji {
		var cmd tea.Cmd
		m.emoji, cmd = m.emoji.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	}

	if m.showHelp {
		var cmd tea.Cmd
		m.help, cmd = m.help.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "h":
		m.focus = focusSidebar
		return m, nil
	case "l":
		m.focus = focusMessages
		return m, nil
	case "r":
		sel := m.messages.SelectedMessage()
		if sel != nil {
			m.showEmoji = true
			m.emoji = emoji.New(m.currentChannelID, sel.Timestamp)
		}
		return m, nil
	}

	switch m.focus {
	case focusSidebar:
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		return m, cmd
	case focusMessages:
		var cmd tea.Cmd
		m.messages, cmd = m.messages.Update(msg)
		return m, cmd
	case focusThread:
		var cmd tea.Cmd
		m.thread, cmd = m.thread.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	innerH := m.height - 3
	if innerH < 1 {
		innerH = 1
	}
	sidebarW := 30
	msgsW := m.width - sidebarW - 4

	sidebarBorder := styles.UnfocusedBorder
	msgsBorder := styles.UnfocusedBorder
	if m.focus == focusSidebar {
		sidebarBorder = styles.FocusedBorder
	} else if m.focus == focusMessages {
		msgsBorder = styles.FocusedBorder
	}

	sidebarView := sidebarBorder.Width(sidebarW).Render(m.sidebar.View())
	messagesView := msgsBorder.Width(msgsW).Height(innerH).Render(m.messages.View())

	var content string
	if m.showThread {
		threadView := m.thread.View()
		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, messagesView, threadView)
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, messagesView)
	}

	statusView := m.statusbar.View(m.width)
	view := lipgloss.JoinVertical(lipgloss.Left, content, statusView)

	if m.showHelp {
		return m.help.View()
	}

	if m.composing {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.input.View(),
			lipgloss.WithWhitespaceBackground(lipgloss.Color("235")))
	}

	if m.showEmoji {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.emoji.View(),
			lipgloss.WithWhitespaceBackground(lipgloss.Color("235")))
	}

	return view
}

func (m Model) loadChannels() tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		log.Printf("loading channels...")
		channels, err := m.client.ListChannels()
		if err != nil {
			log.Printf("error loading channels: %v", err)
			return errMsg{err}
		}
		starred, err := m.client.ListStarredChannelIDs()
		if err != nil {
			log.Printf("error loading stars (continuing without): %v", err)
			starred = nil
		}
		log.Printf("loaded %d channels, %d starred", len(channels), len(starred))
		return channelsLoadedMsg{channels: channels, starred: starred}
	}
}

func (m Model) loadDMs() tea.Cmd {
	if m.client == nil {
		return nil
	}
	cache := m.cache
	return func() tea.Msg {
		convs, err := m.client.ListDMs()
		if err != nil {
			return errMsg{err}
		}
		if cache != nil {
			if err := cache.Load(); err != nil {
				log.Printf("user cache load failed: %v", err)
			}
		}
		return dmsLoadedMsg{convs: convs}
	}
}

func (m Model) fetchHistory(channelID string) tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		result, err := m.client.GetHistory(channelID, "")
		if err != nil {
			return errMsg{err}
		}
		return historyLoadedMsg{result}
	}
}

func (m Model) fetchThread(channelID, threadTS string) tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		msgs, err := m.client.GetThreadReplies(channelID, threadTS)
		if err != nil {
			return errMsg{err}
		}
		return threadLoadedMsg{msgs}
	}
}

func (m Model) sendMessage(channelID, threadTS, text string) tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		var err error
		if threadTS != "" {
			err = m.client.ReplyToThread(channelID, threadTS, text)
		} else {
			err = m.client.SendMessage(channelID, text)
		}
		if err != nil {
			return errMsg{err}
		}
		return messageSentMsg{}
	}
}

func (m Model) addReaction(channelID, messageTS, emoji string) tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		err := m.client.AddReaction(channelID, messageTS, emoji)
		if err != nil {
			return errMsg{err}
		}
		return reactionAddedMsg{}
	}
}

type teaAdapter struct{ m Model }

func (a teaAdapter) Init() tea.Cmd { return a.m.Init() }

func (a teaAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := a.m.Update(msg)
	return teaAdapter{m}, cmd
}

func (a teaAdapter) View() string { return a.m.View() }

func AsTeaModel(m Model) tea.Model { return teaAdapter{m} }
