package help

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	overlay  lipgloss.Style
	title    lipgloss.Style
	tubtitle lipgloss.Style
	helpKey  lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		overlay:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Primary).Padding(1, 2),
		title:    lipgloss.NewStyle().Bold(true).Foreground(th.Primary),
		tubtitle: lipgloss.NewStyle().Bold(false).Foreground(th.Primary),
		helpKey:  lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
	}
}
