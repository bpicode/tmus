package browser

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	titleUnfocused lipgloss.Style
	titleFocused   lipgloss.Style
	separator      lipgloss.Style
	cwd            lipgloss.Style
	dir            lipgloss.Style
	archive        lipgloss.Style
	empty          lipgloss.Style
	selected       lipgloss.Style
	searchInactive lipgloss.Style
	searchActive   lipgloss.Style
	err            lipgloss.Style
	panelFocused   lipgloss.Style
	panelUnfocused lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		titleUnfocused: lipgloss.NewStyle().Bold(true).Foreground(th.Primary),
		titleFocused:   lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
		separator:      lipgloss.NewStyle().Foreground(th.Muted),
		cwd:            lipgloss.NewStyle().Foreground(th.Muted),
		dir:            lipgloss.NewStyle().Foreground(th.Primary),
		archive:        lipgloss.NewStyle().Foreground(th.Secondary),
		empty:          lipgloss.NewStyle().Foreground(th.Muted),
		selected:       lipgloss.NewStyle().Reverse(true),
		searchInactive: lipgloss.NewStyle().Foreground(th.Muted),
		searchActive:   lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
		err:            lipgloss.NewStyle().Foreground(th.Danger),
		panelFocused:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Primary).Padding(0, 1),
		panelUnfocused: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Muted).Padding(0, 1),
	}
}
