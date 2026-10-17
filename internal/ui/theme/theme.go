package theme

import (
	"image/color"
	"math/rand/v2"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/config"
)

// Theme holds the resolved colors and border shape for the UI.
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
	Border     lipgloss.Border
}

// Resolve selects a preset, applies overrides, and constructs a Theme.
func Resolve(cfg config.ThemeConfig) Theme {
	def := resolveDefinition(cfg, rand.IntN)
	return Theme{
		Foreground: toColor(def.foreground),
		Background: toColor(def.background),
		Primary:    toColor(def.primary),
		Secondary:  toColor(def.secondary),
		Muted:      toColor(def.muted),
		Highlight:  toColor(def.highlight),
		Info:       toColor(def.info),
		Danger:     toColor(def.danger),
		Warning:    toColor(def.warning),
		Working:    toColor(def.working),
		Border:     toBorder(def.border),
	}
}

func toColor(s string) color.Color {
	if s == "" {
		return nil
	}
	return lipgloss.Color(s)
}

func toBorder(border string) lipgloss.Border {
	switch border {
	case "rounded":
		return lipgloss.RoundedBorder()
	case "normal":
		return lipgloss.NormalBorder()
	case "ascii":
		return lipgloss.ASCIIBorder()
	case "block":
		return lipgloss.BlockBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "thick":
		return lipgloss.ThickBorder()
	case "hidden":
		return lipgloss.HiddenBorder()
	case "markdown":
		return lipgloss.MarkdownBorder()
	default:
		return lipgloss.RoundedBorder()
	}
}
