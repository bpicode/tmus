package help

import "charm.land/lipgloss/v2"

var (
	styleOverlay  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Cyan).Padding(1, 2)
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightCyan)
	styleSubtitle = lipgloss.NewStyle().Bold(false).Foreground(lipgloss.BrightCyan)
	styleHelpKey  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
)
