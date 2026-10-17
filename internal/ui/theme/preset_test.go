package theme

import (
	"testing"

	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDefinition(t *testing.T) {
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
			cfg := config.ThemeConfig{Preset: tt.preset}

			def := resolveDefinition(cfg, constantZero)

			assert.Equal(t, tt.primary, def.primary)
			assert.Equal(t, tt.background, def.background)
			assert.NotEmpty(t, def.secondary)
			assert.NotEmpty(t, def.muted)
			assert.NotEmpty(t, def.highlight)
			assert.NotEmpty(t, def.info)
			assert.NotEmpty(t, def.danger)
			assert.NotEmpty(t, def.warning)
			assert.NotEmpty(t, def.working)
			assert.Equal(t, "rounded", def.border)
		})
	}
}

func TestResolveDefinitionAppliesOverrides(t *testing.T) {
	cfg := config.ThemeConfig{Preset: "catppuccin-mocha", Primary: "#abcdef", Background: terminalColor, Border: "double"}

	def := resolveDefinition(cfg, constantZero)

	assert.Equal(t, "#abcdef", def.primary)
	assert.Empty(t, def.background)
	assert.Equal(t, "#94e2d5", def.secondary)
	assert.Equal(t, "double", def.border)
}

func TestResolveDefinitionFallsBackForUnknownPreset(t *testing.T) {
	cfg := config.ThemeConfig{Preset: "unknown", Primary: "#abcdef", Border: "thick"}

	def := resolveDefinition(cfg, constantZero)

	assert.Equal(t, "#abcdef", def.primary)
	assert.Equal(t, "14", def.secondary)
	assert.Equal(t, "thick", def.border)
}

func TestRandomCanSelectEveryConcretePreset(t *testing.T) {
	require.NotEmpty(t, builtInPresets)

	for index, want := range builtInPresets {
		t.Run(want.name, func(t *testing.T) {
			cfg := config.ThemeConfig{Preset: randomPreset}
			sequential := func(count int) int {
				assert.Equal(t, len(builtInPresets), count)
				return index
			}

			def := resolveDefinition(cfg, sequential)

			assert.Equal(t, want.definition, def)
		})
	}
}

func TestPresetRegistryHasUniqueNames(t *testing.T) {
	seen := make(map[string]bool, len(builtInPresets))
	for _, preset := range builtInPresets {
		_, duplicate := seen[preset.name]
		assert.False(t, duplicate, "duplicate theme preset %q", preset.name)
		seen[preset.name] = true
	}
}

func constantZero(_ int) int { return 0 }
