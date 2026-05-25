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
		if entry.IsDir() {
			items = append(items, dirEntry{path: entryPath, name: name})
		} else if DefaultArchiveRegistry().FindHandler(entryPath) != nil {
			items = append(items, archiveFile{path: entryPath, name: name})
		} else if isStreamShortcut(entryPath) {
			items = append(items, streamFile{path: entryPath, name: name})
		} else {
			items = append(items, audioFile{path: entryPath, name: name})
		}
	}
	return items, nil
}

func listArchive2(handler ArchiveHandler, path string) ([]Entry2, error) {
	entries, err := handler.List(path, true)
	if err != nil {
		return nil, err
	}
	items := make([]Entry2, 0, len(entries))
	for _, entry := range entries {
		items = append(items, archiveEntry{
			path:  entry.Path,
			name:  entry.Name,
			isDir: entry.IsDir,
		})
	}
	return items, nil
}
