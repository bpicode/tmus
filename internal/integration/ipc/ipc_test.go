package ipc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bpicode/tmus/internal/app/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTracksPreservesStreamShortcutPath(t *testing.T) {
	dir := t.TempDir()
	streamPath := filepath.Join(dir, "radio.stream")
	err := os.WriteFile(streamPath, []byte("https://example.com/radio.mp3\n"), 0644)
	require.NoError(t, err)

	tracks := buildTracks(nil, []string{streamPath})

	assert.Equal(t, []core.Track{
		{Name: "radio.stream", Path: streamPath},
	}, tracks)
}

func TestBuildTracksNormalizesRelativePath(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "song.mp3")
	err := os.WriteFile(audioPath, nil, 0644)
	require.NoError(t, err)

	t.Chdir(dir)
	tracks := buildTracks(nil, []string{"song.mp3"})

	assert.Equal(t, []core.Track{
		{Name: "song.mp3", Path: audioPath},
	}, tracks)
}
