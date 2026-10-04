package lyrics

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	panel      lipgloss.Style
	track      lipgloss.Style
	activeLine lipgloss.Style
	empty      lipgloss.Style
	err        lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		panel:      lipgloss.NewStyle().Padding(0, 1),
		track:      lipgloss.NewStyle().Foreground(th.Muted),
		activeLine: lipgloss.NewStyle().Bold(true).Foreground(th.Highlight),
		empty:      lipgloss.NewStyle().Foreground(th.Muted),
		err:        lipgloss.NewStyle().Foreground(th.Danger),
	}
}
