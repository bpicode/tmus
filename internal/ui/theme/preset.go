package theme

import (
	"math/rand/v2"

	"github.com/bpicode/tmus/internal/config"
)

const (
	defaultPreset = "default"
	randomPreset  = "random"
	terminalColor = "terminal"
)

type preset struct {
	name    string
	palette palette
}

type palette struct {
	foreground string
	background string
	primary    string
	secondary  string
	muted      string
	highlight  string
	info       string
	danger     string
	warning    string
	working    string
}

var builtInPresets = [...]preset{
	{
		name: defaultPreset,
		palette: palette{
			primary: "6", secondary: "14", muted: "8", highlight: "13",
			info: "2", danger: "1", warning: "3", working: "12",
		},
	},
	{
		name: "github-light", // Palette is adapted from GitHub Primer Primitives v11.10.0.
		palette: palette{
			foreground: "#1f2328", background: "#ffffff",
			primary: "#0969da", secondary: "#8250df", muted: "#59636e", highlight: "#bf3989",
			info: "#1a7f37", danger: "#d1242f", warning: "#9a6700", working: "#218bff",
		},
	},
	{
		name: "github-dark", // Palette is adapted from GitHub Primer Primitives v11.10.0.
		palette: palette{
			foreground: "#f0f6fc", background: "#0d1117",
			primary: "#4493f8", secondary: "#ab7df8", muted: "#9198a1", highlight: "#db61a2",
			info: "#3fb950", danger: "#f85149", warning: "#d29922", working: "#79c0ff",
		},
	},
	{
		name: "catppuccin-latte",
		palette: palette{
			foreground: "#4c4f69", background: "#eff1f5",
			primary: "#8839ef", secondary: "#179299", muted: "#9ca0b0", highlight: "#ea76cb",
			info: "#40a02b", danger: "#d20f39", warning: "#df8e1d", working: "#1e66f5",
		},
	},
	{
		name: "catppuccin-frappe",
		palette: palette{
			foreground: "#c6d0f5", background: "#303446",
			primary: "#ca9ee6", secondary: "#81c8be", muted: "#737994", highlight: "#f4b8e4",
			info: "#a6d189", danger: "#e78284", warning: "#e5c890", working: "#8caaee",
		},
	},
	{
		name: "catppuccin-macchiato",
		palette: palette{
			foreground: "#cad3f5", background: "#24273a",
			primary: "#c6a0f6", secondary: "#8bd5ca", muted: "#6e738d", highlight: "#f5bde6",
			info: "#a6da95", danger: "#ed8796", warning: "#eed49f", working: "#8aadf4",
		},
	},
	{
		name: "catppuccin-mocha",
		palette: palette{
			foreground: "#cdd6f4", background: "#1e1e2e",
			primary: "#cba6f7", secondary: "#94e2d5", muted: "#6c7086", highlight: "#f5c2e7",
			info: "#a6e3a1", danger: "#f38ba8", warning: "#f9e2af", working: "#89b4fa",
		},
	},
	{
		name: "matrix",
		palette: palette{
			foreground: "#426644", background: "#0f191c",
			primary: "#50b45a", secondary: "#c1ff8a", muted: "#688060", highlight: "#11ff25",
			info: "#82d967", danger: "#23755a", warning: "#ffd700", working: "#4f7e7e",
		},
	},
}

// Resolve selects a preset, applies color overrides, and constructs a Theme.
func Resolve(cfg config.ThemeConfig) Theme {
	return newTheme(resolveColors(cfg, rand.IntN))
}

func resolveColors(cfg config.ThemeConfig, randomIndex func(int) int) palette {
	name := cfg.Preset
	if name == "" {
		name = defaultPreset
	}
	if name == randomPreset {
		name = builtInPresets[randomIndex(len(builtInPresets))].name
	}

	p, ok := findPreset(name)
	if !ok {
		p, _ = findPreset(defaultPreset)
	}
	applyOverrides(&p, cfg)
	return p
}

func findPreset(name string) (palette, bool) {
	for _, preset := range builtInPresets {
		if preset.name == name {
			return preset.palette, true
		}
	}
	return palette{}, false
}

func applyOverrides(dst *palette, overrides config.ThemeConfig) {
	if overrides.Primary != "" {
		dst.primary = overrides.Primary
	}
	if overrides.Secondary != "" {
		dst.secondary = overrides.Secondary
	}
	if overrides.Muted != "" {
		dst.muted = overrides.Muted
	}
	if overrides.Highlight != "" {
		dst.highlight = overrides.Highlight
	}
	if overrides.Info != "" {
		dst.info = overrides.Info
	}
	if overrides.Danger != "" {
		dst.danger = overrides.Danger
	}
	if overrides.Warning != "" {
		dst.warning = overrides.Warning
	}
	if overrides.Working != "" {
		dst.working = overrides.Working
	}
	if overrides.Background != "" {
		dst.background = resolveTerminalColor(overrides.Background)
	}
	if overrides.Foreground != "" {
		dst.foreground = resolveTerminalColor(overrides.Foreground)
	}
}

func resolveTerminalColor(value string) string {
	if value == terminalColor {
		return ""
	}
	return value
}
