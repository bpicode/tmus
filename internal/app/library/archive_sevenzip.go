package library

import (
	"fmt"
	"io"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/bodgit/sevenzip"
)

type sevenZipHandler struct{}

func newSevenZipHandler() archiveHandler {
	return &sevenZipHandler{}
}

func (h *sevenZipHandler) scheme() string {
	return "7z"
}

func (h *sevenZipHandler) isArchivePath(value string) bool {
	if strings.HasPrefix(value, "arch://7z:") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(value), ".7z")
}

func (h *sevenZipHandler) list(value string, showHidden bool) ([]Entry, error) {
	archivePath, inner, err := splitArchivePath(h.scheme(), value)
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
			var ok bool
			name, ok = strings.CutPrefix(name, inner)
			if !ok {
				continue
			}
		}
		if name == "" {
			continue
		}
		child, _, hasMore := strings.Cut(name, "/")
		if child == "" {
			continue
		}
		if !showHidden && strings.HasPrefix(child, ".") {
			continue
		}
		entryPath := path.Join(inner, child)
		if hasMore || f.FileInfo().IsDir() {
			path := buildArchivePath(h.scheme(), archivePath, strings.TrimSuffix(entryPath, "/"))
			children[child] = archiveEntry{
				name:  child,
				path:  path,
				isDir: true,
			}
		} else {
			path := buildArchivePath(h.scheme(), archivePath, entryPath)
			children[child] = archiveEntry{
				name: child,
				path: path,
			}
		}
	}

	return slices.Collect(maps.Values(children)), nil
}

func (h *sevenZipHandler) open(value string) (io.ReadCloser, error) {
	archivePath, inner, err := splitArchivePath(h.scheme(), value)
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
