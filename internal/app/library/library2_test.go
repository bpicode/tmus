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
)

func TestList2FiltersAndSortsDirectoryEntries(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "z-dir"))
	mustMkdir(t, filepath.Join(dir, "A-dir"))
	mustWriteFile(t, filepath.Join(dir, "notes.txt"), "not audio")
	mustWriteFile(t, filepath.Join(dir, "b.mp3"), "")
	mustWriteFile(t, filepath.Join(dir, "A.flac"), "")
	mustWriteFile(t, filepath.Join(dir, "radio.stream"), "https://example.com/radio.pls\n")
	mustWriteFile(t, filepath.Join(dir, ".hidden.mp3"), "")
	mustWriteFile(t, filepath.Join(dir, "pack.zip"), "")

	entries, err := List2(dir)
	if err != nil {
		t.Fatalf("List2() error = %v", err)
	}

	if got, want := entry2Names(entries), []string{"A-dir", "z-dir", ".hidden.mp3", "A.flac", "b.mp3", "pack.zip", "radio.stream"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry names = %v, want %v", got, want)
	}
	if got, want := entry2Types(entries), []EntryType{EntryDir, EntryDir, EntryMP3, EntryFLAC, EntryMP3, EntryArchive, EntryURL}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry types = %v, want %v", got, want)
	}
}

func TestList2FiltersAndSortsArchiveEntries(t *testing.T) {
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
	entries, err := List2(archiveRoot)
	if err != nil {
		t.Fatalf("List2() error = %v", err)
	}

	if got, want := entry2Names(entries), []string{"A-dir", "z-dir", "A.flac", "b.mp3", "radio.stream", "station.url"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry names = %v, want %v", got, want)
	}
	if got, want := entry2Types(entries), []EntryType{EntryDir, EntryDir, EntryFLAC, EntryMP3, EntryURL, EntryURL}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry types = %v, want %v", got, want)
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

func archivedShortcutEntry(t *testing.T, name, content string) Entry2 {
	t.Helper()
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "shortcuts.zip")
	createZip(t, archivePath, map[string]string{name: content})
	archiveRoot, ok := OpenArchiveRoot(archivePath)
	if !ok {
		t.Fatal("OpenArchiveRoot() did not recognize zip archive")
	}
	entries, err := List2(archiveRoot)
	if err != nil {
		t.Fatalf("List2() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Type() != EntryURL {
		t.Fatalf("entry type = %v, want %v", entries[0].Type(), EntryURL)
	}
	return entries[0]
}

func entry2Names(entries []Entry2) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func entry2Types(entries []Entry2) []EntryType {
	types := make([]EntryType, 0, len(entries))
	for _, entry := range entries {
		types = append(types, entry.Type())
	}
	return types
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
