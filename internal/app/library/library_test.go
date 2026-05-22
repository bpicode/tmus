package library

import (
	"archive/zip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntryFromPathLocalEntries(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "song.mp3"), "")
	mustWriteFile(t, filepath.Join(dir, "station.url"), "[InternetShortcut]\nURL=https://example.com/station.pls\n")
	mustWriteFile(t, filepath.Join(dir, "radio.stream"), "https://example.com/radio.pls\n")
	mustWriteFile(t, filepath.Join(dir, "notes.txt"), "not audio")
	mustWriteFile(t, filepath.Join(dir, "pack.zip"), "")

	tests := []struct {
		name string
		path string
		want entryType
	}{
		{name: "audio", path: filepath.Join(dir, "song.mp3"), want: entryMP3},
		{name: "url", path: filepath.Join(dir, "station.url"), want: entryStream},
		{name: "stream", path: filepath.Join(dir, "radio.stream"), want: entryStream},
		{name: "archive", path: filepath.Join(dir, "pack.zip"), want: entryArchive},
		{name: "unsupported", path: filepath.Join(dir, "notes.txt"), want: entryOther},
		{name: "remote", path: "https://example.com/live.mp3", want: entryStream},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := EntryFromPath(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, testEntryType(t, entry))
			assert.Equal(t, tt.path, entry.Path())
		})
	}
}

func TestEntryFromPathArchiveEntries(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "music.zip")
	createZip(t, archivePath, map[string]string{
		"folder/song.mp3": "",
		"radio.stream":    "https://example.com/radio.pls\n",
	})
	archiveRoot := archiveBrowsePath(t, archivePath)
	songPath := buildArchivePath("zip", archivePath, "folder/song.mp3")
	streamPath := buildArchivePath("zip", archivePath, "radio.stream")

	tests := []struct {
		name string
		path string
		want entryType
	}{
		{name: "archive root", path: archiveRoot, want: entryArchive},
		{name: "archive audio", path: songPath, want: entryMP3},
		{name: "archive stream", path: streamPath, want: entryStream},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := EntryFromPath(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, testEntryType(t, entry))
			if tt.path != archiveRoot {
				assert.Equal(t, tt.path, entry.Path())
			}
		})
	}
}

func TestListFiltersAndSortsDirectoryEntries(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "z-dir"))
	mustMkdir(t, filepath.Join(dir, "A-dir"))
	mustWriteFile(t, filepath.Join(dir, "notes.txt"), "not audio")
	mustWriteFile(t, filepath.Join(dir, "b.mp3"), "")
	mustWriteFile(t, filepath.Join(dir, "A.flac"), "")
	mustWriteFile(t, filepath.Join(dir, "radio.stream"), "https://example.com/radio.pls\n")
	mustWriteFile(t, filepath.Join(dir, ".hidden.mp3"), "")
	mustWriteFile(t, filepath.Join(dir, "pack.zip"), "")

	entries, err := List(dir)
	require.NoError(t, err)

	assert.Equal(t, []string{"A-dir", "z-dir", ".hidden.mp3", "A.flac", "b.mp3", "pack.zip", "radio.stream"}, entryNames(entries))
	assert.Equal(t, []entryType{entryDir, entryDir, entryMP3, entryFLAC, entryMP3, entryArchive, entryStream}, entryTypes(t, entries))
}

func TestListFiltersAndSortsArchiveEntries(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "music.zip")
	createZip(t, archivePath, map[string]string{
		"z-dir/song.mp3": "",
		"A-dir/song.mp3": "",
		"notes.txt":      "not audio",
		"b.mp3":          "",
		"A.flac":         "",
		"radio.stream":   "https://example.com/radio.pls\n",
		"station.url":    "[InternetShortcut]\nURL=https://example.com/station.pls\n",
	})

	archiveRoot := archiveBrowsePath(t, archivePath)
	entries, err := List(archiveRoot)
	require.NoError(t, err)

	assert.Equal(t, []string{"A-dir", "z-dir", "A.flac", "b.mp3", "radio.stream", "station.url"}, entryNames(entries))
	assert.Equal(t, []entryType{entryDir, entryDir, entryFLAC, entryMP3, entryStream, entryStream}, entryTypes(t, entries))
}

