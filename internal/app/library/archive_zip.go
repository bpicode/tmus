package library

import (
	"archive/zip"
	"fmt"
	"io"
	"maps"
	"path"
	"slices"
	"strings"
)

type zipHandler struct{}

func newZipHandler() archiveHandler {
	return &zipHandler{}
}

func (h *zipHandler) scheme() string {
	return "zip"
}

func (h *zipHandler) isArchivePath(value string) bool {
	if strings.HasPrefix(value, "arch://zip:") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(value), ".zip")
}

func (h *zipHandler) list(value string, showHidden bool) ([]Entry, error) {
	archivePath, inner, err := splitArchivePath(h.scheme(), value)
	if err != nil {
		return nil, err
	}

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
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
		if hasMore || strings.HasSuffix(f.Name, "/") {
			archivePath := buildArchivePath(h.scheme(), archivePath, strings.TrimSuffix(entryPath, "/"))
			children[child] = archiveEntry{
				name:  child,
				path:  archivePath,
				isDir: true,
			}
		} else {
			archivePath := buildArchivePath(h.scheme(), archivePath, entryPath)
			children[child] = archiveEntry{
				name: child,
				path: archivePath,
			}
		}
	}

	return slices.Collect(maps.Values(children)), nil
}

func (h *zipHandler) open(value string) (io.ReadCloser, error) {
	archivePath, inner, err := splitArchivePath(h.scheme(), value)
	if err != nil {
		return nil, err
	}
	if inner == "" {
		return nil, fmt.Errorf("zip path missing entry")
	}

	//goland:noinspection GoResourceLeak
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	var entry *zip.File
	for _, f := range reader.File {
		if f.Name == inner {
			entry = f
			break
		}
	}
	if entry == nil {
		_ = reader.Close()
		return nil, fmt.Errorf("zip entry not found: %s", inner)
	}

	rc, err := entry.Open()
	if err != nil {
		_ = reader.Close()
		return nil, fmt.Errorf("open zip entry: %w", err)
	}

	return &archiveReadCloser{ReadCloser: rc, closer: reader.Close}, nil
}

type archiveReadCloser struct {
	io.ReadCloser
	closer func() error
}

func (a *archiveReadCloser) Close() error {
	err := a.ReadCloser.Close()
	if a.closer != nil {
		if cerr := a.closer(); err == nil {
			err = cerr
		}
	}
	return err
}
