package track_info

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	overlay     lipgloss.Style
	title       lipgloss.Style
	subtitle    lipgloss.Style
	error       lipgloss.Style
	metadataKey lipgloss.Style
	artwork     lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		overlay:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Primary).Padding(0, 1),
		title:       lipgloss.NewStyle().Bold(true).Foreground(th.Primary),
		subtitle:    lipgloss.NewStyle().Foreground(th.Muted),
		error:       lipgloss.NewStyle().Foreground(th.Danger),
		metadataKey: lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
		artwork:     lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Margin(),
	}
}
