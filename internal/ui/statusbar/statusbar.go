package statusbar

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/filipeestacio/lazyslack/internal/ui/styles"
)

type Model struct {
	workspace   string
	connected   bool
	rateLimited bool
	hints       string
	lastError   string
}

func New(workspace string) Model {
	return Model{
		workspace: workspace,
		connected: true,
		hints:     "c:compose  t:thread  r:react  ?:help",
	}
}

func (m *Model) SetConnected(v bool)    { m.connected = v }
func (m *Model) SetError(err string)    { m.lastError = err }
func (m *Model) SetRateLimited(v bool) { m.rateLimited = v }
func (m *Model) SetHints(h string)     { m.hints = h }

func (m Model) View(width int) string {
	status := "●"
	statusColor := lipgloss.Color("114")
	if !m.connected {
		status = "○ " + m.lastError
		if status == "○ " {
			status = "○ disconnected"
		}
		statusColor = lipgloss.Color("196")
	} else if m.rateLimited {
		status = "◐ rate limited"
		statusColor = lipgloss.Color("214")
	}

	left := styles.StatusBarKey.Render(" "+m.workspace+" ") +
		" " +
		lipgloss.NewStyle().Foreground(statusColor).Render(status)

	right := styles.StatusBarStyle.Render(m.hints)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	bar := left + lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Render(spaces(gap)) + right

	return styles.StatusBarStyle.Width(width).Render(bar)
}

func spaces(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
