package track_info

import (
	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	panel       lipgloss.Style
	subtitle    lipgloss.Style
	error       lipgloss.Style
	metadataKey lipgloss.Style
	artwork     lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		panel:       lipgloss.NewStyle().Padding(0, 1),
		subtitle:    lipgloss.NewStyle().Foreground(th.Muted),
		error:       lipgloss.NewStyle().Foreground(th.Danger),
		metadataKey: lipgloss.NewStyle().Bold(true).Foreground(th.Secondary),
		artwork:     lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Margin(),
	}
}
