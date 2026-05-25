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

func EntryFromPath(path string) (Entry2, error) {
	if IsRemote(path) {
		return remoteEntry{path: path, name: BaseName(path)}, nil
	}
	if IsArchivePath(path) {
		return archiveEntryFromPath(path)
	}
	return localEntryFromPath(path, false), nil
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
		item := localEntryFromPath(entryPath, entry.IsDir())
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

func localEntryFromPath(path string, isDir bool) Entry2 {
	name := filepath.Base(path)
	switch {
	case isDir:
		return dirEntry{path: path, name: name}
	case DefaultArchiveRegistry().FindHandler(path) != nil:
		return archiveFile{path: path, name: name}
	case isURLFile(path):
		return urlFile{path: path, name: name}
	case isStreamFile(path):
		return streamFile{path: path, name: name}
	default:
		return audioFile{path: path, name: name}
	}
}

func archiveEntryFromPath(value string) (Entry2, error) {
	_, archivePath, inner, err := SplitArchivePath(value)
	if err != nil {
		return nil, err
	}
	if inner == "" {
		return archiveFile{path: archivePath, name: filepath.Base(archivePath)}, nil
	}
	return archiveEntry{path: value, name: BaseName(value)}, nil
}
