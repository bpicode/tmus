package help

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	padding   lipgloss.Style
	title     lipgloss.Style
	subtitle  lipgloss.Style
	helpKey   lipgloss.Style
	separator lipgloss.Style
	footer    lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		padding:   lipgloss.NewStyle().Padding(1, 2),
		title:     lipgloss.NewStyle().Bold(true).Foreground(th.Primary),
		subtitle:  lipgloss.NewStyle().Bold(false).Foreground(th.Primary),
		helpKey:   lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
		separator: lipgloss.NewStyle().Foreground(th.Muted),
		footer:    lipgloss.NewStyle().Foreground(th.Muted),
	}
}
