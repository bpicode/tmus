package view

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/ui/components/notification"
	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	foreground color.Color
	background color.Color
}

func newStyles(th theme.Theme) styles {
	return styles{
		foreground: th.Foreground,
		background: th.Background,
	}
}

func newNotificationStyles(th theme.Theme) notification.Styles {
	styles := notification.DefaultStyles()
	styles.Info = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Secondary).Foreground(th.Secondary).Padding(0, 1)
	styles.Success = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Info).Foreground(th.Info).Padding(0, 1)
	styles.Warn = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Warning).Foreground(th.Warning).Padding(0, 1)
	styles.Error = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Danger).Foreground(th.Danger).Padding(0, 1)
	return styles
}
