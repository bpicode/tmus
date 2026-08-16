package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
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

func newTheme(p palette) Theme {
	return Theme{
		Foreground: toColor(p.foreground),
		Background: toColor(p.background),
		Primary:    toColor(p.primary),
		Secondary:  toColor(p.secondary),
		Muted:      toColor(p.muted),
		Highlight:  toColor(p.highlight),
		Info:       toColor(p.info),
		Danger:     toColor(p.danger),
		Warning:    toColor(p.warning),
		Working:    toColor(p.working),
	}
}

func toColor(s string) color.Color {
	if s == "" {
		return nil
	}
	return lipgloss.Color(s)
}
