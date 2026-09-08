package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFlagSelectsFileAndEnvironmentStillOverridesValues(t *testing.T) {
	t.Setenv("TMUS_TUI_FPS", "45")
	configPath := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[tui]\nFPS = 60\n"), 0o600))

	cmd := &cobra.Command{}
	addConfigFlags(cmd)
	require.NoError(t, cmd.Flags().Set("config", configPath))

	cfg, err := loadConfig(cmd)

	require.NoError(t, err)
	assert.Equal(t, 45, cfg.TUI.FPS)
}
