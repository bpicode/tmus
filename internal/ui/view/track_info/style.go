package track_info

import "charm.land/lipgloss/v2"

var (
	styleOverlay     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Cyan).Padding(0, 1)
	styleTitle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightCyan)
	styleSubtitle    = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	styleError       = lipgloss.NewStyle().Foreground(lipgloss.Red)
	styleMetadataKey = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	styleArtwork     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Margin()
)
