package track_info

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	Overlay     lipgloss.Style
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Error       lipgloss.Style
	MetadataKey lipgloss.Style
	Artwork     lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		Overlay:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Primary).Padding(0, 1),
		Title:       lipgloss.NewStyle().Bold(true).Foreground(th.Primary),
		Subtitle:    lipgloss.NewStyle().Foreground(th.Muted),
		Error:       lipgloss.NewStyle().Foreground(th.Danger),
		MetadataKey: lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
		Artwork:     lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Margin(),
	}
}
