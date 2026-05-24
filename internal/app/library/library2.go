package library

import (
	"os"
	"path/filepath"
)

func List2(path string) ([]Entry2, error) {
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
		} else {
			items = append(items, audioFile{path: entryPath, name: name})
		}
	}
	return items, nil
}
