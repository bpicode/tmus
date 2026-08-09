package core

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/bpicode/tmus/internal/app/library"
	"github.com/bpicode/tmus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppMetadataUsesConfiguredLibraryLimit(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "limited.zip")
	createZip(t, archivePath, map[string]string{"song.mp3": "abcdef"})
	entryPath := "arch://zip:" + archivePath + "::song.mp3"

	cfg := config.Default()
	cfg.Library.MaxArchiveMemberSize = 5
	app := New(cfg)
	t.Cleanup(app.ShutdownAndWait)

	_, err := app.readMetadataExtendedCached(entryPath)
	assert.ErrorIs(t, err, library.ErrArchiveMemberTooLarge)
}

func createZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, file.Close())
	}()

	writer := zip.NewWriter(file)
	defer func() {
		require.NoError(t, writer.Close())
	}()
	for name, content := range files {
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write([]byte(content))
		require.NoError(t, err)
	}
}
