package library

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLibraryOpenArchiveEntryLimit(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "limited.zip")
	createZip(t, archivePath, map[string]string{"song.mp3": "abcdef"})
	entryPath := buildArchivePath("zip", archivePath, "song.mp3")

	tests := []struct {
		name    string
		limit   int64
		wantErr bool
	}{
		{name: "below limit", limit: 7},
		{name: "exact limit", limit: 6},
		{name: "over limit", limit: 5, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib := New(Options{MaxArchiveMemberBytes: tt.limit})
			rc, err := lib.openArchiveEntry(entryPath)
			require.NoError(t, err)
			defer rc.Close()

			got, err := io.ReadAll(rc)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrArchiveMemberTooLarge)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "abcdef", string(got))
		})
	}
}

func TestLibraryArchiveEntryLimitsAreIndependent(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "limited.zip")
	createZip(t, archivePath, map[string]string{"song.mp3": "abcdef"})
	entryPath := buildArchivePath("zip", archivePath, "song.mp3")

	small := New(Options{MaxArchiveMemberBytes: 5})
	large := New(Options{MaxArchiveMemberBytes: 6})

	rc, err := small.openArchiveEntry(entryPath)
	require.NoError(t, err)
	_, err = io.ReadAll(rc)
	assert.NoError(t, rc.Close())
	assert.ErrorIs(t, err, ErrArchiveMemberTooLarge)

	rc, err = large.openArchiveEntry(entryPath)
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	assert.NoError(t, rc.Close())
	require.NoError(t, err)
	assert.Equal(t, "abcdef", string(got))
}

func TestArchiveLimitAppliesToAudioMetadataAndShortcuts(t *testing.T) {
	t.Run("audio", func(t *testing.T) {
		lib, entryPath := limitedArchiveEntry(t, "song.mp3", "abcdef", 5)
		entry, err := lib.EntryFromPath(entryPath)
		require.NoError(t, err)

		_, err = entry.Open(context.Background())
		assert.ErrorIs(t, err, ErrArchiveMemberTooLarge)
	})

	t.Run("metadata", func(t *testing.T) {
		lib, entryPath := limitedArchiveEntry(t, "song.mp3", "abcdef", 5)

		_, err := lib.ReadMetadataExtended(entryPath)
		assert.ErrorIs(t, err, ErrArchiveMemberTooLarge)
	})

	t.Run("shortcut", func(t *testing.T) {
		lib, entryPath := limitedArchiveEntry(t, "radio.stream", "#skip\nhttp://example.com/live.mp3\n", 6)
		entry, err := lib.EntryFromPath(entryPath)
		require.NoError(t, err)

		_, err = entry.Open(context.Background())
		assert.ErrorIs(t, err, ErrArchiveMemberTooLarge)
	})
}

func limitedArchiveEntry(t *testing.T, name, content string, limit int64) (*Library, string) {
	t.Helper()
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "limited.zip")
	createZip(t, archivePath, map[string]string{name: content})
	return New(Options{MaxArchiveMemberBytes: limit}), buildArchivePath("zip", archivePath, name)
}
