package library

import (
	"os"
	"path/filepath"
)

const defaultMaxArchiveMemberBytes int64 = 512 * 1024 * 1024

// Options configures Library behavior.
type Options struct {
	// MaxArchiveMemberBytes is the maximum decoded size for one archive member.
	MaxArchiveMemberBytes int64
}

// DefaultOptions returns the default library options.
func DefaultOptions() Options {
	return Options{
		MaxArchiveMemberBytes: defaultMaxArchiveMemberBytes,
	}
}

// Library provides browsing, entry reconstruction, and metadata reads.
type Library struct {
	opts    Options
	archive *archiveRegistry
}

var defaultLibrary = New(DefaultOptions())

// New constructs a Library with the given options.
// A non-positive MaxArchiveMemberBytes value is replaced with the default.
func New(opts Options) *Library {
	if opts.MaxArchiveMemberBytes <= 0 {
		opts.MaxArchiveMemberBytes = defaultMaxArchiveMemberBytes
	}
	return &Library{
		opts:    opts,
		archive: archiveHandlers(),
	}
}

// List returns browsable child entries for a local directory or archive path.
func List(path string) ([]Entry, error) {
	return defaultLibrary.List(path)
}

// List returns browsable child entries for a local directory or archive path.
func (l *Library) List(path string) ([]Entry, error) {
	if handler := l.archive.findHandler(path); handler != nil {
		return l.listArchive(handler, path)
	}
	return l.listDir(path)
}

// EntryFromPath reconstructs an Entry from its durable path or URI.
//
// The returned entry is classified from the path only; local files and archive
// members are opened or listed later by Entry.Open or List.
func EntryFromPath(path string) (Entry, error) {
	return defaultLibrary.EntryFromPath(path)
}

// EntryFromPath reconstructs an Entry from its durable path or URI.
//
// The returned entry is classified from the path only; local files and archive
// members are opened or listed later by Entry.Open or List.
func (l *Library) EntryFromPath(path string) (Entry, error) {
	if isRemote(path) {
		return remoteEntry{path: path, name: baseName(path)}, nil
	}
	if isArchivePath(path) {
		return l.archiveEntryFromPath(path)
	}
	return l.localEntryFromPath(path, false), nil
}

func (l *Library) listDir(path string) ([]Entry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		entryPath := filepath.Join(path, name)
		item := l.localEntryFromPath(entryPath, entry.IsDir())
		if includeEntry(item) {
			items = append(items, item)
		}
	}
	sortEntries(items)
	return items, nil
}

func (l *Library) listArchive(handler archiveHandler, path string) ([]Entry, error) {
	entries, err := handler.list(path, true)
	if err != nil {
		return nil, err
	}
	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		entry = l.withLibrary(entry)
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

func (l *Library) localEntryFromPath(path string, isDir bool) Entry {
	name := filepath.Base(path)
	switch {
	case isDir:
		return dirEntry{path: path, name: name}
	case l.archive.findHandler(path) != nil:
		return archiveFile{lib: l, path: path, name: name}
	case isURLFile(path):
		return urlFile{path: path, name: name}
	case isStreamFile(path):
		return streamFile{path: path, name: name}
	default:
		return audioFile{path: path, name: name}
	}
}

func (l *Library) archiveEntryFromPath(value string) (Entry, error) {
	_, archivePath, inner, err := splitArchiveURI(value)
	if err != nil {
		return nil, err
	}
	if inner == "" {
		return archiveFile{lib: l, path: archivePath, name: filepath.Base(archivePath)}, nil
	}
	return archiveEntry{lib: l, path: value, name: baseName(value)}, nil
}

func (l *Library) withLibrary(entry Entry) Entry {
	switch e := entry.(type) {
	case archiveFile:
		e.lib = l
		return e
	case archiveEntry:
		e.lib = l
		return e
	default:
		return entry
	}
}
