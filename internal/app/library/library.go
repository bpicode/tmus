package library

import (
	"os"
	"path/filepath"
)

// List returns browsable child entries for a local directory or archive path.
func List(path string) ([]Entry, error) {
	if handler := archiveHandlers().findHandler(path); handler != nil {
		return listArchive(handler, path)
	}
	return listDir(path)
}

// EntryFromPath reconstructs an Entry from its durable path or URI.
//
// The returned entry is classified from the path only; local files and archive
// members are opened or listed later by Entry.Open or List.
func EntryFromPath(path string) (Entry, error) {
	if isRemote(path) {
		return remoteEntry{path: path, name: baseName(path)}, nil
	}
	if isArchivePath(path) {
		return archiveEntryFromPath(path)
	}
	return localEntryFromPath(path, false), nil
}

func listDir(path string) ([]Entry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		entryPath := filepath.Join(path, name)
		item := localEntryFromPath(entryPath, entry.IsDir())
		if includeEntry(item) {
			items = append(items, item)
		}
	}
	sortEntries(items)
	return items, nil
}

func listArchive(handler archiveHandler, path string) ([]Entry, error) {
	entries, err := handler.list(path, true)
	if err != nil {
		return nil, err
	}
	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if includeEntry(entry) {
			items = append(items, entry)
		}
	}
	sortEntries(items)
	return items, nil
}

func includeEntry(entry Entry) bool {
	return entry.CanBrowse() || entry.IsAudio()
}

func localEntryFromPath(path string, isDir bool) Entry {
	name := filepath.Base(path)
	switch {
	case isDir:
		return dirEntry{path: path, name: name}
	case archiveHandlers().findHandler(path) != nil:
		return archiveFile{path: path, name: name}
	case isURLFile(path):
		return urlFile{path: path, name: name}
	case isStreamFile(path):
		return streamFile{path: path, name: name}
	default:
		return audioFile{path: path, name: name}
	}
}

func archiveEntryFromPath(value string) (Entry, error) {
	_, archivePath, inner, err := splitArchiveURI(value)
	if err != nil {
		return nil, err
	}
	if inner == "" {
		return archiveFile{path: archivePath, name: filepath.Base(archivePath)}, nil
	}
	return archiveEntry{path: value, name: baseName(value)}, nil
}
