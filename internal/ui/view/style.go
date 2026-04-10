package view

import (
	"image/color"

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
