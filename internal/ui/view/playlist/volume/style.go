package volume

import (
	"image/color"

	"github.com/bpicode/tmus/internal/ui/theme"
)

type styles struct {
	volumeBarLow  color.Color
	volumeBarHigh color.Color
}

func newStyles(th theme.Theme) styles {
	return styles{
		volumeBarLow:  th.Primary,
		volumeBarHigh: th.Secondary,
	}
}
