package library

import (
	"os"
	"path/filepath"
)

func List2(path string) ([]Entry2, error) {
	if handler := DefaultArchiveRegistry().FindHandler(path); handler != nil {
		return listArchive2(handler, path)
	}
	return listDir2(path)
}

func listDir2(path string) ([]Entry2, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	items := make([]Entry2, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		entryPath := filepath.Join(path, name)
		var item Entry2
		if entry.IsDir() {
			item = dirEntry{path: entryPath, name: name}
		} else if DefaultArchiveRegistry().FindHandler(entryPath) != nil {
			item = archiveFile{path: entryPath, name: name}
		} else if isURLFile(entryPath) {
			item = urlFile{path: entryPath, name: name}
		} else if isStreamFile(entryPath) {
			item = streamFile{path: entryPath, name: name}
		} else {
			item = audioFile{path: entryPath, name: name}
		}
		if includeEntry2(item) {
			items = append(items, item)
		}
	}
	sortEntries2(items)
	return items, nil
}

func listArchive2(handler ArchiveHandler, path string) ([]Entry2, error) {
	entries, err := handler.List(path, true)
	if err != nil {
		return nil, err
	}
	items := make([]Entry2, 0, len(entries))
	for _, entry := range entries {
		item := archiveEntry{
			path:  entry.Path,
			name:  entry.Name,
			isDir: entry.IsDir,
		}
		if includeEntry2(item) {
			items = append(items, item)
		}
	}
	sortEntries2(items)
	return items, nil
}

func includeEntry2(entry Entry2) bool {
	return entry.Type() == EntryDir || entry.Type() == EntryArchive || entry.IsAudio()
}
