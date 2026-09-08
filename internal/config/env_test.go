package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lookupEnv(values map[string]string) envLookup {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

func TestApplyEnvDiscoversFieldsFromTOMLTags(t *testing.T) {
	type futureSection struct {
		NewOption int    `toml:"new_option"`
		Label     string `toml:"label,omitempty"`
	}
	type futureConfig struct {
		Section futureSection `toml:"future"`
	}

	cfg := futureConfig{}
	values := map[string]string{
		"TMUS_FUTURE_NEW_OPTION": "42",
		"TMUS_FUTURE_LABEL":      "automatic",
	}
	err := applyEnv(&cfg, lookupEnv(values))

	require.NoError(t, err)
	assert.Equal(t, 42, cfg.Section.NewOption)
	assert.Equal(t, "automatic", cfg.Section.Label)
}

func TestApplyEnvAllowsAnEmptyStringOverride(t *testing.T) {
	cfg := Default()
	cfg.TUI.Theme.Primary = "#ffffff"

	err := applyEnv(&cfg, func(name string) (string, bool) {
		if name == "TMUS_TUI_THEME_PRIMARY" {
			return "", true
		}
		return "", false
	})

	require.NoError(t, err)
	assert.Empty(t, cfg.TUI.Theme.Primary)
}

func TestApplyEnvRejectsDerivedNameCollisions(t *testing.T) {
	type firstSection struct {
		C string `toml:"c"`
	}
	type secondSection struct {
		BC string `toml:"b_c"`
	}
	type collidingConfig struct {
		First  firstSection  `toml:"a_b"`
		Second secondSection `toml:"a"`
	}

	err := applyEnv(&collidingConfig{}, func(string) (string, bool) { return "", false })

	require.Error(t, err)
	assert.ErrorContains(t, err, "TMUS_A_B_C")
}