func TestEntryBrowseIntent(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "albums"))
	mustWriteFile(t, filepath.Join(dir, "song.mp3"), "")
	archivePath := filepath.Join(dir, "music.zip")
	createZip(t, archivePath, map[string]string{
		"folder/song.mp3": "",
		"root.mp3":        "",
	})

	archiveRoot := archiveBrowsePath(t, archivePath)

	entries, err := List(dir)
	require.NoError(t, err)
	entryByName := mapEntriesByName(entries)

	archiveEntries, err := List(archiveRoot)
	require.NoError(t, err)
	archiveEntryByName := mapEntriesByName(archiveEntries)
	remoteEntry, err := EntryFromPath("https://example.com/live.mp3")
	require.NoError(t, err)

	tests := []struct {
		name               string
		entry              Entry
		wantDir            bool
		wantArchive        bool
		wantBrowse         bool
		wantPath           string
		wantFilesystem     bool
		wantFilesystemPath string
	}{
		{name: "directory", entry: entryByName["albums"], wantDir: true, wantBrowse: true, wantPath: filepath.Join(dir, "albums"), wantFilesystem: true, wantFilesystemPath: filepath.Join(dir, "albums")},
		{name: "archive file", entry: entryByName["music.zip"], wantArchive: true, wantBrowse: true, wantPath: archiveRoot, wantFilesystem: true, wantFilesystemPath: archivePath},
		{name: "archive directory", entry: archiveEntryByName["folder"], wantDir: true, wantBrowse: true, wantPath: buildArchivePath("zip", archivePath, "folder")},
		{name: "audio file", entry: entryByName["song.mp3"], wantFilesystem: true, wantFilesystemPath: filepath.Join(dir, "song.mp3")},
		{name: "archive audio", entry: archiveEntryByName["root.mp3"]},
		{name: "remote audio", entry: remoteEntry},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, tt.entry)
			assert.Equal(t, tt.wantDir, tt.entry.IsDir())
			assert.Equal(t, tt.wantArchive, tt.entry.IsArchive())
			assert.Equal(t, tt.wantBrowse, tt.entry.CanBrowse())

			gotPath, gotOK := tt.entry.BrowsePath()
			assert.Equal(t, tt.wantBrowse, gotOK)
			assert.Equal(t, tt.wantPath, gotPath)

			gotFilesystemPath, gotFilesystemOK := tt.entry.FilesystemPath()
			assert.Equal(t, tt.wantFilesystem, gotFilesystemOK)
			assert.Equal(t, tt.wantFilesystemPath, gotFilesystemPath)
		})
	}
}

func TestArchiveEntryOpenStreamShortcut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("audio data"))
	}))
	defer server.Close()

	entry := archivedShortcutEntry(t, "radio.stream", server.URL+"\n")
	source, err := entry.Open(context.Background())
	require.NoError(t, err)
	defer source.Reader.Close()
	assert.Equal(t, FormatMP3, source.Format)
	data, err := io.ReadAll(source.Reader)
	require.NoError(t, err)
	assert.Equal(t, "audio data", string(data))
}

func TestArchiveEntryOpenURLShortcut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/flac")
		_, _ = w.Write([]byte("flac data"))
	}))
	defer server.Close()

	entry := archivedShortcutEntry(t, "station.url", "[InternetShortcut]\nURL="+server.URL+"\n")
	source, err := entry.Open(context.Background())
	require.NoError(t, err)
	defer source.Reader.Close()
	assert.Equal(t, FormatFLAC, source.Format)
	data, err := io.ReadAll(source.Reader)
	require.NoError(t, err)
	assert.Equal(t, "flac data", string(data))
}

func archivedShortcutEntry(t *testing.T, name, content string) Entry {
	t.Helper()
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "shortcuts.zip")
	createZip(t, archivePath, map[string]string{name: content})
	archiveRoot := archiveBrowsePath(t, archivePath)
	entries, err := List(archiveRoot)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, entryStream, testEntryType(t, entries[0]))
	return entries[0]
}

func archiveBrowsePath(t *testing.T, archivePath string) string {
	t.Helper()
	entry, err := EntryFromPath(archivePath)
	require.NoError(t, err)
	browsePath, ok := entry.BrowsePath()
	require.True(t, ok)
	return browsePath
}

func entryNames(entries []Entry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func entryTypes(t *testing.T, entries []Entry) []entryType {
	t.Helper()
	types := make([]entryType, 0, len(entries))
	for _, entry := range entries {
		types = append(types, testEntryType(t, entry))
	}
	return types
}

func testEntryType(t *testing.T, entry Entry) entryType {
	t.Helper()
	typed, ok := entry.(interface{ entryType() entryType })
	if !ok {
		t.Fatalf("entry %T has no entryType method", entry)
	}
	return typed.entryType()
}

func mapEntriesByName(entries []Entry) map[string]Entry {
	result := make(map[string]Entry, len(entries))
	for _, entry := range entries {
		result[entry.Name()] = entry
	}
	return result
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func createZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", path, err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("Create(%q) in zip error = %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("Write(%q) in zip error = %v", name, err)
		}
	}
}
