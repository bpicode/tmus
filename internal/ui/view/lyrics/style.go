package lyrics

import "charm.land/lipgloss/v2"

var (
	styleOverlay    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Cyan).Padding(0, 1)
	styleTitle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightCyan)
	styleTrack      = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	styleActiveLine = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightYellow)
	styleEmpty      = lipgloss.NewStyle().Foreground(lipgloss.Yellow)
	styleError      = lipgloss.NewStyle().Foreground(lipgloss.Red)
)
