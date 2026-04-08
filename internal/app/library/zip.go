package library

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"strings"
)

type zipHandler struct{}

func NewZipHandler() ArchiveHandler {
	return &zipHandler{}
}

func (h *zipHandler) Scheme() string {
	return "zip"
}

func (h *zipHandler) IsArchivePath(value string) bool {
	if strings.HasPrefix(value, "arch://zip:") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(value), ".zip")
}

func (h *zipHandler) List(value string, showHidden bool) ([]ArchiveEntry, error) {
	archivePath, inner, err := splitArchivePath(h.Scheme(), value)
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

	children := map[string]ArchiveEntry{}
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
		if len(parts) > 1 || strings.HasSuffix(f.Name, "/") {
			children[child] = ArchiveEntry{
				Name:  child,
				Path:  BuildArchivePath(h.Scheme(), archivePath, strings.TrimSuffix(entryPath, "/")),
				IsDir: true,
			}
		} else {
			children[child] = ArchiveEntry{
				Name:  child,
				Path:  BuildArchivePath(h.Scheme(), archivePath, entryPath),
				IsDir: false,
			}
		}
	}

	entries := make([]ArchiveEntry, 0, len(children))
	for _, v := range children {
		entries = append(entries, v)
	}

	sortArchiveEntries(entries)
	return entries, nil
}

func (h *zipHandler) Open(value string) (io.ReadCloser, error) {
	archivePath, inner, err := splitArchivePath(h.Scheme(), value)
	if err != nil {
		return nil, err
	}
	if inner == "" {
		return nil, fmt.Errorf("zip path missing entry")
	}

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
