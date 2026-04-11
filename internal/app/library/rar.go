package library

import (
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/mholt/archives"
)

type rarHandler struct{}

func NewRarHandler() ArchiveHandler {
	return &rarHandler{}
}

func (h *rarHandler) Scheme() string {
	return "rar"
}

func (h *rarHandler) IsArchivePath(value string) bool {
	if strings.HasPrefix(value, "arch://rar:") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(value), ".rar")
}

func (h *rarHandler) List(value string, showHidden bool) ([]Entry, error) {
	archivePath, inner, err := splitArchivePath(h.Scheme(), value)
	if err != nil {
		return nil, err
	}

	fsys := &archives.ArchiveFS{Path: archivePath, Format: archives.Rar{}}
	dir := inner
	if dir == "" {
		dir = "."
	}
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read rar: %w", err)
	}

	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		entryPath := name
		if inner != "" {
			entryPath = path.Join(inner, name)
		}
		path := BuildArchivePath(h.Scheme(), archivePath, entryPath)
		items = append(items, Entry{
			Name:    name,
			Path:    path,
			IsDir:   entry.IsDir(),
			IsAudio: IsAudio(path),
		})
	}

	return items, nil
}

func (h *rarHandler) Open(value string) (io.ReadCloser, error) {
	archivePath, inner, err := splitArchivePath(h.Scheme(), value)
	if err != nil {
		return nil, err
	}
	if inner == "" {
		return nil, fmt.Errorf("rar path missing entry")
	}

	fsys := &archives.ArchiveFS{Path: archivePath, Format: archives.Rar{}}
	file, err := fsys.Open(inner)
	if err != nil {
		return nil, fmt.Errorf("open rar entry: %w", err)
	}
	info, err := file.Stat()
	if err == nil && info.IsDir() {
		_ = file.Close()
		return nil, fmt.Errorf("rar entry is a directory: %s", inner)
	}
	return file, nil
}
