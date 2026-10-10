package view

import (
	"image/color"

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
	return notification.Styles{
		Foreground:  th.Foreground,
		Background:  th.Background,
		InfoBorder:  th.Info,
		ErrorBorder: th.Danger,
	}
}
