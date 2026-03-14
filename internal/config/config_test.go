package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigHasNoErrors(t *testing.T) {
	cfg := Default()
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestWriteAndRead(t *testing.T) {
	tempDir := t.TempDir()
	tmpFilePath := filepath.Join(tempDir, "config.toml")

	err := WriteDefault(tmpFilePath, false)
	assert.NoError(t, err)

	loaded, err := Load(tmpFilePath)
	assert.NoError(t, err)
	assert.Equal(t, Default(), loaded)
}

func TestDefaultPath(t *testing.T) {
	path, err := DefaultPath()
	assert.NoError(t, err)
	assert.Contains(t, path, "tmus")
	assert.Contains(t, path, "config.toml")
}
