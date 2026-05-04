package library

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalResolver handles local filesystem paths and archive URIs (arch://...).
type LocalResolver struct{}

// CanResolve returns true for any URI that is not an HTTP or HTTPS URL.
func (LocalResolver) CanResolve(uri string) bool {
	return !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://")
}

// Resolve opens a local file or archive entry.
// The returned Source.Reader also implements io.ReadSeeker for all local sources.
func (LocalResolver) Resolve(_ context.Context, uri string) (Source, error) {
	if handler := DefaultArchiveRegistry().FindHandler(uri); handler != nil {
		return resolveArchiveEntry(handler, uri)
	}
	f, err := os.Open(uri)
	if err != nil {
		return Source{}, fmt.Errorf("open file: %w", err)
	}
	return Source{Reader: f, Ext: strings.ToLower(filepath.Ext(uri))}, nil
}

// resolveArchiveEntry buffers the archive entry so it can be decoded
// as a seekable byte stream. The returned Reader also implements io.ReadSeeker.
func resolveArchiveEntry(handler ArchiveHandler, uri string) (Source, error) {
	rc, err := handler.Open(uri)
	if err != nil {
		return Source{}, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return Source{}, fmt.Errorf("read archive entry: %w", err)
	}

	ext := filepath.Ext(uri)
	if IsArchivePath(uri) {
		if inner := EntryExt(uri); inner != "" {
			ext = inner
		}
	}
	return Source{Reader: nopSeekCloser{bytes.NewReader(data)}, Ext: strings.ToLower(ext)}, nil
}

// nopSeekCloser wraps an io.ReadSeeker with a no-op Close.
type nopSeekCloser struct{ io.ReadSeeker }

func (nopSeekCloser) Close() error { return nil }
