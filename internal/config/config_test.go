package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfigHasNoErrors(t *testing.T) {
	cfg := Default()
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestDefaultConfigValues(t *testing.T) {
	cfg := Default()

	assert.Equal(t, "auto", cfg.TUI.ArtworkRenderer)
	assert.Equal(t, SingleInstanceAuto, cfg.IPC.SingleInstance)
	assert.Equal(t, ByteSize(512*1024*1024), cfg.Library.MaxArchiveMemberSize)
}

func TestLoadAllowsDisablingIPC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	err := os.WriteFile(path, []byte("[ipc]\nsingle_instance = 'off'\n"), 0o600)
	require.NoError(t, err)

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, SingleInstanceOff, cfg.IPC.SingleInstance)
}

func TestSingleInstanceModeValidation(t *testing.T) {
	for _, mode := range []SingleInstanceMode{
		SingleInstanceAuto,
		SingleInstanceUnixSocket,
		SingleInstanceOff,
	} {
		t.Run(string(mode), func(t *testing.T) {
			cfg := Default()
			cfg.IPC.SingleInstance = mode
			assert.NoError(t, cfg.Validate())
		})
	}

	cfg := Default()
	cfg.IPC.SingleInstance = "magic"
	assert.ErrorContains(t, cfg.Validate(), "ipc.single_instance")
}

func TestArtworkRendererValidation(t *testing.T) {
	for _, renderer := range []string{"auto", "kitty", "blocks", "none"} {
		t.Run(renderer, func(t *testing.T) {
			cfg := Default()
			cfg.TUI.ArtworkRenderer = renderer
			assert.NoError(t, cfg.Validate())
		})
	}

	cfg := Default()
	cfg.TUI.ArtworkRenderer = "sixel"
	assert.ErrorContains(t, cfg.Validate(), "tui.artwork_renderer")
}

func TestLibraryMaxArchiveMemberSizeValidation(t *testing.T) {
	for _, value := range []ByteSize{1, 512 * 1000 * 1000, 512 * 1024 * 1024} {
		t.Run(value.String(), func(t *testing.T) {
			cfg := Default()
			cfg.Library.MaxArchiveMemberSize = value
			assert.NoError(t, cfg.Validate())
		})
	}

	for _, value := range []ByteSize{0, -1} {
		t.Run(value.String(), func(t *testing.T) {
			cfg := Default()
			cfg.Library.MaxArchiveMemberSize = value
			assert.ErrorContains(t, cfg.Validate(), "library.max_archive_member_size")
		})
	}
}

func TestWriteAndRead(t *testing.T) {
	tempDir := t.TempDir()
	tmpFilePath := filepath.Join(tempDir, "config.toml")

	err := WriteDefault(tmpFilePath, false)
	require.NoError(t, err)

	loaded, err := Load(tmpFilePath)
	require.NoError(t, err)
	assert.Equal(t, Default(), loaded)
}

func TestWriteDefaultIncludesLibraryConfig(t *testing.T) {
	tempDir := t.TempDir()
	tmpFilePath := filepath.Join(tempDir, "config.toml")

	err := WriteDefault(tmpFilePath, false)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpFilePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "[library]")
	assert.Contains(t, string(data), `max_archive_member_size = '512MiB'`)
	assert.Contains(t, string(data), "[ipc]")
	assert.Contains(t, string(data), "single_instance = 'auto'")
}

func TestDefaultPath(t *testing.T) {
	path, err := DefaultPath()
	assert.NoError(t, err)
	assert.Contains(t, path, "tmus")
	assert.Contains(t, path, "config.toml")
}
