package theme

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveColors(t *testing.T) {
	tests := []struct {
		name       string
		preset     string
		primary    string
		background string
	}{
		{name: "implicit default", primary: "6"},
		{name: "default", preset: "default", primary: "6"},
		{name: "GitHub light", preset: "github-light", primary: "#0969da", background: "#ffffff"},
		{name: "GitHub dark", preset: "github-dark", primary: "#4493f8", background: "#0d1117"},
		{name: "catppuccin latte", preset: "catppuccin-latte", primary: "#8839ef", background: "#eff1f5"},
		{name: "catppuccin frappe", preset: "catppuccin-frappe", primary: "#ca9ee6", background: "#303446"},
		{name: "catppuccin macchiato", preset: "catppuccin-macchiato", primary: "#c6a0f6", background: "#24273a"},
		{name: "catppuccin mocha", preset: "catppuccin-mocha", primary: "#cba6f7", background: "#1e1e2e"},
		{name: "matrix", preset: "matrix", primary: "#50b45a", background: "#0f191c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colors := resolveColors(config.ThemeConfig{Preset: tt.preset}, func(int) int { return 0 })
			assert.Equal(t, tt.primary, colors.primary)
			assert.Equal(t, tt.background, colors.background)
			assert.NotEmpty(t, colors.secondary)
			assert.NotEmpty(t, colors.muted)
			assert.NotEmpty(t, colors.highlight)
			assert.NotEmpty(t, colors.info)
			assert.NotEmpty(t, colors.danger)
			assert.NotEmpty(t, colors.warning)
			assert.NotEmpty(t, colors.working)
		})
	}
}

func TestGitHubPresetsMatchPrimerPrimitives(t *testing.T) {
	tests := []struct {
		name string
		want palette
	}{
		{
			name: "github-light",
			want: palette{
				foreground: "#1f2328", background: "#ffffff",
				primary: "#0969da", secondary: "#8250df", muted: "#59636e", highlight: "#bf3989",
				info: "#1a7f37", danger: "#d1242f", warning: "#9a6700", working: "#218bff",
			},
		},
		{
			name: "github-dark",
			want: palette{
				foreground: "#f0f6fc", background: "#0d1117",
				primary: "#4493f8", secondary: "#ab7df8", muted: "#9198a1", highlight: "#db61a2",
				info: "#3fb950", danger: "#f85149", warning: "#d29922", working: "#79c0ff",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colors := resolveColors(config.ThemeConfig{Preset: tt.name}, func(int) int { return 0 })
			assert.Equal(t, tt.want, colors)
		})
	}
}

func TestResolveColorsAppliesOverrides(t *testing.T) {
	colors := resolveColors(config.ThemeConfig{
		Preset:     "catppuccin-mocha",
		Primary:    "#abcdef",
		Background: terminalColor,
	}, func(int) int { return 0 })

	assert.Equal(t, "#abcdef", colors.primary)
	assert.Empty(t, colors.background)
	assert.Equal(t, "#94e2d5", colors.secondary)
}

func TestResolveColorsFallsBackForUnknownPreset(t *testing.T) {
	colors := resolveColors(config.ThemeConfig{
		Preset:  "unknown",
		Primary: "#abcdef",
	}, func(int) int { return 0 })

	assert.Equal(t, "#abcdef", colors.primary)
	assert.Equal(t, "14", colors.secondary)
}

func TestRandomCanSelectEveryConcretePreset(t *testing.T) {
	require.NotEmpty(t, builtInPresets)

	for index, want := range builtInPresets {
		t.Run(want.name, func(t *testing.T) {
			colors := resolveColors(config.ThemeConfig{Preset: randomPreset}, func(count int) int {
				assert.Equal(t, len(builtInPresets), count)
				return index
			})
			assert.Equal(t, want.palette.primary, colors.primary)
		})
	}
}

func TestPresetRegistryHasUniqueNames(t *testing.T) {
	seen := make(map[string]struct{}, len(builtInPresets))
	for _, preset := range builtInPresets {
		_, duplicate := seen[preset.name]
		assert.False(t, duplicate, "duplicate theme preset %q", preset.name)
		seen[preset.name] = struct{}{}
	}
}

func TestResolveConstructsTheme(t *testing.T) {
	th := Resolve(config.ThemeConfig{
		Preset:     "matrix",
		Primary:    "#abcdef",
		Background: terminalColor,
	})

	assert.Equal(t, lipgloss.Color("#abcdef"), th.Primary)
	assert.Equal(t, lipgloss.Color("#c1ff8a"), th.Secondary)
	assert.Nil(t, th.Background)
}
