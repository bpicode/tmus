package lyrics

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	overlay    lipgloss.Style
	title      lipgloss.Style
	track      lipgloss.Style
	activeLine lipgloss.Style
	empty      lipgloss.Style
	err        lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		overlay:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Primary).Padding(0, 1),
		title:      lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
		track:      lipgloss.NewStyle().Foreground(th.Muted),
		activeLine: lipgloss.NewStyle().Bold(true).Foreground(th.Highlight),
		empty:      lipgloss.NewStyle().Foreground(th.Muted),
		err:        lipgloss.NewStyle().Foreground(th.Danger),
	}
}
