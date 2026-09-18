package theme

import (
	"image/color"
	"math/rand/v2"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/config"
)

// Theme holds the pre-parsed lipgloss colors for the UI.
type Theme struct {
	Foreground color.Color
	Background color.Color
	Primary    color.Color
	Secondary  color.Color
	Muted      color.Color
	Highlight  color.Color
	Info       color.Color
	Warning    color.Color
	Danger     color.Color
	Working    color.Color
}

// Resolve selects a preset, applies color overrides, and constructs a Theme.
func Resolve(cfg config.ThemeConfig) Theme {
	colors := resolveColors(cfg, rand.IntN)
	return Theme{
		Foreground: toColor(colors.foreground),
		Background: toColor(colors.background),
		Primary:    toColor(colors.primary),
		Secondary:  toColor(colors.secondary),
		Muted:      toColor(colors.muted),
		Highlight:  toColor(colors.highlight),
		Info:       toColor(colors.info),
		Danger:     toColor(colors.danger),
		Warning:    toColor(colors.warning),
		Working:    toColor(colors.working),
	}
}

func toColor(s string) color.Color {
	if s == "" {
		return nil
	}
	return lipgloss.Color(s)
}
