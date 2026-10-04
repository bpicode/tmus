package browser

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/truncate"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	separator      lipgloss.Style
	cwd            truncate.Left
	dir            lipgloss.Style
	archive        lipgloss.Style
	empty          lipgloss.Style
	selected       lipgloss.Style
	searchInactive lipgloss.Style
	searchActive   lipgloss.Style
	err            lipgloss.Style
	panel          lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		separator:      lipgloss.NewStyle().Foreground(th.Muted),
		cwd:            truncate.Left{Style: lipgloss.NewStyle().Foreground(th.Muted)},
		dir:            lipgloss.NewStyle().Foreground(th.Primary),
		archive:        lipgloss.NewStyle().Foreground(th.Secondary),
		empty:          lipgloss.NewStyle().Foreground(th.Muted),
		selected:       lipgloss.NewStyle().Reverse(true),
		searchInactive: lipgloss.NewStyle().Foreground(th.Muted),
		searchActive:   lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
		err:            lipgloss.NewStyle().Foreground(th.Danger),
		panel:          lipgloss.NewStyle().Padding(0, 1),
	}
}
