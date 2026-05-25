package library

import (
	"archive/zip"
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
	})

	archiveRoot, ok := OpenArchiveRoot(archivePath)
	if !ok {
		t.Fatal("OpenArchiveRoot() did not recognize zip archive")
	}
	entries, err := List2(archiveRoot)
	if err != nil {
		t.Fatalf("List2() error = %v", err)
	}

	if got, want := entry2Names(entries), []string{"A-dir", "z-dir", "A.flac", "b.mp3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry names = %v, want %v", got, want)
	}
	if got, want := entry2Types(entries), []EntryType{EntryDir, EntryDir, EntryFLAC, EntryMP3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entry types = %v, want %v", got, want)
	}
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
