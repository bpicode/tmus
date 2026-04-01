package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/config"
)

// Theme holds the pre-parsed lipgloss colors for the UI.
type Theme struct {
	Primary   color.Color
	Secondary color.Color
	Muted     color.Color
	Highlight color.Color
	Info      color.Color
	Warning   color.Color
	Danger    color.Color
	Working   color.Color
}

// New parses the string-based ThemeConfig into a usable lipgloss Theme.
func New(cfg config.ThemeConfig) Theme {
	return Theme{
		Primary:   lipgloss.Color(cfg.Primary),
		Secondary: lipgloss.Color(cfg.Secondary),
		Muted:     lipgloss.Color(cfg.Muted),
		Highlight: lipgloss.Color(cfg.Highlight),
		Info:      lipgloss.Color(cfg.Info),
		Danger:    lipgloss.Color(cfg.Danger),
		Warning:   lipgloss.Color(cfg.Warning),
		Working:   lipgloss.Color(cfg.Working),
	}
}
