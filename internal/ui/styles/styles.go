package styles

import "github.com/charmbracelet/lipgloss"

var (
	AppBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	SidebarStyle = lipgloss.NewStyle().
			Width(30).
			Padding(1, 1)

	SidebarSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true).
			Padding(0, 1)

	SidebarNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	SidebarSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Bold(true).
			MarginTop(1).
			Padding(0, 1)

	MessagesStyle = lipgloss.NewStyle().
			Padding(1, 1)

	MessageUsername = lipgloss.NewStyle().
			Bold(true)

	MessageTimestamp = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	MessageText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	MessageReaction = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	ThreadBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 1)

	OverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	StatusBarKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true).
			Padding(0, 1)

	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	UnreadDot = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	FocusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63"))

	UnfocusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)

var UsernameColors = []lipgloss.Color{
	"204", "166", "178", "114", "81", "147", "212", "209",
}

func ColorForUser(userID string) lipgloss.Color {
	var hash int
	for _, c := range userID {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return UsernameColors[hash%len(UsernameColors)]
}
