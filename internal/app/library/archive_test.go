package library

import (
	"io"
	"path/filepath"
	"testing"

	_ "github.com/bpicode/tmus/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveHandlers(t *testing.T) {
	testArchives := []string{
		"test-archive.zip",
		"test-archive.tar.gz",
		"test-archive.tar.xz",
		"test-archive.7z",
		"test-archive.rar",
	}
	registry := archiveHandlers()

	for _, testArchive := range testArchives {
		t.Run(testArchive, func(t *testing.T) {
			p := filepath.Join("testdata", testArchive)
			h := registry.findHandler(p)
			require.NotNil(t, h)

			l, err := h.list(p, false)
			require.NoError(t, err)
			require.Len(t, l, 1)

			dir := l[0]
			assert.True(t, dir.IsDir())

			l, err = h.list(dir.Path(), false)
			require.NoError(t, err)
			require.Len(t, l, 1)

			file := l[0]
			assert.False(t, file.IsDir())
			reader, err := h.open(file.Path())
			require.NoError(t, err)
			defer reader.Close()
			bytes, err := io.ReadAll(reader)
			require.NoError(t, err)
			assert.Equal(t, "dummy-music-data\n", string(bytes))
		})
	}
}
