package archive

import (
	"io"
	"path/filepath"
	"testing"

	_ "github.com/bpicode/tmus/testing"
	"github.com/stretchr/testify/assert"
)

func TestArchiveHandlers(t *testing.T) {
	testArchives := []string{
		"test-archive.zip",
		"test-archive.tar.gz",
		"test-archive.tar.xz",
		"test-archive.7z",
		"test-archive.rar",
	}
	registry := DefaultRegistry()

	for _, testArchive := range testArchives {
		t.Run(testArchive, func(t *testing.T) {
			p := filepath.Join("testdata", testArchive)
			h := registry.FindHandler(p)
			assert.NotNil(t, h)

			l, err := h.List(p, false)
			assert.NoError(t, err)
			assert.Len(t, l, 1)

			dir := l[0]
			assert.True(t, dir.IsDir)

			l, err = h.List(dir.Path, false)
			assert.NoError(t, err)
			assert.Len(t, l, 1)

			file := l[0]
			assert.False(t, file.IsDir)
			reader, err := h.Open(file.Path)
			assert.NoError(t, err)
			defer reader.Close()
			bytes, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, "dummy-music-data\n", string(bytes))
		})
	}
}
