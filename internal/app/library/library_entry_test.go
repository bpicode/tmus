package library

import (
	"archive/zip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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
		want EntryType
	}{
		{name: "audio", path: filepath.Join(dir, "song.mp3"), want: EntryMP3},
		{name: "url", path: filepath.Join(dir, "station.url"), want: EntryStream},
		{name: "stream", path: filepath.Join(dir, "radio.stream"), want: EntryStream},
		{name: "archive", path: filepath.Join(dir, "pack.zip"), want: EntryArchive},
		{name: "unsupported", path: filepath.Join(dir, "notes.txt"), want: EntryOther},
		{name: "remote", path: "https://example.com/live.mp3", want: EntryStream},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := EntryFromPath(tt.path)
			if err != nil {
				t.Fatalf("EntryFromPath() error = %v", err)
			}
			if entry.Type() != tt.want {
				t.Fatalf("entry type = %v, want %v", entry.Type(), tt.want)
			}
			if entry.Path() != tt.path {
				t.Fatalf("entry path = %q, want %q", entry.Path(), tt.path)
			}
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
	archiveRoot, ok := OpenArchiveRoot(archivePath)
	if !ok {
		t.Fatal("OpenArchiveRoot() did not recognize zip archive")
	}
	songPath := BuildArchivePath("zip", archivePath, "folder/song.mp3")
	streamPath := BuildArchivePath("zip", archivePath, "radio.stream")

	tests := []struct {
		name string
		path string
		want EntryType
	}{
		{name: "archive root", path: archiveRoot, want: EntryArchive},
		{name: "archive audio", path: songPath, want: EntryMP3},
		{name: "archive stream", path: streamPath, want: EntryStream},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := EntryFromPath(tt.path)
			if err != nil {
				t.Fatalf("EntryFromPath() error = %v", err)
			}
			if entry.Type() != tt.want {
				t.Fatalf("entry type = %v, want %v", entry.Type(), tt.want)
			}
			if entry.Path() != tt.path && tt.path != archiveRoot {
				t.Fatalf("entry path = %q, want %q", entry.Path(), tt.path)
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
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if got, want := entryNames(entries), []string{"A-dir", "z-dir", ".hidden.mp3", "A.flac", "b.mp3", "pack.zip", "radio.stream"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry names = %v, want %v", got, want)
	}
	if got, want := entryTypes(entries), []EntryType{EntryDir, EntryDir, EntryMP3, EntryFLAC, EntryMP3, EntryArchive, EntryStream}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry types = %v, want %v", got, want)
	}
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

	archiveRoot, ok := OpenArchiveRoot(archivePath)
	if !ok {
		t.Fatal("OpenArchiveRoot() did not recognize zip archive")
	}
	entries, err := List(archiveRoot)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if got, want := entryNames(entries), []string{"A-dir", "z-dir", "A.flac", "b.mp3", "radio.stream", "station.url"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry names = %v, want %v", got, want)
	}
	if got, want := entryTypes(entries), []EntryType{EntryDir, EntryDir, EntryFLAC, EntryMP3, EntryStream, EntryStream}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry types = %v, want %v", got, want)
	}
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

	archiveRoot, ok := OpenArchiveRoot(archivePath)
	if !ok {
		t.Fatal("OpenArchiveRoot() did not recognize zip archive")
	}

	entries, err := List(dir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	entryByName := mapEntriesByName(entries)

	archiveEntries, err := List(archiveRoot)
	if err != nil {
		t.Fatalf("List() archive error = %v", err)
	}
	archiveEntryByName := mapEntriesByName(archiveEntries)

	tests := []struct {
		name        string
		entry       Entry
		wantDir     bool
		wantArchive bool
		wantBrowse  bool
		wantPath    string
	}{
		{name: "directory", entry: entryByName["albums"], wantDir: true, wantBrowse: true, wantPath: filepath.Join(dir, "albums")},
		{name: "archive file", entry: entryByName["music.zip"], wantArchive: true, wantBrowse: true, wantPath: archiveRoot},
		{name: "archive directory", entry: archiveEntryByName["folder"], wantDir: true, wantBrowse: true, wantPath: BuildArchivePath("zip", archivePath, "folder")},
		{name: "audio file", entry: entryByName["song.mp3"]},
		{name: "archive audio", entry: archiveEntryByName["root.mp3"]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !assert.NotNil(t, tt.entry) {
				return
			}
			assert.Equal(t, tt.wantDir, tt.entry.IsDir())
			assert.Equal(t, tt.wantArchive, tt.entry.IsArchive())
			assert.Equal(t, tt.wantBrowse, tt.entry.CanBrowse())

			gotPath, gotOK := tt.entry.BrowsePath()
			assert.Equal(t, tt.wantBrowse, gotOK)
			assert.Equal(t, tt.wantPath, gotPath)
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
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer source.Reader.Close()
	if source.Format != FormatMP3 {
		t.Fatalf("Format = %v, want %v", source.Format, FormatMP3)
	}
	data, err := io.ReadAll(source.Reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got, want := string(data), "audio data"; got != want {
		t.Fatalf("stream data = %q, want %q", got, want)
	}
}

func TestArchiveEntryOpenURLShortcut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/flac")
		_, _ = w.Write([]byte("flac data"))
	}))
	defer server.Close()

	entry := archivedShortcutEntry(t, "station.url", "[InternetShortcut]\nURL="+server.URL+"\n")
	source, err := entry.Open(context.Background())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer source.Reader.Close()
	if source.Format != FormatFLAC {
		t.Fatalf("Format = %v, want %v", source.Format, FormatFLAC)
	}
	data, err := io.ReadAll(source.Reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got, want := string(data), "flac data"; got != want {
		t.Fatalf("stream data = %q, want %q", got, want)
	}
}

func archivedShortcutEntry(t *testing.T, name, content string) Entry {
	t.Helper()
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "shortcuts.zip")
	createZip(t, archivePath, map[string]string{name: content})
	archiveRoot, ok := OpenArchiveRoot(archivePath)
	if !ok {
		t.Fatal("OpenArchiveRoot() did not recognize zip archive")
	}
	entries, err := List(archiveRoot)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Type() != EntryStream {
		t.Fatalf("entry type = %v, want %v", entries[0].Type(), EntryStream)
	}
	return entries[0]
}

func entryNames(entries []Entry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func entryTypes(entries []Entry) []EntryType {
	types := make([]EntryType, 0, len(entries))
	for _, entry := range entries {
		types = append(types, entry.Type())
	}
	return types
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
