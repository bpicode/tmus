package theme

import (
	"image/color"

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

// New parses the string-based ThemeConfig into a usable lipgloss Theme.
func New(cfg config.ThemeConfig) Theme {
	return Theme{
		Foreground: toColor(cfg.Foreground),
		Background: toColor(cfg.Background),
		Primary:    toColor(cfg.Primary),
		Secondary:  toColor(cfg.Secondary),
		Muted:      toColor(cfg.Muted),
		Highlight:  toColor(cfg.Highlight),
		Info:       toColor(cfg.Info),
		Danger:     toColor(cfg.Danger),
		Warning:    toColor(cfg.Warning),
	}
}

func toColor(s string) color.Color {
	if s == "" {
		return nil
	}
	return lipgloss.Color(s)
}
