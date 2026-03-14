package archive

import (
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/bodgit/sevenzip"
)

type sevenZipHandler struct{}

func NewSevenZipHandler() Handler {
	return &sevenZipHandler{}
}

func (h *sevenZipHandler) Scheme() string {
	return "7z"
}

func (h *sevenZipHandler) IsArchivePath(value string) bool {
	if strings.HasPrefix(value, "arch://7z:") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(value), ".7z")
}

func (h *sevenZipHandler) List(value string, showHidden bool) ([]Entry, error) {
	archivePath, inner, err := splitArchivePath(h.Scheme(), value)
	if err != nil {
		return nil, err
	}

	reader, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open 7z: %w", err)
	}
	defer reader.Close()

	if inner != "" && !strings.HasSuffix(inner, "/") {
		inner += "/"
	}

	children := map[string]Entry{}
	for _, f := range reader.File {
		name := f.Name
		if inner != "" {
			if !strings.HasPrefix(name, inner) {
				continue
			}
			name = strings.TrimPrefix(name, inner)
		}
		if name == "" {
			continue
		}
		parts := strings.Split(name, "/")
		child := parts[0]
		if child == "" {
			continue
		}
		if !showHidden && strings.HasPrefix(child, ".") {
			continue
		}
		entryPath := path.Join(inner, child)
		if len(parts) > 1 || f.FileInfo().IsDir() {
			children[child] = Entry{
				Name:  child,
				Path:  BuildPath(h.Scheme(), archivePath, strings.TrimSuffix(entryPath, "/")),
				IsDir: true,
			}
		} else {
			children[child] = Entry{
				Name:  child,
				Path:  BuildPath(h.Scheme(), archivePath, entryPath),
				IsDir: false,
			}
		}
	}

	entries := make([]Entry, 0, len(children))
	for _, v := range children {
		entries = append(entries, v)
	}
	sortEntries(entries)
	return entries, nil
}

func (h *sevenZipHandler) Open(value string) (io.ReadCloser, error) {
	archivePath, inner, err := splitArchivePath(h.Scheme(), value)
	if err != nil {
		return nil, err
	}
	if inner == "" {
		return nil, fmt.Errorf("7z path missing entry")
	}

	reader, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open 7z: %w", err)
	}

	var entry *sevenzip.File
	for _, f := range reader.File {
		if f.Name == inner {
			entry = f
			break
		}
	}
	if entry == nil {
		_ = reader.Close()
		return nil, fmt.Errorf("7z entry not found: %s", inner)
	}
	if entry.FileInfo().IsDir() {
		_ = reader.Close()
		return nil, fmt.Errorf("7z entry is a directory: %s", inner)
	}

	rc, err := entry.Open()
	if err != nil {
		_ = reader.Close()
		return nil, fmt.Errorf("open 7z entry: %w", err)
	}

	return &archiveReadCloser{ReadCloser: rc, closer: reader.Close}, nil
}
