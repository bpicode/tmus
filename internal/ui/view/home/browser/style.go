package browser

import "charm.land/lipgloss/v2"

var (
	styleTitle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Cyan)
	styleTitleFocused   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightCyan)
	styleSeparator      = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	styleCwd            = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	styleDir            = lipgloss.NewStyle().Foreground(lipgloss.Cyan)
	styleArchive        = lipgloss.NewStyle().Foreground(lipgloss.BrightCyan)
	styleEmpty          = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	styleSelected       = lipgloss.NewStyle().Reverse(true)
	styleSearchInactive = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	styleSearchActive   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightCyan)
	styleError          = lipgloss.NewStyle().Foreground(lipgloss.Red)
	stylePanelFocused   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.BrightCyan).Padding(0, 1)
	stylePanelUnfocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.BrightBlack).Padding(0, 1)
)
