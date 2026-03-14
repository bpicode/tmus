package library

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bpicode/tmus/internal/app/archive"
)

// Entry represents a browsable item.
type Entry struct {
	Name      string
	Path      string
	IsDir     bool
	IsArchive bool
	IsAudio   bool
}

// List returns directory or archive entries filtered to audio files, archives, and directories.
func List(path string, showHidden bool) ([]Entry, error) {
	if handler := archive.DefaultRegistry().FindHandler(path); handler != nil {
		entries, err := handler.List(path, showHidden)
		if err != nil {
			return nil, err
		}
		items := make([]Entry, 0, len(entries))
		for _, entry := range entries {
			isAudio := !entry.IsDir && IsAudio(entry.Path)
			if !entry.IsDir && !isAudio {
				continue
			}
			items = append(items, Entry{
				Name:    entry.Name,
				Path:    entry.Path,
				IsDir:   entry.IsDir,
				IsAudio: isAudio,
			})
		}
		sortEntries(items)
		return items, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		fullPath := filepath.Join(path, name)
		if entry.IsDir() {
			items = append(items, Entry{
				Name:  name,
				Path:  fullPath,
				IsDir: true,
			})
			continue
		}

		if handler := archive.DefaultRegistry().FindHandler(fullPath); handler != nil {
			items = append(items, Entry{
				Name:      name,
				Path:      fullPath,
				IsArchive: true,
			})
			continue
		}

		if !IsAudio(fullPath) {
			continue
		}
		items = append(items, Entry{
			Name:    name,
			Path:    fullPath,
			IsAudio: true,
		})
	}

	sortEntries(items)
	return items, nil
}

// OpenArchiveRoot returns an archive URI for the root of the archive if the path is a supported archive file.
func OpenArchiveRoot(path string) (string, bool) {
	handler := archive.DefaultRegistry().FindHandler(path)
	if handler == nil {
		return "", false
	}
	return archive.BuildPath(handler.Scheme(), path, ""), true
}

func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}
