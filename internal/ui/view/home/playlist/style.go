package playlist

import "charm.land/lipgloss/v2"

var (
	styleTitle          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Cyan)
	styleTitleFocused   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightCyan)
	styleSearchInactive = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	styleSearchActive   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightCyan)
	styleError          = lipgloss.NewStyle().Foreground(lipgloss.Red)
	styleEmpty          = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	styleSelected       = lipgloss.NewStyle().Reverse(true)
	stylePlaying        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Cyan)
	stylePaused         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Yellow)
	styleStatusMeta     = lipgloss.NewStyle().Foreground(lipgloss.BrightMagenta)
	styleStatusPlay     = lipgloss.NewStyle().Foreground(lipgloss.Green)
	styleStatusPause    = lipgloss.NewStyle().Foreground(lipgloss.Yellow)
	styleStatusStop     = lipgloss.NewStyle().Foreground(lipgloss.Red)
	styleStatusNone     = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	styleStatusTime     = lipgloss.NewStyle().Foreground(lipgloss.BrightBlue)
	styleSeparator      = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	stylePanelFocused   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.BrightCyan).Padding(0, 1)
	stylePanelUnfocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.BrightBlack).Padding(0, 1)

	colorVolumeBarLow  = lipgloss.Cyan
	colorVolumeBarHigh = lipgloss.BrightCyan
)
